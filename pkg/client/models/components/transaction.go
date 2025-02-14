// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"math/big"
	"openapi/internal/utils"
	"time"
)

type Transaction struct {
	Timestamp time.Time         `json:"timestamp"`
	Postings  []Posting         `json:"postings"`
	Reference *string           `json:"reference,omitempty"`
	Metadata  map[string]string `json:"metadata"`
	ID        *big.Int          `json:"id"`
	Reverted  bool              `json:"reverted"`
}

func (t Transaction) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(t, "", false)
}

func (t *Transaction) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &t, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *Transaction) GetTimestamp() time.Time {
	if o == nil {
		return time.Time{}
	}
	return o.Timestamp
}

func (o *Transaction) GetPostings() []Posting {
	if o == nil {
		return []Posting{}
	}
	return o.Postings
}

func (o *Transaction) GetReference() *string {
	if o == nil {
		return nil
	}
	return o.Reference
}

func (o *Transaction) GetMetadata() map[string]string {
	if o == nil {
		return map[string]string{}
	}
	return o.Metadata
}

func (o *Transaction) GetID() *big.Int {
	if o == nil {
		return big.NewInt(0)
	}
	return o.ID
}

func (o *Transaction) GetReverted() bool {
	if o == nil {
		return false
	}
	return o.Reverted
}
