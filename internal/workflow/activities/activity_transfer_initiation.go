package activities

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

type CreateTransferInitiationRequest struct {
	Amount      *big.Int `json:"amount,omitempty"`
	Asset       *string  `json:"asset,omitempty"`
	ConnectorID *string  `json:"connectorID,omitempty"`
	// Provider is the PSP name (e.g., "stripe", "wise", "mangopay", etc.)
	// Passed directly to the Payments service which validates supported providers.
	Provider    *string `json:"provider,omitempty"`
	Destination *string `json:"destination,omitempty"`
	// Source is optional - only required for TRANSFER type (internal to internal)
	Source *string `json:"source,omitempty"`
	// Type is either "TRANSFER" (internal to internal) or "PAYOUT" (internal to external)
	// Defaults to "TRANSFER" if not specified.
	Type string `json:"type,omitempty"`
	// Description for the transfer initiation
	Description string `json:"description,omitempty"`
	// A set of key/value pairs that you can attach to a transfer object.
	// It can be useful for storing additional information about the transfer in a structured format.
	Metadata          map[string]string `json:"metadata"`
	WaitingValidation *bool             `default:"false" json:"waitingValidation"`
}

func (a Activities) CreateTransferInitiation(ctx context.Context, request CreateTransferInitiationRequest) error {
	validated := request.WaitingValidation == nil || !*request.WaitingValidation
	if request.ConnectorID == nil || *request.ConnectorID == "" {
		return fmt.Errorf("connectorID is required")
	}

	activityInfo := activity.GetInfo(ctx)

	// Determine the transfer type - default to TRANSFER
	transferType := shared.TransferInitiationRequestTypeTransfer
	if request.Type != "" {
		switch strings.ToUpper(request.Type) {
		case "PAYOUT":
			transferType = shared.TransferInitiationRequestTypePayout
		case "TRANSFER":
			transferType = shared.TransferInitiationRequestTypeTransfer
		default:
			return fmt.Errorf("invalid transfer type: %s (must be TRANSFER or PAYOUT)", request.Type)
		}
	}

	// Build the transfer initiation request
	ti := shared.TransferInitiationRequest{
		Amount:               request.Amount,
		Asset:                *request.Asset,
		DestinationAccountID: *request.Destination,
		Description:          request.Description,
		ConnectorID:          request.ConnectorID,
		Type:                 transferType,
		Reference:            activityInfo.WorkflowExecution.ID + activityInfo.ActivityID,
		Validated:            validated,
		Metadata:             request.Metadata,
	}

	// Set source account if provided (required for TRANSFER type)
	if request.Source != nil {
		ti.SourceAccountID = *request.Source
	}

	// Set provider if specified - pass directly to Payments service for validation
	if request.Provider != nil && *request.Provider != "" {
		// Normalize to uppercase to match connector naming convention
		connector := shared.Connector(strings.ToUpper(*request.Provider))
		ti.Provider = &connector
	}

	// Set default description if not provided
	if ti.Description == "" {
		if request.Provider != nil {
			ti.Description = fmt.Sprintf("%s %s", *request.Provider, transferType)
		} else {
			ti.Description = fmt.Sprintf("Transfer Initiation (%s)", transferType)
		}
	}

	_, err := a.client.Payments.V1.CreateTransferInitiation(ctx, ti)
	if err != nil {
		return err
	}

	return nil
}

var CreateTransferInitiationActivity = Activities{}.CreateTransferInitiation

func CreateTransferInitiation(ctx workflow.Context, request CreateTransferInitiationRequest) error {
	return executeActivity(ctx, CreateTransferInitiationActivity, nil, request)
}
