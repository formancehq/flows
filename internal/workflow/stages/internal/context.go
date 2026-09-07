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

// commonNonRetryableErrorCodes are the error codes both retry contexts below treat as
// non-retryable: VALIDATION and CONFLICT can surface from either the ledger operations
// InfiniteRetryContext guards or the PSP activities PaymentInitiationRetryContext guards.
var commonNonRetryableErrorCodes = []string{
	ErrorCodeValidation,
	ErrorCodeConflict,
}

func InfiniteRetryContext(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 60 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    100 * time.Second,
			// NO_SCRIPT/COMPILATION_FAILED are Numscript compile-time errors that only
			// CreateTransaction (a ledger operation this context guards) can return.
			NonRetryableErrorTypes: append(append([]string{}, commonNonRetryableErrorCodes...),
				ErrorCodeNoScript, ErrorCodeCompilationFailed),
		},
	})
}

// PaymentInitiationRetryContext is InfiniteRetryContext's bounded counterpart for activities that
// call out to a PSP (CreateTransferInitiation, StripeTransfer). Unlike the internal ledger
// operations InfiniteRetryContext is meant for, these activities can hit real-world PSP latency
// and conflicts that self-heal (see createTransferInitiationWithSelfHeal) rather than fail
// outright, but a request that never succeeds - or a self-heal fetch that itself keeps failing -
// must still eventually stop retrying rather than loop forever. MaximumAttempts bounds that. With
// a 2s initial interval doubling up to a 200s cap, the 14 backoff waits between 15 attempts alone
// add up to ~28 minutes; with StartToCloseTimeout factored in (each attempt can itself take up to
// 60s before failing), worst case reaches ~43 minutes - giving up after around 40 minutes.
func PaymentInitiationRetryContext(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 60 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        2 * time.Second,
			BackoffCoefficient:     2,
			MaximumInterval:        200 * time.Second,
			MaximumAttempts:        15,
			NonRetryableErrorTypes: commonNonRetryableErrorCodes,
		},
	})
}
