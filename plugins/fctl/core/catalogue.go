package core

import (
	"strings"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

const (
	productMajor      uint32 = 2
	readRequestBytes  int64  = 64 << 10
	writeRequestBytes int64  = 256 << 10
	responseBytes     int64  = 512 << 10
)

type spec struct {
	path                              []string
	summary, operation, method, route string
	args                              []sdk.Argument
	flags                             []sdk.Flag
	paginated, mutation, body         bool
	extra                             []operation
}
type operation struct {
	id, method, route string
	body              bool
}

func arg(name, usage string) sdk.Argument {
	return sdk.Argument{Name: name, Usage: usage, Type: sdk.ArgumentString, Required: true, Completion: sdk.CompletionSpec{Kind: sdk.CompletionNone}}
}
func str(name, usage string) sdk.Flag {
	return sdk.Flag{Name: name, Usage: usage, Type: sdk.FlagString, Completion: sdk.CompletionSpec{Kind: sdk.CompletionNone}}
}
func boolean(name, usage string) sdk.Flag {
	return sdk.Flag{Name: name, Usage: usage, Type: sdk.FlagBool, Completion: sdk.CompletionSpec{Kind: sdk.CompletionNone}}
}
func integer(name, usage string) sdk.Flag {
	return sdk.Flag{Name: name, Usage: usage, Type: sdk.FlagInt32, Completion: sdk.CompletionSpec{Kind: sdk.CompletionNone}}
}

func catalogueSpecs() []spec {
	return []spec{
		{[]string{"triggers", "list"}, "List triggers", "v2ListTriggers", "GET", "/v2/triggers", nil, []sdk.Flag{str("name", "Filter by trigger name"), integer("page-size", "Items per page")}, true, false, false, nil},
		{[]string{"triggers", "show"}, "Show a trigger", "v2ReadTrigger", "GET", "/v2/triggers/{triggerID}", []sdk.Argument{arg("trigger-id", "Trigger id")}, nil, false, false, false, nil},
		{[]string{"triggers", "create"}, "Create a trigger", "v2CreateTrigger", "POST", "/v2/triggers", []sdk.Argument{arg("event", "Event name"), arg("workflow-id", "Workflow id")}, []sdk.Flag{str("name", "Trigger name"), str("version", "Trigger version"), str("filter", "Filter expression"), str("vars", "Variables as a JSON object")}, false, true, true, nil},
		{[]string{"triggers", "delete"}, "Delete a trigger", "v2DeleteTrigger", "DELETE", "/v2/triggers/{triggerID}", []sdk.Argument{arg("trigger-id", "Trigger id")}, nil, false, true, false, nil},
		{[]string{"triggers", "test"}, "Test a trigger against an event", "testTrigger", "POST", "/v2/triggers/{triggerID}/test", []sdk.Argument{arg("trigger-id", "Trigger id"), arg("event", "Event as a JSON object")}, nil, false, false, true, nil},
		{[]string{"triggers", "occurrences", "list"}, "List trigger occurrences", "v2ListTriggersOccurrences", "GET", "/v2/triggers/{triggerID}/occurrences", []sdk.Argument{arg("trigger-id", "Trigger id")}, []sdk.Flag{integer("page-size", "Items per page")}, true, false, false, nil},
		{[]string{"workflows", "list"}, "List workflows", "v2ListWorkflows", "GET", "/v2/workflows", nil, []sdk.Flag{integer("page-size", "Items per page")}, true, false, false, nil},
		{[]string{"workflows", "show"}, "Show a workflow", "v2GetWorkflow", "GET", "/v2/workflows/{flowId}", []sdk.Argument{arg("workflow-id", "Workflow id")}, nil, false, false, false, nil},
		{[]string{"workflows", "create"}, "Create a workflow", "v2CreateWorkflow", "POST", "/v2/workflows", []sdk.Argument{arg("definition", "Workflow definition as JSON")}, nil, false, true, true, nil},
		{[]string{"workflows", "delete"}, "Soft-delete a workflow", "v2DeleteWorkflow", "DELETE", "/v2/workflows/{flowId}", []sdk.Argument{arg("workflow-id", "Workflow id")}, nil, false, true, false, nil},
		{[]string{"workflows", "run"}, "Run a workflow", "v2RunWorkflow", "POST", "/v2/workflows/{workflowID}/instances", []sdk.Argument{arg("workflow-id", "Workflow id")}, []sdk.Flag{str("variables", "Variables as a JSON object"), boolean("wait", "Wait for workflow termination")}, false, true, true, []operation{{"v2GetWorkflow", "GET", "/v2/workflows/{flowId}", false}}},
		{[]string{"instances", "list"}, "List workflow instances", "v2ListInstances", "GET", "/v2/instances", nil, []sdk.Flag{str("workflow-id", "Filter by workflow id"), boolean("running", "Filter running instances"), integer("page-size", "Items per page")}, true, false, false, nil},
		{[]string{"instances", "show"}, "Show a workflow instance", "v2GetInstance", "GET", "/v2/instances/{instanceID}", []sdk.Argument{arg("instance-id", "Instance id")}, nil, false, false, false, []operation{{"v2GetWorkflow", "GET", "/v2/workflows/{flowId}", false}}},
		{[]string{"instances", "describe"}, "Describe workflow instance history", "v2GetInstanceHistory", "GET", "/v2/instances/{instanceID}/history", []sdk.Argument{arg("instance-id", "Instance id")}, nil, false, false, false, []operation{{"v2GetInstanceStageHistory", "GET", "/v2/instances/{instanceID}/stages/{number}/history", false}}},
		{[]string{"instances", "send-event"}, "Send an event to a workflow instance", "v2SendEvent", "POST", "/v2/instances/{instanceID}/events", []sdk.Argument{arg("instance-id", "Instance id"), arg("event", "Event name")}, nil, false, true, true, nil},
		{[]string{"instances", "stop"}, "Stop a workflow instance", "v2CancelEvent", "PUT", "/v2/instances/{instanceID}/abort", []sdk.Argument{arg("instance-id", "Instance id")}, nil, false, true, false, nil},
	}
}

// tableColumns are the compact display columns the host may render for a
// command. A column names one scalar field the adapter actually emits, using a
// dotted object path where the useful value is nested. Unbounded and low-signal
// fields are deliberately left out, and a command whose result carries no
// stable scalar leaf declares no table at all. The exhaustive
// result contract stays in PublicOutputSchema, which these hints do not narrow.
var tableColumns = map[string][]sdk.TableColumn{
	"triggers.list":             triggerColumns,
	"triggers.show":             triggerColumns,
	"triggers.create":           triggerColumns,
	"triggers.test":             {{Header: "Match", Field: "filter.match"}},
	"triggers.occurrences.list": {{Header: "Date", Field: "date"}, {Header: "Trigger ID", Field: "triggerID"}, {Header: "Instance ID", Field: "workflowInstanceID"}},
	"workflows.list":            workflowColumns,
	"workflows.show":            workflowColumns,
	"workflows.create":          workflowColumns,
	"workflows.run":             compositeInstanceColumns,
	"instances.list":            {{Header: "ID", Field: "id"}, {Header: "Workflow ID", Field: "workflowID"}, {Header: "Created At", Field: "createdAt"}, {Header: "Updated At", Field: "updatedAt"}, {Header: "Terminated", Field: "terminated"}},
	"instances.show":            compositeInstanceColumns,
}

var triggerColumns = []sdk.TableColumn{{Header: "ID", Field: "id"}, {Header: "Name", Field: "name"}, {Header: "Event", Field: "event"}, {Header: "Workflow ID", Field: "workflowID"}, {Header: "Created At", Field: "createdAt"}}

var workflowColumns = []sdk.TableColumn{{Header: "ID", Field: "id"}, {Header: "Created At", Field: "createdAt"}, {Header: "Updated At", Field: "updatedAt"}}

var compositeInstanceColumns = []sdk.TableColumn{{Header: "ID", Field: "instance.id"}, {Header: "Workflow ID", Field: "instance.workflowID"}, {Header: "Workflow Name", Field: "workflow.config.name"}, {Header: "Terminated", Field: "instance.terminated"}}

func renderHints(path string) sdk.RenderHints {
	columns, ok := tableColumns[path]
	if !ok {
		return sdk.RenderHints{}
	}
	// Copied so a descriptor consumer cannot mutate the shared declaration.
	return sdk.RenderHints{Table: &sdk.TableRenderHint{Columns: append([]sdk.TableColumn(nil), columns...)}}
}

func Catalogue() []sdk.Command {
	specs := catalogueSpecs()
	out := make([]sdk.Command, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.command())
	}
	return out
}
func (s spec) command() sdk.Command {
	ops := []operation{{s.operation, s.method, s.route, s.body}}
	ops = append(ops, s.extra...)
	policies := make([]sdk.OperationPolicy, 0, len(ops))
	for _, op := range ops {
		policies = append(policies, op.policy(s.mutation))
	}
	path := strings.Join(s.path, ".")
	shape := outputSchema(path)
	max := uint32(len(ops))
	if s.paginated {
		max = sdk.DefaultAllPagesMaxPages
	}
	if path == "instances.describe" {
		max = sdk.PortableMaxHostRequests
	}
	risk := sdk.RiskRead
	if s.mutation {
		risk = sdk.RiskMutation
	}
	return sdk.Command{ID: "flows.v2." + strings.Join(s.path, "."), ExecutionKind: sdk.ExecutionKindService, AuthMode: sdk.AuthModeCapability, Path: s.path, Target: sdk.TargetRequirement{Kind: sdk.TargetStack}, Summary: s.summary, Long: s.summary + ". Endpoint, authentication and transport are host-owned.", Example: strings.Join(s.path, " ") + " --help", Arguments: s.args, Flags: s.flags, Auth: []sdk.AuthRequirement{{Capability: "auth.stack"}}, Operations: policies, Compatibility: []sdk.ServiceCompatibility{{Service: sdk.ServiceFlows, Majors: []uint32{productMajor}}}, Risk: risk, InputSchema: inputSchema(s.args, s.flags), RawOutputSchema: shape, PublicOutputSchema: shape, Pagination: sdk.PaginationSpec{Supported: s.paginated}, OutputMediaType: "application/json", Render: renderHints(path), ExecutionPolicy: &sdk.CommandExecutionPolicy{MaxHostRequests: max}}
}
func (op operation) policy(_ bool) sdk.OperationPolicy {
	scope := "orchestration:read"
	maxRequestBytes := readRequestBytes
	if op.method != "GET" {
		scope = "orchestration:write"
		maxRequestBytes = writeRequestBytes
	}
	contents := []string(nil)
	if op.body {
		contents = []string{"application/json"}
	}
	return sdk.OperationPolicy{ID: op.id, Service: sdk.ServiceFlows, Scopes: []string{scope}, HTTP: &sdk.HTTPOperationPolicy{Method: op.method, GeneratedClient: &sdk.HTTPGeneratedClientPolicy{PathTemplate: op.route, RequestContentTypes: contents, RequestHeaders: []string{"Accept"}, MaxRequestBytes: maxRequestBytes, ResponseLimits: sdk.ResponseLimits{MaxMessageBytes: responseBytes, MaxMessages: 1, MaxAggregateBytes: responseBytes}}}}
}
