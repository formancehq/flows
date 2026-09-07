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

func TestParseTransferType(t *testing.T) {
	testCases := []struct {
		name        string
		requestType string
		want        shared.V3PaymentInitiationTypeEnum
		wantErr     bool
	}{
		{name: "empty defaults to transfer", requestType: "", want: shared.V3PaymentInitiationTypeEnumTransfer},
		{name: "transfer", requestType: "TRANSFER", want: shared.V3PaymentInitiationTypeEnumTransfer},
		{name: "payout", requestType: "PAYOUT", want: shared.V3PaymentInitiationTypeEnumPayout},
		{name: "lowercase is case-insensitive", requestType: "payout", want: shared.V3PaymentInitiationTypeEnumPayout},
		{name: "invalid", requestType: "REFUND", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseTransferType(tc.requestType, shared.V3PaymentInitiationTypeEnumTransfer, shared.V3PaymentInitiationTypeEnumPayout)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestDefaultDescription(t *testing.T) {
	provider := "stripe"

	testCases := []struct {
		name        string
		description string
		provider    *string
		want        string
	}{
		{name: "explicit description is kept", description: "invoice #123", provider: &provider, want: "invoice #123"},
		{name: "falls back to provider and type", description: "", provider: &provider, want: "stripe PAYOUT"},
		{name: "falls back without provider", description: "", provider: nil, want: "Transfer Initiation (PAYOUT)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := defaultDescription(tc.description, tc.provider, shared.V3PaymentInitiationTypeEnumPayout)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestConnectorIDOrProviderRequired(t *testing.T) {
	connectorID := "connector_id"
	provider := "stripe"
	empty := ""

	t.Run("connectorID set wins", func(t *testing.T) {
		id, resolved, err := connectorIDOrProviderRequired(&connectorID, nil)
		require.NoError(t, err)
		require.True(t, resolved)
		require.Equal(t, connectorID, id)
	})

	t.Run("no connectorID, no provider is non-retryable", func(t *testing.T) {
		_, resolved, err := connectorIDOrProviderRequired(nil, nil)
		require.False(t, resolved)
		var appErr *temporal.ApplicationError
		require.ErrorAs(t, err, &appErr)
		require.True(t, appErr.NonRetryable())
	})

	t.Run("empty connectorID and empty provider is non-retryable", func(t *testing.T) {
		_, resolved, err := connectorIDOrProviderRequired(&empty, &empty)
		require.False(t, resolved)
		require.Error(t, err)
	})

	t.Run("no connectorID, provider set defers to caller", func(t *testing.T) {
		id, resolved, err := connectorIDOrProviderRequired(nil, &provider)
		require.NoError(t, err)
		require.False(t, resolved)
		require.Empty(t, id)
	})
}

func TestResolveProviderMatch(t *testing.T) {
	t.Run("no matches is non-retryable", func(t *testing.T) {
		_, err := resolveProviderMatch(nil, "stripe")
		var appErr *temporal.ApplicationError
		require.ErrorAs(t, err, &appErr)
		require.True(t, appErr.NonRetryable())
	})

	t.Run("single match wins", func(t *testing.T) {
		id, err := resolveProviderMatch([]string{"connector_id"}, "stripe")
		require.NoError(t, err)
		require.Equal(t, "connector_id", id)
	})

	t.Run("multiple matches is non-retryable", func(t *testing.T) {
		_, err := resolveProviderMatch([]string{"c1", "c2"}, "stripe")
		var appErr *temporal.ApplicationError
		require.ErrorAs(t, err, &appErr)
		require.True(t, appErr.NonRetryable())
	})
}

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

	t.Run("approve fails with a genuine validation error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(sdkerrors.V3ErrorResponse{
				ErrorCode:    shared.V3ErrorsEnumValidation,
				ErrorMessage: "amount must be positive",
			})
		}))
		defer ts.Close()

		a := Activities{client: sdk.New(sdk.WithServerURL(ts.URL))}
		err := a.classifyExistingPaymentInitiation(context.Background(), existing)

		// Not the already-approved race - a real validation failure, so this must be
		// non-retryable rather than silently assumed benign.
		var appErr *temporal.ApplicationError
		require.ErrorAs(t, err, &appErr)
		require.True(t, appErr.NonRetryable())
		require.Equal(t, string(shared.V3ErrorsEnumValidation), appErr.Type())
		require.Equal(t, "amount must be positive", appErr.Message())
	})

	t.Run("approve fails with a typed 4xx like NOT_FOUND", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(sdkerrors.V3ErrorResponse{
				ErrorCode:    shared.V3ErrorsEnumNotFound,
				ErrorMessage: "payment initiation not found",
			})
		}))
		defer ts.Close()

		a := Activities{client: sdk.New(sdk.WithServerURL(ts.URL))}
		err := a.classifyExistingPaymentInitiation(context.Background(), existing)

		// Must not be blindly treated as a transient/retryable failure - classified via
		// classifyV3Error like any other typed v3 API error.
		var appErr *temporal.ApplicationError
		require.ErrorAs(t, err, &appErr)
		require.True(t, appErr.NonRetryable())
		require.Equal(t, string(shared.V3ErrorsEnumNotFound), appErr.Type())
	})
}
