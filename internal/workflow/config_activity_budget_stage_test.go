package workflow

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/formancehq/orchestration/internal/temporalworker"
	"github.com/formancehq/orchestration/internal/workflow/stages"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	temporalworkflow "go.temporal.io/sdk/workflow"
)

func TestConfigRunStageActivityBudgetBounded(t *testing.T) {
	for _, tc := range []struct {
		name     string
		attempts *int
		want     int32
	}{
		{"configured", new(3), 3},
		{"legacy-fallback", nil, 15},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
			env.SetWorkerOptions(worker.Options{Interceptors: []interceptor.WorkerInterceptor{
				&temporalworker.StagePauseInterceptor{Attempts: 0},
			}})
			var calls atomic.Int32
			env.RegisterActivityWithOptions(func(context.Context) error {
				calls.Add(1)
				return errors.New("service unavailable")
			}, activity.RegisterOptions{Name: "CreateTransaction"})
			// Replace the noop child with a retrying stage activity. The parent still
			// resolves and schedules it through the real Config.runStage path.
			env.RegisterWorkflowWithOptions(func(ctx temporalworkflow.Context, _ stages.NoOp) error {
				ctx = temporalworkflow.WithActivityOptions(ctx, temporalworkflow.ActivityOptions{
					StartToCloseTimeout: time.Minute,
					RetryPolicy:         &temporal.RetryPolicy{InitialInterval: time.Second, MaximumAttempts: 15},
				})
				return temporalworkflow.ExecuteActivity(ctx, "CreateTransaction").Get(ctx, nil)
			}, temporalworkflow.RegisterOptions{Name: "RunNoOp"})
			env.ExecuteWorkflow(func(ctx temporalworkflow.Context, config Config) error {
				return config.runStage(ctx, NewStage("budget-instance", "run", 0), RawStage{"noop": {}}, nil)
			}, Config{ActivityMaxAttempts: tc.attempts})
			require.True(t, env.IsWorkflowCompleted())
			require.ErrorContains(t, env.GetWorkflowError(), "service unavailable")
			require.Equal(t, tc.want, calls.Load())
		})
	}
}
