package triggers

import (
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/publish"
	"github.com/stretchr/testify/require"
)

// A Temporal activity retry after a lost ack replays the exact same occurrence,
// since the id is part of the recorded activity input. It used to fail forever on
// triggers_occurrences_pkey, wedging ExecuteTrigger.
func TestInsertTriggerOccurrenceIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	db := setupTestDB(t)
	w := insertNoOpWorkflow(t, db)
	trigger := insertTrigger(t, db, w.ID, "NEW_TRANSACTION", nil, nil)

	activities := NewActivities(db, nil, NewDefaultExpressionEvaluator(), publish.NoOpPublisher)
	occurrence := NewTriggerOccurrence("test-workflow-id", "run-id", trigger.ID, publish.EventMessage{
		Type:    "NEW_TRANSACTION",
		Version: "v1",
		Payload: map[string]any{},
	}, time.Now().Round(time.Microsecond).UTC())

	require.NoError(t, activities.InsertTriggerOccurrence(ctx, occurrence))
	require.NoError(t, activities.InsertTriggerOccurrence(ctx, occurrence))

	count, err := db.NewSelect().Model((*Occurrence)(nil)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

// The flip side of the idempotency guarantee: ON CONFLICT DO NOTHING must only
// swallow replays of the same occurrence, never distinct ones. The occurrence id
// is keyed on workflow id + run id precisely so that a reused workflow id cannot
// make a new occurrence collide with an old one and disappear.
func TestInsertTriggerOccurrenceDistinctIDsAreSeparateRows(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	db := setupTestDB(t)
	w := insertNoOpWorkflow(t, db)
	trigger := insertTrigger(t, db, w.ID, "NEW_TRANSACTION", nil, nil)

	activities := NewActivities(db, nil, NewDefaultExpressionEvaluator(), publish.NoOpPublisher)
	newOccurrence := func(workflowId string, runId string) Occurrence {
		return NewTriggerOccurrence(workflowId, runId, trigger.ID, publish.EventMessage{
			Type:    "NEW_TRANSACTION",
			Version: "v1",
			Payload: map[string]any{},
		}, time.Now().Round(time.Microsecond).UTC())
	}

	// Same workflow id, different run id: two runs of the same logical workflow.
	require.NoError(t, activities.InsertTriggerOccurrence(ctx, newOccurrence("workflow-id", "run-id-1")))
	require.NoError(t, activities.InsertTriggerOccurrence(ctx, newOccurrence("workflow-id", "run-id-2")))

	count, err := db.NewSelect().Model((*Occurrence)(nil)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}
