package triggers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.temporal.io/sdk/temporal"

	"github.com/formancehq/go-libs/v3/collectionutils"

	"github.com/expr-lang/expr"
	"github.com/formancehq/go-libs/v3/api"
	"github.com/pkg/errors"
)

type expressionEvaluator struct {
	httpClient *http.Client
}

func (h *expressionEvaluator) link(params ...any) (any, error) {
	if len(params) != 2 {
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("expect two arguments, got %d", len(params)),
			"APPLICATION",
			fmt.Errorf("expect two arguments, got %d", len(params)),
		)
	}

	data, _ := json.Marshal(params[0])

	type object struct {
		Links []api.Link `json:"links"`
	}
	o := &object{}
	if err := json.Unmarshal(data, o); err != nil {
		return nil, err
	}

	rel, ok := params[1].(string)
	if !ok {
		return nil, errors.New("second parameter must be a string")
	}

	filteredLinks := collectionutils.Filter(o.Links, func(link api.Link) bool {
		return link.Name == rel
	})

	switch len(filteredLinks) {
	case 0:
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("link '%s' not defined for object", rel),
			"APPLICATION",
			fmt.Errorf("link '%s' not defined for object", rel),
		)
	case 1:
		rsp, err := h.httpClient.Get(filteredLinks[0].URI)
		if err != nil {
			return nil, errors.Wrapf(err, "reading resource: %s", filteredLinks[0].URI)
		}
		if rsp.StatusCode >= 400 {
			return nil, fmt.Errorf("unexpected status code when reading resource: %d", rsp.StatusCode)
		}

		apiResponse := api.BaseResponse[map[string]any]{}
		if err := json.NewDecoder(rsp.Body).Decode(&apiResponse); err != nil {
			return nil, errors.Wrap(err, "decoding response")
		}

		return apiResponse.Data, nil
	default:
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("multiple link '%s' found for object", rel),
			"APPLICATION",
			fmt.Errorf("multiple link '%s' found for object", rel),
		)
	}
}

// ExpressionEvaluationErrorType is the Temporal ApplicationError type used for
// errors raised by the expression engine itself (compile errors, runtime errors
// such as fetching a field on nil). Those errors are deterministic: evaluating
// the same expression against the same payload always fails the same way, so
// they are returned as non-retryable to avoid wedging the calling workflow.
const ExpressionEvaluationErrorType = "EXPRESSION_EVALUATION"

// linkFunctionError marks an error returned by the link() custom function so
// eval can tell it apart from errors produced by the expression engine. expr
// wraps (via *file.Error.Unwrap) any error returned by a custom function, but
// also some of its own deterministic errors (e.g. invalid regexp), so the
// marker is the only reliable way to identify link() errors.
type linkFunctionError struct {
	err error
}

func (e *linkFunctionError) Error() string { return e.err.Error() }
func (e *linkFunctionError) Unwrap() error { return e.err }

func (h *expressionEvaluator) linkFunction(params ...any) (any, error) {
	ret, err := h.link(params...)
	if err != nil {
		return nil, &linkFunctionError{err: err}
	}
	return ret, nil
}

func (h *expressionEvaluator) eval(rawObject any, e string) (any, error) {
	p, err := expr.Compile(e, expr.Function("link", h.linkFunction))
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("compiling expression: %s", err),
			ExpressionEvaluationErrorType,
			nil,
		)
	}

	ret, err := expr.Run(p, map[string]any{
		"event": rawObject,
	})
	if err != nil {
		// Errors from link() are returned as is: transient ones (HTTP failure,
		// bad status, decoding) stay retryable, while link() already returns
		// non-retryable ApplicationErrors for deterministic cases.
		var linkErr *linkFunctionError
		if errors.As(err, &linkErr) {
			return nil, linkErr.err
		}
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("evaluating expression: %s", err),
			ExpressionEvaluationErrorType,
			nil,
		)
	}

	return ret, nil
}

func (h *expressionEvaluator) evalFilter(event any, filter string) (bool, error) {
	ret, err := h.eval(event, filter)
	if err != nil {
		return false, err
	}

	switch ret := ret.(type) {
	case bool:
		return ret, nil
	default:
		return false, nil
	}
}

func (h *expressionEvaluator) evalVariable(rawObject any, e string) (string, error) {
	ret, err := h.eval(rawObject, e)
	if err != nil {
		return "", err
	}

	switch ret.(type) {
	case float64, float32:
		data, err := json.Marshal(ret)
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return fmt.Sprint(ret), nil
	}
}

func (h *expressionEvaluator) evalVariables(rawObject any, vars map[string]string) (map[string]string, error) {
	results := make(map[string]string)
	for k, v := range vars {
		var err error
		results[k], err = h.evalVariable(rawObject, v)
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

func NewExpressionEvaluator(httpClient *http.Client) *expressionEvaluator {
	return &expressionEvaluator{
		httpClient: httpClient,
	}
}

func NewDefaultExpressionEvaluator() *expressionEvaluator {
	return NewExpressionEvaluator(http.DefaultClient)
}
