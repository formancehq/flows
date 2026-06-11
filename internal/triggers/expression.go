package triggers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"go.temporal.io/sdk/temporal"

	"github.com/formancehq/go-libs/v3/collectionutils"

	"github.com/expr-lang/expr"
	"github.com/formancehq/go-libs/v3/api"
	"github.com/pkg/errors"
)

type expressionEvaluator struct {
	httpClient *http.Client
	// allowedHosts is the set of hosts link() is permitted to call. It exists
	// to prevent the (credential-bearing) HTTP client from being pointed at an
	// arbitrary, attacker-controlled host via a user-defined trigger
	// expression (SSRF + bearer-token exfiltration). An empty set denies every
	// network call.
	allowedHosts map[string]struct{}
}

// checkLinkURL enforces that a link() target uses an http(s) scheme and points
// at an allow-listed host (typically the stack gateway the HTTP client is
// scoped to).
func (h *expressionEvaluator) checkLinkURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("invalid link url: %s", raw), "APPLICATION", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("link url scheme not allowed: %q", u.Scheme), "APPLICATION",
			fmt.Errorf("scheme %q not allowed", u.Scheme))
	}
	if _, ok := h.allowedHosts[strings.ToLower(u.Host)]; !ok {
		return temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("link url host not allowed: %q", u.Host), "APPLICATION",
			fmt.Errorf("host %q is not in the allowlist", u.Host))
	}
	return nil
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
		if err := h.checkLinkURL(filteredLinks[0].URI); err != nil {
			return nil, err
		}
		rsp, err := h.httpClient.Get(filteredLinks[0].URI)
		if err != nil {
			return nil, errors.Wrapf(err, "reading resource: %s", filteredLinks[0].URI)
		}
		defer func() {
			_ = rsp.Body.Close()
		}()
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

func (h *expressionEvaluator) eval(rawObject any, e string) (any, error) {
	p, err := expr.Compile(e, expr.Function("link", h.link))
	if err != nil {
		return "", err
	}

	ret, err := expr.Run(p, map[string]any{
		"event": rawObject,
	})
	if err != nil {
		if err := errors.Unwrap(err); err != nil {
			return nil, err
		}
		return nil, err
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

// NewExpressionEvaluator builds an evaluator whose link() function may only
// reach the provided hosts. Each entry may be a bare host ("example.com:8080")
// or a full URL, in which case only its host is retained. With no allowed host,
// link() network calls are denied.
func NewExpressionEvaluator(httpClient *http.Client, allowedHosts ...string) *expressionEvaluator {
	hosts := make(map[string]struct{}, len(allowedHosts))
	for _, h := range allowedHosts {
		if h == "" {
			continue
		}
		if u, err := url.Parse(h); err == nil && u.Host != "" {
			hosts[strings.ToLower(u.Host)] = struct{}{}
			continue
		}
		hosts[strings.ToLower(h)] = struct{}{}
	}
	return &expressionEvaluator{
		httpClient:   httpClient,
		allowedHosts: hosts,
	}
}

func NewDefaultExpressionEvaluator() *expressionEvaluator {
	return NewExpressionEvaluator(http.DefaultClient)
}
