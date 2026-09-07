package activities

import (
	"context"

	"github.com/formancehq/formance-sdk-go/v5/pkg/models/ledger"
	"github.com/formancehq/formance-sdk-go/v5/pkg/models/operations"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type GetAccountRequest struct {
	Ledger string `json:"ledger"`
	ID     string `json:"id"`
}

func (a Activities) GetAccount(ctx context.Context, request GetAccountRequest) (*ledger.AccountResponse, error) {
	response, err := a.client.Ledger.V1.GetAccountLedger(
		ctx,
		operations.GetAccountLedgerRequest{
			Address: request.ID,
			Ledger:  request.Ledger,
		},
	)
	if err != nil {
		switch err := err.(type) {
		case *ledger.ErrorResponseError:
			return nil, temporal.NewApplicationError(err.ErrorMessage, string(err.ErrorCode), err.Details)
		default:
			return nil, err
		}
	}

	return response.AccountResponse, nil
}

var GetAccountActivity = Activities{}.GetAccount

func GetAccount(ctx workflow.Context, ledgerName, id string) (*ledger.AccountWithVolumesAndBalances, error) {
	ret := &ledger.AccountResponse{}
	if err := executeActivity(ctx, GetAccountActivity, ret, GetAccountRequest{
		Ledger: ledgerName,
		ID:     id,
	}); err != nil {
		return nil, err
	}
	return &ret.Data, nil
}
