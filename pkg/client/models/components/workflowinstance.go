// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"openapi/internal/utils"
	"time"
)

type WorkflowInstance struct {
	WorkflowID   string        `json:"workflowID"`
	ID           string        `json:"id"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	Status       []StageStatus `json:"status,omitempty"`
	Terminated   bool          `json:"terminated"`
	TerminatedAt *time.Time    `json:"terminatedAt,omitempty"`
	Error        *string       `json:"error,omitempty"`
}

func (w WorkflowInstance) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(w, "", false)
}

func (w *WorkflowInstance) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &w, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *WorkflowInstance) GetWorkflowID() string {
	if o == nil {
		return ""
	}
	return o.WorkflowID
}

func (o *WorkflowInstance) GetID() string {
	if o == nil {
		return ""
	}
	return o.ID
}

func (o *WorkflowInstance) GetCreatedAt() time.Time {
	if o == nil {
		return time.Time{}
	}
	return o.CreatedAt
}

func (o *WorkflowInstance) GetUpdatedAt() time.Time {
	if o == nil {
		return time.Time{}
	}
	return o.UpdatedAt
}

func (o *WorkflowInstance) GetStatus() []StageStatus {
	if o == nil {
		return nil
	}
	return o.Status
}

func (o *WorkflowInstance) GetTerminated() bool {
	if o == nil {
		return false
	}
	return o.Terminated
}

func (o *WorkflowInstance) GetTerminatedAt() *time.Time {
	if o == nil {
		return nil
	}
	return o.TerminatedAt
}

func (o *WorkflowInstance) GetError() *string {
	if o == nil {
		return nil
	}
	return o.Error
}
