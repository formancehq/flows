package triggers

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
)

// An expression that fails deterministically against the event payload must not
// wedge ExecuteTrigger in an endless activity retry loop: the evaluation is
// attempted once, and the failure is recorded as a failed occurrence.
func TestExecuteTriggerRecordsExpressionEvaluationFailure(t *testing.T) {
	t.Parallel()

	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	env.SetTestTimeout(30 * time.Second)

	w := NewWorkflow("test", "default", false)
	activities := NewActivities(nil, nil, NewDefaultExpressionEvaluator(), publish.NoOpPublisher)
	for _, def := range activities.DefinitionSet() {
		env.RegisterActivityWithOptions(def.Func, activity.RegisterOptions{Name: def.Name})
	}

	var evalCalls atomic.Int32
	env.SetOnActivityStartedListener(func(info *activity.Info, _ context.Context, _ converter.EncodedValues) {
		if info.ActivityType.Name == "EvalTriggerVariables" {
			evalCalls.Add(1)
		}
	})

	var inserted, sent Occurrence
	env.OnActivity(InsertTriggerOccurrence, mock.Anything, mock.Anything).
		Return(func(_ context.Context, o Occurrence) error {
			inserted = o
			return nil
		}).Once()
	env.OnActivity(SendEventForTriggerTermination, mock.Anything, mock.Anything).
		Return(func(_ context.Context, o Occurrence) error {
			sent = o
			return nil
		}).Once()

	req := ProcessEventRequest{
		Event: publish.EventMessage{
			Type:    "SAVED_PAYMENT_INITIATION_ADJUSTMENT",
			Payload: savedPaymentInitiationAdjustmentPayload(),
		},
	}
	trigger := Trigger{
		ID: "trigger-1",
		TriggerData: TriggerData{
			Event: "SAVED_PAYMENT_INITIATION_ADJUSTMENT",
			Vars: map[string]string{
				"payout_cycle": payoutCycleExpression,
			},
		},
	}

	env.ExecuteWorkflow(w.ExecuteTrigger, req, trigger)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)

	require.EqualValues(t, 1, evalCalls.Load(), "EvalTriggerVariables must not be retried")
	require.NotNil(t, inserted.Error)
	require.Contains(t, *inserted.Error, "cannot fetch 0 from <nil> (1:23)")
	require.Nil(t, inserted.WorkflowInstanceID)
	require.Equal(t, "trigger-1", inserted.TriggerID)
	require.Equal(t, inserted, sent)
}
