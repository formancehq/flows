package activities

import (
	"testing"

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
		{name: "waiting for validation is treated as success", status: shared.V3PaymentInitiationStatusEnumWaitingForValidation, expectSuccess: true},
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

			err := classifyExistingPaymentInitiation(existing)

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
