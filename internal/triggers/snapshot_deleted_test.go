package triggers

import (
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/orchestration/internal/temporalworker"
	"github.com/formancehq/orchestration/internal/workflow"
	"github.com/formancehq/orchestration/internal/workflow/stages"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func TestLoadedTriggerSnapshotRejectedAfterDeletion(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	definition := insertNoOpWorkflow(t, db)
	trigger := insertTrigger(t, db, definition.ID, "NEW_TRANSACTION", new("v2"), nil)
	// Load the exact snapshot delivered by both listener and RunTrigger paths.
	require.NoError(t, db.NewSelect().Model(&trigger).Relation("Workflow").Where("trigger.id = ?", trigger.ID).Scan(t.Context()))
	require.Nil(t, trigger.Workflow.DeletedAt)
	taskQueue := uuid.NewString()
	manager := workflow.NewManager(db, devServer.Client(), "test", taskQueue, false)
	require.NoError(t, manager.DeleteWorkflow(t.Context(), definition.ID))
	publisher := publish.InMemory()
	w := temporalworker.New(logging.Testing(), devServer.Client(), taskQueue,
		[]temporalworker.DefinitionSet{
			NewWorkflow("test", taskQueue, false).DefinitionSet(),
			workflow.NewWorkflows("test", false).DefinitionSet(),
			temporalworker.NewDefinitionSet().Append(temporalworker.Definition{Name: "NoOp", Func: (&stages.NoOp{}).GetWorkflow()}),
		}, []temporalworker.DefinitionSet{
			workflow.NewActivities(publisher, db).DefinitionSet(),
			NewActivities(db, manager, NewDefaultExpressionEvaluator(), publisher).DefinitionSet(),
		}, worker.Options{})
	require.NoError(t, w.Start())
	t.Cleanup(w.Stop)
	run, err := devServer.Client().ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{TaskQueue: taskQueue}, ExecuteTrigger,
		ProcessEventRequest{Event: publish.EventMessage{Type: "NEW_TRANSACTION", Version: "v2"}}, trigger)
	require.NoError(t, err)
	require.NoError(t, run.Get(t.Context(), nil))
	var occurrences []Occurrence
	require.NoError(t, db.NewSelect().Model(&occurrences).Where("trigger_id = ?", trigger.ID).Scan(t.Context()))
	require.Len(t, occurrences, 1)
	require.Nil(t, occurrences[0].WorkflowInstanceID)
	require.NotNil(t, occurrences[0].Error)
	require.Equal(t, workflow.ErrWorkflowNotFound.Error(), *occurrences[0].Error)
	count, err := db.NewSelect().Model((*workflow.Instance)(nil)).Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, count)
	count, err = db.NewSelect().Model((*workflow.Stage)(nil)).Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, count, "rejected snapshot must not execute any stage")
	require.Len(t, publisher.AllMessages(), 1, "only FAILED_TRIGGER should be published")
	// Both discovery implementations exclude the now-deleted definition.
	matches, err := listMatchingTriggers(t.Context(), db, NewDefaultExpressionEvaluator(), publish.EventMessage{Type: "NEW_TRANSACTION", Version: "v2"})
	require.NoError(t, err)
	require.Empty(t, matches)
}
