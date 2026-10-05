// Package retry holds the bounded activity retry policy shared by every activity Flows schedules.
// Temporal's default RetryPolicy has no MaximumAttempts, so a deterministic failure would be
// retried for the life of the workflow.
package retry

import (
	"slices"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	initialInterval    = 2 * time.Second
	backoffCoefficient = 2
	maximumInterval    = 200 * time.Second
	// The 14 backoff waits between 15 attempts add up to ~28 minutes; with each attempt's
	// StartToCloseTimeout, an activity gives up after ~30 to ~43 minutes.
	maximumAttempts = 15

	shortStartToCloseTimeout = 10 * time.Second
)

// ActivityContext sets the per-attempt timeout and the bounded retry policy, treating the given
// error types as non-retryable.
func ActivityContext(ctx workflow.Context, startToCloseTimeout time.Duration, nonRetryableErrorTypes ...string) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: startToCloseTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        initialInterval,
			BackoffCoefficient:     backoffCoefficient,
			MaximumInterval:        maximumInterval,
			MaximumAttempts:        maximumAttempts,
			NonRetryableErrorTypes: slices.Clone(nonRetryableErrorTypes),
		},
	})
}

// ShortActivityContext is ActivityContext for quick database and publisher activities.
func ShortActivityContext(ctx workflow.Context) workflow.Context {
	return ActivityContext(ctx, shortStartToCloseTimeout)
}
