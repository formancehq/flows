// Package retry holds the bounded activity retry policy shared by every activity Flows schedules.
//
// Temporal's default activity RetryPolicy has no MaximumAttempts, so an activity that fails
// deterministically (a bad expression, a constraint violation, a downstream service that keeps
// rejecting the request) is retried for the life of the workflow: the workflow never makes
// progress and never reports the failure. Every ActivityOptions in this repository should
// therefore set RetryPolicy to Policy(...) rather than rely on the default.
package retry

import (
	"time"

	"go.temporal.io/sdk/temporal"
)

const (
	InitialInterval    = 2 * time.Second
	BackoffCoefficient = 2
	MaximumInterval    = 200 * time.Second
	// MaximumAttempts bounds the retries. With a 2s initial interval doubling up to a 200s cap,
	// the 14 backoff waits between 15 attempts add up to ~28 minutes. Each attempt can also run
	// up to its StartToCloseTimeout before failing (60s for stage activities, 10s for the
	// bookkeeping ones), so the worst case is ~30 to ~43 minutes before giving up.
	MaximumAttempts = 15
)

// Policy returns a fresh bounded RetryPolicy treating the given error types as non-retryable.
// The slice is copied, so callers can pass a shared package-level slice without the returned
// policy aliasing it.
func Policy(nonRetryableErrorTypes ...string) *temporal.RetryPolicy {
	return &temporal.RetryPolicy{
		InitialInterval:        InitialInterval,
		BackoffCoefficient:     BackoffCoefficient,
		MaximumInterval:        MaximumInterval,
		MaximumAttempts:        MaximumAttempts,
		NonRetryableErrorTypes: append([]string(nil), nonRetryableErrorTypes...),
	}
}
