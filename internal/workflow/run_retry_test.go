package workflow

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// The instance/stage bookkeeping activities used to run with no RetryPolicy, i.e. Temporal's
// unlimited default. These tests pin that they now give up after 15 attempts and fail the
// workflow instead of retrying for its whole life.

func TestInitiateGivesUpOnInsertNewInstanceAfterBoundedAttempts(t *testing.T) {
	t.Parallel()

	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	env.OnActivity(InsertNewInstanceActivity, mock.Anything, mock.Anything).
		Return(nil, errors.New("database unavailable"))

	env.ExecuteWorkflow(NewWorkflows("test", false).Initiate, Input{Workflow: New(Config{})})

	require.True(t, env.IsWorkflowCompleted())
	require.ErrorContains(t, env.GetWorkflowError(), "database unavailable")
	env.AssertActivityNumberOfCalls(t, "InsertNewInstance", 15)
}

func TestRunGivesUpOnUpdateInstanceAfterBoundedAttempts(t *testing.T) {
	t.Parallel()

	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	env.OnActivity(UpdateInstanceActivity, mock.Anything, mock.Anything).
		Return(errors.New("database unavailable"))
	env.OnActivity(SendWorkflowTerminationEventActivity, mock.Anything, mock.Anything).
		Return(nil)

	wf := New(Config{})
	env.ExecuteWorkflow(NewWorkflows("test", false).Run, Input{Workflow: wf}, NewInstance("instance", wf.ID))

	require.True(t, env.IsWorkflowCompleted())
	require.ErrorContains(t, env.GetWorkflowError(), "database unavailable")
	env.AssertActivityNumberOfCalls(t, "UpdateInstance", 15)
	env.AssertActivityNumberOfCalls(t, "SendWorkflowTerminationEvent", 0)
}
