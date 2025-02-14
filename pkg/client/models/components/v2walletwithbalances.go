// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"openapi/internal/utils"
	"time"
)

type V2WalletWithBalancesBalances struct {
	Main V2AssetHolder `json:"main"`
}

func (o *V2WalletWithBalancesBalances) GetMain() V2AssetHolder {
	if o == nil {
		return V2AssetHolder{}
	}
	return o.Main
}

type V2WalletWithBalances struct {
	// The unique ID of the wallet.
	ID string `json:"id"`
	// Metadata associated with the wallet.
	Metadata  map[string]string            `json:"metadata"`
	Name      string                       `json:"name"`
	CreatedAt time.Time                    `json:"createdAt"`
	Balances  V2WalletWithBalancesBalances `json:"balances"`
	Ledger    string                       `json:"ledger"`
}

func (v V2WalletWithBalances) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(v, "", false)
}

func (v *V2WalletWithBalances) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &v, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *V2WalletWithBalances) GetID() string {
	if o == nil {
		return ""
	}
	return o.ID
}

func (o *V2WalletWithBalances) GetMetadata() map[string]string {
	if o == nil {
		return map[string]string{}
	}
	return o.Metadata
}

func (o *V2WalletWithBalances) GetName() string {
	if o == nil {
		return ""
	}
	return o.Name
}

func (o *V2WalletWithBalances) GetCreatedAt() time.Time {
	if o == nil {
		return time.Time{}
	}
	return o.CreatedAt
}

func (o *V2WalletWithBalances) GetBalances() V2WalletWithBalancesBalances {
	if o == nil {
		return V2WalletWithBalancesBalances{}
	}
	return o.Balances
}

func (o *V2WalletWithBalances) GetLedger() string {
	if o == nil {
		return ""
	}
	return o.Ledger
}
