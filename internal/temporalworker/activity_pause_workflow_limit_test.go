package temporalworker

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	workflowLimitRootName       = "WorkflowLimitRoot"
	workflowLimitChildName      = "WorkflowLimitChild"
	workflowLimitGrandchildName = "WorkflowLimitGrandchild"
)

type workflowLimitHeaderKey struct{}

type workflowLimitHeaderCapture struct {
	interceptor.WorkerInterceptorBase
}

func (i *workflowLimitHeaderCapture) InterceptActivity(ctx context.Context, next interceptor.ActivityInboundInterceptor) interceptor.ActivityInboundInterceptor {
	return &workflowLimitActivityCapture{ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next}}
}

type workflowLimitActivityCapture struct {
	interceptor.ActivityInboundInterceptorBase
}

func (i *workflowLimitActivityCapture) ExecuteActivity(ctx context.Context, in *interceptor.ExecuteActivityInput) (any, error) {
	ctx = context.WithValue(ctx, workflowLimitHeaderKey{}, interceptor.Header(ctx)[pauseAttemptsHeader])
	return i.Next.ExecuteActivity(ctx, in)
}

func workflowLimitRoot(ctx workflow.Context) (string, error) {
	ctx = WithStageActivityAttempts(ctx, 3)
	ctx = workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{WorkflowID: workflow.GetInfo(ctx).WorkflowExecution.ID + "/child"})
	var result string
	err := workflow.ExecuteChildWorkflow(ctx, workflowLimitChildName).Get(ctx, &result)
	return result, err
}

func workflowLimitChild(ctx workflow.Context) (string, error) {
	ctx = workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{WorkflowID: workflow.GetInfo(ctx).WorkflowExecution.ID + "/grandchild"})
	var result string
	err := workflow.ExecuteChildWorkflow(ctx, workflowLimitGrandchildName).Get(ctx, &result)
	return result, err
}

func workflowLimitGrandchild(ctx workflow.Context) (string, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{InitialInterval: 20 * time.Millisecond, MaximumInterval: 20 * time.Millisecond, BackoffCoefficient: 1, MaximumAttempts: 15},
	})
	var result string
	err := workflow.ExecuteActivity(ctx, "CreateTransaction").Get(ctx, &result)
	return result, err
}

func registerWorkflowLimitWorkflows(registry worker.WorkflowRegistry) {
	registry.RegisterWorkflowWithOptions(workflowLimitRoot, workflow.RegisterOptions{Name: workflowLimitRootName})
	registry.RegisterWorkflowWithOptions(workflowLimitChild, workflow.RegisterOptions{Name: workflowLimitChildName})
	registry.RegisterWorkflowWithOptions(workflowLimitGrandchild, workflow.RegisterOptions{Name: workflowLimitGrandchildName})
}

func TestWorkflowLimitNativePausePropagatesAndSurvivesWorkerChange(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	server, err := testsuite.StartDevServer(ctx, testsuite.DevServerOptions{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, server.Stop()) })
	c := server.Client()
	queue := "workflow-limit-test"
	var calls atomic.Int32
	var ready atomic.Bool
	type observation struct {
		limit   int
		attempt int32
		key     string
	}
	observations := make(chan observation, 20)
	newWorker := func(limit int) worker.Worker {
		w := worker.New(c, queue, worker.Options{Interceptors: []interceptor.WorkerInterceptor{&StagePauseInterceptor{Attempts: limit}, &workflowLimitHeaderCapture{}}})
		registerWorkflowLimitWorkflows(w)
		w.RegisterActivityWithOptions(func(ctx context.Context) (string, error) {
			var captured int
			payload, _ := ctx.Value(workflowLimitHeaderKey{}).(*commonpb.Payload)
			if payload == nil {
				return "", temporal.NewNonRetryableApplicationError("missing captured budget", "TEST_HEADER", nil)
			}
			if err := converter.GetDefaultDataConverter().FromPayload(payload, &captured); err != nil {
				return "", temporal.NewNonRetryableApplicationError("invalid captured budget", "TEST_HEADER", err)
			}
			info := activity.GetInfo(ctx)
			observations <- observation{captured, info.Attempt, info.WorkflowExecution.RunID + "/" + info.ActivityID}
			calls.Add(1)
			if ready.Load() {
				return "posted", nil
			}
			return "", temporal.NewApplicationError("synthetic insufficient funds", "INSUFFICIENT_FUND")
		}, activity.RegisterOptions{Name: "CreateTransaction"})
		return w
	}
	w := newWorker(15)
	require.NoError(t, w.Start())
	// Keep cleanup attached to the current worker when replacing it below.
	t.Cleanup(func() { w.Stop() })
	run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: queue, TaskQueue: queue}, workflowLimitRootName)
	require.NoError(t, err)
	grandchildID := run.GetID() + "/child/grandchild"
	var execution *commonpb.WorkflowExecution
	var activityID string
	waitForPause := func(expectedCalls int32) {
		t.Helper()
		require.Eventually(t, func() bool {
			d, err := c.DescribeWorkflowExecution(ctx, grandchildID, "")
			if err != nil || len(d.PendingActivities) != 1 {
				return false
			}
			pending := d.PendingActivities[0]
			if !pending.Paused || pending.LastFailure == nil {
				return false
			}
			if pending.GetPauseInfo().GetManual().GetReason() != "ACTIVITY_ATTEMPT_LIMIT:3" {
				return false
			}
			execution = d.WorkflowExecutionInfo.Execution
			activityID = pending.ActivityId
			return calls.Load() == expectedCalls
		}, 15*time.Second, 20*time.Millisecond)
		// Several retry intervals must pass without another business dispatch.
		require.Never(t, func() bool { return calls.Load() != expectedCalls }, 150*time.Millisecond, 20*time.Millisecond)
	}
	unpause := func() {
		t.Helper()
		_, err := c.WorkflowService().UnpauseActivity(ctx, &workflowservice.UnpauseActivityRequest{
			Namespace: "default", Execution: execution,
			Activity: &workflowservice.UnpauseActivityRequest_Id{Id: activityID}, ResetAttempts: true,
		})
		require.NoError(t, err)
	}
	waitForPause(3)
	originalExecution, originalActivityID := execution, activityID
	w.Stop()
	w = newWorker(0)
	require.NoError(t, w.Start())
	unpause()
	waitForPause(6)
	require.Equal(t, originalExecution.GetWorkflowId(), execution.GetWorkflowId())
	require.Equal(t, originalExecution.GetRunId(), execution.GetRunId())
	require.Equal(t, originalActivityID, activityID)
	ready.Store(true)
	unpause()
	var result string
	require.NoError(t, run.Get(ctx, &result))
	require.Equal(t, "posted", result)
	require.EqualValues(t, 7, calls.Load())
	for index := range 7 {
		got := <-observations
		require.Equal(t, 3, got.limit)
		require.EqualValues(t, index%3+1, got.attempt)
		require.Equal(t, originalExecution.GetRunId()+"/"+originalActivityID, got.key)
	}

	for _, id := range []string{run.GetID(), run.GetID() + "/child", grandchildID} {
		t.Run(id, func(t *testing.T) {
			history := &historypb.History{}
			iterator := c.GetWorkflowHistory(ctx, id, "", false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
			for iterator.HasNext() {
				event, err := iterator.Next()
				require.NoError(t, err)
				history.Events = append(history.Events, event)
			}
			var capturedHeaders, snapshots int
			for _, event := range history.Events {
				if started := event.GetWorkflowExecutionStartedEventAttributes(); started != nil && id != run.GetID() {
					var inherited int
					require.NotNil(t, started.Header.GetFields()[workflowAttemptsHeader])
					require.NoError(t, converter.GetDefaultDataConverter().FromPayload(started.Header.GetFields()[workflowAttemptsHeader], &inherited))
					require.Equal(t, 3, inherited)
				}
				if marker := event.GetMarkerRecordedEventAttributes(); marker != nil && marker.MarkerName == "SideEffect" {
					var captured int
					require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(marker.Details["data"], &captured))
					require.Equal(t, 3, captured)
					snapshots++
				}
				var payload *commonpb.Payload
				if child := event.GetStartChildWorkflowExecutionInitiatedEventAttributes(); child != nil {
					payload = child.Header.GetFields()[workflowAttemptsHeader]
				}
				if scheduled := event.GetActivityTaskScheduledEventAttributes(); scheduled != nil {
					payload = scheduled.Header.GetFields()[pauseAttemptsHeader]
					require.Zero(t, scheduled.RetryPolicy.MaximumAttempts)
				}
				if payload != nil {
					var captured int
					require.NoError(t, converter.GetDefaultDataConverter().FromPayload(payload, &captured))
					require.Equal(t, 3, captured)
					capturedHeaders++
				}
			}
			require.Equal(t, 1, capturedHeaders)
			if id == grandchildID {
				require.Equal(t, 1, snapshots)
			} else {
				require.Zero(t, snapshots)
			}
			for _, limit := range []int{0, 1, 15} {
				t.Run(fmt.Sprintf("worker-%d", limit), func(t *testing.T) {
					replayer, err := worker.NewWorkflowReplayerWithOptions(worker.WorkflowReplayerOptions{Interceptors: []interceptor.WorkerInterceptor{&StagePauseInterceptor{Attempts: limit}}})
					require.NoError(t, err)
					registerWorkflowLimitWorkflows(replayer)
					require.NoError(t, replayer.ReplayWorkflowHistoryWithOptions(nil, history, worker.ReplayWorkflowHistoryOptions{OriginalExecution: workflow.Execution{ID: id}}))
				})
			}
		})
	}
}

func TestWorkflowLimitDisabledWorkerFailsAfterConfiguredAttempts(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetTestTimeout(10 * time.Second)
	env.SetWorkerOptions(worker.Options{Interceptors: []interceptor.WorkerInterceptor{&StagePauseInterceptor{Attempts: 0}, &workflowLimitHeaderCapture{}}})
	var calls atomic.Int32
	var pauseHeaders atomic.Int32
	env.RegisterActivityWithOptions(func(ctx context.Context) (string, error) {
		calls.Add(1)
		if payload, _ := ctx.Value(workflowLimitHeaderKey{}).(*commonpb.Payload); payload != nil {
			pauseHeaders.Add(1)
		}
		return "", temporal.NewApplicationError("synthetic unavailable", "UNAVAILABLE")
	}, activity.RegisterOptions{Name: "CreateTransaction"})
	env.ExecuteWorkflow(func(ctx workflow.Context) (string, error) {
		return workflowLimitGrandchild(WithStageActivityAttempts(ctx, 3))
	})
	require.True(t, env.IsWorkflowCompleted())
	require.EqualValues(t, 3, calls.Load())
	require.Zero(t, pauseHeaders.Load())
	err := env.GetWorkflowError()
	require.Error(t, err)
	app, ok := errors.AsType[*temporal.ApplicationError](err)
	require.True(t, ok)
	require.Equal(t, "UNAVAILABLE", app.Type())
}

func TestWorkflowLimitNativeDisabledSnapshotReplaysWithPauseEnabled(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	server, err := testsuite.StartDevServer(ctx, testsuite.DevServerOptions{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, server.Stop()) })
	c := server.Client()
	queue := "workflow-limit-disabled-test"
	w := worker.New(c, queue, worker.Options{Interceptors: []interceptor.WorkerInterceptor{&StagePauseInterceptor{Attempts: 0}, &workflowLimitHeaderCapture{}}})
	registerWorkflowLimitWorkflows(w)
	var calls, pauseHeaders atomic.Int32
	w.RegisterActivityWithOptions(func(ctx context.Context) (string, error) {
		calls.Add(1)
		if payload, _ := ctx.Value(workflowLimitHeaderKey{}).(*commonpb.Payload); payload != nil {
			pauseHeaders.Add(1)
		}
		return "", temporal.NewApplicationError("synthetic unavailable", "UNAVAILABLE")
	}, activity.RegisterOptions{Name: "CreateTransaction"})
	require.NoError(t, w.Start())
	t.Cleanup(w.Stop)
	run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: queue, TaskQueue: queue}, workflowLimitRootName)
	require.NoError(t, err)
	err = run.Get(ctx, nil)
	require.Error(t, err)
	app, ok := errors.AsType[*temporal.ApplicationError](err)
	require.True(t, ok)
	require.Equal(t, "UNAVAILABLE", app.Type())
	require.EqualValues(t, 3, calls.Load())
	require.Zero(t, pauseHeaders.Load())
	for _, id := range []string{run.GetID(), run.GetID() + "/child", run.GetID() + "/child/grandchild"} {
		t.Run(id, func(t *testing.T) {
			history := &historypb.History{}
			iterator := c.GetWorkflowHistory(ctx, id, "", false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
			for iterator.HasNext() {
				event, err := iterator.Next()
				require.NoError(t, err)
				history.Events = append(history.Events, event)
			}
			var snapshots, scheduledActivities int
			for _, event := range history.Events {
				if marker := event.GetMarkerRecordedEventAttributes(); marker != nil && marker.MarkerName == "SideEffect" {
					var captured int
					require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(marker.Details["data"], &captured))
					require.Zero(t, captured)
					snapshots++
				}
				if scheduled := event.GetActivityTaskScheduledEventAttributes(); scheduled != nil {
					require.EqualValues(t, 3, scheduled.RetryPolicy.MaximumAttempts)
					require.Nil(t, scheduled.Header.GetFields()[pauseAttemptsHeader])
					scheduledActivities++
				}
			}
			if id == run.GetID()+"/child/grandchild" {
				require.Equal(t, 1, snapshots)
				require.Equal(t, 1, scheduledActivities)
			}
			for _, limit := range []int{0, 1, 15} {
				t.Run(fmt.Sprintf("worker-%d", limit), func(t *testing.T) {
					replayer, err := worker.NewWorkflowReplayerWithOptions(worker.WorkflowReplayerOptions{Interceptors: []interceptor.WorkerInterceptor{&StagePauseInterceptor{Attempts: limit}}})
					require.NoError(t, err)
					registerWorkflowLimitWorkflows(replayer)
					require.NoError(t, replayer.ReplayWorkflowHistoryWithOptions(nil, history, worker.ReplayWorkflowHistoryOptions{OriginalExecution: workflow.Execution{ID: id}}))
				})
			}
		})
	}
}
