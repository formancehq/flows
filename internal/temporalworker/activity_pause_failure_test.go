package temporalworker

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPauseFailsClosedWhenControlAPIUnavailable(t *testing.T) {
	server, err := testsuite.StartDevServer(t.Context(), testsuite.DevServerOptions{ClientOptions: &client.Options{ConnectionOptions: client.ConnectionOptions{DialOptions: []grpc.DialOption{grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if strings.HasSuffix(method, "/PauseActivity") {
			return status.Error(codes.Unimplemented, "pause unavailable")
		}
		return invoke(ctx, method, req, reply, cc, opts...)
	})}}}})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, server.Stop()) })
	c := server.Client()
	w := worker.New(c, "unavailable-pause", worker.Options{Interceptors: []interceptor.WorkerInterceptor{&StagePauseInterceptor{Attempts: 1}}})
	var calls atomic.Int32
	w.RegisterActivityWithOptions(func() error { calls.Add(1); return temporal.NewApplicationError("unavailable", "UNAVAILABLE") }, activity.RegisterOptions{Name: "CreateTransaction"})
	w.RegisterWorkflowWithOptions(pauseFailureWorkflow, workflow.RegisterOptions{Name: "PauseFailureTest"})
	require.NoError(t, w.Start())
	t.Cleanup(w.Stop)
	run, err := c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{ID: "unavailable-pause", TaskQueue: "unavailable-pause"}, "PauseFailureTest")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	err = run.Get(ctx, nil)
	require.ErrorContains(t, err, "ACTIVITY_PAUSE_FAILED")
	require.EqualValues(t, 1, calls.Load())
}

func TestPauseGuardsBudgetAfterActivityPanic(t *testing.T) {
	testPauseGuardsBudget(t, false)
}

func TestPauseWithExpiredAttemptContext(t *testing.T) {
	testPauseGuardsBudget(t, true)
}

func testPauseGuardsBudget(t *testing.T, expire bool) {
	server, err := testsuite.StartDevServer(t.Context(), testsuite.DevServerOptions{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, server.Stop()) })
	c := server.Client()
	w := worker.New(c, "panic-pause", worker.Options{Interceptors: []interceptor.WorkerInterceptor{&StagePauseInterceptor{Attempts: 1}}})
	var calls atomic.Int32
	w.RegisterActivityWithOptions(func(ctx context.Context) error {
		calls.Add(1)
		if expire {
			<-ctx.Done()
			return ctx.Err()
		}
		panic("worker activity panic")
	}, activity.RegisterOptions{Name: "CreateTransaction"})
	w.RegisterWorkflowWithOptions(pauseFailureWorkflow, workflow.RegisterOptions{Name: "PauseFailureTest"})
	require.NoError(t, w.Start())
	t.Cleanup(w.Stop)
	run, err := c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{ID: "panic-pause", TaskQueue: "panic-pause"}, "PauseFailureTest")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		d, e := c.DescribeWorkflowExecution(t.Context(), run.GetID(), run.GetRunID())
		return e == nil && len(d.PendingActivities) == 1 && d.PendingActivities[0].Paused
	}, 10*time.Second, 100*time.Millisecond)
	require.EqualValues(t, 1, calls.Load())
	require.NoError(t, c.CancelWorkflow(t.Context(), run.GetID(), run.GetRunID()))
}

func pauseFailureWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 5 * time.Second, RetryPolicy: &temporal.RetryPolicy{InitialInterval: time.Second, MaximumAttempts: 15}})
	return workflow.ExecuteActivity(ctx, "CreateTransaction").Get(ctx, nil)
}
