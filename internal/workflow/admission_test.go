package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunconnect"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/orchestration/internal/storage"
	"github.com/formancehq/orchestration/internal/temporalworker"
	"github.com/formancehq/orchestration/internal/workflow/stages"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	temporalworkflow "go.temporal.io/sdk/workflow"
)

func admissionDB(t *testing.T) (*bun.DB, Workflow) {
	t.Helper()
	db, err := bunconnect.OpenSQLDB(t.Context(), bunconnect.ConnectionOptions{DatabaseSourceName: srv.NewDatabase(t).ConnString()})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, storage.Migrate(t.Context(), db))
	definition := New(Config{})
	_, err = db.NewInsert().Model(&definition).Exec(t.Context())
	require.NoError(t, err)
	return db, definition
}

func admissionActivity(db *bun.DB, workflowID string) (*Instance, error) {
	env := (&testsuite.WorkflowTestSuite{}).NewTestActivityEnvironment()
	env.RegisterActivity(NewActivities(publish.NoOpPublisher, db).InsertNewInstance)
	value, err := env.ExecuteActivity(InsertNewInstanceActivity, workflowID)
	if err != nil {
		return nil, err
	}
	var instance Instance
	err = value.Get(&instance)
	return &instance, err
}

func TestAdmissionRetryAfterDeletion(t *testing.T) {
	t.Parallel()
	db, definition := admissionDB(t)
	first, err := admissionActivity(db, definition.ID)
	require.NoError(t, err)
	// A lost activity acknowledgement must not turn an admitted instance into
	// a rejection, or reset its stored execution state after deletion.
	_, err = db.NewUpdate().Model((*Instance)(nil)).Set("terminated = true").Where("id = ?", first.ID).Exec(t.Context())
	require.NoError(t, err)
	require.NoError(t, NewManager(db, nil, "", "", false).DeleteWorkflow(t.Context(), definition.ID))
	replay, err := admissionActivity(db, definition.ID)
	require.NoError(t, err)
	require.Equal(t, first.ID, replay.ID)
	require.True(t, replay.Terminated)
	require.WithinDuration(t, first.CreatedAt, replay.CreatedAt, time.Microsecond)
	count, err := db.NewSelect().Model((*Instance)(nil)).Count(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func waitForAdmissionLock(t *testing.T, db *bun.DB) {
	t.Helper()
	require.Eventually(t, func() bool {
		var blocked bool
		err := db.NewRaw(`SELECT EXISTS (SELECT 1 FROM pg_stat_activity
   WHERE datname = current_database() AND pid <> pg_backend_pid()
   AND wait_event_type = 'Lock')`).Scan(t.Context(), &blocked)
		return err == nil && blocked
	}, 5*time.Second, 10*time.Millisecond, "operation must actually contend on the database lock")
}

func TestDeletionWinsAdmissionRace(t *testing.T) {
	t.Parallel()
	db, definition := admissionDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	_, err = tx.NewUpdate().Model((*Workflow)(nil)).Set("deleted_at = ?", time.Now()).Where("id = ?", definition.ID).Exec(t.Context())
	require.NoError(t, err)
	result := make(chan error, 1)
	go func() { _, err := admissionActivity(db, definition.ID); result <- err }()
	waitForAdmissionLock(t, db)
	require.NoError(t, tx.Commit())
	select {
	case err := <-result:
		appErr, ok := errors.AsType[*temporal.ApplicationError](err)
		require.True(t, ok, "%v", err)
		require.Equal(t, WorkflowNotFoundErrorType, appErr.Type())
		require.True(t, appErr.NonRetryable())
	case <-time.After(10 * time.Second):
		t.Fatal("admission did not unblock")
	}
	count, err := db.NewSelect().Model((*Instance)(nil)).Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, count)
}

type pauseAdmission struct{ inserted, release chan struct{} }

func (h pauseAdmission) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
	return ctx
}
func (h pauseAdmission) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	if strings.HasPrefix(event.Query, `INSERT INTO "workflow_instances"`) {
		close(h.inserted)
		select {
		case <-h.release:
		case <-ctx.Done():
		}
	}
}

func TestAdmissionWinsDeletionRace(t *testing.T) {
	t.Parallel()
	db, definition := admissionDB(t)
	hook := pauseAdmission{inserted: make(chan struct{}), release: make(chan struct{})}
	db.AddQueryHook(hook)
	// Always release on an assertion failure too, before the DB cleanup.
	defer func() {
		select {
		case <-hook.release:
		default:
			close(hook.release)
		}
	}()
	admitted := make(chan error, 1)
	go func() { _, err := admissionActivity(db, definition.ID); admitted <- err }()
	select {
	case <-hook.inserted:
	case <-time.After(10 * time.Second):
		t.Fatal("instance was not inserted")
	}
	deleted := make(chan error, 1)
	go func() { deleted <- NewManager(db, nil, "", "", false).DeleteWorkflow(t.Context(), definition.ID) }()
	waitForAdmissionLock(t, db)
	close(hook.release)
	require.NoError(t, <-admitted)
	require.NoError(t, <-deleted)
	count, err := db.NewSelect().Model((*Instance)(nil)).Count(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, count)
	// Same activity identity remains an admitted retry after deletion.
	_, err = admissionActivity(db, definition.ID)
	require.NoError(t, err)
}

// Register a blocking implementation for this worker's NoOp stage. It has no
// external effects and lets the test prove the instance was active at deletion.
func admissionBlockedStage(ctx temporalworkflow.Context, _ stages.NoOp) error {
	var event Event
	temporalworkflow.GetSignalChannel(ctx, EventSignalName).Receive(ctx, &event)
	return nil
}

func TestAdmittedInstanceContinuesAfterDeletion(t *testing.T) {
	t.Parallel()
	db, definition := admissionDB(t)
	definition.Config.Stages = []RawStage{{"noop": map[string]any{}}}
	_, err := db.NewUpdate().Model(&definition).WherePK().Exec(t.Context())
	require.NoError(t, err)
	taskQueue := uuid.NewString()
	w := temporalworker.New(logging.Testing(), devServer.Client(), taskQueue,
		[]temporalworker.DefinitionSet{NewWorkflows("test", false).DefinitionSet(),
			temporalworker.NewDefinitionSet().Append(temporalworker.Definition{Name: "RunNoOp", Func: admissionBlockedStage})},
		[]temporalworker.DefinitionSet{NewActivities(publish.NoOpPublisher, db).DefinitionSet()}, worker.Options{})
	require.NoError(t, w.Start())
	t.Cleanup(w.Stop)
	manager := NewManager(db, devServer.Client(), "test", taskQueue, false)
	instance, err := manager.RunWorkflow(t.Context(), definition.ID, nil)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		current, err := manager.GetInstance(t.Context(), instance.ID)
		if err != nil || current.Terminated || len(current.Statuses) != 1 || current.Statuses[0].TerminatedAt != nil {
			return false
		}
		// InsertNewStage commits before the child workflow is started. Wait
		// for Temporal too, so PostEvent cannot race that child's creation.
		_, err = devServer.Client().DescribeWorkflowExecution(t.Context(), current.Statuses[0].TemporalWorkflowID(), "")
		return err == nil
	}, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, manager.DeleteWorkflow(t.Context(), definition.ID))
	current, err := manager.GetInstance(t.Context(), instance.ID)
	require.NoError(t, err)
	require.False(t, current.Terminated)
	require.NoError(t, manager.PostEvent(t.Context(), instance.ID, Event{Name: "continue"}))
	require.Eventually(t, func() bool {
		current, err := manager.GetInstance(t.Context(), instance.ID)
		return err == nil && current.Terminated && current.Error == ""
	}, 5*time.Second, 10*time.Millisecond)
	history, err := manager.ReadInstanceHistory(t.Context(), instance.ID)
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Empty(t, history[0].Error)
}

// Pause after the manager has read an active definition, before it dispatches
// the Temporal workflow, to exercise the HTTP/backend error mapping on a race.
type pauseDefinitionRead struct{ loaded, release chan struct{} }

func (h pauseDefinitionRead) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
	return ctx
}
func (h pauseDefinitionRead) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	if strings.HasPrefix(event.Query, "SELECT") && strings.Contains(event.Query, `FROM "workflows"`) && strings.Contains(event.Query, "deleted_at IS NULL") {
		close(h.loaded)
		select {
		case <-h.release:
		case <-ctx.Done():
		}
	}
}

func TestManagerRejectsDeletionAfterLookup(t *testing.T) {
	t.Parallel()
	db, definition := admissionDB(t)
	taskQueue := uuid.NewString()
	w := temporalworker.New(logging.Testing(), devServer.Client(), taskQueue,
		[]temporalworker.DefinitionSet{NewWorkflows("test", false).DefinitionSet()},
		[]temporalworker.DefinitionSet{NewActivities(publish.NoOpPublisher, db).DefinitionSet()}, worker.Options{})
	require.NoError(t, w.Start())
	t.Cleanup(w.Stop)
	hook := pauseDefinitionRead{loaded: make(chan struct{}), release: make(chan struct{})}
	db.AddQueryHook(hook)
	defer func() {
		select {
		case <-hook.release:
		default:
			close(hook.release)
		}
	}()
	manager := NewManager(db, devServer.Client(), "test", taskQueue, false)
	result := make(chan error, 1)
	go func() { _, err := manager.RunWorkflow(t.Context(), definition.ID, nil); result <- err }()
	select {
	case <-hook.loaded:
	case <-time.After(10 * time.Second):
		t.Fatal("definition was not loaded")
	}
	require.NoError(t, manager.DeleteWorkflow(t.Context(), definition.ID))
	close(hook.release)
	select {
	case err := <-result:
		require.ErrorIs(t, err, ErrWorkflowNotFound)
	case <-time.After(10 * time.Second):
		t.Fatal("start did not reject deletion")
	}
	count, err := db.NewSelect().Model((*Instance)(nil)).Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, count)
}
