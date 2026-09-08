package activities

import (
	"context"

	"github.com/formancehq/formance-sdk-go/v5/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v5/pkg/models/payments"
	"go.temporal.io/sdk/workflow"
)

type GetPaymentRequest struct {
	ID string `json:"id"`
}

func (a Activities) GetPayment(ctx context.Context, request GetPaymentRequest) (*payments.PaymentResponse, error) {
	response, err := a.client.Payments.V1.GetPayment(
		ctx,
		operations.GetPaymentRequest{
			PaymentID: request.ID,
		},
	)
	if err != nil {
		return nil, err
	}

	return response.PaymentResponse, nil
}

var GetPaymentActivity = Activities{}.GetPayment

func GetPayment(ctx workflow.Context, id string) (*payments.Payment, error) {
	ret := &payments.PaymentResponse{}
	if err := executeActivity(ctx, GetPaymentActivity, ret, GetPaymentRequest{
		ID: id,
	}); err != nil {
		return nil, err
	}
	return &ret.Data, nil
}
