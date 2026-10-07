package triggers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/orchestration/internal/workflow"
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	successSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"metadata":{"payout_cycle":"cycle-1"}}}`))
	}))
	t.Cleanup(successSrv.Close)

	for name, tc := range map[string]struct {
		payload    map[string]any
		expression string
		message    string
	}{
		"permanent error after fetch": {linkPayload(successSrv.URL), `[link(event, "source_account"), link(event, "unknown")]`, "link 'unknown' not defined for object"},
		"expression error":            {savedPaymentInitiationAdjustmentPayload(), payoutCycleExpression, "cannot fetch 0 from <nil> (1:23)"},
		"missing linked resource":     {linkPayload(srv.URL), `link(event, "source_account").role`, "unexpected status code when reading resource: 404"},
	} {
		t.Run(name, func(t *testing.T) {
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
					Payload: tc.payload,
				},
			}
			trigger := Trigger{
				ID: "trigger-1",
				TriggerData: TriggerData{
					Event: "SAVED_PAYMENT_INITIATION_ADJUSTMENT",
					Vars: map[string]string{
						"payout_cycle": tc.expression,
					},
				},
			}

			env.ExecuteWorkflow(w.ExecuteTrigger, req, trigger)

			require.True(t, env.IsWorkflowCompleted())
			require.NoError(t, env.GetWorkflowError())
			env.AssertExpectations(t)

			require.EqualValues(t, 1, evalCalls.Load(), "EvalTriggerVariables must not be retried")
			require.NotNil(t, inserted.Error)
			require.Contains(t, *inserted.Error, tc.message)
			require.Nil(t, inserted.WorkflowInstanceID)
			require.Equal(t, "trigger-1", inserted.TriggerID)
			require.Equal(t, inserted, sent)
		})
	}
}

func TestExecuteTriggerRetriesMutableLinkedData(t *testing.T) {
	t.Parallel()
	for name, succeeds := range map[string]bool{"eventually available": true, "always incomplete": false} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				attempt := requests.Add(1)
				if succeeds && attempt > 1 {
					_, _ = w.Write([]byte(`{"data":{"metadata":{"payout_cycle":"cycle-1"}}}`))
				} else {
					_, _ = w.Write([]byte(`{"data":{"metadata":null}}`))
				}
			}))
			t.Cleanup(srv.Close)
			env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
			env.SetTestTimeout(30 * time.Second)
			activities := NewActivities(nil, nil, NewExpressionEvaluator(srv.Client()), publish.NoOpPublisher)
			for _, def := range activities.DefinitionSet() {
				env.RegisterActivityWithOptions(def.Func, activity.RegisterOptions{Name: def.Name})
			}
			env.RegisterWorkflow(workflow.Initiate)
			if succeeds {
				env.OnWorkflow(workflow.Initiate, mock.Anything, mock.MatchedBy(func(input workflow.Input) bool {
					return input.Variables["payout_cycle"] == "cycle-1"
				})).Return(&workflow.Instance{ID: "instance-1"}, nil).Once()
			}
			var inserted, sent Occurrence
			env.OnActivity(InsertTriggerOccurrence, mock.Anything, mock.Anything).
				Return(func(_ context.Context, o Occurrence) error { inserted = o; return nil }).Once()
			env.OnActivity(SendEventForTriggerTermination, mock.Anything, mock.Anything).
				Return(func(_ context.Context, o Occurrence) error { sent = o; return nil }).Once()
			env.ExecuteWorkflow(NewWorkflow("test", "default", false).ExecuteTrigger,
				ProcessEventRequest{Event: publish.EventMessage{Payload: linkPayload(srv.URL)}},
				Trigger{ID: "trigger-1", TriggerData: TriggerData{
					Workflow: new(workflow.New(workflow.Config{})),
					Vars:     map[string]string{"payout_cycle": `link(event, "source_account").metadata.payout_cycle`},
				}})
			require.True(t, env.IsWorkflowCompleted())
			require.NoError(t, env.GetWorkflowError())
			env.AssertExpectations(t)
			require.Equal(t, inserted, sent)
			if succeeds {
				require.EqualValues(t, 2, requests.Load())
				require.Nil(t, inserted.Error)
				require.NotNil(t, inserted.WorkflowInstanceID)
				require.Equal(t, "instance-1", *inserted.WorkflowInstanceID)
			} else {
				require.EqualValues(t, 15, requests.Load())
				require.NotNil(t, inserted.Error)
				require.Contains(t, *inserted.Error, "cannot fetch payout_cycle from <nil>")
				require.Nil(t, inserted.WorkflowInstanceID)
			}
		})
	}
}
