package workflow

import (
	"testing"
	"time"

	bundebug "github.com/formancehq/go-libs/v5/pkg/storage/bun/debug"
	"github.com/uptrace/bun"

	"github.com/formancehq/go-libs/v5/pkg/messaging/publish"
	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	bunconnect "github.com/formancehq/go-libs/v5/pkg/storage/bun/connect"
	"github.com/formancehq/orchestration/internal/storage"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestActivities(t *testing.T) {

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

	publisher := publish.InMemory()
	activities := NewActivities(publisher, db)
	require.NoError(t, storage.Migrate(logging.TestingContext(), db))

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	env.RegisterActivity(activities.SendWorkflowTerminationEvent)
	_, err = env.ExecuteActivity(SendWorkflowTerminationEventActivity, NewInstance("vvv", "xxx"))
	require.NoError(t, err)
	require.NotEmpty(t, publisher.AllMessages())

	env.RegisterActivity(activities.InsertNewInstance)
	workflowModel := New(Config{})
	_, err = db.NewInsert().Model(&workflowModel).Exec(logging.TestingContext())
	require.NoError(t, err)

	firstValue, err := env.ExecuteActivity(InsertNewInstanceActivity, workflowModel.ID)
	require.NoError(t, err)
	var firstInstance Instance
	require.NoError(t, firstValue.Get(&firstInstance))

	time.Sleep(time.Millisecond)
	secondValue, err := env.ExecuteActivity(InsertNewInstanceActivity, workflowModel.ID)
	require.NoError(t, err)
	var secondInstance Instance
	require.NoError(t, secondValue.Get(&secondInstance))
	require.True(t, firstInstance.CreatedAt.Equal(secondInstance.CreatedAt))
	require.True(t, firstInstance.UpdatedAt.Equal(secondInstance.UpdatedAt))

	env.RegisterActivity(activities.InsertNewStage)
	firstStageValue, err := env.ExecuteActivity(InsertNewStageActivity, firstInstance, 0)
	require.NoError(t, err)
	var firstStage Stage
	require.NoError(t, firstStageValue.Get(&firstStage))

	time.Sleep(time.Millisecond)
	secondStageValue, err := env.ExecuteActivity(InsertNewStageActivity, firstInstance, 0)
	require.NoError(t, err)
	var secondStage Stage
	require.NoError(t, secondStageValue.Get(&secondStage))
	require.True(t, firstStage.StartedAt.Equal(secondStage.StartedAt))
}
