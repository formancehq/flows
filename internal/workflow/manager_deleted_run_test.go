package workflow

import (
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunconnect"
	"github.com/formancehq/go-libs/v3/bun/bundebug"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/orchestration/internal/storage"
	"github.com/formancehq/orchestration/internal/temporalworker"
	"github.com/formancehq/orchestration/internal/workflow/stages"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/worker"
)

// SEC-94 regression at the shared backend boundary (api.Backend -> WorkflowManager.RunWorkflow,
// used by both API v1 and v2 run handlers): RunWorkflow on a soft-deleted workflow
// must fail instead of starting a new Temporal execution. Product decision
// 2026-11-16: 404/410 semantics are handled at the HTTP layer; at this layer the
// contract is a non-nil error and no new instance.
func TestRunWorkflowOnDeletedWorkflowFails(t *testing.T) {
	t.Parallel()

	hooks := make([]bun.QueryHook, 0)
	if testing.Verbose() {
		hooks = append(hooks, bundebug.NewQueryHook())
	}

	ctx := logging.TestingContext()
	database := srv.NewDatabase(t)
	db, err := bunconnect.OpenSQLDB(ctx, bunconnect.ConnectionOptions{
		DatabaseSourceName: database.ConnString(),
	}, hooks...)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	require.NoError(t, storage.Migrate(ctx, db))

	taskQueue := uuid.NewString()
	w := temporalworker.New(logging.Testing(), devServer.Client(), taskQueue,
		[]temporalworker.DefinitionSet{
			NewWorkflows("test", false).DefinitionSet(),
			temporalworker.NewDefinitionSet().Append(temporalworker.Definition{
				Name: "NoOp",
				Func: (&stages.NoOp{}).GetWorkflow(),
			}),
		},
		[]temporalworker.DefinitionSet{
			NewActivities(publish.NoOpPublisher, db).DefinitionSet(),
		},
		worker.Options{},
	)
	require.NoError(t, w.Start())
	t.Cleanup(w.Stop)

	manager := NewManager(db, devServer.Client(), "test", taskQueue, false)

	workflow := New(Config{
		Stages: []RawStage{{
			"noop": map[string]any{},
		}},
	})
	_, err = db.NewInsert().Model(&workflow).Exec(ctx)
	require.NoError(t, err)

	// Legitimate flow preserved: running a live workflow works.
	instance, err := manager.RunWorkflow(ctx, workflow.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, instance)

	// Soft-delete the definition.
	require.NoError(t, manager.DeleteWorkflow(ctx, workflow.ID))

	// Historical access preserved: the pre-deletion instance is still readable.
	previous, err := manager.GetInstance(ctx, instance.ID)
	require.NoError(t, err)
	require.Equal(t, workflow.ID, previous.WorkflowID)

	// Attack: RunWorkflow with the retained ID of the deleted definition.
	newInstance, err := manager.RunWorkflow(ctx, workflow.ID, nil)
	require.Error(t, err, "SEC-94: RunWorkflow on a deleted workflow must return an error")
	require.Nil(t, newInstance)

	// No new instance row for the deleted definition.
	count, err := db.NewSelect().Model((*Instance)(nil)).
		Where("workflow_id = ?", workflow.ID).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count,
		"SEC-94: deleted workflow must not gain new instances (only the 1 pre-deletion instance expected)")

	// Repeated deletion retains the existing idempotent API behavior.
	require.NoError(t, manager.DeleteWorkflow(ctx, workflow.ID))
}

// SEC-94 boundary sanity: deleting a workflow must not disturb a running
// instance (no cancellation) — the product decision explicitly preserves
// already-running instances.
func TestDeleteWorkflowKeepsRunningInstanceAccessible(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	database := srv.NewDatabase(t)
	db, err := bunconnect.OpenSQLDB(ctx, bunconnect.ConnectionOptions{
		DatabaseSourceName: database.ConnString(),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	require.NoError(t, storage.Migrate(ctx, db))

	taskQueue := uuid.NewString()
	w := temporalworker.New(logging.Testing(), devServer.Client(), taskQueue,
		[]temporalworker.DefinitionSet{
			NewWorkflows("test", false).DefinitionSet(),
			temporalworker.NewDefinitionSet().Append(temporalworker.Definition{
				Name: "NoOp",
				Func: (&stages.NoOp{}).GetWorkflow(),
			}),
		},
		[]temporalworker.DefinitionSet{
			NewActivities(publish.NoOpPublisher, db).DefinitionSet(),
		},
		worker.Options{},
	)
	require.NoError(t, w.Start())
	t.Cleanup(w.Stop)

	manager := NewManager(db, devServer.Client(), "test", taskQueue, false)

	wf := New(Config{
		Stages: []RawStage{{
			"noop": map[string]any{},
		}},
	})
	_, err = db.NewInsert().Model(&wf).Exec(ctx)
	require.NoError(t, err)

	instance, err := manager.RunWorkflow(ctx, wf.ID, nil)
	require.NoError(t, err)

	require.NoError(t, manager.DeleteWorkflow(ctx, wf.ID))

	// The already-created instance must still be retrievable after deletion.
	got, err := manager.GetInstance(ctx, instance.ID)
	require.NoError(t, err)
	require.Equal(t, wf.ID, got.WorkflowID)

	// The workflow row itself is soft-deleted, not removed.
	var row Workflow
	require.NoError(t, db.NewSelect().Model(&row).Where("id = ?", wf.ID).Scan(ctx))
	require.NotNil(t, row.DeletedAt)
	require.False(t, row.DeletedAt.IsZero())
	require.WithinDuration(t, time.Now(), *row.DeletedAt, time.Minute)
}
