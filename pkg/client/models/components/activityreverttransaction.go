// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

type ActivityRevertTransaction struct {
	Ledger string `json:"ledger"`
	ID     string `json:"id"`
}

func (o *ActivityRevertTransaction) GetLedger() string {
	if o == nil {
		return ""
	}
	return o.Ledger
}

func (o *ActivityRevertTransaction) GetID() string {
	if o == nil {
		return ""
	}
	return o.ID
}
