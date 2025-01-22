// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"openapi/internal/utils"
	"time"
)

type V2Wallet struct {
	// The unique ID of the wallet.
	ID string `json:"id"`
	// Metadata associated with the wallet.
	Metadata  map[string]string `json:"metadata"`
	Name      string            `json:"name"`
	CreatedAt time.Time         `json:"createdAt"`
	Ledger    string            `json:"ledger"`
}

func (v V2Wallet) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(v, "", false)
}

func (v *V2Wallet) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &v, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *V2Wallet) GetID() string {
	if o == nil {
		return ""
	}
	return o.ID
}

func (o *V2Wallet) GetMetadata() map[string]string {
	if o == nil {
		return map[string]string{}
	}
	return o.Metadata
}

func (o *V2Wallet) GetName() string {
	if o == nil {
		return ""
	}
	return o.Name
}

func (o *V2Wallet) GetCreatedAt() time.Time {
	if o == nil {
		return time.Time{}
	}
	return o.CreatedAt
}

func (o *V2Wallet) GetLedger() string {
	if o == nil {
		return ""
	}
	return o.Ledger
}
