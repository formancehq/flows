// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

type V2GetWorkflowInstanceResponse struct {
	Data V2WorkflowInstance `json:"data"`
}

func (o *V2GetWorkflowInstanceResponse) GetData() V2WorkflowInstance {
	if o == nil {
		return V2WorkflowInstance{}
	}
	return o.Data
}
