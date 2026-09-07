package activities

import (
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

// TestPaymentsErrorsEnumUnmarshalsConflict is a narrow regression test for the actual gap that
// motivated the v5.0.1 bump: a v1/v2 payments server returning errorCode "CONFLICT" must decode
// into PaymentsErrorResponse rather than failing PaymentsErrorsEnum.UnmarshalJSON, which is what
// silently degraded a CONFLICT into a plain, unclassified retryable error under v3.8.1.
func TestPaymentsErrorsEnumUnmarshalsConflict(t *testing.T) {
	var e payments.PaymentsErrorsEnum
	require.NoError(t, e.UnmarshalJSON([]byte(`"CONFLICT"`)))
	require.Equal(t, payments.PaymentsErrorsEnumConflict, e)
}
