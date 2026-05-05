package send

import (
	"math/big"
	"testing"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/formancehq/orchestration/internal/schema"
	"github.com/stretchr/testify/require"
)

func TestPaymentDestinationRequiresConnectorID(t *testing.T) {
	t.Parallel()

	t.Run("valid connectorID", func(t *testing.T) {
		t.Parallel()

		resolved, err := schema.Resolve(schema.Context{}, map[string]any{
			"source": map[string]any{
				"account": map[string]any{
					"id": "source",
				},
			},
			"destination": map[string]any{
				"payment": map[string]any{
					"psp":         "stripe",
					"metadata":    "formanceAccountID",
					"connectorID": "conn_123",
				},
			},
			"amount": map[string]any{
				"amount": float64(100),
				"asset":  "USD/2",
			},
		}, "send")
		require.NoError(t, err)

		send := resolved.(*Send)
		require.Equal(t, &Send{
			Source: Source{
				Account: &LedgerAccountSource{
					ID:             "source",
					Ledger:         "default",
					ThroughAccount: "world",
					AllowOverdraft: false,
				},
			},
			Destination: Destination{
				Payment: &PaymentDestination{
					PSP:               "stripe",
					Type:              "TRANSFER",
					Metadata:          "formanceAccountID",
					WaitingValidation: false,
					ConnectorID:       stringPtr("conn_123"),
				},
			},
			Amount: &shared.Monetary{
				Amount: big.NewInt(100),
				Asset:  "USD/2",
			},
		}, send)
	})

	t.Run("missing connectorID", func(t *testing.T) {
		t.Parallel()

		resolved, err := schema.Resolve(schema.Context{}, map[string]any{
			"source": map[string]any{
				"account": map[string]any{
					"id": "source",
				},
			},
			"destination": map[string]any{
				"payment": map[string]any{
					"psp":      "stripe",
					"metadata": "formanceAccountID",
				},
			},
			"amount": map[string]any{
				"amount": float64(100),
				"asset":  "USD/2",
			},
		}, "send")
		require.NoError(t, err)
		require.Error(t, schema.ValidateRequirements(resolved))
	})
}

func stringPtr(value string) *string {
	return &value
}
