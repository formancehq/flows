package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	common "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/mocks"
)

func TestResumeActivitySelectsParentRunAmongHistoricalStages(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name              string
		parentID          string
		parentRun         string
		currentTerminated bool
		historicalActive  bool
		wantConflict      bool
	}{
		{name: "historical terminated rows", parentID: "instance-main", parentRun: "parent-run"},
		{name: "historical unfinished rows", parentID: "instance-main", parentRun: "parent-run", historicalActive: true},
		{name: "current terminated historical active", parentID: "instance-main", parentRun: "parent-run", currentTerminated: true, historicalActive: true, wantConflict: true},
		{name: "missing parent metadata", wantConflict: true},
		{name: "unrecorded parent run", parentID: "instance-main", parentRun: "unrecorded", wantConflict: true},
		{name: "wrong parent workflow", parentID: "other-main", parentRun: "parent-run", wantConflict: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := resumeActivityFixture(t)
			terminatedAt := time.Now().Add(-time.Minute)
			for _, parentRun := range []string{"aaa-historical-parent", "zzz-historical-parent"} {
				historical := NewStage("instance", parentRun, 2)
				historical.StartedAt = time.Now().Add(-time.Hour)
				if !tc.historicalActive {
					historical.TerminatedAt = &terminatedAt
				}
				_, err := fixture.db.NewInsert().Model(&historical).Exec(t.Context())
				require.NoError(t, err)
			}
			if tc.currentTerminated {
				_, err := fixture.db.NewUpdate().Model((*Stage)(nil)).Set("terminated_at = ?", terminatedAt).Where("instance_id = ? AND stage = ? AND temporal_run_id = ?", "instance", 2, "parent-run").Exec(t.Context())
				require.NoError(t, err)
			}
			pausedAt := time.Now().UTC()
			description := resumeActivityDescription(pausedAt)
			description.WorkflowExecutionInfo.ParentExecution = &common.WorkflowExecution{WorkflowId: tc.parentID, RunId: tc.parentRun}
			temporalClient := &mocks.Client{}
			manager := newResumeActivityManager(t, fixture, temporalClient)
			temporalClient.On("DescribeWorkflowExecution", mock.Anything, "instance-2", "child-run").Run(func(mock.Arguments) {
				// Every historical row must stay locked across the Temporal transition,
				// regardless of which row is chosen as the current parent run.
				for _, parentRun := range []string{"aaa-historical-parent", "parent-run", "zzz-historical-parent"} {
					err := fixture.db.RunInTx(t.Context(), nil, func(ctx context.Context, tx bun.Tx) error {
						var stage Stage
						return tx.NewSelect().Model(&stage).Where("instance_id = ? AND stage = ? AND temporal_run_id = ?", "instance", 2, parentRun).For("UPDATE NOWAIT").Scan(ctx)
					})
					require.ErrorContains(t, err, "could not obtain lock")
				}
			}).Return(description, nil).Once()
			service := &resumeActivityService{t: t, fixture: fixture}
			if !tc.wantConflict {
				temporalClient.On("WorkflowService").Return(service).Once()
			}
			err := manager.ResumeActivity(t.Context(), "instance", 2, "activity", "child-run", pausedAt)
			if tc.wantConflict {
				require.ErrorIs(t, err, ErrActivityResumeConflict)
				require.Empty(t, service.requests)
				temporalClient.AssertNotCalled(t, "WorkflowService")
			} else {
				require.NoError(t, err)
				require.Len(t, service.requests, 1)
				require.Equal(t, "child-run", service.requests[0].Execution.RunId)
			}
			fixture.assertUnlocked()
		})
	}
}
