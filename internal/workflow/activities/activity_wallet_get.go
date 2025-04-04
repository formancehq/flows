package activities

import (
	"context"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"go.temporal.io/sdk/workflow"
)

type GetWalletRequest struct {
	ID string `json:"id"`
}

func (a Activities) GetWallet(ctx context.Context, request GetWalletRequest) (*shared.GetWalletResponse, error) {
	response, err := a.client.Wallets.V1.GetWallet(
		ctx,
		operations.GetWalletRequest{
			ID: request.ID,
		},
	)
	if err != nil {
		return nil, err
	}

	return response.GetWalletResponse, nil
}

var GetWalletActivity = Activities{}.GetWallet

func GetWallet(ctx workflow.Context, id string) (*shared.WalletWithBalances, error) {
	ret := &shared.GetWalletResponse{}
	if err := executeActivity(ctx, GetWalletActivity, ret, GetWalletRequest{
		ID: id,
	}); err != nil {
		return nil, err
	}
	return &ret.Data, nil
}
