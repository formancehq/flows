package triggers

import (
	"testing"
	"time"

	"github.com/formancehq/go-libs/v2/bun/bunconnect"
	"github.com/formancehq/go-libs/v2/bun/bundebug"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/orchestration/internal/storage"
	"github.com/formancehq/orchestration/internal/temporalworker"
	"github.com/formancehq/orchestration/internal/workflow"
	"github.com/formancehq/orchestration/internal/workflow/stages"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/client"
	worker "go.temporal.io/sdk/worker"
)

func TestWorkflow(t *testing.T) {
	t.Parallel()

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

	taskQueue := uuid.NewString()
	workflowManager := workflow.NewManager(db, devServer.Client(), "test", taskQueue, false)

	worker := temporalworker.New(logging.Testing(), devServer.Client(), taskQueue,
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
	require.NoError(t, worker.Start())
	t.Cleanup(worker.Stop)

	type testCase struct {
		EventType      string
		EventVersion   string
		EventDate      time.Time
		TriggerEvent   string
		TriggerVersion *string
	}

	testCases := []testCase{
		{
			EventType:    "NEW_TRANSACTION",
			EventVersion: "v2",
			EventDate:    time.Now().Round(time.Second).UTC(),
			TriggerEvent: "NEW_TRANSACTION",
		},
		{
			EventType:      "NEW_TRANSACTION",
			EventVersion:   "v2",
			EventDate:      time.Now().Round(time.Second).UTC(),
			TriggerEvent:   "NEW_TRANSACTION",
			TriggerVersion: pointer.For("v2"),
		},
	}

	for _, tc := range testCases {
		req := ProcessEventRequest{
			Event: publish.EventMessage{
				Type: tc.EventType,
				Date: tc.EventDate,
			},
		}

		workflow := workflow.New(workflow.Config{
			Stages: []workflow.RawStage{{
				"noop": map[string]any{},
			}},
		})
		_, err = db.
			NewInsert().
			Model(&workflow).
			Exec(logging.TestingContext())
		require.NoError(t, err)

		trigger := Trigger{
			TriggerData: TriggerData{
				Event:      tc.TriggerEvent,
				Version:    &tc.EventVersion,
				WorkflowID: workflow.ID,
			},
			ID: uuid.NewString(),
		}
		_, err = db.NewInsert().Model(&trigger).Exec(logging.TestingContext())
		require.NoError(t, err)

		ret, err := devServer.Client().
			ExecuteWorkflow(logging.TestingContext(), client.StartWorkflowOptions{
				TaskQueue: taskQueue,
			}, RunTrigger, req)
		require.NoError(t, err)
		require.NoError(t, ret.Get(logging.TestingContext(), nil))
	}
}
