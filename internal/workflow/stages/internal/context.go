package internal

import (
	"slices"
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

// commonNonRetryableErrorCodes can surface from both the ledger/wallet operations and the PSP
// activities.
var commonNonRetryableErrorCodes = []string{
	ErrorCodeValidation,
	ErrorCodeConflict,
}

// ledgerNonRetryableErrorCodes adds ledger-only terminal errors: NO_SCRIPT/COMPILATION_FAILED are
// Numscript compile errors, and INSUFFICIENT_FUND fails identically until the balance changes.
var ledgerNonRetryableErrorCodes = slices.Concat(commonNonRetryableErrorCodes,
	[]string{ErrorCodeNoScript, ErrorCodeCompilationFailed, ErrorCodeInsufficientFund})

const stageActivityStartToCloseTimeout = 60 * time.Second

// LedgerRetryContext is the bounded retry context for the ledger, wallet and payment read
// operations of the send and update stages. Retries of one activity reuse its idempotency key
// (RunID + ActivityID), so they never double-post.
func LedgerRetryContext(ctx workflow.Context) workflow.Context {
	return retry.ActivityContext(ctx, stageActivityStartToCloseTimeout, ledgerNonRetryableErrorCodes...)
}

// PaymentInitiationRetryContext is the bounded retry context for activities that call a PSP
// (CreateTransferInitiation, StripeTransfer). Only the common codes are terminal: PSP latency and
// conflicts can self-heal (see createTransferInitiationWithSelfHeal).
func PaymentInitiationRetryContext(ctx workflow.Context) workflow.Context {
	return retry.ActivityContext(ctx, stageActivityStartToCloseTimeout, commonNonRetryableErrorCodes...)
}
