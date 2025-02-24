// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

type V2StageSendDestination struct {
	Wallet  *V2StageSendDestinationWallet  `json:"wallet,omitempty"`
	Account *V2StageSendDestinationAccount `json:"account,omitempty"`
	Payment *V2StageSendDestinationPayment `json:"payment,omitempty"`
}

func (o *V2StageSendDestination) GetWallet() *V2StageSendDestinationWallet {
	if o == nil {
		return nil
	}
	return o.Wallet
}

func (o *V2StageSendDestination) GetAccount() *V2StageSendDestinationAccount {
	if o == nil {
		return nil
	}
	return o.Account
}

func (o *V2StageSendDestination) GetPayment() *V2StageSendDestinationPayment {
	if o == nil {
		return nil
	}
	return o.Payment
}
