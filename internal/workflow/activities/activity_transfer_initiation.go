package activities

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/sdkerrors"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type CreateTransferInitiationRequest struct {
	Amount      *big.Int `json:"amount,omitempty"`
	Asset       *string  `json:"asset,omitempty"`
	ConnectorID *string  `json:"connectorID,omitempty"`
	// Provider is the PSP name (e.g., "stripe", "wise", "mangopay", etc.).
	// Used to resolve ConnectorID via resolveConnectorID when ConnectorID is not set directly.
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

// CreateTransferInitiation picks between the v3 payment-initiations API and the legacy v1
// transfer-initiations API based on the target stack's payments module version (see
// getPaymentsVersion) - v3 only exists from payments major release v3 onward.
func (a Activities) CreateTransferInitiation(ctx context.Context, request CreateTransferInitiationRequest) error {
	pv, err := a.getPaymentsVersion(ctx)
	if err != nil {
		return fmt.Errorf("checking payments version: %w", err)
	}

	if !pv.supportsV3 {
		return a.createTransferInitiationV1(ctx, request)
	}
	return a.createTransferInitiationV3(ctx, request)
}

func (a Activities) createTransferInitiationV3(ctx context.Context, request CreateTransferInitiationRequest) error {
	validated := request.WaitingValidation == nil || !*request.WaitingValidation

	activityInfo := activity.GetInfo(ctx)

	transferType := shared.V3PaymentInitiationTypeEnumTransfer
	if request.Type != "" {
		switch strings.ToUpper(request.Type) {
		case "PAYOUT":
			transferType = shared.V3PaymentInitiationTypeEnumPayout
		case "TRANSFER":
			transferType = shared.V3PaymentInitiationTypeEnumTransfer
		default:
			return fmt.Errorf("invalid transfer type: %s (must be TRANSFER or PAYOUT)", request.Type)
		}
	}

	connectorID, err := a.resolveConnectorID(ctx, request.ConnectorID, request.Provider)
	if err != nil {
		return err
	}

	description := request.Description
	if description == "" {
		if request.Provider != nil {
			description = fmt.Sprintf("%s %s", *request.Provider, transferType)
		} else {
			description = fmt.Sprintf("Transfer Initiation (%s)", transferType)
		}
	}

	// Reference is the idempotency key: stable across retries of this same activity invocation,
	// so a create that conflicts means a previous attempt already reached the payments service.
	reference := activityInfo.WorkflowExecution.ID + activityInfo.ActivityID

	_, err = a.client.Payments.V3.InitiatePayment(ctx, operations.V3InitiatePaymentRequest{
		V3InitiatePaymentRequest: &shared.V3InitiatePaymentRequest{
			Amount:               request.Amount,
			Asset:                *request.Asset,
			ConnectorID:          connectorID,
			Description:          description,
			DestinationAccountID: request.Destination,
			Metadata:             request.Metadata,
			Reference:            reference,
			SourceAccountID:      request.Source,
			Type:                 transferType,
		},
		// NoValidation mirrors the same flag the v1/v2 API exposed as the "validated" body field:
		// true skips the manual validation step and forwards the request to the PSP directly.
		NoValidation: &validated,
	})
	if err == nil {
		return nil
	}

	v3Err, ok := err.(*sdkerrors.V3ErrorResponse)
	if !ok {
		return err
	}

	if v3Err.ErrorCode != shared.V3ErrorsEnumConflict {
		return temporal.NewApplicationError(v3Err.ErrorMessage, string(v3Err.ErrorCode), v3Err.Details)
	}

	// A payment initiation with this reference already exists - most likely a previous attempt
	// reached the payments service and was recorded, but its response never made it back here
	// (e.g. we hit our own deadline first). Retrying the create would only hit the same conflict
	// again, so fetch the existing record instead of retrying.
	existing, ferr := a.getPaymentInitiationByReference(ctx, reference)
	if ferr != nil {
		// Could not confirm the existing record - surface the original conflict rather than
		// silently retrying it forever.
		return temporal.NewApplicationError(v3Err.ErrorMessage, string(v3Err.ErrorCode), v3Err.Details)
	}

	switch existing.Status {
	case shared.V3PaymentInitiationStatusEnumFailed, shared.V3PaymentInitiationStatusEnumRejected:
		msg := fmt.Sprintf("payment initiation %s already exists and is in a terminal failure state (%s)", existing.ID, existing.Status)
		if existing.Error != nil {
			msg = fmt.Sprintf("%s: %s", msg, *existing.Error)
		}
		return temporal.NewApplicationError(msg, string(shared.V3ErrorsEnumConflict))
	default:
		// Already created, and not in a failure state - treat this as success.
		return nil
	}
}

// resolveConnectorID returns connectorID unchanged when set, otherwise resolves it from provider
// by listing installed connectors. Errors when zero or more than one connector matches the
// provider - callers with multiple connectors for the same provider must pass connectorID
// explicitly.
func (a Activities) resolveConnectorID(ctx context.Context, connectorID, provider *string) (string, error) {
	if connectorID != nil && *connectorID != "" {
		return *connectorID, nil
	}
	if provider == nil || *provider == "" {
		return "", temporal.NewApplicationError("either connectorID or provider must be specified", "VALIDATION")
	}

	resp, err := a.client.Payments.V3.ListConnectors(ctx, operations.V3ListConnectorsRequest{
		Query: map[string]any{
			"$match": map[string]any{
				"provider": *provider,
			},
		},
	})
	if err != nil {
		if v3Err, ok := err.(*sdkerrors.V3ErrorResponse); ok {
			return "", temporal.NewApplicationError(v3Err.ErrorMessage, string(v3Err.ErrorCode), v3Err.Details)
		}
		return "", err
	}

	connectors := resp.V3ConnectorsCursorResponse.Cursor.Data
	switch len(connectors) {
	case 0:
		return "", temporal.NewApplicationError(fmt.Sprintf("no connector installed for provider %q", *provider), "VALIDATION")
	case 1:
		return connectors[0].ID, nil
	default:
		return "", temporal.NewApplicationError(
			fmt.Sprintf("%d connectors installed for provider %q, specify connectorID explicitly", len(connectors), *provider),
			"VALIDATION",
		)
	}
}

// createTransferInitiationV1 is the fallback for stacks whose payments module predates the
// v3 payment-initiations API (getPaymentsVersion reports major < 3). It mirrors the original
// v1 CreateTransferInitiation call. Unlike v3, it does not self-heal on CONFLICT by fetching
// the existing record - v1's list/query DSL differs from v3's query builder used elsewhere in
// this file, and guessing at it risks a subtly wrong filter. It does still classify SDK errors
// into a typed ApplicationError so the existing NonRetryableErrorTypes policy (VALIDATION,
// CONFLICT, ...) actually engages instead of retrying a terminal error forever.
func (a Activities) createTransferInitiationV1(ctx context.Context, request CreateTransferInitiationRequest) error {
	validated := request.WaitingValidation == nil || !*request.WaitingValidation

	activityInfo := activity.GetInfo(ctx)

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

	// The v2.1.0-era payments service (the actual population this fallback targets) resolved
	// ConnectorID from Provider server-side when ConnectorID was left unset (see
	// cmd/connectors/internal/api/service/transfer_initiation.go in formancehq/stack
	// releases/v2.1.0: ListConnectorsByProvider, erroring on 0 or >1 matches). The current
	// SDK's TransferInitiationRequest no longer has a Provider field to carry that hint, so
	// resolveConnectorIDV1 replicates the same resolution client-side instead.
	connectorID, err := a.resolveConnectorIDV1(ctx, request.ConnectorID, request.Provider)
	if err != nil {
		return err
	}

	description := request.Description
	if description == "" {
		if request.Provider != nil {
			description = fmt.Sprintf("%s %s", *request.Provider, transferType)
		} else {
			description = fmt.Sprintf("Transfer Initiation (%s)", transferType)
		}
	}

	ti := shared.TransferInitiationRequest{
		Amount:               request.Amount,
		Asset:                *request.Asset,
		DestinationAccountID: *request.Destination,
		Description:          description,
		ConnectorID:          &connectorID,
		Type:                 transferType,
		Reference:            activityInfo.WorkflowExecution.ID + activityInfo.ActivityID,
		Validated:            validated,
		Metadata:             request.Metadata,
	}
	if request.Source != nil {
		ti.SourceAccountID = *request.Source
	}

	_, err = a.client.Payments.V1.CreateTransferInitiation(ctx, ti)
	if err != nil {
		if v1Err, ok := err.(*sdkerrors.PaymentsErrorResponse); ok {
			return temporal.NewApplicationError(v1Err.ErrorMessage, string(v1Err.ErrorCode))
		}
		return err
	}

	return nil
}

// resolveConnectorIDV1 is resolveConnectorID's counterpart for stacks without the v3 API:
// v1 has no filtered connector listing, so it lists everything and matches provider client-side.
func (a Activities) resolveConnectorIDV1(ctx context.Context, connectorID, provider *string) (string, error) {
	if connectorID != nil && *connectorID != "" {
		return *connectorID, nil
	}
	if provider == nil || *provider == "" {
		return "", temporal.NewApplicationError("either connectorID or provider must be specified", "VALIDATION")
	}

	resp, err := a.client.Payments.V1.ListAllConnectors(ctx)
	if err != nil {
		if v1Err, ok := err.(*sdkerrors.PaymentsErrorResponse); ok {
			return "", temporal.NewApplicationError(v1Err.ErrorMessage, string(v1Err.ErrorCode))
		}
		return "", err
	}

	var matches []string
	if resp.ConnectorsResponse != nil {
		for _, c := range resp.ConnectorsResponse.Data {
			if strings.EqualFold(string(c.Provider), *provider) {
				matches = append(matches, c.ConnectorID)
			}
		}
	}

	switch len(matches) {
	case 0:
		return "", temporal.NewApplicationError(fmt.Sprintf("no connector installed for provider %q", *provider), "VALIDATION")
	case 1:
		return matches[0], nil
	default:
		return "", temporal.NewApplicationError(
			fmt.Sprintf("%d connectors installed for provider %q, specify connectorID explicitly", len(matches), *provider),
			"VALIDATION",
		)
	}
}

func (a Activities) getPaymentInitiationByReference(ctx context.Context, reference string) (*shared.V3PaymentInitiation, error) {
	resp, err := a.client.Payments.V3.ListPaymentInitiations(ctx, operations.V3ListPaymentInitiationsRequest{
		Query: map[string]any{
			"$match": map[string]any{
				"reference": reference,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	data := resp.V3PaymentInitiationsCursorResponse.Cursor.Data
	if len(data) != 1 {
		return nil, fmt.Errorf("expected exactly one payment initiation for reference %q, found %d", reference, len(data))
	}
	return &data[0], nil
}

var CreateTransferInitiationActivity = Activities{}.CreateTransferInitiation

func CreateTransferInitiation(ctx workflow.Context, request CreateTransferInitiationRequest) error {
	return executeActivity(ctx, CreateTransferInitiationActivity, nil, request)
}
