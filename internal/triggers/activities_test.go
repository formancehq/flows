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
	occurrence := NewTriggerOccurrence("test-workflow-execution-id", trigger.ID, publish.EventMessage{
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
