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
	worker "go.temporal.io/sdk/worker"
)

func setupTestDB(t *testing.T) *bun.DB {
	t.Helper()

	hooks := make([]bun.QueryHook, 0)
	if testing.Verbose() {
		hooks = append(hooks, bundebug.NewQueryHook())
	}

	database := srv.NewDatabase(t)
	db, err := bunconnect.OpenSQLDB(logging.TestingContext(), bunconnect.ConnectionOptions{
		DatabaseSourceName: database.ConnString(),
	}, hooks...)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	require.NoError(t, storage.Migrate(logging.TestingContext(), db))

	return db
}

func insertNoOpWorkflow(t *testing.T, db *bun.DB) workflow.Workflow {
	t.Helper()

	w := workflow.New(workflow.Config{
		Stages: []workflow.RawStage{{
			"noop": map[string]any{},
		}},
	})
	_, err := db.NewInsert().Model(&w).Exec(logging.TestingContext())
	require.NoError(t, err)

	return w
}

func insertTrigger(t *testing.T, db *bun.DB, workflowID, event string, version, filter *string) Trigger {
	t.Helper()

	trigger := Trigger{
		TriggerData: TriggerData{
			Event:      event,
			Version:    version,
			Filter:     filter,
			WorkflowID: workflowID,
		},
		ID:        uuid.NewString(),
		CreatedAt: time.Now().Round(time.Microsecond).UTC(),
	}
	_, err := db.NewInsert().Model(&trigger).Exec(logging.TestingContext())
	require.NoError(t, err)

	return trigger
}

func TestListMatchingTriggers(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name            string
		triggers        func(t *testing.T, db *bun.DB, workflowID string)
		event           publish.EventMessage
		expectedMatched int
	}

	testCases := []testCase{
		{
			name:     "no triggers",
			triggers: func(t *testing.T, db *bun.DB, workflowID string) {},
			event: publish.EventMessage{
				Type:    "NEW_TRANSACTION",
				Version: "v1",
			},
			expectedMatched: 0,
		},
		{
			name: "event type match",
			triggers: func(t *testing.T, db *bun.DB, workflowID string) {
				insertTrigger(t, db, workflowID, "NEW_TRANSACTION", nil, nil)
			},
			event: publish.EventMessage{
				Type:    "NEW_TRANSACTION",
				Version: "v1",
			},
			expectedMatched: 1,
		},
		{
			name: "event type mismatch",
			triggers: func(t *testing.T, db *bun.DB, workflowID string) {
				insertTrigger(t, db, workflowID, "SAVED_PAYMENT", nil, nil)
			},
			event: publish.EventMessage{
				Type:    "NEW_TRANSACTION",
				Version: "v1",
			},
			expectedMatched: 0,
		},
		{
			name: "version match",
			triggers: func(t *testing.T, db *bun.DB, workflowID string) {
				insertTrigger(t, db, workflowID, "NEW_TRANSACTION", pointer.For("v2"), nil)
			},
			event: publish.EventMessage{
				Type:    "NEW_TRANSACTION",
				Version: "v2",
			},
			expectedMatched: 1,
		},
		{
			name: "version mismatch",
			triggers: func(t *testing.T, db *bun.DB, workflowID string) {
				insertTrigger(t, db, workflowID, "NEW_TRANSACTION", pointer.For("v1"), nil)
			},
			event: publish.EventMessage{
				Type:    "NEW_TRANSACTION",
				Version: "v2",
			},
			expectedMatched: 0,
		},
		{
			name: "nil version matches any",
			triggers: func(t *testing.T, db *bun.DB, workflowID string) {
				insertTrigger(t, db, workflowID, "NEW_TRANSACTION", nil, nil)
			},
			event: publish.EventMessage{
				Type:    "NEW_TRANSACTION",
				Version: "v2",
			},
			expectedMatched: 1,
		},
		{
			name: "filter match",
			triggers: func(t *testing.T, db *bun.DB, workflowID string) {
				insertTrigger(t, db, workflowID, "NEW_TRANSACTION", nil, pointer.For("event.amount > 50"))
			},
			event: publish.EventMessage{
				Type:    "NEW_TRANSACTION",
				Version: "v1",
				Payload: map[string]any{"amount": 100},
			},
			expectedMatched: 1,
		},
		{
			name: "filter no match",
			triggers: func(t *testing.T, db *bun.DB, workflowID string) {
				insertTrigger(t, db, workflowID, "NEW_TRANSACTION", nil, pointer.For("event.amount > 50"))
			},
			event: publish.EventMessage{
				Type:    "NEW_TRANSACTION",
				Version: "v1",
				Payload: map[string]any{"amount": 10},
			},
			expectedMatched: 0,
		},
		{
			name: "filter error skips trigger",
			triggers: func(t *testing.T, db *bun.DB, workflowID string) {
				insertTrigger(t, db, workflowID, "NEW_TRANSACTION", nil, pointer.For("event.missing.deep"))
			},
			event: publish.EventMessage{
				Type:    "NEW_TRANSACTION",
				Version: "v1",
				Payload: map[string]any{"amount": 10},
			},
			expectedMatched: 0,
		},
		{
			name: "multiple triggers mixed",
			triggers: func(t *testing.T, db *bun.DB, workflowID string) {
				// This one matches: correct event type, no filter
				insertTrigger(t, db, workflowID, "NEW_TRANSACTION", nil, nil)
				// This one does not match: wrong event type
				insertTrigger(t, db, workflowID, "SAVED_PAYMENT", nil, nil)
				// This one does not match: filter evaluates to false
				insertTrigger(t, db, workflowID, "NEW_TRANSACTION", nil, pointer.For("event.amount > 200"))
			},
			event: publish.EventMessage{
				Type:    "NEW_TRANSACTION",
				Version: "v1",
				Payload: map[string]any{"amount": 100},
			},
			expectedMatched: 1,
		},
		{
			name: "soft deleted trigger excluded",
			triggers: func(t *testing.T, db *bun.DB, workflowID string) {
				trigger := insertTrigger(t, db, workflowID, "NEW_TRANSACTION", nil, nil)
				_, err := db.NewUpdate().
					Model(&Trigger{}).
					Where("id = ?", trigger.ID).
					Set("deleted_at = ?", time.Now()).
					Exec(logging.TestingContext())
				require.NoError(t, err)
			},
			event: publish.EventMessage{
				Type:    "NEW_TRANSACTION",
				Version: "v1",
			},
			expectedMatched: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			w := insertNoOpWorkflow(t, db)

			tc.triggers(t, db, w.ID)

			evaluator := NewDefaultExpressionEvaluator()
			matched, err := listMatchingTriggers(logging.TestingContext(), db, evaluator, tc.event)
			require.NoError(t, err)
			require.Len(t, matched, tc.expectedMatched)
		})
	}
}

func TestHandleMessage(t *testing.T) {
	t.Parallel()

	setupWorker := func(t *testing.T, db *bun.DB) string {
		t.Helper()

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

		return taskQueue
	}

	makeMessage := func(eventType, version string, payload any) *publish.EventMessage {
		return &publish.EventMessage{
			Date:    time.Now().Round(time.Second).UTC(),
			App:     "test",
			Version: version,
			Type:    eventType,
			Payload: payload,
		}
	}

	t.Run("no match returns nil", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		_ = setupWorker(t, db)

		event := makeMessage("NEW_TRANSACTION", "v1", map[string]any{})
		msg := publish.NewMessage(logging.TestingContext(), *event)

		evaluator := NewDefaultExpressionEvaluator()
		err := handleMessage(devServer.Client(), db, evaluator, "test", "test", uuid.NewString(), false, msg)
		require.NoError(t, err)
	})

	t.Run("one match creates workflow", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		taskQueue := setupWorker(t, db)

		w := insertNoOpWorkflow(t, db)
		insertTrigger(t, db, w.ID, "NEW_TRANSACTION", nil, nil)

		event := makeMessage("NEW_TRANSACTION", "v1", map[string]any{})
		msg := publish.NewMessage(logging.TestingContext(), *event)

		evaluator := NewDefaultExpressionEvaluator()
		err := handleMessage(devServer.Client(), db, evaluator, "test", "test", taskQueue, false, msg)
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			count, err := db.NewSelect().
				Model((*Occurrence)(nil)).
				Count(logging.TestingContext())
			return err == nil && count == 1
		}, 10*time.Second, 200*time.Millisecond)
	})

	t.Run("multiple matches create multiple workflows", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		taskQueue := setupWorker(t, db)

		w := insertNoOpWorkflow(t, db)
		insertTrigger(t, db, w.ID, "NEW_TRANSACTION", nil, nil)
		insertTrigger(t, db, w.ID, "NEW_TRANSACTION", nil, nil)

		event := makeMessage("NEW_TRANSACTION", "v1", map[string]any{})
		msg := publish.NewMessage(logging.TestingContext(), *event)

		evaluator := NewDefaultExpressionEvaluator()
		err := handleMessage(devServer.Client(), db, evaluator, "test", "test", taskQueue, false, msg)
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			count, err := db.NewSelect().
				Model((*Occurrence)(nil)).
				Count(logging.TestingContext())
			return err == nil && count == 2
		}, 10*time.Second, 200*time.Millisecond)
	})

	t.Run("duplicate SAVED_PAYMENT is skipped", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		taskQueue := setupWorker(t, db)

		w := insertNoOpWorkflow(t, db)
		insertTrigger(t, db, w.ID, "SAVED_PAYMENT", nil, nil)

		paymentID := uuid.NewString()
		event := makeMessage("SAVED_PAYMENT", "v1", map[string]any{"id": paymentID})

		evaluator := NewDefaultExpressionEvaluator()

		// First call
		msg1 := publish.NewMessage(logging.TestingContext(), *event)
		err := handleMessage(devServer.Client(), db, evaluator, "test", "test", taskQueue, false, msg1)
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			count, err := db.NewSelect().
				Model((*Occurrence)(nil)).
				Count(logging.TestingContext())
			return err == nil && count == 1
		}, 10*time.Second, 200*time.Millisecond)

		// Second call with same event — should be skipped (duplicate workflow ID)
		msg2 := publish.NewMessage(logging.TestingContext(), *event)
		err = handleMessage(devServer.Client(), db, evaluator, "test", "test", taskQueue, false, msg2)
		require.NoError(t, err)

		// Still only 1 occurrence after a short wait
		time.Sleep(2 * time.Second)
		count, err := db.NewSelect().
			Model((*Occurrence)(nil)).
			Count(logging.TestingContext())
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("malformed payment payload returns an error instead of being dropped", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		taskQueue := setupWorker(t, db)

		w := insertNoOpWorkflow(t, db)
		insertTrigger(t, db, w.ID, "SAVED_PAYMENT", nil, nil)

		// "id" is a number, so extracting the dedup id fails. This used to
		// panic and be silently acked; it must now surface as an error so the
		// message is NACKed.
		event := makeMessage("SAVED_PAYMENT", "v1", map[string]any{"id": 123})
		msg := publish.NewMessage(logging.TestingContext(), *event)

		evaluator := NewDefaultExpressionEvaluator()
		err := handleMessage(devServer.Client(), db, evaluator, "test", "test", taskQueue, false, msg)
		require.Error(t, err)
	})
}
