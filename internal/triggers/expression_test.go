package triggers

import (
	"errors"
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

func requireApplicationError(t *testing.T, err error) *temporal.ApplicationError {
	t.Helper()
	require.Error(t, err)
	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected *temporal.ApplicationError, got %T: %v", err, err)
	return appErr
}

func TestEvalRuntimeErrorIsNonRetryable(t *testing.T) {
	t.Parallel()

	e := NewDefaultExpressionEvaluator()
	_, err := e.evalVariables(savedPaymentInitiationAdjustmentPayload(), map[string]string{
		"payout_cycle": payoutCycleExpression,
	})

	appErr := requireApplicationError(t, err)
	require.True(t, appErr.NonRetryable())
	require.Equal(t, ExpressionEvaluationErrorType, appErr.Type())
	require.Contains(t, err.Error(), "cannot fetch 0 from <nil> (1:23)")
}

func TestEvalCompileErrorIsNonRetryable(t *testing.T) {
	t.Parallel()

	e := NewDefaultExpressionEvaluator()
	_, err := e.evalVariable(map[string]any{}, `event.foo +`)

	appErr := requireApplicationError(t, err)
	require.True(t, appErr.NonRetryable())
	require.Equal(t, ExpressionEvaluationErrorType, appErr.Type())
	require.Contains(t, err.Error(), "compiling expression")
}

// Deterministic errors expr raises with an error value (here an invalid regexp)
// are wrapped by expr the same way as custom function errors; they must not be
// mistaken for link() errors.
func TestEvalEngineWrappedErrorIsNonRetryable(t *testing.T) {
	t.Parallel()

	e := NewDefaultExpressionEvaluator()
	_, err := e.evalVariable(map[string]any{"foo": "bar", "re": "("}, `event.foo matches event.re`)

	appErr := requireApplicationError(t, err)
	require.True(t, appErr.NonRetryable())
	require.Equal(t, ExpressionEvaluationErrorType, appErr.Type())
}

func linkPayload(uri string) map[string]any {
	return map[string]any{
		"links": []map[string]any{{
			"name": "source_account",
			"uri":  uri,
		}},
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

			e := NewExpressionEvaluator(http.DefaultClient)
			_, err := e.evalVariable(linkPayload(tc.uri), `link(event, "source_account").role`)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.message)

			var appErr *temporal.ApplicationError
			require.False(t, errors.As(err, &appErr),
				"transient link error must be returned as a plain (retryable) error, got %v", err)
		})
	}
}

func TestEvalLinkNonRetryableErrorIsPreserved(t *testing.T) {
	t.Parallel()

	e := NewDefaultExpressionEvaluator()
	_, err := e.evalVariable(linkPayload("http://localhost"), `link(event, "unknown").role`)

	appErr := requireApplicationError(t, err)
	require.True(t, appErr.NonRetryable())
	require.Equal(t, "APPLICATION", appErr.Type())
	require.Contains(t, err.Error(), "link 'unknown' not defined for object")
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
