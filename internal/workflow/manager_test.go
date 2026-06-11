package workflow

import (
	"testing"
	"time"

	bundebug "github.com/formancehq/go-libs/v5/pkg/storage/bun/debug"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/worker"

	"github.com/formancehq/go-libs/v5/pkg/messaging/publish"
	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/orchestration/internal/temporalworker"
	"github.com/formancehq/orchestration/internal/workflow/stages"
	"github.com/google/uuid"

	bunconnect "github.com/formancehq/go-libs/v5/pkg/storage/bun/connect"

	"github.com/formancehq/orchestration/internal/storage"
	"github.com/stretchr/testify/require"
	common "go.temporal.io/api/common/v1"
)

func TestUnmarshalFirstPayload(t *testing.T) {
	t.Parallel()

	var v map[string]any
	// nil and empty payload sets are tolerated (no panic, no error).
	require.NoError(t, unmarshalFirstPayload(nil, &v))
	require.NoError(t, unmarshalFirstPayload(&common.Payloads{}, &v))

	// A malformed payload returns an error instead of panicking.
	err := unmarshalFirstPayload(&common.Payloads{
		Payloads: []*common.Payload{{Data: []byte("{not-json")}},
	}, &v)
	require.Error(t, err)

	// A well-formed payload decodes.
	require.NoError(t, unmarshalFirstPayload(&common.Payloads{
		Payloads: []*common.Payload{{Data: []byte(`{"a":"b"}`)}},
	}, &v))
	require.Equal(t, map[string]any{"a": "b"}, v)
}

func TestConfig(t *testing.T) {
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
	worker := temporalworker.New(logging.Testing(), devServer.Client(), taskQueue,
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
	require.NoError(t, worker.Start())
	t.Cleanup(worker.Stop)

	manager := NewManager(db, devServer.Client(), "test", taskQueue, false)

	config := Config{
		Stages: []RawStage{
			{
				"noop": map[string]any{},
			},
		},
	}
	w, err := manager.Create(logging.TestingContext(), config)
	require.NoError(t, err)

	i, err := manager.RunWorkflow(logging.TestingContext(), w.ID, map[string]string{})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		updatedInstance, err := manager.GetInstance(logging.TestingContext(), i.ID)
		require.NoError(t, err)
		return len(updatedInstance.Statuses) == 1
	}, 2*time.Second, 100*time.Millisecond)
}

func TestWait(t *testing.T) {
	t.Parallel()

	database := srv.NewDatabase(t)
	db, err := bunconnect.OpenSQLDB(logging.TestingContext(), bunconnect.ConnectionOptions{
		DatabaseSourceName: database.ConnString(),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	require.NoError(t, storage.Migrate(logging.TestingContext(), db))

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

	t.Run("waits for the -main run to terminate", func(t *testing.T) {
		config := Config{Stages: []RawStage{{"noop": map[string]any{}}}}
		wf, err := manager.Create(logging.TestingContext(), config)
		require.NoError(t, err)
		i, err := manager.RunWorkflow(logging.TestingContext(), wf.ID, map[string]string{})
		require.NoError(t, err)

		// Wait must block on the detached "-main" child, not the Initiate
		// workflow (which returns immediately). Once it returns, the instance
		// must already be terminated.
		require.NoError(t, manager.Wait(logging.TestingContext(), i.ID))

		updated, err := manager.GetInstance(logging.TestingContext(), i.ID)
		require.NoError(t, err)
		require.True(t, updated.Terminated)
	})

	t.Run("unknown instance returns ErrInstanceNotFound", func(t *testing.T) {
		err := manager.Wait(logging.TestingContext(), "does-not-exist")
		require.ErrorIs(t, err, ErrInstanceNotFound)
	})
}

func TestSoftDeletedWorkflowIsNotUsable(t *testing.T) {
	t.Parallel()

	database := srv.NewDatabase(t)
	db, err := bunconnect.OpenSQLDB(logging.TestingContext(), bunconnect.ConnectionOptions{
		DatabaseSourceName: database.ConnString(),
	}, []bun.QueryHook{})
	require.NoError(t, err)
	require.NoError(t, storage.Migrate(logging.TestingContext(), db))

	manager := NewManager(db, devServer.Client(), "test", uuid.NewString(), false)

	w, err := manager.Create(logging.TestingContext(), Config{
		Stages: []RawStage{{"noop": map[string]any{}}},
	})
	require.NoError(t, err)
	require.NoError(t, manager.DeleteWorkflow(logging.TestingContext(), w.ID))

	t.Run("ReadWorkflow excludes it", func(t *testing.T) {
		_, err := manager.ReadWorkflow(logging.TestingContext(), w.ID)
		require.Error(t, err)
	})

	t.Run("RunWorkflow refuses it", func(t *testing.T) {
		// The select filters deleted_at, so this fails before reaching Temporal.
		_, err := manager.RunWorkflow(logging.TestingContext(), w.ID, map[string]string{})
		require.Error(t, err)
	})

	t.Run("re-delete returns not found", func(t *testing.T) {
		err := manager.DeleteWorkflow(logging.TestingContext(), w.ID)
		require.ErrorIs(t, err, ErrWorkflowNotFound)
	})
}
