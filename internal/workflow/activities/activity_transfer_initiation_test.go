package activities

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/sdkerrors"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestClassifyV3Error(t *testing.T) {
	testCases := []struct {
		name         string
		errorCode    shared.V3ErrorsEnum
		nonRetryable bool
	}{
		{name: "internal is retryable", errorCode: shared.V3ErrorsEnumInternal, nonRetryable: false},
		{name: "validation is non-retryable", errorCode: shared.V3ErrorsEnumValidation, nonRetryable: true},
		{name: "invalid id is non-retryable", errorCode: shared.V3ErrorsEnumInvalidID, nonRetryable: true},
		{name: "missing or invalid body is non-retryable", errorCode: shared.V3ErrorsEnumMissingOrInvalidBody, nonRetryable: true},
		{name: "conflict is non-retryable", errorCode: shared.V3ErrorsEnumConflict, nonRetryable: true},
		{name: "not found is non-retryable", errorCode: shared.V3ErrorsEnumNotFound, nonRetryable: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			details := "extra context"
			sdkErr := &sdkerrors.V3ErrorResponse{
				ErrorCode:    tc.errorCode,
				ErrorMessage: "boom",
				Details:      &details,
			}

			var appErr *temporal.ApplicationError
			require.ErrorAs(t, classifyV3Error(sdkErr), &appErr)
			require.Equal(t, string(tc.errorCode), appErr.Type())
			require.Equal(t, tc.nonRetryable, appErr.NonRetryable())
			require.Equal(t, "boom", appErr.Message())
		})
	}
}

func TestClassifyV1Error(t *testing.T) {
	// PaymentsErrorsEnum (v1) only declares these three values as of the latest published
	// formance-sdk-go (v3.8.1) - see classifyV1Error's docstring for the known gap covering
	// CONFLICT, INVALID_ID, MISSING_OR_INVALID_BODY and CONNECTOR_CAPABILITY_NOT_SUPPORTED.
	testCases := []struct {
		name         string
		errorCode    shared.PaymentsErrorsEnum
		nonRetryable bool
	}{
		{name: "internal is retryable", errorCode: shared.PaymentsErrorsEnumInternal, nonRetryable: false},
		{name: "validation is non-retryable", errorCode: shared.PaymentsErrorsEnumValidation, nonRetryable: true},
		{name: "not found is non-retryable", errorCode: shared.PaymentsErrorsEnumNotFound, nonRetryable: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sdkErr := &sdkerrors.PaymentsErrorResponse{
				ErrorCode:    tc.errorCode,
				ErrorMessage: "boom",
			}

			var appErr *temporal.ApplicationError
			require.ErrorAs(t, classifyV1Error(sdkErr), &appErr)
			require.Equal(t, string(tc.errorCode), appErr.Type())
			require.Equal(t, tc.nonRetryable, appErr.NonRetryable())
			require.Equal(t, "boom", appErr.Message())
		})
	}
}

func TestClassifyExistingPaymentInitiation(t *testing.T) {
	pspError := "psp rejected the payout"

	testCases := []struct {
		name          string
		status        shared.V3PaymentInitiationStatusEnum
		existingError *string
		expectSuccess bool
	}{
		{name: "scheduled for processing is treated as success", status: shared.V3PaymentInitiationStatusEnumScheduledForProcessing, expectSuccess: true},
		{name: "processing is treated as success", status: shared.V3PaymentInitiationStatusEnumProcessing, expectSuccess: true},
		{name: "processed is treated as success", status: shared.V3PaymentInitiationStatusEnumProcessed, expectSuccess: true},
		{name: "failed is non-retryable", status: shared.V3PaymentInitiationStatusEnumFailed, existingError: &pspError},
		{name: "rejected is non-retryable", status: shared.V3PaymentInitiationStatusEnumRejected},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			existing := &shared.V3PaymentInitiation{
				ID:     "payment_initiation_id",
				Status: tc.status,
				Error:  tc.existingError,
			}

			var a Activities
			err := a.classifyExistingPaymentInitiation(context.Background(), existing)

			if tc.expectSuccess {
				require.NoError(t, err)
				return
			}

			var appErr *temporal.ApplicationError
			require.ErrorAs(t, err, &appErr)
			require.True(t, appErr.NonRetryable())
			// Labeled with the actual terminal status, not "CONFLICT" - see the comment on
			// classifyExistingPaymentInitiation for why.
			require.Equal(t, string(tc.status), appErr.Type())
			require.Contains(t, appErr.Message(), existing.ID)
			if tc.existingError != nil {
				require.Contains(t, appErr.Message(), *tc.existingError)
			}
		})
	}
}

// TestClassifyExistingPaymentInitiationWaitingForValidation covers the self-heal path added for
// the "CreateTransfer workflow never actually started" incident: a payment initiation stuck at
// WAITING_FOR_VALIDATION never received any adjustment past its initial one (nothing else retries
// it on payments' side), so classifyExistingPaymentInitiation re-triggers it via /approve rather
// than treating it as done.
func TestClassifyExistingPaymentInitiationWaitingForValidation(t *testing.T) {
	existing := &shared.V3PaymentInitiation{
		ID:     "payment_initiation_id",
		Status: shared.V3PaymentInitiationStatusEnumWaitingForValidation,
	}

	t.Run("approve succeeds", func(t *testing.T) {
		var approvedID string
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			approvedID = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(shared.V3ApprovePaymentInitiationResponse{
				Data: shared.V3ApprovePaymentInitiationResponseData{TaskID: "task_id"},
			})
		}))
		defer ts.Close()

		a := Activities{client: sdk.New(sdk.WithServerURL(ts.URL))}
		err := a.classifyExistingPaymentInitiation(context.Background(), existing)

		require.Contains(t, approvedID, existing.ID)

		var appErr *temporal.ApplicationError
		require.ErrorAs(t, err, &appErr)
		require.False(t, appErr.NonRetryable())
		require.Equal(t, string(existing.Status), appErr.Type())
		require.Contains(t, appErr.Message(), existing.ID)
	})

	t.Run("approve races with a concurrent retry and gets already-approved", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(sdkerrors.V3ErrorResponse{
				ErrorCode:    shared.V3ErrorsEnumValidation,
				ErrorMessage: "cannot approve an already approved payment initiation",
			})
		}))
		defer ts.Close()

		a := Activities{client: sdk.New(sdk.WithServerURL(ts.URL))}
		err := a.classifyExistingPaymentInitiation(context.Background(), existing)

		// Still retryable, and doesn't surface the benign race as a failure - the next
		// attempt re-checks the payment initiation's status from scratch.
		var appErr *temporal.ApplicationError
		require.ErrorAs(t, err, &appErr)
		require.False(t, appErr.NonRetryable())
		require.Equal(t, string(existing.Status), appErr.Type())
		require.NotContains(t, appErr.Message(), "already approved")
	})
}
