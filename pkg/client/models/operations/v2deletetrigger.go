// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"openapi/models/components"
)

type V2DeleteTriggerRequest struct {
	// The trigger id
	TriggerID string `pathParam:"style=simple,explode=false,name=triggerID"`
}

func (o *V2DeleteTriggerRequest) GetTriggerID() string {
	if o == nil {
		return ""
	}
	return o.TriggerID
}

type V2DeleteTriggerResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
}

func (o *V2DeleteTriggerResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}
