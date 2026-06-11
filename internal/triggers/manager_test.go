package triggers

import (
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func insertNamedTrigger(t *testing.T, db *bun.DB, workflowID, name string) Trigger {
	t.Helper()

	trigger := Trigger{
		TriggerData: TriggerData{
			Name:       name,
			Event:      "NEW_TRANSACTION",
			WorkflowID: workflowID,
		},
		ID:        uuid.NewString(),
		CreatedAt: time.Now().Round(time.Microsecond).UTC(),
	}
	_, err := db.NewInsert().Model(&trigger).Exec(logging.TestingContext())
	require.NoError(t, err)

	return trigger
}

func TestListTriggersNameFilter(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	w := insertNoOpWorkflow(t, db)
	insertNamedTrigger(t, db, w.ID, "payment-processor")
	insertNamedTrigger(t, db, w.ID, "ledger-sync")

	manager := &TriggerManager{db: db}

	listWithName := func(name string) []Trigger {
		t.Helper()
		cursor, err := manager.ListTriggers(logging.TestingContext(), ListTriggersQuery{
			PageSize: 15,
			Options:  ListTriggerParams{Name: name},
		})
		require.NoError(t, err)
		return cursor.Data
	}

	t.Run("substring match", func(t *testing.T) {
		got := listWithName("payment")
		require.Len(t, got, 1)
		require.Equal(t, "payment-processor", got[0].Name)
	})

	t.Run("case insensitive", func(t *testing.T) {
		got := listWithName("LEDGER")
		require.Len(t, got, 1)
		require.Equal(t, "ledger-sync", got[0].Name)
	})

	t.Run("no match", func(t *testing.T) {
		require.Empty(t, listWithName("does-not-exist"))
	})

	t.Run("no filter returns all", func(t *testing.T) {
		cursor, err := manager.ListTriggers(logging.TestingContext(), ListTriggersQuery{
			PageSize: 15,
		})
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
	})
}
