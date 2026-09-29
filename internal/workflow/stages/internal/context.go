package internal

import (
	"time"

	"github.com/formancehq/orchestration/internal/retry"
	"go.temporal.io/sdk/workflow"
)

const (
	ErrorCodeValidation        = "VALIDATION"
	ErrorCodeConflict          = "CONFLICT"
	ErrorCodeNoScript          = "NO_SCRIPT"
	ErrorCodeCompilationFailed = "COMPILATION_FAILED"
	// Singular, matching ledger.V2ErrorsEnumInsufficientFund and wallets.ErrorCodeInsufficientFund.
	ErrorCodeInsufficientFund = "INSUFFICIENT_FUND"
)

// commonNonRetryableErrorCodes are the error codes both retry contexts below treat as
// non-retryable: VALIDATION and CONFLICT can surface from either the ledger/wallet operations
// LedgerRetryContext guards or the PSP activities PaymentInitiationRetryContext guards.
var commonNonRetryableErrorCodes = []string{
	ErrorCodeValidation,
	ErrorCodeConflict,
}

// ledgerNonRetryableErrorCodes extends the common codes with ledger-specific terminal errors.
// NO_SCRIPT/COMPILATION_FAILED are Numscript compile-time errors that only CreateTransaction
// can return. INSUFFICIENT_FUND is a settled business outcome from CreateTransaction or
// DebitWallet - nothing between attempts changes the source balance, so retrying it only burns
// the attempt budget before failing identically.
var ledgerNonRetryableErrorCodes = append(append([]string{}, commonNonRetryableErrorCodes...),
	ErrorCodeNoScript, ErrorCodeCompilationFailed, ErrorCodeInsufficientFund)

// stageActivityStartToCloseTimeout caps a single attempt of any stage activity.
const stageActivityStartToCloseTimeout = 60 * time.Second

// LedgerRetryContext is the retry context for the internal ledger, wallet and payment read
// operations the send and update stages run (CreateTransaction, DebitWallet, CreditWallet,
// AddAccountMetadata, GetAccount, GetWallet, GetPayment, ...). It uses the bounded
// retry.Policy: a transient outage is ridden out for ~40 minutes (15 attempts), after which the
// activity - and so the stage and the workflow instance - fails with the last error instead of
// retrying forever. Retries of one activity invocation reuse the same idempotency key
// (RunID + ActivityID), so they never double-post.
func LedgerRetryContext(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: stageActivityStartToCloseTimeout,
		RetryPolicy:         retry.Policy(ledgerNonRetryableErrorCodes...),
	})
}

// PaymentInitiationRetryContext is the retry context for activities that call out to a PSP
// (CreateTransferInitiation, StripeTransfer). It shares LedgerRetryContext's bounded
// retry.Policy but only treats the common codes as non-retryable: these activities can hit
// real-world PSP latency and conflicts that self-heal (see createTransferInitiationWithSelfHeal)
// rather than fail outright, yet a request that never succeeds - or a self-heal fetch that
// itself keeps failing - still stops retrying after ~40 minutes (15 attempts).
func PaymentInitiationRetryContext(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: stageActivityStartToCloseTimeout,
		RetryPolicy:         retry.Policy(commonNonRetryableErrorCodes...),
	})
}
