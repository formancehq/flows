package workflow

import (
	"time"

	"github.com/formancehq/orchestration/internal/retry"
	"go.temporal.io/sdk/workflow"
)

// bookkeepingActivityContext is the activity context for the instance/stage bookkeeping
// activities Initiate, Run and Config.run schedule (inserting and updating instances and
// stages in the database, publishing lifecycle events). Each attempt keeps a short 10s
// StartToCloseTimeout, and retries are bounded by retry.Policy so a deterministic failure
// fails the workflow after ~30 minutes instead of retrying forever under Temporal's default
// (unlimited) policy.
func bookkeepingActivityContext(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy:         retry.Policy(),
	})
}
