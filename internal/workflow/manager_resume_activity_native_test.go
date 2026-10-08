package workflow

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunconnect"
	"github.com/formancehq/orchestration/internal/storage"
	"github.com/formancehq/orchestration/internal/temporalworker"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	sdkworkflow "go.temporal.io/sdk/workflow"
)

func TestResumeActivityNativePauseGenerations(t *testing.T) {
	ctx := t.Context()
	database := srv.NewDatabase(t)
	db, err := bunconnect.OpenSQLDB(ctx, database.ConnectionOptions())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, storage.Migrate(ctx, db))

	w := New(Config{})
	instance := NewInstance(uuid.NewString(), w.ID)
	// Stage rows contain the parent's run ID; resume must target the actual
	// stage execution and the child run returned by GetInstance instead.
	stage := NewStage(instance.ID, "parent-run", 0)
	_, err = db.NewInsert().Model(&w).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().Model(&instance).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().Model(&stage).Exec(ctx)
	require.NoError(t, err)
	require.Equal(t, instance.ID+"-0", stage.TemporalWorkflowID())

	var calls atomic.Int64
	var ready atomic.Bool
	var keysMu sync.Mutex
	var keys []string
	held := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }

	taskQueue := "resume-native-" + uuid.NewString()
	nativeWorker := worker.New(devServer.Client(), taskQueue, worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{
			&temporalworker.StagePauseInterceptor{Attempts: 15},
		},
	})
	nativeWorker.RegisterActivityWithOptions(func(ctx context.Context) error {
		info := activity.GetInfo(ctx)
		key := info.WorkflowExecution.RunID + "-" + info.ActivityID
		keysMu.Lock()
		keys = append(keys, key)
		keysMu.Unlock()
		if calls.Add(1) == 16 {
			// Hold the first resumed call so a duplicate request is guaranteed
			// to observe an active activity, rather than racing the next pause.
			close(held)
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if ready.Load() {
			return nil
		}
		return temporal.NewApplicationError("synthetic insufficient balance", "INSUFFICIENT_FUND")
	}, activity.RegisterOptions{Name: "CreateTransaction"})
	nativeWorker.RegisterWorkflowWithOptions(func(ctx sdkworkflow.Context) error {
		// Accelerate only this synthetic workflow; product retry settings stay
		// untouched. The interceptor supplies the native 15-attempt budget.
		ctx = sdkworkflow.WithActivityOptions(ctx, sdkworkflow.ActivityOptions{
			ActivityID:          "transaction",
			StartToCloseTimeout: 30 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    20 * time.Millisecond,
				BackoffCoefficient: 1,
				MaximumInterval:    20 * time.Millisecond,
			},
		})
		return sdkworkflow.ExecuteActivity(ctx, "CreateTransaction").Get(ctx, nil)
	}, sdkworkflow.RegisterOptions{Name: "ResumeActivityNative"})
	require.NoError(t, nativeWorker.Start())
	t.Cleanup(nativeWorker.Stop)
	t.Cleanup(unblock)

	run, err := devServer.Client().ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID: stage.TemporalWorkflowID(), TaskQueue: taskQueue,
		WorkflowExecutionTimeout: 2 * time.Minute,
	}, "ResumeActivityNative")
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = devServer.Client().CancelWorkflow(cleanupCtx, run.GetID(), run.GetRunID())
	})
	require.NotEqual(t, stage.TemporalRunID, run.GetRunID())
	idempotencyKey := run.GetRunID() + "-transaction"
	manager := NewManager(db, devServer.Client(), "test", taskQueue, false)

	waitPaused := func(wantCalls int64, previous time.Time) ActivityPause {
		t.Helper()
		var observed ActivityPause
		require.Eventually(t, func() bool {
			description, err := devServer.Client().DescribeWorkflowExecution(ctx, run.GetID(), run.GetRunID())
			if err != nil || len(description.GetPendingActivities()) != 1 {
				return false
			}
			pending := description.GetPendingActivities()[0]
			state := pending.GetState()
			pauseTime := pending.GetPauseInfo().GetPauseTime()
			if !pending.GetPaused() || pauseTime == nil || pauseTime.AsTime().IsZero() ||
				(state != enums.PENDING_ACTIVITY_STATE_PAUSED && state != enums.PENDING_ACTIVITY_STATE_SCHEDULED) ||
				pauseTime.AsTime().Equal(previous) || calls.Load() != wantCalls {
				return false
			}
			updated, err := manager.GetInstance(ctx, instance.ID)
			if err != nil || len(updated.Statuses) != 1 || updated.Statuses[0].PauseStateUnavailable ||
				len(updated.Statuses[0].PendingActivities) != 1 {
				return false
			}
			observed = updated.Statuses[0].PendingActivities[0]
			return observed.PausedAt != nil && observed.PausedAt.Equal(pauseTime.AsTime())
		}, 20*time.Second, 20*time.Millisecond, "activity must settle at business call %d", wantCalls)
		require.Equal(t, "transaction", observed.ActivityID)
		require.Equal(t, "CreateTransaction", observed.Name)
		require.Equal(t, run.GetRunID(), observed.TemporalRunID)
		require.Equal(t, "INSUFFICIENT_FUND", observed.LastFailureType)
		require.Equal(t, "ACTIVITY_ATTEMPT_LIMIT:15", observed.Reason)
		require.NotNil(t, observed.MaxAttempts)
		require.Equal(t, 15, *observed.MaxAttempts)
		require.Never(t, func() bool { return calls.Load() != wantCalls }, 200*time.Millisecond, 20*time.Millisecond,
			"paused activity must not dispatch more business calls")
		return observed
	}

	first := waitPaused(15, time.Time{})
	require.NoError(t, manager.ResumeActivity(ctx, instance.ID, 0, first.ActivityID, first.TemporalRunID, *first.PausedAt))
	select {
	case <-held:
	case <-time.After(20 * time.Second):
		t.Fatal("resumed activity did not reach its first business call")
	}
	description, err := devServer.Client().DescribeWorkflowExecution(ctx, run.GetID(), run.GetRunID())
	require.NoError(t, err)
	require.Len(t, description.GetPendingActivities(), 1)
	require.False(t, description.GetPendingActivities()[0].GetPaused())
	require.Equal(t, enums.PENDING_ACTIVITY_STATE_STARTED, description.GetPendingActivities()[0].GetState())
	require.NoError(t, manager.ResumeActivity(ctx, instance.ID, 0, first.ActivityID, first.TemporalRunID, *first.PausedAt))
	require.Equal(t, int64(16), calls.Load())
	unblock()

	second := waitPaused(30, *first.PausedAt)
	require.False(t, second.PausedAt.Equal(*first.PausedAt), "a fresh budget must produce a new pause generation")
	// ErrActivityResumeConflict is the manager error mapped to HTTP 409 by
	// both API versions. An old pause token must not grant another budget.
	require.ErrorIs(t, manager.ResumeActivity(ctx, instance.ID, 0, first.ActivityID, first.TemporalRunID, *first.PausedAt), ErrActivityResumeConflict)
	require.Never(t, func() bool { return calls.Load() != 30 }, 200*time.Millisecond, 20*time.Millisecond)
	unchanged := waitPaused(30, *first.PausedAt)
	require.True(t, unchanged.PausedAt.Equal(*second.PausedAt))

	ready.Store(true)
	require.NoError(t, manager.ResumeActivity(ctx, instance.ID, 0, second.ActivityID, second.TemporalRunID, *second.PausedAt))
	completionCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	require.NoError(t, run.Get(completionCtx, nil))
	require.Equal(t, int64(31), calls.Load())
	description, err = devServer.Client().DescribeWorkflowExecution(ctx, run.GetID(), run.GetRunID())
	require.NoError(t, err)
	require.Equal(t, enums.WORKFLOW_EXECUTION_STATUS_COMPLETED, description.GetWorkflowExecutionInfo().GetStatus())
	require.Empty(t, description.GetPendingActivities())
	keysMu.Lock()
	defer keysMu.Unlock()
	require.Len(t, keys, 31)
	for _, key := range keys {
		require.Equal(t, idempotencyKey, key, "all attempts and resumes must retain the original idempotency key")
	}
}
