// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

type V2ActivityGetAccountOutput struct {
	Data V2Account `json:"data"`
}

func (o *V2ActivityGetAccountOutput) GetData() V2Account {
	if o == nil {
		return V2Account{}
	}
	return o.Data
}
