package triggers

import (
	"errors"
	"testing"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// Every trigger activity gives up after 15 attempts and fails the workflow.

func TestRunTriggerGivesUpOnListTriggersAfterBoundedAttempts(t *testing.T) {
	t.Parallel()

	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	env.OnActivity(ListTriggersActivity, mock.Anything, mock.Anything).
		Return(nil, errors.New("database unavailable"))

	env.ExecuteWorkflow(NewWorkflow("test", "test", false).RunTrigger, ProcessEventRequest{
		Event: publish.EventMessage{Type: "NEW_TRANSACTION"},
	})

	require.True(t, env.IsWorkflowCompleted())
	require.ErrorContains(t, env.GetWorkflowError(), "database unavailable")
	env.AssertActivityNumberOfCalls(t, "ListTriggers", 15)
}

func TestExecuteTriggerGivesUpAfterBoundedAttempts(t *testing.T) {
	t.Parallel()

	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	// A variables evaluation that keeps failing is recorded on the occurrence once its attempts
	// are exhausted, rather than retried forever...
	env.OnActivity(EvalTriggerVariables, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("cannot evaluate variables"))
	// ...and an occurrence insert that keeps failing fails the workflow once its attempts are
	// exhausted, so the termination event is never sent.
	env.OnActivity(InsertTriggerOccurrence, mock.Anything, mock.Anything).
		Return(errors.New("duplicate key value violates unique constraint"))
	env.OnActivity(SendEventForTriggerTermination, mock.Anything, mock.Anything).
		Return(nil)

	env.ExecuteWorkflow(NewWorkflow("test", "test", false).ExecuteTrigger, ProcessEventRequest{
		Event: publish.EventMessage{Type: "NEW_TRANSACTION"},
	}, Trigger{ID: "trigger"})

	require.True(t, env.IsWorkflowCompleted())
	require.ErrorContains(t, env.GetWorkflowError(), "duplicate key value violates unique constraint")
	env.AssertActivityNumberOfCalls(t, "EvalTriggerVariables", 15)
	env.AssertActivityNumberOfCalls(t, "InsertTriggerOccurrence", 15)
	env.AssertActivityNumberOfCalls(t, "SendEventForTriggerTermination", 0)
}
