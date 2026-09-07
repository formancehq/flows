package activities

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/formancehq/formance-sdk-go/v5/pkg/models/payments"
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

// classifyPaymentError converts a payments API error into a temporal.ApplicationError. Every
// declared error code maps to an HTTP 4xx except INTERNAL (500), and a 4xx won't succeed by
// retrying the exact same request, so it's marked non-retryable directly here. formance-sdk-go
// v5.0.1's PaymentsErrorsEnum declares CONFLICT, INVALID_ID, MISSING_OR_INVALID_BODY and
// CONNECTOR_CAPABILITY_NOT_SUPPORTED alongside INTERNAL/VALIDATION/NOT_FOUND - v3.8.1's enum only
// had the latter three, so those four codes used to fail PaymentsErrorsEnum.UnmarshalJSON before
// this function was ever reached and fell through as a plain, unclassified retryable error.
func classifyPaymentError(err *payments.PaymentsErrorResponse) error {
	if err.ErrorCode == payments.PaymentsErrorsEnumInternal {
		return temporal.NewApplicationError(err.ErrorMessage, string(err.ErrorCode))
	}
	return temporal.NewNonRetryableApplicationError(err.ErrorMessage, string(err.ErrorCode), nil)
}

// parseTransferType maps the workflow-facing "TRANSFER"/"PAYOUT" string onto the SDK's enum type.
func parseTransferType(requestType string) (payments.TransferInitiationRequestType, error) {
	if requestType == "" {
		return payments.TransferInitiationRequestTypeTransfer, nil
	}
	switch strings.ToUpper(requestType) {
	case "PAYOUT":
		return payments.TransferInitiationRequestTypePayout, nil
	case "TRANSFER":
		return payments.TransferInitiationRequestTypeTransfer, nil
	default:
		return "", fmt.Errorf("invalid transfer type: %s (must be TRANSFER or PAYOUT)", requestType)
	}
}

// defaultDescription returns request.Description unchanged when set, otherwise a fallback
// mentioning the provider when known.
func defaultDescription(description string, provider *string, transferType payments.TransferInitiationRequestType) string {
	if description != "" {
		return description
	}
	if provider != nil {
		return fmt.Sprintf("%s %s", *provider, transferType)
	}
	return fmt.Sprintf("Transfer Initiation (%s)", transferType)
}

// connectorIDOrProviderRequired validates the (connectorID, provider) pair: connectorID wins when
// set, otherwise a non-empty provider is required to look one up. resolved reports whether id is
// already the final answer - on false with a nil error, the caller must list connectors and
// resolve provider matches itself.
func connectorIDOrProviderRequired(connectorID, provider *string) (id string, resolved bool, err error) {
	if connectorID != nil && *connectorID != "" {
		return *connectorID, true, nil
	}
	if provider == nil || *provider == "" {
		return "", false, temporal.NewNonRetryableApplicationError("either connectorID or provider must be specified", "VALIDATION", nil)
	}
	return "", false, nil
}

// resolveProviderMatch applies the 0/1/many match-count cascade once resolveConnectorID has built
// its own list of connector IDs matching provider.
func resolveProviderMatch(matches []string, provider string) (string, error) {
	switch len(matches) {
	case 0:
		return "", temporal.NewNonRetryableApplicationError(fmt.Sprintf("no connector installed for provider %q", provider), "VALIDATION", nil)
	case 1:
		return matches[0], nil
	default:
		return "", temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("%d connectors installed for provider %q, specify connectorID explicitly", len(matches), provider),
			"VALIDATION", nil,
		)
	}
}

// CreateTransferInitiation always targets the legacy v1 transfer-initiations API. This used to
// pick between a v1 and a v3 (payment-initiations) path based on the target stack's payments
// module version, but the v3 path was dropped after it was found to silently swallow failures
// instead of reporting them. It does not self-heal on CONFLICT by fetching the existing record -
// v1's list/query DSL differs from v3's query builder that self-heal was built against, and
// guessing at it risks a subtly wrong filter - but CONFLICT and every other payments error code
// is properly decodable and classified now (see classifyPaymentError).
func (a Activities) CreateTransferInitiation(ctx context.Context, request CreateTransferInitiationRequest) error {
	validated := request.WaitingValidation == nil || !*request.WaitingValidation

	activityInfo := activity.GetInfo(ctx)

	transferType, err := parseTransferType(request.Type)
	if err != nil {
		return err
	}

	// The v2.1.0-era payments service (the actual population this targets) resolved ConnectorID
	// from Provider server-side when ConnectorID was left unset (see
	// cmd/connectors/internal/api/service/transfer_initiation.go in formancehq/stack
	// releases/v2.1.0: ListConnectorsByProvider, erroring on 0 or >1 matches). The current SDK's
	// TransferInitiationRequest no longer has a Provider field to carry that hint, so
	// resolveConnectorID replicates the same resolution client-side instead.
	connectorID, err := a.resolveConnectorID(ctx, request.ConnectorID, request.Provider)
	if err != nil {
		return err
	}

	description := defaultDescription(request.Description, request.Provider, transferType)

	ti := payments.TransferInitiationRequest{
		Amount:               request.Amount,
		Asset:                *request.Asset,
		DestinationAccountID: *request.Destination,
		Description:          description,
		ConnectorID:          &connectorID,
		Type:                 transferType,
		// Reference is the idempotency key: stable across retries of this same activity
		// invocation (RunID doesn't change across activity retries within one workflow
		// execution), so a create that conflicts means a previous attempt already reached the
		// payments service. RunID is required, not just WorkflowID: a Temporal reset restarts
		// the same WorkflowID under a new RunID, and WorkflowID alone would then collide with
		// the payment initiation the pre-reset run already created, silently skipping what
		// should be a fresh attempt. Matches the RunID + ActivityID scheme getIK already uses
		// for the same reason (see activity.go).
		Reference: activityInfo.WorkflowExecution.RunID + activityInfo.ActivityID,
		Validated: validated,
		Metadata:  request.Metadata,
	}
	if request.Source != nil {
		ti.SourceAccountID = *request.Source
	}

	_, err = a.client.Payments.V1.CreateTransferInitiation(ctx, ti)
	if err != nil {
		if pErr, ok := err.(*payments.PaymentsErrorResponse); ok {
			return classifyPaymentError(pErr)
		}
		return err
	}

	return nil
}

// resolveConnectorID returns connectorID unchanged when set, otherwise resolves it from provider.
// v1 has no filtered connector listing, so it lists everything and matches provider client-side.
func (a Activities) resolveConnectorID(ctx context.Context, connectorID, provider *string) (string, error) {
	if id, resolved, err := connectorIDOrProviderRequired(connectorID, provider); resolved || err != nil {
		return id, err
	}

	resp, err := a.client.Payments.V1.ListAllConnectors(ctx)
	if err != nil {
		if pErr, ok := err.(*payments.PaymentsErrorResponse); ok {
			return "", classifyPaymentError(pErr)
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

	return resolveProviderMatch(matches, *provider)
}

var CreateTransferInitiationActivity = Activities{}.CreateTransferInitiation

func CreateTransferInitiation(ctx workflow.Context, request CreateTransferInitiationRequest) error {
	return executeActivity(ctx, CreateTransferInitiationActivity, nil, request)
}
