package workflow

import (
	"time"

	"github.com/uptrace/bun"
)

type Instance struct {
	bun.BaseModel `bun:"table:workflow_instances,alias:u"`

	WorkflowID    string     `json:"workflowID" bun:"workflow_id,type:varchar"`
	Workflow *Workflow `json:"workflow,omitempty" bun:"rel:belongs-to,join:workflow_id=id"`
	ID            string     `json:"id" bun:"id,pk"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	Terminated    bool       `json:"terminated"`
	TerminatedAt  *time.Time `json:"terminatedAt,omitempty"`
	//TODO: change JSON tag from status to statuses
	Statuses []Stage `json:"status,omitempty" bun:"rel:has-many,join:id=instance_id"`
	Error    string  `json:"error,omitempty"`
}

func (i *Instance) SetTerminated(at time.Time) {
	i.Terminated = true
	i.TerminatedAt = &at
}

func (i *Instance) SetTerminatedWithError(at time.Time, err error) {
	i.SetTerminated(at)
	i.Error = err.Error()
}

func NewInstance(id, workflowID string) Instance {
	now := time.Now().Round(time.Nanosecond)
	return Instance{
		BaseModel:  bun.BaseModel{},
		WorkflowID: workflowID,
		ID:         id,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}
