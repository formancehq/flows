package send

import (
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/formancehq/go-libs/v3/metadata"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/orchestration/internal/schema"
	"github.com/formancehq/orchestration/internal/workflow/stages"
)

type WalletReference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WalletSource struct {
	WalletReference
	Balance string `json:"balance" spec:"default:main" validate:"required"`
}

type WalletDestination = WalletSource

type LedgerAccountSource struct {
	ID     string `json:"id" validate:"required"`
	Ledger string `json:"ledger" spec:"default:default" validate:"required"`
	// ThroughAccount is used when this account interacts with external systems (payments, cross-ledger).
	// - As SOURCE going to payment: funds are sent to this account (e.g., "liabilities:payouts-pending")
	// - As DESTINATION from payment: funds come from this account (e.g., "assets:stripe:incoming")
	// - For cross-ledger transfers: replaces "world" on both sides
	// Defaults to "world"
	ThroughAccount string `json:"throughAccount" spec:"default:world"`
	// AllowOverdraft enables unbounded overdraft on the throughAccount when set to true.
	// This is useful when the throughAccount represents a liability or bridge account that
	// needs to go negative (e.g., "liabilities:payouts-pending").
	// Only applies when throughAccount is not "world" (which already has unbounded overdraft).
	// Defaults to false.
	AllowOverdraft bool `json:"allowOverdraft" spec:"default:false"`
}

type LedgerAccountDestination = LedgerAccountSource

type PaymentSource struct {
	ID string `json:"id" validate:"required"`
	// Ledger specifies which ledger to use for payment ingestion.
	// Defaults to the internal orchestration ledger ("orchestration-000-internal").
	Ledger string `json:"ledger,omitempty"`
	// HoldingAccount is the intermediate account where payment funds are held.
	// Defaults to "payment:{paymentID}" format.
	HoldingAccount string `json:"holdingAccount,omitempty"`
	// ThroughAccount is the source account for the payment ingestion transaction.
	// Defaults to "world".
	ThroughAccount string `json:"throughAccount" spec:"default:world"`
	// AllowOverdraft enables unbounded overdraft on the throughAccount when set to true.
	// Only applies when throughAccount is not "world" (which already has unbounded overdraft).
	// Defaults to false.
	AllowOverdraft bool `json:"allowOverdraft" spec:"default:false"`
}

type PaymentDestination struct {
	// PSP is the payment service provider name (e.g., "stripe", "wise", "mangopay", "modulr", etc.)
	PSP string `json:"psp"`
	// Type is either "TRANSFER" (internal to internal) or "PAYOUT" (internal to external)
	// Defaults to "TRANSFER" if not specified.
	Type string `json:"type" spec:"default:TRANSFER"`
	// SourceAccount is the Formance Payments account ID for the source (internal PSP account).
	// If not specified, the Payments service may use a default account for the connector.
	SourceAccount *string `json:"sourceAccount,omitempty"`
	// Metadata is the key to look up in the source wallet/account metadata to get the destination account ID.
	Metadata          string  `json:"metadata" spec:"default:formanceAccountID"`
	WaitingValidation bool    `json:"waitingValidation" spec:"default:false"`
	ConnectorID       *string `json:"connectorId,omitempty"`
}

type Source struct {
	Wallet  *WalletSource        `json:"wallet,omitempty"`
	Account *LedgerAccountSource `json:"account,omitempty"`
	Payment *PaymentSource       `json:"payment,omitempty"`
}

func NewSource() *Source {
	return &Source{}
}

func (s Source) WithWallet(src *WalletSource) Source {
	s.Wallet = src
	return s
}

func (s Source) WithPayment(src *PaymentSource) Source {
	s.Payment = src
	return s
}

func (s Source) WithAccount(src *LedgerAccountSource) Source {
	s.Account = src
	return s
}

type Destination struct {
	Wallet  *WalletDestination        `json:"wallet,omitempty"`
	Account *LedgerAccountDestination `json:"account,omitempty"`
	Payment *PaymentDestination       `json:"payment,omitempty"`
}

func NewDestination() *Destination {
	return &Destination{}
}

func (s Destination) WithWallet(src *WalletDestination) Destination {
	s.Wallet = src
	return s
}

func (s Destination) WithPayment(src *PaymentDestination) Destination {
	s.Payment = src
	return s
}

func (s Destination) WithAccount(src *LedgerAccountDestination) Destination {
	s.Account = src
	return s
}

type Send struct {
	Source      Source            `json:"source"`
	Destination Destination       `json:"destination"`
	Amount      *shared.Monetary  `json:"amount,omitempty"`
	Metadata    metadata.Metadata `json:"metadata,omitempty"`
	Timestamp   *time.Time        `json:"timestamp"`
}

func (s Send) GetWorkflow() any {
	return RunSend
}

func init() {
	schema.RegisterOneOf(&Source{}, &Destination{}, &WalletReference{})
	stages.Register("send", Send{})
}
