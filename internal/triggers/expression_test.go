package triggers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

// savedPaymentInitiationAdjustmentPayload mirrors a Connectivity
// SAVED_PAYMENT_INITIATION_ADJUSTMENT event: it has no `transactions` field.
func savedPaymentInitiationAdjustmentPayload() map[string]any {
	return map[string]any{
		"id":                  "adj-1",
		"paymentInitiationID": "pi-1",
		"status":              "FAILED",
		"amount":              float64(100),
		"asset":               "EUR/2",
		"error":               "provider rejected",
	}
}

const payoutCycleExpression = `get(event.transactions[0].metadata, "payout_cycle") ?? ""`

func requireApplicationError(t *testing.T, err error, errType string, nonRetryable bool) {
	t.Helper()
	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, errType, appErr.Type())
	require.Equal(t, nonRetryable, appErr.NonRetryable())
}

func linkPayload(uri string) map[string]any {
	return map[string]any{
		"links": []map[string]any{{
			"name": "source_account",
			"uri":  uri,
		}},
	}
}

func TestEvalDeterministicErrorsAreNonRetryable(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		payload map[string]any
		expr    string
		errType string
		message string
	}{
		"runtime error": {
			savedPaymentInitiationAdjustmentPayload(), payoutCycleExpression,
			ExpressionEvaluationErrorType, "cannot fetch 0 from <nil> (1:23)",
		},
		"compile error": {
			map[string]any{}, `event.foo +`,
			ExpressionEvaluationErrorType, "unexpected token",
		},
		// expr wraps this engine error like a custom function error; it must not pass as a link() error.
		"invalid regexp": {
			map[string]any{"foo": "bar", "re": "("}, `event.foo matches event.re`,
			ExpressionEvaluationErrorType, "missing closing )",
		},
		"unknown link": {
			linkPayload("http://localhost"), `link(event, "unknown").role`,
			"APPLICATION", "link 'unknown' not defined for object",
		},
		"invalid link argument": {
			linkPayload("http://localhost"), `link(event, 1).role`,
			"APPLICATION", "second parameter must be a string",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			a := NewActivities(nil, nil, NewDefaultExpressionEvaluator(), publish.NoOpPublisher)
			_, err := a.EvalTriggerVariables(t.Context(), Trigger{
				TriggerData: TriggerData{Vars: map[string]string{"v": tc.expr}},
			}, ProcessEventRequest{Event: publish.EventMessage{Payload: tc.payload}})
			requireApplicationError(t, err, tc.errType, true)
			require.Contains(t, err.Error(), tc.message)
		})
	}
}

func TestEvalLinkTransientErrorsStayRetryable(t *testing.T) {
	t.Parallel()

	failingSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failingSrv.Close)

	badBodySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	t.Cleanup(badBodySrv.Close)

	closedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := closedSrv.URL
	closedSrv.Close()

	for name, tc := range map[string]struct {
		uri     string
		message string
	}{
		"status code >= 400": {failingSrv.URL, "unexpected status code when reading resource: 500"},
		"decoding failure":   {badBodySrv.URL, "decoding response"},
		"http get failure":   {closedURL, "reading resource"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := NewDefaultExpressionEvaluator().evalVariable(linkPayload(tc.uri), `link(event, "source_account").role`)
			requireApplicationError(t, err, "LINK", false)
			require.Contains(t, err.Error(), tc.message)
		})
	}
}

// Filter errors are swallowed into a non-match; the non-retryable wrapping must
// not change that.
func TestFilterEvaluationErrorDoesNotMatch(t *testing.T) {
	t.Parallel()

	filter := `event.transactions[0].metadata.foo == "bar"`
	e := NewDefaultExpressionEvaluator()

	ok, err := e.evalFilter(savedPaymentInitiationAdjustmentPayload(), filter)
	require.Error(t, err)
	require.False(t, ok)

	a := NewActivities(nil, nil, e, publish.NoOpPublisher)
	require.False(t, a.processTrigger(t.Context(), ProcessEventRequest{
		Event: publish.EventMessage{Payload: savedPaymentInitiationAdjustmentPayload()},
	}, Trigger{
		TriggerData: TriggerData{Filter: &filter},
	}))
}
