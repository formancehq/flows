package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// retryPolicyOf runs `build` inside a real workflow goroutine - WithActivityOptions
// needs one - and returns the RetryPolicy it installed on the context.
func retryPolicyOf(t *testing.T, build func(workflow.Context) workflow.Context) workflow.ActivityOptions {
	t.Helper()

	var captured workflow.ActivityOptions
	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		captured = workflow.GetActivityOptions(build(ctx))
		return nil
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	return captured
}

// TestInfiniteRetryContextDoesNotRetryInsufficientFund pins the terminal treatment of
// INSUFFICIENT_FUND. This context deliberately sets no MaximumAttempts, so a code that is
// missing from NonRetryableErrorTypes is retried for the life of the workflow - the ledger
// re-evaluates the same postings against the same balance and fails identically every time.
func TestInfiniteRetryContextDoesNotRetryInsufficientFund(t *testing.T) {
	t.Parallel()

	opts := retryPolicyOf(t, InfiniteRetryContext)

	require.Zero(t, opts.RetryPolicy.MaximumAttempts,
		"InfiniteRetryContext is unbounded, so NonRetryableErrorTypes is the only thing that stops a retry loop")
	require.Contains(t, opts.RetryPolicy.NonRetryableErrorTypes, ErrorCodeInsufficientFund)
	require.Contains(t, opts.RetryPolicy.NonRetryableErrorTypes, ErrorCodeValidation)
	require.Contains(t, opts.RetryPolicy.NonRetryableErrorTypes, ErrorCodeConflict)
	require.Contains(t, opts.RetryPolicy.NonRetryableErrorTypes, ErrorCodeNoScript)
	require.Contains(t, opts.RetryPolicy.NonRetryableErrorTypes, ErrorCodeCompilationFailed)
}

// TestPaymentInitiationRetryContextKeepsCommonCodes guards against the append() in
// InfiniteRetryContext mutating the shared commonNonRetryableErrorCodes backing array: the
// PSP context must keep exactly the common codes and gain none of the ledger-only ones.
func TestPaymentInitiationRetryContextKeepsCommonCodes(t *testing.T) {
	t.Parallel()

	opts := retryPolicyOf(t, PaymentInitiationRetryContext)

	require.Equal(t, commonNonRetryableErrorCodes, opts.RetryPolicy.NonRetryableErrorTypes)
	require.EqualValues(t, 15, opts.RetryPolicy.MaximumAttempts)
}
