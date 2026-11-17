package triggers

import (
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunconnect"
	"github.com/formancehq/go-libs/v3/bun/bundebug"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/orchestration/internal/storage"
	"github.com/formancehq/orchestration/internal/temporalworker"
	"github.com/formancehq/orchestration/internal/workflow"
	"github.com/formancehq/orchestration/internal/workflow/stages"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// SEC-94 analysis/regression for the trigger path: ListTriggersActivity loads
// triggers with Relation("Workflow") but never checks workflow.deleted_at, and
// ExecuteTrigger then starts workflow.Initiate with the embedded workflow
// snapshot (*trigger.Workflow). After the workflow definition is soft-deleted,
// an event matching the trigger still starts a new instance of the deleted
// definition.
//
// NOTE on semantics (open decision surfaced to the orchestrator, per product
// 2026-11-16): a deleted workflow reached through a trigger is currently a
// *successful* occurrence that starts an instance. The fix options are
// "skip silently" vs "failed occurrence"; this test pins the security
// invariant only — no new instance of a deleted definition may be started —
// and leaves the occurrence-status choice to the correction phase.
func TestRunTriggerDoesNotStartDeletedWorkflow(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()

	hooks := make([]bun.QueryHook, 0)
	if testing.Verbose() {
		hooks = append(hooks, bundebug.NewQueryHook())
	}

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
	workflowManager := workflow.NewManager(db, devServer.Client(), "test", taskQueue, false)

	w := temporalworker.New(logging.Testing(), devServer.Client(), taskQueue,
		[]temporalworker.DefinitionSet{
			NewWorkflow("test", taskQueue, false).DefinitionSet(),
			workflow.NewWorkflows("test", false).DefinitionSet(),
			temporalworker.NewDefinitionSet().Append(temporalworker.Definition{
				Name: "NoOp",
				Func: (&stages.NoOp{}).GetWorkflow(),
			}),
		},
		[]temporalworker.DefinitionSet{
			workflow.NewActivities(publish.NoOpPublisher, db).DefinitionSet(),
			NewActivities(db, workflowManager, NewDefaultExpressionEvaluator(), publish.NoOpPublisher).DefinitionSet(),
		},
		worker.Options{},
	)
	require.NoError(t, w.Start())
	t.Cleanup(w.Stop)

	// Workflow definition + trigger on NEW_TRANSACTION.
	wf := workflow.New(workflow.Config{
		Stages: []workflow.RawStage{{
			"noop": map[string]any{},
		}},
	})
	_, err = db.NewInsert().Model(&wf).Exec(ctx)
	require.NoError(t, err)

	version := "v2"
	trigger := Trigger{
		TriggerData: TriggerData{
			Event:      "NEW_TRANSACTION",
			Version:    &version,
			WorkflowID: wf.ID,
		},
		ID:        uuid.NewString(),
		CreatedAt: time.Now().Round(time.Microsecond).UTC(),
	}
	_, err = db.NewInsert().Model(&trigger).Exec(ctx)
	require.NoError(t, err)

	// Sanity: with a live workflow, the trigger starts an instance (observed via
	// the recorded trigger occurrence, which carries the instance ID).
	req := ProcessEventRequest{
		Event: publish.EventMessage{
			Type:    "NEW_TRANSACTION",
			Version: "v2",
			Date:    time.Now().Round(time.Second).UTC(),
			Payload: map[string]any{},
		},
	}
	run, err := devServer.Client().ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		TaskQueue: taskQueue,
	}, RunTrigger, req)
	require.NoError(t, err)
	require.NoError(t, run.Get(ctx, nil))
	// Wait for the occurrence row to be visible (the run completing implies the
	// InsertTriggerOccurrence activity already executed).
	var first *Occurrence
	require.Eventually(t, func() bool {
		occurrences := make([]Occurrence, 0)
		if err := db.NewSelect().Model(&occurrences).
			Where("trigger_id = ?", trigger.ID).
			Scan(ctx); err != nil {
			return false
		}
		if len(occurrences) == 0 {
			return false
		}
		first = &occurrences[0]
		return true
	}, 10*time.Second, 50*time.Millisecond)
	require.NotNil(t, first.WorkflowInstanceID, "live workflow: trigger should have started an instance")
	require.Empty(t, deref(first.Error))

	count, err := db.NewSelect().Model((*workflow.Instance)(nil)).
		Where("workflow_id = ?", wf.ID).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count, "live workflow: trigger should have started 1 instance before deletion")

	// Soft-delete the workflow definition; the trigger row is untouched.
	require.NoError(t, workflowManager.DeleteWorkflow(ctx, wf.ID))

	// Attack: a new matching event arrives after the definition was deleted.
	run, err = devServer.Client().ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		TaskQueue: taskQueue,
	}, RunTrigger, req)
	require.NoError(t, err)
	require.NoError(t, run.Get(ctx, nil))

	// Security invariant: no NEW instance of the deleted definition may exist.
	count, err = db.NewSelect().Model((*workflow.Instance)(nil)).
		Where("workflow_id = ?", wf.ID).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count,
		"SEC-94: after deletion only the pre-deletion instance may exist "+
			"(embedded workflow snapshot in the triggers row bypasses the workflows table)")

	// The pre-deletion instance is preserved (historical access, no cancellation).
	instances := make([]workflow.Instance, 0)
	require.NoError(t, db.NewSelect().Model(&instances).
		Where("workflow_id = ?", wf.ID).
		Scan(ctx))
	require.Len(t, instances, 1)
	require.Equal(t, *first.WorkflowInstanceID, instances[0].ID)
}

// SEC-94: ListTriggersActivity itself must exclude triggers whose workflow is
// soft-deleted. This pins the read side at the DB query level, independently of
// the Temporal workflow layer. (Does not touch the trigger name filter/query
// owned by SEC-85 in manager.go.)
func TestListTriggersExcludesDeletedWorkflow(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	db := setupTestDB(t)

	w := insertNoOpWorkflow(t, db)
	trigger := insertTrigger(t, db, w.ID, "NEW_TRANSACTION", pointer.For("v2"), nil)

	// Soft-delete the workflow directly (same UPDATE the manager performs).
	_, err := db.NewUpdate().
		Model((*workflow.Workflow)(nil)).
		Where("id = ?", w.ID).
		Set("deleted_at = ?", time.Now()).
		Exec(ctx)
	require.NoError(t, err)

	activities := NewActivities(db, nil, NewDefaultExpressionEvaluator(), publish.NoOpPublisher)
	triggers, err := activities.ListTriggers(ctx, ProcessEventRequest{
		Event: publish.EventMessage{
			Type:    "NEW_TRANSACTION",
			Version: "v2",
			Payload: map[string]any{},
		},
	})
	require.NoError(t, err)
	for _, got := range triggers {
		require.NotEqual(t, trigger.ID, got.ID,
			"SEC-94: ListTriggers must not return triggers bound to a soft-deleted workflow")
	}
}
