package temporalworker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"time"
)

func TestPauseKeepsTerminalErrors(t *testing.T) {
	policy := &temporal.RetryPolicy{NonRetryableErrorTypes: []string{"VALIDATION", "CONFLICT"}}
	for _, code := range []string{"VALIDATION", "CONFLICT"} {
		require.True(t, terminalActivityError(temporal.NewApplicationError("invalid", code), policy))
	}
	require.True(t, terminalActivityError(temporal.NewNonRetryableApplicationError("permanent", "OTHER", nil), policy))
	require.True(t, terminalActivityError(context.Canceled, policy))
	require.False(t, terminalActivityError(temporal.NewApplicationError("balance too low", "INSUFFICIENT_FUND"), policy))
}

func TestPauseDisabledPreservesBoundedFailure(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetWorkerOptions(worker.Options{Interceptors: []interceptor.WorkerInterceptor{&StagePauseInterceptor{Attempts: 0}}})
	calls := 0
	env.RegisterActivityWithOptions(func() error { calls++; return temporal.NewApplicationError("unavailable", "UNAVAILABLE") }, activity.RegisterOptions{Name: "CreateTransaction"})
	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Second, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 15}})
		return workflow.ExecuteActivity(ctx, "CreateTransaction").Get(ctx, nil)
	})
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	require.Equal(t, 15, calls)
}
