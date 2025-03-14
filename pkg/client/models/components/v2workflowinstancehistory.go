// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"openapi/internal/utils"
	"time"
)

type V2WorkflowInstanceHistory struct {
	Name         string     `json:"name"`
	Input        V2Stage    `json:"input"`
	Error        *string    `json:"error,omitempty"`
	Terminated   bool       `json:"terminated"`
	StartedAt    time.Time  `json:"startedAt"`
	TerminatedAt *time.Time `json:"terminatedAt,omitempty"`
}

func (v V2WorkflowInstanceHistory) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(v, "", false)
}

func (v *V2WorkflowInstanceHistory) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &v, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *V2WorkflowInstanceHistory) GetName() string {
	if o == nil {
		return ""
	}
	return o.Name
}

func (o *V2WorkflowInstanceHistory) GetInput() V2Stage {
	if o == nil {
		return V2Stage{}
	}
	return o.Input
}

func (o *V2WorkflowInstanceHistory) GetError() *string {
	if o == nil {
		return nil
	}
	return o.Error
}

func (o *V2WorkflowInstanceHistory) GetTerminated() bool {
	if o == nil {
		return false
	}
	return o.Terminated
}

func (o *V2WorkflowInstanceHistory) GetStartedAt() time.Time {
	if o == nil {
		return time.Time{}
	}
	return o.StartedAt
}

func (o *V2WorkflowInstanceHistory) GetTerminatedAt() *time.Time {
	if o == nil {
		return nil
	}
	return o.TerminatedAt
}
