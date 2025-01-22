// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

type V2ActivityCreateTransaction struct {
	Ledger *string            `json:"ledger,omitempty"`
	Data   *V2PostTransaction `json:"data,omitempty"`
}

func (o *V2ActivityCreateTransaction) GetLedger() *string {
	if o == nil {
		return nil
	}
	return o.Ledger
}

func (o *V2ActivityCreateTransaction) GetData() *V2PostTransaction {
	if o == nil {
		return nil
	}
	return o.Data
}
