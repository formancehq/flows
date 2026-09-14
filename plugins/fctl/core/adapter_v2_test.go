package core

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
	"github.com/formancehq/orchestration/plugins/fctl/audit"
)

func executeRequest(id string, args []string, flags ...sdk.FlagOccurrence) sdk.ExecuteRequest {
	return sdk.ExecuteRequest{CommandID: id, Arguments: args, Flags: flags, Target: sdk.TargetSelection{OrganizationID: "org", StackID: "stack"}, ServiceVersions: []sdk.ServiceVersion{{Service: sdk.ServiceFlows, Version: "2.0.0", Major: 2}}, Continuation: sdk.SinglePageContinuationControl()}
}

func TestInstancesStopUsesOpenAPIMethod(t *testing.T) {
	report, err := audit.Build("../../../openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	method := ""
	for _, operation := range report.Operations {
		if operation.OperationID == "v2CancelEvent" {
			method = operation.Method
			break
		}
	}
	if method == "" {
		t.Fatal("OpenAPI has no v2CancelEvent operation")
	}

	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		if request.HTTP.Method != method {
			t.Fatalf("instances stop method=%q, OpenAPI method=%q", request.HTTP.Method, method)
		}
		return sdk.NewResponseStream(sdk.Response{Status: 204}), nil
	})
	if err := (Plugin{}).Execute(context.Background(), executeRequest("flows.v2.instances.stop", []string{"instance"}), host); err != nil {
		t.Fatal(err)
	}
}

func TestExecuteCoversEveryCommandFamily(t *testing.T) {
	tests := []struct {
		id, operation string
		args          []string
		flags         []sdk.FlagOccurrence
		method, path  string
		body          bool
		status        int32
		shape         sdk.ResultShape
	}{
		{"flows.v2.triggers.list", "v2ListTriggers", nil, []sdk.FlagOccurrence{{Name: "name", Value: "hello"}, {Name: "page-size", Value: "20"}}, "GET", "/v2/triggers", false, 200, sdk.ResultCollection},
		{"flows.v2.triggers.show", "v2ReadTrigger", []string{"t 1"}, nil, "GET", "/v2/triggers/t%201", false, 200, sdk.ResultObject},
		{"flows.v2.triggers.create", "v2CreateTrigger", []string{"payments", "wf1"}, []sdk.FlagOccurrence{{Name: "vars", Value: `{"account":"event.account"}`}}, "POST", "/v2/triggers", true, 201, sdk.ResultObject},
		{"flows.v2.triggers.delete", "v2DeleteTrigger", []string{"t1"}, nil, "DELETE", "/v2/triggers/t1", false, 204, sdk.ResultEmpty},
		{"flows.v2.triggers.test", "testTrigger", []string{"t1", `{"type":"payments"}`}, nil, "POST", "/v2/triggers/t1/test", true, 200, sdk.ResultObject},
		{"flows.v2.triggers.occurrences.list", "v2ListTriggersOccurrences", []string{"t1"}, nil, "GET", "/v2/triggers/t1/occurrences", false, 200, sdk.ResultCollection},
		{"flows.v2.workflows.list", "v2ListWorkflows", nil, nil, "GET", "/v2/workflows", false, 200, sdk.ResultCollection},
		{"flows.v2.workflows.show", "v2GetWorkflow", []string{"wf1"}, nil, "GET", "/v2/workflows/wf1", false, 200, sdk.ResultObject},
		{"flows.v2.workflows.create", "v2CreateWorkflow", []string{`{"name":"wf","stages":[]}`}, nil, "POST", "/v2/workflows", true, 201, sdk.ResultObject},
		{"flows.v2.workflows.delete", "v2DeleteWorkflow", []string{"wf1"}, nil, "DELETE", "/v2/workflows/wf1", false, 204, sdk.ResultEmpty},
		{"flows.v2.workflows.run", "v2RunWorkflow", []string{"wf1"}, []sdk.FlagOccurrence{{Name: "variables", Value: `{"x":"y"}`}}, "POST", "/v2/workflows/wf1/instances", true, 201, sdk.ResultObject},
		{"flows.v2.instances.list", "v2ListInstances", nil, []sdk.FlagOccurrence{{Name: "workflow-id", Value: "wf1"}, {Name: "running", Value: "true"}}, "GET", "/v2/instances", false, 200, sdk.ResultCollection},
		{"flows.v2.instances.show", "v2GetInstance", []string{"i1"}, nil, "GET", "/v2/instances/i1", false, 200, sdk.ResultObject},
		{"flows.v2.instances.describe", "v2GetInstanceHistory", []string{"i1"}, nil, "GET", "/v2/instances/i1/history", false, 200, sdk.ResultObject},
		{"flows.v2.instances.send-event", "v2SendEvent", []string{"i1", "approved"}, nil, "POST", "/v2/instances/i1/events", true, 204, sdk.ResultEmpty},
		{"flows.v2.instances.stop", "v2CancelEvent", []string{"i1"}, nil, "PUT", "/v2/instances/i1/abort", false, 204, sdk.ResultEmpty},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			host := sdk.NewMemoryHost(func(_ context.Context, r sdk.Request) (sdk.Responses, error) {
				if _, exists := r.HTTP.Headers["authorization"]; exists {
					t.Fatalf("generated client injected product auth: headers=%#v", r.HTTP.Headers)
				}
				if r.Operation == "v2GetWorkflow" {
					return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"data":{"id":"wf1"}}`)}), nil
				}
				if r.Operation == "v2GetInstanceStageHistory" {
					return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"data":[]}`)}), nil
				}
				if r.Operation != tt.operation || r.Service != sdk.ServiceFlows || r.Capability != "auth.stack" {
					t.Fatalf("identity=%#v", r)
				}
				if r.HTTP.Method != tt.method || r.HTTP.Path != tt.path {
					t.Fatalf("http=%#v", r.HTTP)
				}
				if tt.body != (len(r.HTTP.Body) > 0) {
					t.Fatalf("body=%s", r.HTTP.Body)
				}
				body := []byte(`{"data":{"id":"one","workflowID":"wf1"}}`)
				if tt.shape == sdk.ResultCollection {
					body = []byte(`{"cursor":{"pageSize":1,"hasMore":false,"data":[]}}`)
				}
				if tt.id == "flows.v2.instances.describe" {
					body = []byte(`{"data":[{"name":"stage-one","input":{"duration":"1s"},"terminated":false,"startedAt":"2026-01-01T00:00:00Z"}]}`)
				}
				if tt.status == 204 {
					body = nil
				}
				ct := "application/json"
				if len(body) == 0 {
					ct = ""
				}
				return sdk.NewResponseStream(sdk.Response{Status: tt.status, ContentType: ct, Body: body}), nil
			})
			err := (Plugin{}).Execute(context.Background(), executeRequest(tt.id, tt.args, tt.flags...), host)
			if err != nil {
				t.Fatalf("execute: %v", err)
			}
			events := host.Events()
			if len(events) != 1 || events[0].Result == nil {
				t.Fatalf("events=%#v", events)
			}
			if events[0].Result.OperationID != tt.id {
				t.Fatalf("result operation=%q want command id %q", events[0].Result.OperationID, tt.id)
			}
			if events[0].Result.Shape != tt.shape || events[0].Result.MediaType != "application/json" || !json.Valid(events[0].Result.Data) {
				t.Fatalf("result form=%#v", events[0].Result)
			}
		})
	}
}

func TestSinglePageEmptyCollectionIsJSONArrayAndCursorIsOpaque(t *testing.T) {
	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		if _, exposed := request.HTTP.Query["page"]; exposed {
			t.Fatal("numeric page leaked into request")
		}
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"pageSize":1,"hasMore":true,"next":"opaque-token","data":[]}}`)}), nil
	})
	request := executeRequest("flows.v2.workflows.list", nil)
	if err := (Plugin{}).Execute(context.Background(), request, host); err != nil {
		t.Fatal(err)
	}
	result := host.Events()[0].Result
	if result.OperationID != request.CommandID || string(result.Data) != "[]" || result.Page == nil || result.Page.NextCursor != "opaque-token" || !result.Page.HasMore {
		t.Fatalf("result=%#v", result)
	}
}

func TestAllPagesUsesOpaqueCursorAndAggregates(t *testing.T) {
	calls := 0
	host := sdk.NewMemoryHost(func(_ context.Context, r sdk.Request) (sdk.Responses, error) {
		calls++
		if calls == 1 {
			if !reflect.DeepEqual(r.HTTP.Query, map[string][]string{"pageSize": {"7"}}) {
				t.Fatalf("first query=%#v", r.HTTP.Query)
			}
			return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"pageSize":1,"hasMore":true,"next":"opaque==","data":[{"id":"1"}]}}`)}), nil
		}
		if !reflect.DeepEqual(r.HTTP.Query, map[string][]string{"cursor": {"opaque=="}}) {
			t.Fatalf("cursor=%#v", r.HTTP.Query)
		}
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"pageSize":1,"hasMore":false,"data":[{"id":"2"}]}}`)}), nil
	})
	request := executeRequest("flows.v2.workflows.list", nil, sdk.FlagOccurrence{Name: "page-size", Value: "7"})
	request.Continuation = sdk.AllPagesContinuationControl()
	if err := (Plugin{}).Execute(context.Background(), request, host); err != nil {
		t.Fatal(err)
	}
	var values []map[string]any
	if json.Unmarshal(host.Events()[0].Result.Data, &values) != nil || len(values) != 2 {
		t.Fatalf("result=%s", host.Events()[0].Result.Data)
	}
}

func TestPaginationCursorStateFailsClosed(t *testing.T) {
	for _, continuation := range []sdk.ContinuationControl{sdk.SinglePageContinuationControl(), sdk.AllPagesContinuationControl()} {
		for _, body := range []string{
			`{"cursor":{"pageSize":1,"hasMore":true,"data":[]}}`,
			`{"cursor":{"pageSize":1,"hasMore":false,"next":"unexpected","data":[]}}`,
		} {
			host := sdk.NewMemoryHost(func(context.Context, sdk.Request) (sdk.Responses, error) {
				return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(body)}), nil
			})
			request := executeRequest("flows.v2.workflows.list", nil)
			request.Continuation = continuation
			if err := (Plugin{}).Execute(context.Background(), request, host); err == nil {
				t.Errorf("mode=%v accepted incoherent cursor %s", continuation.Mode, body)
			}
			if len(host.Events()) != 0 {
				t.Fatalf("mode=%v emitted result for incoherent cursor: %#v", continuation.Mode, host.Events())
			}
		}
	}
}

func TestAllPagesSecondPageCursorFailureEmitsNoPartialResult(t *testing.T) {
	calls := 0
	host := sdk.NewMemoryHost(func(context.Context, sdk.Request) (sdk.Responses, error) {
		calls++
		if calls == 1 {
			return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"pageSize":1,"hasMore":true,"next":"page-two","data":[{"id":"first"}]}}`)}), nil
		}
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"pageSize":1,"hasMore":false,"next":"impossible","data":[{"id":"second"}]}}`)}), nil
	})
	request := executeRequest("flows.v2.workflows.list", nil)
	request.Continuation = sdk.AllPagesContinuationControl()
	if err := (Plugin{}).Execute(context.Background(), request, host); err == nil {
		t.Fatal("accepted incoherent cursor metadata on page two")
	}
	if calls != 2 {
		t.Fatalf("host calls=%d, want 2", calls)
	}
	if events := host.Events(); len(events) != 0 {
		t.Fatalf("emitted partial result after page-two failure: %#v", events)
	}
}

func TestArbitraryJSONNumbersRemainLosslessAcrossGeneratedAdapter(t *testing.T) {
	const large = "9007199254740993"
	tests := []struct {
		id       string
		args     []string
		flags    []sdk.FlagOccurrence
		status   int32
		response string
	}{
		{
			id:       "flows.v2.triggers.create",
			args:     []string{"payments", "wf1"},
			flags:    []sdk.FlagOccurrence{{Name: "vars", Value: `{"large":` + large + `}`}},
			status:   201,
			response: `{"data":{"id":"trigger"}}`,
		},
		{
			id:       "flows.v2.triggers.test",
			args:     []string{"trigger", `{"nested":{"large":` + large + `}}`},
			status:   200,
			response: `{"data":{}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
				if !strings.Contains(string(request.HTTP.Body), large) {
					t.Fatalf("large integer lost in generated request: %s", request.HTTP.Body)
				}
				return sdk.NewResponseStream(sdk.Response{Status: tt.status, ContentType: "application/json", Body: []byte(tt.response)}), nil
			})
			if err := (Plugin{}).Execute(context.Background(), executeRequest(tt.id, tt.args, tt.flags...), host); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestExecuteRejectsInvalidInputsAndBudgets(t *testing.T) {
	host := sdk.NewMemoryHost(nil)
	cases := []sdk.ExecuteRequest{{CommandID: "missing"}, executeRequest("flows.v2.triggers.test", []string{"t", "no-json"}), executeRequest("flows.v2.triggers.create", []string{"e", "w"}, sdk.FlagOccurrence{Name: "vars", Value: "[]"}), executeRequest("flows.v2.workflows.create", []string{`{"name":"missing-stages"}`}), executeRequest("flows.v2.instances.list", nil, sdk.FlagOccurrence{Name: "running", Value: "sometimes"}), executeRequest("flows.v2.workflows.list", nil, sdk.FlagOccurrence{Name: "page-size", Value: "0"})}
	for _, r := range cases {
		if err := (Plugin{}).Execute(context.Background(), r, host); err == nil {
			t.Errorf("accepted %#v", r)
		}
	}
	cycle := sdk.NewMemoryHost(func(context.Context, sdk.Request) (sdk.Responses, error) {
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"pageSize":1,"hasMore":true,"next":"same","data":[]}}`)}), nil
	})
	r := executeRequest("flows.v2.workflows.list", nil)
	r.Continuation = sdk.ContinuationControl{Mode: sdk.ContinuationAllPages, MaxPages: 3, MaxItems: 10, MaxBytes: 100}
	if err := (Plugin{}).Execute(context.Background(), r, cycle); err == nil {
		t.Fatal("accepted cursor cycle")
	}
}

func TestWorkflowCreateRequiresOpenAPIStagesField(t *testing.T) {
	calls := 0
	host := sdk.NewMemoryHost(func(context.Context, sdk.Request) (sdk.Responses, error) {
		calls++
		return sdk.NewResponseStream(sdk.Response{Status: 201, ContentType: "application/json", Body: []byte(`{"data":{"id":"wf"}}`)}), nil
	})
	err := (Plugin{}).Execute(context.Background(), executeRequest("flows.v2.workflows.create", []string{`{"name":"missing-stages"}`}), host)
	if err == nil {
		t.Fatal("accepted workflow definition without required stages")
	}
	if calls != 0 {
		t.Fatalf("invalid workflow reached host: calls=%d", calls)
	}
}

func TestHostErrorsRemainErrors(t *testing.T) {
	sentinel := errors.New("boom")
	calls := 0
	host := sdk.NewMemoryHost(func(context.Context, sdk.Request) (sdk.Responses, error) {
		calls++
		return nil, sentinel
	})
	err := (Plugin{}).Execute(context.Background(), executeRequest("flows.v2.workflows.show", []string{"wf"}), host)
	if err == nil {
		t.Fatal("missing error")
	}
	if calls != 1 {
		t.Fatalf("generated client retried a host-owned request: calls=%d", calls)
	}
}

func TestMalformedProductResultsFailClosed(t *testing.T) {
	for _, test := range []struct {
		id   string
		args []string
		body string
	}{
		{"flows.v2.workflows.show", []string{"wf"}, `[]`},
		{"flows.v2.workflows.list", nil, `{}`},
	} {
		host := sdk.NewMemoryHost(func(context.Context, sdk.Request) (sdk.Responses, error) {
			return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(test.body)}), nil
		})
		if err := (Plugin{}).Execute(context.Background(), executeRequest(test.id, test.args), host); err == nil {
			t.Errorf("%s accepted %s", test.id, test.body)
		}
	}
}

// TestProductHTTPErrorsSurfaceTypedFromTheAdapter pins the failure value the
// adapter returns for a non-2xx product response. The shared bridge classifies
// it, so the adapter must forward the typed code, the HTTP status and the
// retryability verdict unchanged instead of flattening them into its own
// diagnostic.
//
// Scope: this is the adapter's in-process return value, not what a host sees.
// Flows' only entrypoint is the portable component, whose terminal frame
// carries a failure code alone, so the status and retryability asserted here do
// not cross that boundary. The host-visible contract is pinned separately by
// component.TestOnlyTheTypedFailureCodeCrossesThePortableHostBoundary.
func TestProductHTTPErrorsSurfaceTypedFromTheAdapter(t *testing.T) {
	for _, test := range []struct {
		status        int32
		wantRetryable bool
	}{
		{404, false},
		{409, false},
		{503, true},
	} {
		host := sdk.NewMemoryHost(func(context.Context, sdk.Request) (sdk.Responses, error) {
			return sdk.NewResponseStream(sdk.Response{Status: test.status, ContentType: "application/json", Body: []byte(`{"errorCode":"NOT_FOUND"}`)}), nil
		})

		err := (Plugin{}).Execute(context.Background(), executeRequest("flows.v2.workflows.show", []string{"wf"}), host)
		var failure sdk.Failure
		if !errors.As(err, &failure) {
			t.Errorf("HTTP %d produced %v, want an sdk.Failure", test.status, err)
			continue
		}
		if failure.Code != string(sdk.FailureProductHTTPError) {
			t.Errorf("HTTP %d failed with code %q, want %q", test.status, failure.Code, sdk.FailureProductHTTPError)
		}
		if failure.Retryable != test.wantRetryable {
			t.Errorf("HTTP %d retryable=%t, want %t", test.status, failure.Retryable, test.wantRetryable)
		}

		var details struct {
			HTTPStatus int32 `json:"httpStatus"`
		}
		if err := json.Unmarshal(failure.Details, &details); err != nil {
			t.Errorf("HTTP %d details are not JSON: %v", test.status, err)
			continue
		}
		if details.HTTPStatus != test.status {
			t.Errorf("failure details report status %d, want %d", details.HTTPStatus, test.status)
		}
	}
}
