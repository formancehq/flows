package activities

import (
	"context"
	"math/big"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// stripeProvider is passed to resolveConnectorIDV1 in place of the old Provider request
// field - see the comment on StripeTransfer below.
var stripeProvider = string(shared.ConnectorStripe)

type StripeTransferRequest struct {
	Amount      *big.Int `json:"amount,omitempty"`
	Asset       *string  `json:"asset,omitempty"`
	ConnectorID *string  `json:"connectorID,omitempty"`
	Destination *string  `json:"destination,omitempty"`
	// A set of key/value pairs that you can attach to a transfer object.
	// It can be useful for storing additional information about the transfer in a structured format.
	//
	Metadata          map[string]string `json:"metadata"`
	WaitingValidation *bool             `default:"false" json:"waitingValidation"`
}

func (a Activities) StripeTransfer(ctx context.Context, request StripeTransferRequest) error {
	validated := request.WaitingValidation == nil || !*request.WaitingValidation

	activityInfo := activity.GetInfo(ctx)

	// The request used to carry Provider=STRIPE as a hint for the server to resolve the
	// connector when ConnectorID was left unset (confirmed against the v2.1.0 payments
	// service: cmd/connectors/internal/api/service/transfer_initiation.go resolved
	// ConnectorID from Provider via ListConnectorsByProvider, erroring on 0 or >1 matches).
	// The current SDK's TransferInitiationRequest dropped that field entirely, so resolve it
	// the same way client-side instead of silently sending no connector at all.
	connectorID, err := a.resolveConnectorIDV1(ctx, request.ConnectorID, &stripeProvider)
	if err != nil {
		return err
	}

	ti := shared.TransferInitiationRequest{
		Amount:               request.Amount,
		Asset:                *request.Asset,
		DestinationAccountID: *request.Destination,
		Description:          "Stripe Transfer",
		ConnectorID:          &connectorID,
		Type:                 shared.TransferInitiationRequestTypeTransfer,
		Reference:            activityInfo.WorkflowExecution.ID + activityInfo.ActivityID,
		Validated:            validated,
	}

	_, err = a.client.Payments.V1.CreateTransferInitiation(ctx, ti)
	if err != nil {
		return err
	}

	return nil
}

var StripeTransferActivity = Activities{}.StripeTransfer

func StripeTransfer(ctx workflow.Context, request StripeTransferRequest) error {
	return executeActivity(ctx, StripeTransferActivity, nil, request)
}
