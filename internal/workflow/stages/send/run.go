package send

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/formancehq/go-libs/v5/pkg/types/time"

	collectionutils "github.com/formancehq/go-libs/v5/pkg/types/collections"
	"github.com/formancehq/go-libs/v5/pkg/types/metadata"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/formancehq/orchestration/internal/workflow/activities"
	"github.com/formancehq/orchestration/internal/workflow/stages/internal"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	internalLedger         = "orchestration-000-internal"
	moveToLedgerMetadata   = "orchestration/move-to-ledger"
	moveFromLedgerMetadata = "orchestration/move-from-ledger"
)

// generateNumscriptWithSourceOverdraft creates a Numscript program with unbounded overdraft on the source.
// This is used when the source account may not have sufficient funds (e.g., bridge accounts, asset tracking accounts).
// Note: In Numscript, overdraft only applies to sources, not destinations.
func generateNumscriptWithSourceOverdraft(source, destination, asset string, amount string) string {
	return fmt.Sprintf(`send [%s %s] (
  source = @%s allowing unbounded overdraft
  destination = @%s
)`, asset, amount, source, destination)
}

func extractFormanceAccountID[V any](metadataKey string, metadata map[string]V) (string, error) {
	formanceAccountID, ok := metadata[metadataKey]
	if !ok {
		return "", fmt.Errorf("expected '%s' metadata containing formance account ID", metadataKey)
	}
	if reflect.ValueOf(formanceAccountID).IsZero() {
		return "", errors.New("formance account ID empty")
	}
	return fmt.Sprint(formanceAccountID), nil
}

func justError[T any](v T, err error) error {
	return err
}

func getWalletFromReference(ctx workflow.Context, ref WalletReference) (*shared.Wallet, error) {
	if ref.ID != "" {
		walletSource, err := activities.GetWallet(internal.InfiniteRetryContext(ctx), ref.ID)
		if err != nil {
			return nil, err
		}
		return &shared.Wallet{
			CreatedAt: walletSource.CreatedAt,
			ID:        walletSource.ID,
			Ledger:    walletSource.Ledger,
			Metadata:  walletSource.Metadata,
			Name:      walletSource.Name,
		}, nil
	} else {
		wallets, err := activities.ListWallets(internal.InfiniteRetryContext(ctx), activities.ListWalletsRequest{
			Name: ref.Name,
		})
		if err != nil {
			return nil, err
		}
		switch len(wallets.Cursor.Data) {
		case 0:
			return nil, errors.New("wallet not found")
		case 1:
			return &wallets.Cursor.Data[0], nil
		default:
			return nil, errors.New("found multiple wallets with the same name")
		}
	}
}

func RunSend(ctx workflow.Context, send Send) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors.WithStack(fmt.Errorf("%s", e))
		}
	}()
	amount := send.Amount
	metadata := send.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}
	switch {
	case send.Source.Account != nil && send.Destination.Account != nil:
		return runAccountToAccount(ctx, send.Timestamp, send.Source.Account, send.Destination.Account, amount, metadata)
	case send.Source.Account != nil && send.Destination.Payment != nil:
		return runAccountToPayment(ctx, send.Timestamp, send.Source.Account, send.Destination.Payment, amount, metadata)
	case send.Source.Account != nil && send.Destination.Wallet != nil:
		return runAccountToWallet(ctx, send.Timestamp, send.Source.Account, send.Destination.Wallet, amount, metadata)
	case send.Source.Wallet != nil && send.Destination.Account != nil:
		return runWalletToAccount(ctx, send.Timestamp, send.Source.Wallet, send.Destination.Account, amount, send.Metadata)
	case send.Source.Wallet != nil && send.Destination.Payment != nil:
		return runWalletToPayment(ctx, send.Timestamp, send.Source.Wallet, send.Destination.Payment, amount, metadata)
	case send.Source.Wallet != nil && send.Destination.Wallet != nil:
		return runWalletToWallet(ctx, send.Timestamp, send.Source.Wallet, send.Destination.Wallet, amount, metadata)
	case send.Source.Payment != nil && send.Destination.Account != nil:
		return runPaymentToAccount(ctx, send.Timestamp, send.Source.Payment, send.Destination.Account, amount, metadata)
	case send.Source.Payment != nil && send.Destination.Wallet != nil:
		return runPaymentToWallet(ctx, send.Timestamp, send.Source.Payment, send.Destination.Wallet, amount, metadata)
	case send.Source.Payment != nil && send.Destination.Payment != nil:
		return errors.New("send from payment to payment is not supported")
	}
	panic("should not happen")
}

func runPaymentToWallet(ctx workflow.Context, timestamp *time.Time, source *PaymentSource, destination *WalletSource, amount *shared.Monetary, m metadata.Metadata) error {
	payment, err := savePayment(ctx, timestamp, source, m)
	if err != nil {
		return err
	}
	if amount == nil {
		amount = &shared.Monetary{
			Amount: payment.InitialAmount,
			Asset:  payment.Asset,
		}
	}
	// Determine the ledger and holding account for the payment
	ledger := source.Ledger
	if ledger == "" {
		ledger = internalLedger
	}
	holdingAccount := source.HoldingAccount
	if holdingAccount == "" {
		holdingAccount = paymentAccountName(source.ID)
	}
	return runAccountToWallet(ctx, timestamp, &LedgerAccountSource{
		ID:             holdingAccount,
		Ledger:         ledger,
		ThroughAccount: "world", // The payment was already ingested, no need for custom throughAccount here
	}, destination, amount, m)
}

func paymentAccountName(paymentID string) string {
	paymentID = strings.ReplaceAll(paymentID, "-", "_")
	return fmt.Sprintf("payment:%s", paymentID)
}

func savePayment(ctx workflow.Context, timestamp *time.Time, source *PaymentSource, m metadata.Metadata) (*shared.Payment, error) {
	payment, err := activities.GetPayment(internal.InfiniteRetryContext(ctx), source.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving payment: %s", source.ID)
	}

	// Determine ledger, holding account, and through account
	ledger := source.Ledger
	if ledger == "" {
		ledger = internalLedger
	}
	holdingAccount := source.HoldingAccount
	if holdingAccount == "" {
		holdingAccount = paymentAccountName(source.ID)
	}
	throughAccount := source.ThroughAccount
	if throughAccount == "" {
		throughAccount = "world"
	}

	reference := holdingAccount

	// Use Numscript with overdraft if allowOverdraft is true and throughAccount is not "world"
	// Here throughAccount is the SOURCE, so overdraft makes sense
	var txRequest activities.PostTransaction
	if source.AllowOverdraft && throughAccount != "world" {
		script := generateNumscriptWithSourceOverdraft(
			throughAccount, holdingAccount,
			payment.Asset, payment.InitialAmount.String(),
		)
		txRequest = activities.PostTransaction{
			Script: &shared.V2PostTransactionScript{
				Plain: script,
			},
			Timestamp: timestamp,
			Metadata:  m,
			Reference: &reference,
		}
	} else {
		txRequest = activities.PostTransaction{
			Postings: []shared.V2Posting{{
				Amount:      payment.InitialAmount,
				Asset:       payment.Asset,
				Destination: holdingAccount,
				Source:      throughAccount,
			}},
			Timestamp: timestamp,
			Metadata:  m,
			Reference: &reference,
		}
	}

	_, err = activities.CreateTransaction(internal.InfiniteRetryContext(ctx), ledger, txRequest)
	if err != nil {
		applicationError := &temporal.ApplicationError{}
		if errors.As(err, &applicationError) {
			if applicationError.Type() != "CONFLICT" {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return payment, nil
}

func runPaymentToAccount(ctx workflow.Context, timestamp *time.Time, source *PaymentSource, destination *LedgerAccountDestination, amount *shared.Monetary, m metadata.Metadata) error {
	payment, err := savePayment(ctx, timestamp, source, m)
	if err != nil {
		return err
	}
	if amount == nil {
		amount = &shared.Monetary{
			Amount: payment.InitialAmount,
			Asset:  payment.Asset,
		}
	}
	// Determine the ledger and holding account for the payment
	ledger := source.Ledger
	if ledger == "" {
		ledger = internalLedger
	}
	holdingAccount := source.HoldingAccount
	if holdingAccount == "" {
		holdingAccount = paymentAccountName(source.ID)
	}
	return runAccountToAccount(ctx, timestamp, &LedgerAccountSource{
		ID:             holdingAccount,
		Ledger:         ledger,
		ThroughAccount: "world", // The payment was already ingested, no throughAccount needed for intermediate transfer
	}, destination, amount, m)
}

func runWalletToWallet(ctx workflow.Context, timestamp *time.Time, source *WalletSource, destination *WalletDestination, amount *shared.Monetary, m metadata.Metadata) error {
	if amount == nil {
		return errors.New("amount must be specified")
	}
	sourceWallet, err := getWalletFromReference(ctx, source.WalletReference)
	if err != nil {
		return err
	}
	destinationWallet, err := getWalletFromReference(ctx, destination.WalletReference)
	if err != nil {
		return err
	}
	if sourceWallet.Ledger == destinationWallet.Ledger {
		mainBalance := "main"
		sourceSubject := shared.WalletSubject{
			Balance:    &mainBalance,
			Identifier: sourceWallet.ID,
			Type:       "WALLET",
		}
		return activities.CreditWallet(internal.InfiniteRetryContext(ctx), destinationWallet.ID, &activities.CreditWalletRequestPayload{
			Amount:    *amount,
			Balance:   &destination.Balance,
			Metadata:  m,
			Sources:   []shared.Subject{{WalletSubject: &sourceSubject}},
			Timestamp: timestamp,
		})
	}

	if err := justError(activities.DebitWallet(internal.InfiniteRetryContext(ctx), sourceWallet.ID, &activities.DebitWalletRequestPayload{
		Amount:    *amount,
		Balances:  []string{source.Balance},
		Timestamp: timestamp,
		Metadata: collectionutils.MergeMaps(m, map[string]string{
			moveToLedgerMetadata: destinationWallet.Ledger,
		}),
	})); err != nil {
		return err
	}

	return activities.CreditWallet(internal.InfiniteRetryContext(ctx), destinationWallet.ID, &activities.CreditWalletRequestPayload{
		Amount:    *amount,
		Balance:   &destination.Balance,
		Timestamp: timestamp,
		Metadata: collectionutils.MergeMaps(m, map[string]string{
			moveFromLedgerMetadata: sourceWallet.Ledger,
		}),
	})
}

func runWalletToPayment(ctx workflow.Context, timestamp *time.Time, source *WalletSource, destination *PaymentDestination, amount *shared.Monetary, m metadata.Metadata) error {
	if amount == nil {
		return errors.New("amount must be specified")
	}
	sourceWallet, err := getWalletFromReference(ctx, source.WalletReference)
	if err != nil {
		return errors.Wrapf(err, "reading account: %s", source.ID)
	}

	formanceAccountID, err := extractFormanceAccountID(destination.Metadata, sourceWallet.Metadata)
	if err != nil {
		return err
	}

	// IMPORTANT: Debit wallet FIRST to validate balance before initiating external transfer
	if err := justError(activities.DebitWallet(internal.InfiniteRetryContext(ctx), sourceWallet.ID, &activities.DebitWalletRequestPayload{
		Amount:    *amount,
		Balances:  []string{source.Balance},
		Metadata:  m,
		Timestamp: timestamp,
	})); err != nil {
		return err
	}

	// Version 1: Use generic CreateTransferInitiation (supports all PSPs)
	// Version 0 (default): Use legacy StripeTransfer (Stripe only)
	v := workflow.GetVersion(ctx, "generic-transfer-initiation", workflow.DefaultVersion, 1)
	if v == workflow.DefaultVersion {
		// Legacy behavior: Stripe only
		if destination.PSP != "stripe" {
			return errors.New("only stripe actually supported")
		}
		return activities.StripeTransfer(internal.InfiniteRetryContext(ctx), activities.StripeTransferRequest{
			Amount:            amount.Amount,
			Asset:             &amount.Asset,
			Destination:       &formanceAccountID,
			WaitingValidation: &destination.WaitingValidation,
			ConnectorID:       destination.ConnectorID,
			Metadata:          m,
		})
	}

	// New behavior: Generic transfer initiation for all supported PSPs
	return activities.CreateTransferInitiation(internal.InfiniteRetryContext(ctx), activities.CreateTransferInitiationRequest{
		Amount:            amount.Amount,
		Asset:             &amount.Asset,
		Provider:          &destination.PSP,
		Type:              destination.Type,
		Source:            destination.SourceAccount,
		Destination:       &formanceAccountID,
		WaitingValidation: &destination.WaitingValidation,
		ConnectorID:       destination.ConnectorID,
		Metadata:          m,
	})
}

func runWalletToAccount(ctx workflow.Context, timestamp *time.Time, source *WalletSource, destination *LedgerAccountDestination, amount *shared.Monetary, m metadata.Metadata) error {
	if amount == nil {
		return errors.New("amount must be specified")
	}
	sourceWallet, err := getWalletFromReference(ctx, source.WalletReference)
	if err != nil {
		return err
	}
	if sourceWallet.Ledger == destination.Ledger {
		return justError(activities.DebitWallet(internal.InfiniteRetryContext(ctx), sourceWallet.ID, &activities.DebitWalletRequestPayload{
			Amount: *amount,
			Destination: &shared.Subject{
				LedgerAccountSubject: &shared.LedgerAccountSubject{
					Identifier: destination.ID,
					Type:       "ACCOUNT",
				},
			},
			Timestamp: timestamp,
			Balances:  []string{source.Balance},
			Metadata:  m,
		}))
	}

	if err := justError(activities.DebitWallet(internal.InfiniteRetryContext(ctx), sourceWallet.ID, &activities.DebitWalletRequestPayload{
		Amount:    *amount,
		Balances:  []string{source.Balance},
		Timestamp: timestamp,
		Metadata: collectionutils.MergeMaps(m, map[string]string{
			moveToLedgerMetadata: destination.Ledger,
		}),
	})); err != nil {
		return err
	}

	// Use destination's throughAccount instead of hardcoded "world"
	throughAccount := destination.ThroughAccount
	if throughAccount == "" {
		throughAccount = "world"
	}

	txMetadata := collectionutils.MergeMaps(m, map[string]string{
		moveFromLedgerMetadata: sourceWallet.Ledger,
	})

	// Use Numscript with overdraft if allowOverdraft is true and throughAccount is not "world"
	// Here throughAccount is the SOURCE, so overdraft makes sense
	var txRequest activities.PostTransaction
	if destination.AllowOverdraft && throughAccount != "world" {
		script := generateNumscriptWithSourceOverdraft(
			throughAccount, destination.ID,
			amount.Asset, amount.Amount.String(),
		)
		txRequest = activities.PostTransaction{
			Script: &shared.V2PostTransactionScript{
				Plain: script,
			},
			Timestamp: timestamp,
			Metadata:  txMetadata,
		}
	} else {
		txRequest = activities.PostTransaction{
			Postings: []shared.V2Posting{{
				Amount:      amount.Amount,
				Asset:       amount.Asset,
				Destination: destination.ID,
				Source:      throughAccount,
			}},
			Timestamp: timestamp,
			Metadata:  txMetadata,
		}
	}

	return justError(activities.CreateTransaction(internal.InfiniteRetryContext(ctx), destination.Ledger, txRequest))
}

func runAccountToWallet(ctx workflow.Context, timestamp *time.Time, source *LedgerAccountSource, destination *WalletDestination, amount *shared.Monetary, m metadata.Metadata) error {
	if amount == nil {
		return errors.New("amount must be specified")
	}
	destinationWallet, err := getWalletFromReference(ctx, destination.WalletReference)
	if err != nil {
		return err
	}
	if destinationWallet.Ledger == source.Ledger {
		return activities.CreditWallet(internal.InfiniteRetryContext(ctx), destinationWallet.ID, &activities.CreditWalletRequestPayload{
			Amount: *amount,
			Sources: []shared.Subject{{
				LedgerAccountSubject: &shared.LedgerAccountSubject{
					Identifier: source.ID,
					Type:       "ACCOUNT",
				},
			}},
			Timestamp: timestamp,
			Balance:   &destination.Balance,
			Metadata:  m,
		})
	}

	// Use source's throughAccount instead of hardcoded "world"
	throughAccount := source.ThroughAccount
	if throughAccount == "" {
		throughAccount = "world"
	}

	// If allowOverdraft is true, use Numscript with overdraft on source.ID
	txMetadata := collectionutils.MergeMaps(
		m,
		map[string]string{
			moveToLedgerMetadata: destinationWallet.Ledger,
		},
	)

	var txRequest activities.PostTransaction
	if source.AllowOverdraft {
		script := generateNumscriptWithSourceOverdraft(
			source.ID, throughAccount,
			amount.Asset, amount.Amount.String(),
		)
		txRequest = activities.PostTransaction{
			Script: &shared.V2PostTransactionScript{
				Plain: script,
			},
			Timestamp: timestamp,
			Metadata:  txMetadata,
		}
	} else {
		txRequest = activities.PostTransaction{
			Postings: []shared.V2Posting{{
				Amount:      amount.Amount,
				Asset:       amount.Asset,
				Destination: throughAccount,
				Source:      source.ID,
			}},
			Timestamp: timestamp,
			Metadata:  txMetadata,
		}
	}

	if err := justError(activities.CreateTransaction(internal.InfiniteRetryContext(ctx), source.Ledger, txRequest)); err != nil {
		return err
	}

	return activities.CreditWallet(internal.InfiniteRetryContext(ctx), destinationWallet.ID, &activities.CreditWalletRequestPayload{
		Amount: *amount,
		Sources: []shared.Subject{{
			LedgerAccountSubject: &shared.LedgerAccountSubject{
				Identifier: throughAccount,
				Type:       "ACCOUNT",
			},
		}},
		Timestamp: timestamp,
		Balance:   &destination.Balance,
		Metadata: collectionutils.MergeMaps(m, map[string]string{
			moveFromLedgerMetadata: source.Ledger,
		}),
	})
}

func runAccountToAccount(ctx workflow.Context, timestamp *time.Time, source *LedgerAccountSource, destination *LedgerAccountDestination, amount *shared.Monetary, m metadata.Metadata) error {
	if amount == nil {
		return errors.New("amount must be specified")
	}
	if source.Ledger == destination.Ledger {
		return justError(activities.CreateTransaction(internal.InfiniteRetryContext(ctx), destination.Ledger, activities.PostTransaction{
			Postings: []shared.V2Posting{{
				Amount:      amount.Amount,
				Asset:       amount.Asset,
				Destination: destination.ID,
				Source:      source.ID,
			}},
			Timestamp: timestamp,
			Metadata:  m,
		}))
	}

	// Use source's throughAccount for exit point
	sourceThroughAccount := source.ThroughAccount
	if sourceThroughAccount == "" {
		sourceThroughAccount = "world"
	}

	// Use destination's throughAccount for entry point
	destThroughAccount := destination.ThroughAccount
	if destThroughAccount == "" {
		destThroughAccount = "world"
	}

	// First transaction: source ledger (source.ID -> sourceThroughAccount)
	// If source.AllowOverdraft is true, apply overdraft on source.ID
	sourceTxMetadata := collectionutils.MergeMaps(m, map[string]string{
		moveToLedgerMetadata: destination.Ledger,
	})

	var sourceTxRequest activities.PostTransaction
	if source.AllowOverdraft {
		script := generateNumscriptWithSourceOverdraft(
			source.ID, sourceThroughAccount,
			amount.Asset, amount.Amount.String(),
		)
		sourceTxRequest = activities.PostTransaction{
			Script: &shared.V2PostTransactionScript{
				Plain: script,
			},
			Timestamp: timestamp,
			Metadata:  sourceTxMetadata,
		}
	} else {
		sourceTxRequest = activities.PostTransaction{
			Postings: []shared.V2Posting{{
				Amount:      amount.Amount,
				Asset:       amount.Asset,
				Destination: sourceThroughAccount,
				Source:      source.ID,
			}},
			Timestamp: timestamp,
			Metadata:  sourceTxMetadata,
		}
	}

	if err := justError(activities.CreateTransaction(internal.InfiniteRetryContext(ctx), source.Ledger, sourceTxRequest)); err != nil {
		return err
	}

	// Second transaction: destination ledger (destThroughAccount -> destination.ID)
	// destThroughAccount is the SOURCE here, so overdraft makes sense if needed
	destTxMetadata := collectionutils.MergeMaps(m, map[string]string{
		moveFromLedgerMetadata: source.Ledger,
	})

	var destTxRequest activities.PostTransaction
	if destination.AllowOverdraft && destThroughAccount != "world" {
		script := generateNumscriptWithSourceOverdraft(
			destThroughAccount, destination.ID,
			amount.Asset, amount.Amount.String(),
		)
		destTxRequest = activities.PostTransaction{
			Script: &shared.V2PostTransactionScript{
				Plain: script,
			},
			Timestamp: timestamp,
			Metadata:  destTxMetadata,
		}
	} else {
		destTxRequest = activities.PostTransaction{
			Postings: []shared.V2Posting{{
				Amount:      amount.Amount,
				Asset:       amount.Asset,
				Destination: destination.ID,
				Source:      destThroughAccount,
			}},
			Timestamp: timestamp,
			Metadata:  destTxMetadata,
		}
	}

	return justError(activities.CreateTransaction(internal.InfiniteRetryContext(ctx), destination.Ledger, destTxRequest))
}

func runAccountToPayment(ctx workflow.Context, timestamp *time.Time, source *LedgerAccountSource, destination *PaymentDestination, amount *shared.Monetary, m metadata.Metadata) error {
	if amount == nil {
		return errors.New("amount must be specified")
	}
	account, err := activities.GetAccount(internal.InfiniteRetryContext(ctx), source.Ledger, source.ID)
	if err != nil {
		return errors.Wrapf(err, "reading account: %s", source.ID)
	}
	formanceAccountID, err := extractFormanceAccountID(destination.Metadata, account.Metadata)
	if err != nil {
		return err
	}

	// Use source's throughAccount instead of hardcoded "world"
	throughAccount := source.ThroughAccount
	if throughAccount == "" {
		throughAccount = "world"
	}

	// IMPORTANT: Debit ledger FIRST to validate balance before initiating external transfer
	var txRequest activities.PostTransaction
	if source.AllowOverdraft {
		script := generateNumscriptWithSourceOverdraft(
			source.ID, throughAccount,
			amount.Asset, amount.Amount.String(),
		)
		txRequest = activities.PostTransaction{
			Script: &shared.V2PostTransactionScript{
				Plain: script,
			},
			Timestamp: timestamp,
			Metadata:  m,
		}
	} else {
		txRequest = activities.PostTransaction{
			Postings: []shared.V2Posting{{
				Amount:      amount.Amount,
				Asset:       amount.Asset,
				Destination: throughAccount,
				Source:      source.ID,
			}},
			Timestamp: timestamp,
			Metadata:  m,
		}
	}

	if err := justError(activities.CreateTransaction(internal.InfiniteRetryContext(ctx), source.Ledger, txRequest)); err != nil {
		return err
	}

	// Version 1: Use generic CreateTransferInitiation (supports all PSPs)
	// Version 0 (default): Use legacy StripeTransfer (Stripe only)
	v := workflow.GetVersion(ctx, "generic-transfer-initiation", workflow.DefaultVersion, 1)
	if v == workflow.DefaultVersion {
		// Legacy behavior: Stripe only
		if destination.PSP != "stripe" {
			return errors.New("only stripe actually supported")
		}
		return activities.StripeTransfer(internal.InfiniteRetryContext(ctx), activities.StripeTransferRequest{
			Amount:            amount.Amount,
			Asset:             &amount.Asset,
			Destination:       &formanceAccountID,
			WaitingValidation: &destination.WaitingValidation,
			ConnectorID:       destination.ConnectorID,
			Metadata:          m,
		})
	}

	// New behavior: Generic transfer initiation for all supported PSPs
	return activities.CreateTransferInitiation(internal.InfiniteRetryContext(ctx), activities.CreateTransferInitiationRequest{
		Amount:            amount.Amount,
		Asset:             &amount.Asset,
		Provider:          &destination.PSP,
		Type:              destination.Type,
		Source:            destination.SourceAccount,
		Destination:       &formanceAccountID,
		WaitingValidation: &destination.WaitingValidation,
		ConnectorID:       destination.ConnectorID,
		Metadata:          m,
	})
}
