package temporalworker

import (
	"context"
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
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func TestPausePreservesActivityAndStopsRetryDispatch(t *testing.T) {
	server, err := testsuite.StartDevServer(t.Context(), testsuite.DevServerOptions{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, server.Stop()) })
	c := server.Client()
	w := worker.New(c, "pause-test", worker.Options{Interceptors: []interceptor.WorkerInterceptor{&StagePauseInterceptor{Attempts: 15}}})
	var calls atomic.Int32
	var ready atomic.Bool
	keys := make(chan string, 20)
	w.RegisterActivityWithOptions(func(ctx context.Context) (string, error) {
		info := activity.GetInfo(ctx)
		keys <- info.WorkflowExecution.RunID + "-" + info.ActivityID
		calls.Add(1)
		if ready.Load() {
			return "posted", nil
		}
		return "", temporal.NewApplicationError("insufficient funds", "INSUFFICIENT_FUND")
	}, activity.RegisterOptions{Name: "CreateTransaction"})
	pauseWorkflow := func(ctx workflow.Context) (string, error) {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 5 * time.Second, RetryPolicy: &temporal.RetryPolicy{InitialInterval: time.Second, BackoffCoefficient: 1, MaximumAttempts: 15, NonRetryableErrorTypes: []string{"INSUFFICIENT_FUND"}}})
		var result string
		err := workflow.ExecuteActivity(ctx, "CreateTransaction").Get(ctx, &result)
		return result, err
	}
	w.RegisterWorkflowWithOptions(pauseWorkflow, workflow.RegisterOptions{Name: funcNameForPauseTest})
	require.NoError(t, w.Start())
	t.Cleanup(w.Stop)
	run, err := c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{ID: "pause-test", TaskQueue: "pause-test"}, funcNameForPauseTest)
	require.NoError(t, err)
	var pendingID string
	require.Eventually(t, func() bool {
		d, e := c.DescribeWorkflowExecution(t.Context(), run.GetID(), run.GetRunID())
		if e != nil || len(d.PendingActivities) != 1 || !d.PendingActivities[0].Paused {
			return false
		}
		pendingID = d.PendingActivities[0].ActivityId
		return d.PendingActivities[0].LastFailure != nil
	}, 35*time.Second, 100*time.Millisecond)
	require.EqualValues(t, 15, calls.Load())
	// Longer than the retry interval: no sixteenth business operation is dispatched.
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	require.EqualValues(t, 15, calls.Load())
	ready.Store(true)
	_, err = c.WorkflowService().UnpauseActivity(t.Context(), &workflowservice.UnpauseActivityRequest{Namespace: "default", Execution: &commonpb.WorkflowExecution{WorkflowId: run.GetID(), RunId: run.GetRunID()}, Activity: &workflowservice.UnpauseActivityRequest_Id{Id: pendingID}, ResetAttempts: true})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var result string
	require.NoError(t, run.Get(ctx, &result))
	require.Equal(t, "posted", result)
	require.EqualValues(t, 16, calls.Load())
	iterator := c.GetWorkflowHistory(t.Context(), run.GetID(), run.GetRunID(), false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	history := &historypb.History{}
	for iterator.HasNext() {
		event, err := iterator.Next()
		require.NoError(t, err)
		history.Events = append(history.Events, event)
	}
	for _, limit := range []int{0, 1, 15} {
		replayer, err := worker.NewWorkflowReplayerWithOptions(worker.WorkflowReplayerOptions{Interceptors: []interceptor.WorkerInterceptor{&StagePauseInterceptor{Attempts: limit}}})
		require.NoError(t, err)
		replayer.RegisterWorkflowWithOptions(pauseWorkflow, workflow.RegisterOptions{Name: funcNameForPauseTest})
		require.NoError(t, replayer.ReplayWorkflowHistory(nil, history))
	}

	// Replay a pre-feature execution with no pause version or side-effect markers.
	legacyWorker := worker.New(c, "legacy-pause-test", worker.Options{})
	legacyWorker.RegisterWorkflowWithOptions(pauseWorkflow, workflow.RegisterOptions{Name: funcNameForPauseTest})
	legacyWorker.RegisterActivityWithOptions(func() (string, error) { return "legacy", nil }, activity.RegisterOptions{Name: "CreateTransaction"})
	require.NoError(t, legacyWorker.Start())
	defer legacyWorker.Stop()
	legacyRun, err := c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{ID: "legacy-pause-test", TaskQueue: "legacy-pause-test"}, funcNameForPauseTest)
	require.NoError(t, err)
	require.NoError(t, legacyRun.Get(ctx, &result))
	legacyHistory := &historypb.History{}
	legacyIterator := c.GetWorkflowHistory(t.Context(), legacyRun.GetID(), legacyRun.GetRunID(), false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for legacyIterator.HasNext() {
		event, err := legacyIterator.Next()
		require.NoError(t, err)
		legacyHistory.Events = append(legacyHistory.Events, event)
	}
	legacyReplayer, err := worker.NewWorkflowReplayerWithOptions(worker.WorkflowReplayerOptions{Interceptors: []interceptor.WorkerInterceptor{&StagePauseInterceptor{Attempts: 15}}})
	require.NoError(t, err)
	legacyReplayer.RegisterWorkflowWithOptions(pauseWorkflow, workflow.RegisterOptions{Name: funcNameForPauseTest})
	require.NoError(t, legacyReplayer.ReplayWorkflowHistory(nil, legacyHistory))

	close(keys)
	first := ""
	for key := range keys {
		if first == "" {
			first = key
		}
		require.Equal(t, first, key)
	}
}

const funcNameForPauseTest = "PauseTest"
