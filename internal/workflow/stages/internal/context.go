package internal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ErrorCodeValidation        = "VALIDATION"
	ErrorCodeConflict          = "CONFLICT"
	ErrorCodeNoScript          = "NO_SCRIPT"
	ErrorCodeCompilationFailed = "COMPILATION_FAILED"
)

func InfiniteRetryContext(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 60 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    100 * time.Second,
			NonRetryableErrorTypes: []string{
				ErrorCodeValidation,
				ErrorCodeConflict,
				ErrorCodeNoScript,
				ErrorCodeCompilationFailed,
			},
		},
	})
}

// PaymentInitiationRetryContext is InfiniteRetryContext's bounded counterpart for activities that
// call out to a PSP (CreateTransferInitiation, StripeTransfer). Unlike the internal ledger
// operations InfiniteRetryContext is meant for, these activities' error classification has known
// gaps (see classifyV1Error's docstring: some payments error codes aren't retryable but also
// aren't recognized as non-retryable by this SDK version), so an unclassified error must still
// eventually stop retrying rather than loop forever against a request that will never succeed.
// MaximumAttempts bounds that. With a 2s initial interval doubling up to a 200s cap, the 14
// backoff waits between 15 attempts alone add up to ~28 minutes; with StartToCloseTimeout
// factored in (each attempt can itself take up to 60s before failing), worst case reaches
// ~43 minutes - giving up after around 40 minutes.
func PaymentInitiationRetryContext(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 60 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    200 * time.Second,
			MaximumAttempts:    15,
			NonRetryableErrorTypes: []string{
				ErrorCodeValidation,
				ErrorCodeConflict,
				ErrorCodeNoScript,
				ErrorCodeCompilationFailed,
			},
		},
	})
}
