// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"openapi/models/components"
)

type RunWorkflowRequest struct {
	// The flow id
	WorkflowID string `pathParam:"style=simple,explode=false,name=workflowID"`
	// Wait end of the workflow before return
	Wait        *bool             `queryParam:"style=form,explode=true,name=wait"`
	RequestBody map[string]string `request:"mediaType=application/json"`
}

func (o *RunWorkflowRequest) GetWorkflowID() string {
	if o == nil {
		return ""
	}
	return o.WorkflowID
}

func (o *RunWorkflowRequest) GetWait() *bool {
	if o == nil {
		return nil
	}
	return o.Wait
}

func (o *RunWorkflowRequest) GetRequestBody() map[string]string {
	if o == nil {
		return nil
	}
	return o.RequestBody
}

type RunWorkflowResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// The workflow instance
	RunWorkflowResponse *components.RunWorkflowResponse
}

func (o *RunWorkflowResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *RunWorkflowResponse) GetRunWorkflowResponse() *components.RunWorkflowResponse {
	if o == nil {
		return nil
	}
	return o.RunWorkflowResponse
}
