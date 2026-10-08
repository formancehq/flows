package workflow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/uptrace/bun"
	common "go.temporal.io/api/common/v1"
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	temporalworkflow "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
)

var (
	ErrActivityNotFound       = errors.New("activity or stage not found")
	ErrActivityResumeConflict = errors.New("activity cannot be resumed from the observed pause; refresh the instance")
)

// ResumeActivity resumes one activity in its original execution. The observed
// run and pause timestamp prevent an old request from granting another budget
// after the activity has paused again. Stage row locks serialize this API across
// replicas; the Temporal transition itself is atomic and no-ops if unpaused.
func (m *WorkflowManager) ResumeActivity(ctx context.Context, instanceID string, number int, activityID, runID string, pausedAt time.Time) (result error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	outcome := "resumed"
	defer func() {
		if result != nil {
			switch {
			case errors.Is(result, ErrActivityResumeConflict):
				outcome = "conflict"
			case errors.Is(result, ErrInstanceNotFound), errors.Is(result, ErrActivityNotFound):
				outcome = "not_found"
			default:
				outcome = "failed"
			}
		}
		logging.FromContext(ctx).WithFields(map[string]any{
			"instanceID": instanceID, "stage": number, "activityID": activityID,
			"temporalRunID": runID, "pausedAt": pausedAt, "outcome": outcome,
		}).Info("Workflow activity resume outcome")
	}()
	return m.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		instance := Instance{}
		if err := tx.NewSelect().Model(&instance).Where("u.id = ?", instanceID).Scan(ctx); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInstanceNotFound
			}
			return err
		}
		if instance.Terminated {
			return ErrActivityResumeConflict
		}
		// Lock every recorded run before reading Temporal, so replicas remain
		// serialized even when a stage has historical runs. Lock in a stable order.
		var stages []Stage
		if err := tx.NewSelect().Model(&stages).
			Where("instance_id = ? AND stage = ?", instanceID, number).
			OrderExpr("temporal_run_id ASC").For("UPDATE").Scan(ctx); err != nil {
			return err
		}
		if len(stages) == 0 {
			return ErrActivityNotFound
		}
		active := false
		for _, stage := range stages {
			active = active || stage.TerminatedAt == nil
		}
		if !active {
			return ErrActivityResumeConflict
		}
		stageID := fmt.Sprintf("%s-%d", instanceID, number)
		described, err := m.temporalClient.DescribeWorkflowExecution(ctx, stageID, runID)
		if err != nil {
			var missing *serviceerror.NotFound
			if errors.As(err, &missing) {
				return ErrActivityNotFound
			}
			return err
		}
		info := described.GetWorkflowExecutionInfo()
		if info.GetExecution().GetRunId() != runID || info.GetStatus() != enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
			return ErrActivityResumeConflict
		}
		parent := info.GetParentExecution()
		var selected *Stage
		if parent.GetRunId() != "" {
			if parent.GetWorkflowId() != instanceID+"-main" {
				return ErrActivityResumeConflict
			}
			for index := range stages {
				if stages[index].TemporalRunID == parent.GetRunId() {
					selected = &stages[index]
					break
				}
			}
		} else if len(stages) == 1 {
			// Legacy or standalone executions without parent metadata are only
			// unambiguous when there is exactly one recorded stage run.
			selected = &stages[0]
		}
		if selected == nil || selected.TerminatedAt != nil {
			return ErrActivityResumeConflict
		}
		for _, pending := range described.GetPendingActivities() {
			if pending.GetActivityId() != activityID {
				continue
			}
			if pending.GetState() == enums.PENDING_ACTIVITY_STATE_CANCEL_REQUESTED {
				return ErrActivityResumeConflict
			}
			if !pending.GetPaused() {
				// A repeated request must never reset an active activity's budget.
				outcome = "already_active"
				return nil
			}
			if !resumablePause(pending, pausedAt) {
				return ErrActivityResumeConflict
			}
			_, err := m.temporalClient.WorkflowService().UnpauseActivity(ctx, &workflowservice.UnpauseActivityRequest{
				Namespace:     m.namespace,
				Execution:     &common.WorkflowExecution{WorkflowId: stageID, RunId: runID},
				Activity:      &workflowservice.UnpauseActivityRequest_Id{Id: activityID},
				Identity:      "flows-api",
				ResetAttempts: true,
				// Keep heartbeat checkpoints and the original payment idempotency key.
				ResetHeartbeat: false,
			})
			if err != nil {
				var missing *serviceerror.NotFound
				var conflict *serviceerror.FailedPrecondition
				if errors.As(err, &missing) || errors.As(err, &conflict) {
					return ErrActivityResumeConflict
				}
				return err
			}
			return nil
		}
		return ErrActivityNotFound
	})
}

func resumablePause(pending *temporalworkflow.PendingActivityInfo, observed time.Time) bool {
	state := pending.GetState()
	if state != enums.PENDING_ACTIVITY_STATE_PAUSED && state != enums.PENDING_ACTIVITY_STATE_SCHEDULED {
		return false
	}
	pauseTime := pending.GetPauseInfo().GetPauseTime()
	return pauseTime != nil && !observed.IsZero() && pauseTime.AsTime().Equal(observed)
}
