package internal

import (
	"testing"
	"time"

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

// requireBoundedPolicy pins a context to the shared bounded policy against literals, not the
// retry package constants: a test comparing the constants to themselves could never catch an
// accidental change to the numbers the ~40 minute give-up budget is derived from.
func requireBoundedPolicy(t *testing.T, opts workflow.ActivityOptions) {
	t.Helper()

	require.Equal(t, 60*time.Second, opts.StartToCloseTimeout)
	require.NotNil(t, opts.RetryPolicy)
	require.Equal(t, 2*time.Second, opts.RetryPolicy.InitialInterval)
	require.Equal(t, 2.0, opts.RetryPolicy.BackoffCoefficient)
	require.Equal(t, 200*time.Second, opts.RetryPolicy.MaximumInterval)
	require.EqualValues(t, 15, opts.RetryPolicy.MaximumAttempts,
		"a zero MaximumAttempts means Temporal retries forever")
}

// TestLedgerRetryContextIsBoundedAndKeepsLedgerCodes pins the ledger context to the bounded
// policy and to its terminal error codes. INSUFFICIENT_FUND and the Numscript compile errors
// are settled outcomes: retrying them only burns the ~40 minute attempt budget before failing
// identically, so they must stay non-retryable even now that the policy is bounded.
func TestLedgerRetryContextIsBoundedAndKeepsLedgerCodes(t *testing.T) {
	t.Parallel()

	opts := retryPolicyOf(t, LedgerRetryContext)

	requireBoundedPolicy(t, opts)
	require.Equal(t, []string{
		ErrorCodeValidation,
		ErrorCodeConflict,
		ErrorCodeNoScript,
		ErrorCodeCompilationFailed,
		ErrorCodeInsufficientFund,
	}, opts.RetryPolicy.NonRetryableErrorTypes)
}

// TestPaymentInitiationRetryContextKeepsCommonCodes pins the PSP context to the bounded policy
// and to exactly the common codes. It builds the ledger context first, so a future change that
// lets the ledger codes write into the shared commonNonRetryableErrorCodes backing array shows
// up here as ledger-only codes leaking into the PSP policy. The expected value is a literal
// rather than commonNonRetryableErrorCodes itself: comparing the package variable against a
// policy built from that same variable can never fail.
func TestPaymentInitiationRetryContextKeepsCommonCodes(t *testing.T) {
	t.Parallel()

	_ = retryPolicyOf(t, LedgerRetryContext)
	opts := retryPolicyOf(t, PaymentInitiationRetryContext)

	requireBoundedPolicy(t, opts)
	require.Equal(t, []string{ErrorCodeValidation, ErrorCodeConflict},
		opts.RetryPolicy.NonRetryableErrorTypes)
}

// TestRetryContextsDoNotShareNonRetryableSlices guards against a policy aliasing the package
// level code slices: mutating one context's NonRetryableErrorTypes must not leak into the
// next policy either context builds.
func TestRetryContextsDoNotShareNonRetryableSlices(t *testing.T) {
	t.Parallel()

	first := retryPolicyOf(t, PaymentInitiationRetryContext)
	first.RetryPolicy.NonRetryableErrorTypes[0] = "MUTATED"

	second := retryPolicyOf(t, PaymentInitiationRetryContext)
	require.Equal(t, ErrorCodeValidation, second.RetryPolicy.NonRetryableErrorTypes[0])
}
