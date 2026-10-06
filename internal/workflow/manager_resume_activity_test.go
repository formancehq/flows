package workflow

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunconnect"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/orchestration/internal/storage"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	common "go.temporal.io/api/common/v1"
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	temporalworkflow "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/mocks"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Each test uses the real PostgreSQL server provisioned by TestMain. A second
// connection probes the row lock with NOWAIT while Temporal calls are in flight.
type resumeActivityDatabase struct {
	t          *testing.T
	db         *bun.DB
	connString string
}

func (f *resumeActivityDatabase) assertLocked() {
	f.t.Helper()
	ctx, cancel := context.WithTimeout(f.t.Context(), 2*time.Second)
	defer cancel()
	err := f.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var stage Stage
		return tx.NewSelect().Model(&stage).Where("instance_id = ? AND stage = ?", "instance", 2).For("UPDATE NOWAIT").Scan(ctx)
	})
	require.ErrorContains(f.t, err, "could not obtain lock", "Temporal must execute while the stage row is locked")
}

func (f *resumeActivityDatabase) assertUnlocked() {
	f.t.Helper()
	ctx, cancel := context.WithTimeout(f.t.Context(), 2*time.Second)
	defer cancel()
	require.NoError(f.t, f.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var stage Stage
		return tx.NewSelect().Model(&stage).Where("instance_id = ? AND stage = ?", "instance", 2).For("UPDATE NOWAIT").Scan(ctx)
	}))
}

// Embedding the interface makes any unexpected RPC fail; only the requested
// single-activity unpause is implemented.
type resumeActivityService struct {
	workflowservice.WorkflowServiceClient
	t            *testing.T
	fixture      *resumeActivityDatabase
	requests     []*workflowservice.UnpauseActivityRequest
	err          error
	beforeReturn func(context.Context) error
}

func (s *resumeActivityService) UnpauseActivity(ctx context.Context, request *workflowservice.UnpauseActivityRequest, _ ...grpc.CallOption) (*workflowservice.UnpauseActivityResponse, error) {
	s.fixture.assertLocked()
	deadline, ok := ctx.Deadline()
	require.True(s.t, ok)
	require.InDelta(s.t, 10, time.Until(deadline).Seconds(), 1)
	s.requests = append(s.requests, proto.Clone(request).(*workflowservice.UnpauseActivityRequest))
	if s.beforeReturn != nil {
		if err := s.beforeReturn(ctx); err != nil {
			return nil, err
		}
	}
	if s.err != nil {
		return nil, s.err
	}
	return &workflowservice.UnpauseActivityResponse{}, nil
}

func newResumeActivityManager(t *testing.T, fixture *resumeActivityDatabase, temporalClient *mocks.Client) *WorkflowManager {
	t.Helper()
	t.Cleanup(func() { temporalClient.AssertExpectations(t) })
	return NewManager(fixture.db, temporalClient, "stack", "task-queue", false, WithNamespace("custom-namespace"))
}

func resumeActivityFixture(t *testing.T) *resumeActivityDatabase {
	t.Helper()
	database := srv.NewDatabase(t)
	db, err := bunconnect.OpenSQLDB(logging.TestingContext(), bunconnect.ConnectionOptions{DatabaseSourceName: database.ConnString()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, storage.Migrate(logging.TestingContext(), db))
	definition := New(Config{Stages: []RawStage{{"noop": map[string]any{}}}})
	_, err = db.NewInsert().Model(&definition).Exec(t.Context())
	require.NoError(t, err)
	instance := NewInstance("instance", definition.ID)
	_, err = db.NewInsert().Model(&instance).Exec(t.Context())
	require.NoError(t, err)
	stage := NewStage("instance", "parent-run", 2)
	_, err = db.NewInsert().Model(&stage).Exec(t.Context())
	require.NoError(t, err)
	return &resumeActivityDatabase{t: t, db: db, connString: database.ConnString()}
}

func resumeActivityDescription(pausedAt time.Time) *workflowservice.DescribeWorkflowExecutionResponse {
	return &workflowservice.DescribeWorkflowExecutionResponse{
		WorkflowExecutionInfo: &temporalworkflow.WorkflowExecutionInfo{
			Execution: &common.WorkflowExecution{WorkflowId: "instance-2", RunId: "child-run"},
			Status:    enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
		},
		PendingActivities: []*temporalworkflow.PendingActivityInfo{
			{ActivityId: "unrelated", Paused: true},
			{ActivityId: "activity", Paused: true, Attempt: 15,
				State:     enums.PENDING_ACTIVITY_STATE_PAUSED,
				PauseInfo: &temporalworkflow.PendingActivityInfo_PauseInfo{PauseTime: timestamppb.New(pausedAt)},
			},
		},
	}
}

func expectResumeActivityDescribe(t *testing.T, temporalClient *mocks.Client, fixture *resumeActivityDatabase, description *workflowservice.DescribeWorkflowExecutionResponse, err error) {
	t.Helper()
	temporalClient.On("DescribeWorkflowExecution", mock.Anything, "instance-2", "child-run").
		Run(func(mock.Arguments) {
			fixture.assertLocked()
		}).Return(description, err).Once()
}

func TestResumeActivityExactRPCAndPauseGeneration(t *testing.T) {
	t.Parallel()
	for _, state := range []enums.PendingActivityState{enums.PENDING_ACTIVITY_STATE_PAUSED, enums.PENDING_ACTIVITY_STATE_SCHEDULED} {
		t.Run(state.String(), func(t *testing.T) {
			fixture := resumeActivityFixture(t)
			temporalClient := &mocks.Client{}
			manager := newResumeActivityManager(t, fixture, temporalClient)
			service := &resumeActivityService{t: t, fixture: fixture}
			temporalClient.On("WorkflowService").Return(service).Once()
			pausedAt := time.Date(2026, 10, 6, 12, 0, 0, 123456789, time.UTC)
			description := resumeActivityDescription(pausedAt)
			description.PendingActivities[1].State = state
			expectResumeActivityDescribe(t, temporalClient, fixture, description, nil)
			// Equality is by instant, preserving nanoseconds across timezone offsets.
			err := manager.ResumeActivity(t.Context(), "instance", 2, "activity", "child-run", pausedAt.In(time.FixedZone("offset", 7200)))
			require.NoError(t, err)
			require.Len(t, service.requests, 1)
			expected := &workflowservice.UnpauseActivityRequest{
				Namespace: "custom-namespace", Identity: "flows-api",
				Execution:     &common.WorkflowExecution{WorkflowId: "instance-2", RunId: "child-run"},
				Activity:      &workflowservice.UnpauseActivityRequest_Id{Id: "activity"},
				ResetAttempts: true, ResetHeartbeat: false,
			}
			require.True(t, proto.Equal(expected, service.requests[0]), "request: %v", service.requests[0])
			fixture.assertUnlocked()
		})
	}
}

func TestResumeActivityRepeatedRequestAndDelayedNewPause(t *testing.T) {
	t.Parallel()
	fixture := resumeActivityFixture(t)
	temporalClient := &mocks.Client{}
	manager := newResumeActivityManager(t, fixture, temporalClient)
	service := &resumeActivityService{t: t, fixture: fixture}
	temporalClient.On("WorkflowService").Return(service).Once()
	pausedAt := time.Unix(100, 123456789).UTC()
	expectResumeActivityDescribe(t, temporalClient, fixture, resumeActivityDescription(pausedAt), nil)
	require.NoError(t, manager.ResumeActivity(t.Context(), "instance", 2, "activity", "child-run", pausedAt))

	active := resumeActivityDescription(pausedAt)
	active.PendingActivities[1].Paused = false
	active.PendingActivities[1].State = enums.PENDING_ACTIVITY_STATE_STARTED
	active.PendingActivities[1].PauseInfo = nil
	expectResumeActivityDescribe(t, temporalClient, fixture, active, nil)
	require.NoError(t, manager.ResumeActivity(t.Context(), "instance", 2, "activity", "child-run", pausedAt))
	require.Len(t, service.requests, 1, "active duplicate must not reset the budget")

	// Same run and activity, same attempt count, but a new pause generation.
	expectResumeActivityDescribe(t, temporalClient, fixture, resumeActivityDescription(pausedAt.Add(time.Nanosecond)), nil)
	require.ErrorIs(t, manager.ResumeActivity(t.Context(), "instance", 2, "activity", "child-run", pausedAt), ErrActivityResumeConflict)
	require.Len(t, service.requests, 1, "delayed retry must not unpause the new generation")
	fixture.assertUnlocked()
}

func TestResumeActivityConcurrentReplicasSerializeOnStageRow(t *testing.T) {
	t.Parallel()
	fixture := resumeActivityFixture(t)
	firstClient := &mocks.Client{}
	first := newResumeActivityManager(t, fixture, firstClient)
	secondDB, err := bunconnect.OpenSQLDB(logging.TestingContext(), bunconnect.ConnectionOptions{DatabaseSourceName: fixture.connString})
	require.NoError(t, err)
	secondDB.SetMaxOpenConns(1)
	secondDB.SetMaxIdleConns(1)
	t.Cleanup(func() { require.NoError(t, secondDB.Close()) })
	var secondPID int
	require.NoError(t, secondDB.NewRaw("SELECT pg_backend_pid()").Scan(t.Context(), &secondPID))
	secondClient := &mocks.Client{}
	t.Cleanup(func() { secondClient.AssertExpectations(t) })
	second := NewManager(secondDB, secondClient, "stack", "task-queue", false, WithNamespace("custom-namespace"))
	pausedAt := time.Unix(100, 123456789).UTC()
	expectResumeActivityDescribe(t, firstClient, fixture, resumeActivityDescription(pausedAt), nil)
	enteredRPC := make(chan struct{})
	releaseRPC := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseRPC) }) }
	t.Cleanup(release)
	service := &resumeActivityService{t: t, fixture: fixture, beforeReturn: func(ctx context.Context) error {
		close(enteredRPC)
		select {
		case <-releaseRPC:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}}
	firstClient.On("WorkflowService").Return(service).Once()
	active := resumeActivityDescription(pausedAt)
	active.PendingActivities[1].Paused = false
	active.PendingActivities[1].State = enums.PENDING_ACTIVITY_STATE_STARTED
	active.PendingActivities[1].PauseInfo = nil
	secondDescribed := make(chan struct{})
	secondClient.On("DescribeWorkflowExecution", mock.Anything, "instance-2", "child-run").Run(func(mock.Arguments) {
		close(secondDescribed)
	}).Return(active, nil).Once()
	firstResult := make(chan error, 1)
	secondResult := make(chan error, 1)
	go func() {
		firstResult <- first.ResumeActivity(t.Context(), "instance", 2, "activity", "child-run", pausedAt)
	}()
	select {
	case <-enteredRPC:
	case <-time.After(3 * time.Second):
		t.Fatal("first manager did not reach UnpauseActivity")
	}
	go func() {
		secondResult <- second.ResumeActivity(t.Context(), "instance", 2, "activity", "child-run", pausedAt)
	}()
	// Observe an actual PostgreSQL lock wait, rather than infer blocking from a sleep.
	require.Eventually(t, func() bool {
		var waiting bool
		err := fixture.db.NewRaw("SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pid = ? AND wait_event_type = 'Lock')", secondPID).Scan(t.Context(), &waiting)
		return err == nil && waiting
	}, 3*time.Second, 10*time.Millisecond)
	select {
	case <-secondDescribed:
		t.Fatal("second replica described Temporal before acquiring the row lock")
	default:
	}
	release()
	for _, result := range []<-chan error{firstResult, secondResult} {
		select {
		case err := <-result:
			require.NoError(t, err)
		case <-time.After(3 * time.Second):
			t.Fatal("resume did not finish after releasing the row lock")
		}
	}
	require.Len(t, service.requests, 1)
	secondClient.AssertNotCalled(t, "WorkflowService")
	fixture.assertUnlocked()
}

func TestResumeActivityRejectsInvalidTemporalState(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		change       func(*workflowservice.DescribeWorkflowExecutionResponse)
		zeroObserved bool
		want         error
	}{
		{name: "different child run", change: func(d *workflowservice.DescribeWorkflowExecutionResponse) {
			d.WorkflowExecutionInfo.Execution.RunId = "new-run"
		}, want: ErrActivityResumeConflict},
		{name: "completed workflow", change: func(d *workflowservice.DescribeWorkflowExecutionResponse) {
			d.WorkflowExecutionInfo.Status = enums.WORKFLOW_EXECUTION_STATUS_COMPLETED
		}, want: ErrActivityResumeConflict},
		{name: "canceled workflow", change: func(d *workflowservice.DescribeWorkflowExecutionResponse) {
			d.WorkflowExecutionInfo.Status = enums.WORKFLOW_EXECUTION_STATUS_CANCELED
		}, want: ErrActivityResumeConflict},
		{name: "missing workflow info", change: func(d *workflowservice.DescribeWorkflowExecutionResponse) { d.WorkflowExecutionInfo = nil }, want: ErrActivityResumeConflict},
		{name: "missing target activity", change: func(d *workflowservice.DescribeWorkflowExecutionResponse) {
			d.PendingActivities = d.PendingActivities[:1]
		}, want: ErrActivityNotFound},
		{name: "completed activity", change: func(d *workflowservice.DescribeWorkflowExecutionResponse) { d.PendingActivities = nil }, want: ErrActivityNotFound},
		{name: "still executing", change: func(d *workflowservice.DescribeWorkflowExecutionResponse) {
			d.PendingActivities[1].State = enums.PENDING_ACTIVITY_STATE_STARTED
		}, want: ErrActivityResumeConflict},
		{name: "cancel requested", change: func(d *workflowservice.DescribeWorkflowExecutionResponse) {
			d.PendingActivities[1].State = enums.PENDING_ACTIVITY_STATE_CANCEL_REQUESTED
		}, want: ErrActivityResumeConflict},
		{name: "unpaused cancel requested", change: func(d *workflowservice.DescribeWorkflowExecutionResponse) {
			d.PendingActivities[1].Paused = false
			d.PendingActivities[1].State = enums.PENDING_ACTIVITY_STATE_CANCEL_REQUESTED
			d.PendingActivities[1].PauseInfo = nil
		}, want: ErrActivityResumeConflict},
		{name: "pause requested", change: func(d *workflowservice.DescribeWorkflowExecutionResponse) {
			d.PendingActivities[1].State = enums.PENDING_ACTIVITY_STATE_PAUSE_REQUESTED
		}, want: ErrActivityResumeConflict},
		{name: "unknown activity state", change: func(d *workflowservice.DescribeWorkflowExecutionResponse) {
			d.PendingActivities[1].State = enums.PENDING_ACTIVITY_STATE_UNSPECIFIED
		}, want: ErrActivityResumeConflict},
		{name: "legacy pause without info", change: func(d *workflowservice.DescribeWorkflowExecutionResponse) { d.PendingActivities[1].PauseInfo = nil }, want: ErrActivityResumeConflict},
		{name: "legacy pause without timestamp", change: func(d *workflowservice.DescribeWorkflowExecutionResponse) {
			d.PendingActivities[1].PauseInfo.PauseTime = nil
		}, want: ErrActivityResumeConflict},
		{name: "missing observed pause", zeroObserved: true, want: ErrActivityResumeConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := resumeActivityFixture(t)
			temporalClient := &mocks.Client{}
			manager := newResumeActivityManager(t, fixture, temporalClient)
			pausedAt := time.Unix(100, 123456789).UTC()
			description := resumeActivityDescription(pausedAt)
			if tc.change != nil {
				tc.change(description)
			}
			expectResumeActivityDescribe(t, temporalClient, fixture, description, nil)
			if tc.zeroObserved {
				pausedAt = time.Time{}
			}
			require.ErrorIs(t, manager.ResumeActivity(t.Context(), "instance", 2, "activity", "child-run", pausedAt), tc.want)
			temporalClient.AssertNotCalled(t, "WorkflowService")
			fixture.assertUnlocked()
		})
	}
}

func TestResumeActivitySQLBoundaries(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		query string
		want  error
	}{
		{"instance missing", "DELETE FROM workflow_instance_stage_statuses; DELETE FROM workflow_instances", ErrInstanceNotFound},
		{"instance terminal", "UPDATE workflow_instances SET terminated = true", ErrActivityResumeConflict},
		{"stage missing", "DELETE FROM workflow_instance_stage_statuses", ErrActivityNotFound},
		{"stage terminal", "UPDATE workflow_instance_stage_statuses SET terminated_at = now()", ErrActivityResumeConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := resumeActivityFixture(t)
			_, err := fixture.db.ExecContext(t.Context(), tc.query)
			require.NoError(t, err)
			temporalClient := &mocks.Client{}
			manager := newResumeActivityManager(t, fixture, temporalClient)
			require.ErrorIs(t, manager.ResumeActivity(t.Context(), "instance", 2, "activity", "child-run", time.Unix(100, 0)), tc.want)
			require.Empty(t, temporalClient.Calls, "SQL boundary failures must not contact Temporal")
			if tc.name == "instance terminal" || tc.name == "stage terminal" {
				fixture.assertUnlocked()
			}
		})
	}
}

func TestResumeActivityDatabaseFailure(t *testing.T) {
	t.Parallel()
	fixture := resumeActivityFixture(t)
	temporalClient := &mocks.Client{}
	manager := newResumeActivityManager(t, fixture, temporalClient)
	// A missing table is a real query failure, not a missing business entity.
	_, err := fixture.db.ExecContext(t.Context(), "DROP TABLE workflow_instance_stage_statuses")
	require.NoError(t, err)
	err = manager.ResumeActivity(t.Context(), "instance", 2, "activity", "child-run", time.Unix(100, 0))
	require.ErrorContains(t, err, "does not exist")
	require.NotErrorIs(t, err, ErrActivityNotFound)
	require.Empty(t, temporalClient.Calls)
}

func TestResumeActivityTemporalFailures(t *testing.T) {
	t.Parallel()
	unavailable := serviceerror.NewUnavailable("Temporal unavailable")
	denied := serviceerror.NewPermissionDenied("not authorized", "policy")
	for _, tc := range []struct {
		name        string
		describeErr error
		unpauseErr  error
		want        error
	}{
		{name: "describe missing", describeErr: serviceerror.NewNotFound("expired"), want: ErrActivityNotFound},
		{name: "describe unavailable", describeErr: unavailable, want: unavailable},
		{name: "describe denied", describeErr: denied, want: denied},
		{name: "activity disappears after describe", unpauseErr: serviceerror.NewNotFound("completed"), want: ErrActivityResumeConflict},
		{name: "workflow closes after describe", unpauseErr: serviceerror.NewFailedPrecondition("closed"), want: ErrActivityResumeConflict},
		{name: "unpause unavailable", unpauseErr: unavailable, want: unavailable},
		{name: "unpause unsupported", unpauseErr: serviceerror.NewUnimplemented("unsupported"), want: nil},
		{name: "unpause denied", unpauseErr: denied, want: denied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := resumeActivityFixture(t)
			temporalClient := &mocks.Client{}
			manager := newResumeActivityManager(t, fixture, temporalClient)
			pausedAt := time.Unix(100, 0).UTC()
			expectResumeActivityDescribe(t, temporalClient, fixture, resumeActivityDescription(pausedAt), tc.describeErr)
			service := &resumeActivityService{t: t, fixture: fixture, err: tc.unpauseErr}
			if tc.describeErr == nil {
				temporalClient.On("WorkflowService").Return(service).Once()
			}
			want := tc.want
			if want == nil {
				want = tc.unpauseErr
			}
			require.ErrorIs(t, manager.ResumeActivity(t.Context(), "instance", 2, "activity", "child-run", pausedAt), want)
			if tc.describeErr == nil {
				require.Len(t, service.requests, 1)
			} else {
				require.Empty(t, service.requests)
			}
			fixture.assertUnlocked()
		})
	}
}
