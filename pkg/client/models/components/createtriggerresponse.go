// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

type CreateTriggerResponse struct {
	Data Trigger `json:"data"`
}

func (o *CreateTriggerResponse) GetData() Trigger {
	if o == nil {
		return Trigger{}
	}
	return o.Data
}
