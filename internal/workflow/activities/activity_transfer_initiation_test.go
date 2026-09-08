package activities

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/formancehq/formance-sdk-go/v5/pkg/models/payments"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestParseTransferType(t *testing.T) {
	testCases := []struct {
		name        string
		requestType string
		want        payments.TransferInitiationRequestType
		wantErr     bool
	}{
		{name: "empty defaults to transfer", requestType: "", want: payments.TransferInitiationRequestTypeTransfer},
		{name: "transfer", requestType: "TRANSFER", want: payments.TransferInitiationRequestTypeTransfer},
		{name: "payout", requestType: "PAYOUT", want: payments.TransferInitiationRequestTypePayout},
		{name: "lowercase is case-insensitive", requestType: "payout", want: payments.TransferInitiationRequestTypePayout},
		{name: "invalid", requestType: "REFUND", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseTransferType(tc.requestType)
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
			got := defaultDescription(tc.description, tc.provider, payments.TransferInitiationRequestTypePayout)
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

func TestClassifyPaymentError(t *testing.T) {
	testCases := []struct {
		name         string
		errorCode    payments.PaymentsErrorsEnum
		nonRetryable bool
	}{
		{name: "internal is retryable", errorCode: payments.PaymentsErrorsEnumInternal, nonRetryable: false},
		{name: "validation is non-retryable", errorCode: payments.PaymentsErrorsEnumValidation, nonRetryable: true},
		{name: "not found is non-retryable", errorCode: payments.PaymentsErrorsEnumNotFound, nonRetryable: true},
		// These four are the ones formance-sdk-go v3.8.1's PaymentsErrorsEnum didn't declare -
		// a response carrying any of them used to fail UnmarshalJSON before this function was
		// ever reached, and fell through as a plain, unclassified retryable error instead. v5.0.1
		// declares all seven, so they now decode and classify the same as any other code.
		{name: "conflict is non-retryable", errorCode: payments.PaymentsErrorsEnumConflict, nonRetryable: true},
		{name: "invalid id is non-retryable", errorCode: payments.PaymentsErrorsEnumInvalidID, nonRetryable: true},
		{name: "missing or invalid body is non-retryable", errorCode: payments.PaymentsErrorsEnumMissingOrInvalidBody, nonRetryable: true},
		{name: "connector capability not supported is non-retryable", errorCode: payments.PaymentsErrorsEnumConnectorCapabilityNotSupported, nonRetryable: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sdkErr := &payments.PaymentsErrorResponse{
				ErrorCode:    tc.errorCode,
				ErrorMessage: "boom",
			}

			var appErr *temporal.ApplicationError
			require.ErrorAs(t, classifyPaymentError(sdkErr), &appErr)
			require.Equal(t, string(tc.errorCode), appErr.Type())
			require.Equal(t, tc.nonRetryable, appErr.NonRetryable())
			require.Equal(t, "boom", appErr.Message())
		})
	}
}

func TestClassifyV3Error(t *testing.T) {
	testCases := []struct {
		name         string
		errorCode    payments.V3ErrorsEnum
		nonRetryable bool
	}{
		{name: "internal is retryable", errorCode: payments.V3ErrorsEnumInternal, nonRetryable: false},
		{name: "validation is non-retryable", errorCode: payments.V3ErrorsEnumValidation, nonRetryable: true},
		{name: "not found is non-retryable", errorCode: payments.V3ErrorsEnumNotFound, nonRetryable: true},
		{name: "invalid id is non-retryable", errorCode: payments.V3ErrorsEnumInvalidID, nonRetryable: true},
		{name: "missing or invalid body is non-retryable", errorCode: payments.V3ErrorsEnumMissingOrInvalidBody, nonRetryable: true},
		{name: "conflict is non-retryable", errorCode: payments.V3ErrorsEnumConflict, nonRetryable: true},
		{name: "connector capability not supported is non-retryable", errorCode: payments.V3ErrorsEnumConnectorCapabilityNotSupported, nonRetryable: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sdkErr := &payments.V3ErrorResponse{
				ErrorCode:    tc.errorCode,
				ErrorMessage: "boom",
			}

			var appErr *temporal.ApplicationError
			require.ErrorAs(t, classifyV3Error(sdkErr), &appErr)
			require.Equal(t, string(tc.errorCode), appErr.Type())
			require.Equal(t, tc.nonRetryable, appErr.NonRetryable())
			require.Equal(t, "boom", appErr.Message())
		})
	}
}

// TestConnectorEnumRejectsUnknownProvider pins the SDK-side defect resolveConnectorID routes
// around: payments' v1-2 openapi declares Connector as a closed enum, its v2 handler emits any
// newer provider verbatim (toV2Provider's default branch), and the generated UnmarshalJSON then
// fails the entire GET /connectors response - which is how a payout to any PSP started failing
// with "invalid value for Connector: routable" on stacks with a Routable connector installed.
// V3Connector.Provider is a plain string, so the v3 listing decodes the same value fine.
//
// If the first assertion ever starts failing, formance-sdk-go relaxed the enum: the v1 listing is
// safe again and resolveConnectorID's version dispatch can be reconsidered.
func TestConnectorEnumRejectsUnknownProvider(t *testing.T) {
	var c payments.Connector
	require.ErrorContains(t, c.UnmarshalJSON([]byte(`"routable"`)), "invalid value for Connector: routable")

	var v3 payments.V3Connector
	require.NoError(t, json.Unmarshal([]byte(`{"provider":"routable"}`), &v3))
	require.Equal(t, "routable", v3.Provider)
}

// TestPaymentsErrorsEnumUnmarshalsConflict is a narrow regression test for the actual gap that
// motivated the v5.0.1 bump: a v1/v2 payments server returning errorCode "CONFLICT" must decode
// into PaymentsErrorResponse rather than failing PaymentsErrorsEnum.UnmarshalJSON, which is what
// silently degraded a CONFLICT into a plain, unclassified retryable error under v3.8.1.
func TestPaymentsErrorsEnumUnmarshalsConflict(t *testing.T) {
	var e payments.PaymentsErrorsEnum
	require.NoError(t, e.UnmarshalJSON([]byte(`"CONFLICT"`)))
	require.Equal(t, payments.PaymentsErrorsEnumConflict, e)
}

// TestClassifyExistingTransferInitiation covers the self-heal branches that don't require a live
// payments client (the WAITING_FOR_VALIDATION "re-trigger validation" branch does call the client
// and isn't covered here). This is what CreateTransferInitiation/StripeTransfer fall back to after
// a CONFLICT resolves to an existing record - load testing showed that without it, a response
// merely arriving late (past the activity's StartToCloseTimeout) turns into a permanent workflow
// failure on retry, even though the transfer was already recorded successfully.
func TestClassifyExistingTransferInitiation(t *testing.T) {
	t.Run("failed is non-retryable", func(t *testing.T) {
		errMsg := "insufficient funds"
		existing := &payments.TransferInitiation{ID: "ti_1", Status: payments.TransferInitiationStatusFailed, Error: &errMsg}
		err := classifyExistingTransferInitiation(context.Background(), Activities{}, existing, false)
		var appErr *temporal.ApplicationError
		require.ErrorAs(t, err, &appErr)
		require.True(t, appErr.NonRetryable())
		require.Equal(t, string(payments.TransferInitiationStatusFailed), appErr.Type())
	})

	t.Run("rejected is non-retryable", func(t *testing.T) {
		existing := &payments.TransferInitiation{ID: "ti_2", Status: payments.TransferInitiationStatusRejected}
		err := classifyExistingTransferInitiation(context.Background(), Activities{}, existing, false)
		var appErr *temporal.ApplicationError
		require.ErrorAs(t, err, &appErr)
		require.True(t, appErr.NonRetryable())
	})

	t.Run("waiting for validation, caller asked for it, is success", func(t *testing.T) {
		existing := &payments.TransferInitiation{ID: "ti_3", Status: payments.TransferInitiationStatusWaitingForValidation}
		err := classifyExistingTransferInitiation(context.Background(), Activities{}, existing, true)
		require.NoError(t, err)
	})

	t.Run("processing (already created, no failure) is success", func(t *testing.T) {
		existing := &payments.TransferInitiation{ID: "ti_4", Status: payments.TransferInitiationStatusProcessing}
		err := classifyExistingTransferInitiation(context.Background(), Activities{}, existing, false)
		require.NoError(t, err)
	})
}
