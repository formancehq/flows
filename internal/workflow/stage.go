package workflow

import (
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"

	"github.com/uptrace/bun"
)

type Stage struct {
	bun.BaseModel         `bun:"table:workflow_instance_stage_statuses"`
	Number                int             `json:"stage" bun:"stage,pk"`
	InstanceID            string          `json:"instanceID" bun:"instance_id,pk"`
	TemporalRunID         string          `json:"temporalRunID" bun:"temporal_run_id,pk"`
	StartedAt             time.Time       `json:"startedAt" bun:"started_at"`
	TerminatedAt          *time.Time      `json:"terminatedAt,omitempty" bun:"terminated_at"`
	Error                 *string         `json:"error,omitempty" bun:"error"`
	PendingActivities     []ActivityPause `json:"pendingActivities,omitempty" bun:"-"`
	PauseStateUnavailable bool            `json:"pauseStateUnavailable,omitempty" bun:"-"`
}

// ActivityPause is a read-only projection of a paused Temporal activity.
type ActivityPause struct {
	ActivityID      string     `json:"activityID"`
	TemporalRunID   string     `json:"temporalRunID"`
	Name            string     `json:"name"`
	Attempt         int        `json:"attempt"`
	LastFailure     string     `json:"lastFailure,omitempty"`
	LastFailureType string     `json:"lastFailureType,omitempty"`
	Reason          string     `json:"reason,omitempty"`
	MaxAttempts     *int       `json:"maxAttempts,omitempty"`
	PausedAt        *time.Time `json:"pausedAt,omitempty"`
}

func (s *Stage) SetTerminated(err error, date time.Time) {
	s.TerminatedAt = &date
	if err != nil {
		s.Error = pointer.For(err.Error())
	}
}

func (s *Stage) TemporalWorkflowID() string {
	return fmt.Sprintf("%s-%d", s.InstanceID, s.Number)
}

func NewStage(instanceID, temporalRunID string, number int) Stage {
	return Stage{
		BaseModel:     bun.BaseModel{},
		TemporalRunID: temporalRunID,
		Number:        number,
		InstanceID:    instanceID,
		StartedAt:     time.Now(),
	}
}
