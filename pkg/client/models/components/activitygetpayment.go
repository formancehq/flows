// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

type ActivityGetPayment struct {
	ID string `json:"id"`
}

func (o *ActivityGetPayment) GetID() string {
	if o == nil {
		return ""
	}
	return o.ID
}
