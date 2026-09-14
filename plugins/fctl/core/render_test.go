package core

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

// column is one expected table column, in its declared order.
type column struct{ header, field string }

// wantTableColumns is the exact, ordered table each command declares. A command
// absent from this map declares no table hint at all; wantNoTable records why.
// Both maps are exhaustive over the catalogue and are proved against real
// emitted results by TestTableRenderHintsDescribeActualResultFields.
var wantTableColumns = map[string][]column{
	"flows.v2.triggers.list": {
		{"ID", "id"}, {"Name", "name"}, {"Event", "event"},
		{"Workflow ID", "workflowID"}, {"Created At", "createdAt"},
	},
	"flows.v2.triggers.show": {
		{"ID", "id"}, {"Name", "name"}, {"Event", "event"},
		{"Workflow ID", "workflowID"}, {"Created At", "createdAt"},
	},
	"flows.v2.triggers.create": {
		{"ID", "id"}, {"Name", "name"}, {"Event", "event"},
		{"Workflow ID", "workflowID"}, {"Created At", "createdAt"},
	},
	"flows.v2.triggers.occurrences.list": {
		{"Date", "date"}, {"Trigger ID", "triggerID"}, {"Instance ID", "workflowInstanceID"},
	},
	"flows.v2.workflows.list": {
		{"ID", "id"}, {"Created At", "createdAt"}, {"Updated At", "updatedAt"},
	},
	"flows.v2.workflows.show": {
		{"ID", "id"}, {"Created At", "createdAt"}, {"Updated At", "updatedAt"},
	},
	"flows.v2.workflows.create": {
		{"ID", "id"}, {"Created At", "createdAt"}, {"Updated At", "updatedAt"},
	},
	"flows.v2.instances.list": {
		{"ID", "id"}, {"Workflow ID", "workflowID"}, {"Created At", "createdAt"},
		{"Updated At", "updatedAt"}, {"Terminated", "terminated"},
	},
	"flows.v2.triggers.test": {
		{"Match", "filter.match"},
	},
	"flows.v2.workflows.run": {
		{"ID", "instance.id"}, {"Workflow ID", "instance.workflowID"},
		{"Workflow Name", "workflow.config.name"}, {"Terminated", "instance.terminated"},
	},
	"flows.v2.instances.show": {
		{"ID", "instance.id"}, {"Workflow ID", "instance.workflowID"},
		{"Workflow Name", "workflow.config.name"}, {"Terminated", "instance.terminated"},
	},
}

// wantNoTable is every command that deliberately declares no table hint, with
// the exact reason. Nothing is invented for these: their results carry no flat
// field a stable column could name.
var wantNoTable = map[string]string{
	"flows.v2.triggers.delete":      "no-content mutation: the canonical empty result has no field",
	"flows.v2.workflows.delete":     "no-content mutation: the canonical empty result has no field",
	"flows.v2.instances.send-event": "no-content mutation: the canonical empty result has no field",
	"flows.v2.instances.stop":       "no-content mutation: the canonical empty result has no field",
	"flows.v2.instances.describe":   "composite presentation read: history and stages are both nested arrays",
}

func TestTableRenderHintsAreTheExactOrderedColumns(t *testing.T) {
	declared := map[string][]column{}
	for _, command := range Catalogue() {
		reason, silent := wantNoTable[command.ID]
		if silent && command.Render.Table != nil {
			t.Errorf("%s declares a table hint, but it has none to declare: %s", command.ID, reason)
			continue
		}
		if command.Render.Table == nil {
			if !silent {
				t.Errorf("%s declares no table hint and no recorded reason", command.ID)
			}
			continue
		}
		got := make([]column, 0, len(command.Render.Table.Columns))
		for _, c := range command.Render.Table.Columns {
			got = append(got, column{c.Header, c.Field})
		}
		declared[command.ID] = got
	}
	if !reflect.DeepEqual(declared, wantTableColumns) {
		t.Errorf("declared table columns =\n%#v\nwant\n%#v", declared, wantTableColumns)
	}
	if got, want := len(wantTableColumns)+len(wantNoTable), len(Catalogue()); got != want {
		t.Errorf("render decisions cover %d commands, catalogue has %d", got, want)
	}
}

func TestTableRenderHintColumnsAreCompactAndWellFormed(t *testing.T) {
	for _, command := range Catalogue() {
		if command.Render.Table == nil {
			continue
		}
		columns := command.Render.Table.Columns
		// Ceilings accepted by the pinned SDK for the portable payload.
		if len(columns) == 0 || len(columns) > 64 {
			t.Errorf("%s declares %d columns", command.ID, len(columns))
		}
		if len(columns) > 5 {
			t.Errorf("%s declares %d columns, want a compact table of at most 5", command.ID, len(columns))
		}
		headers, fields := map[string]struct{}{}, map[string]struct{}{}
		for _, c := range columns {
			if c.Header == "" || len(c.Header) > 2048 || c.Field == "" || len(c.Field) > 256 {
				t.Errorf("%s column %#v is out of bounds", command.ID, c)
			}
			for _, segment := range strings.Split(c.Field, ".") {
				if segment == "" || strings.ContainsAny(segment, "/[]") {
					t.Errorf("%s column field %q is not a dotted object path", command.ID, c.Field)
				}
			}
			if _, repeated := headers[c.Header]; repeated {
				t.Errorf("%s repeats header %q", command.ID, c.Header)
			}
			if _, repeated := fields[c.Field]; repeated {
				t.Errorf("%s repeats field %q", command.ID, c.Field)
			}
			headers[c.Header], fields[c.Field] = struct{}{}, struct{}{}
		}
	}
	if err := sdk.ValidateCatalogue(Catalogue(), nil); err != nil {
		t.Fatalf("catalogue with render hints is invalid: %v", err)
	}
}

// TestTableRenderHintsDescribeActualResultFields catches a renamed, missing or
// container-valued column path by resolving it through the real result emitted
// by the adapter. The host accepts dotted object paths, so nested scalar leaves
// are valid columns.
func TestTableRenderHintsDescribeActualResultFields(t *testing.T) {
	for _, test := range renderEvidenceCases() {
		t.Run(test.id, func(t *testing.T) {
			data := executeForRender(t, test)
			result := decodeResultRow(t, data)
			columns, hinted := wantTableColumns[test.id]
			if !hinted {
				if _, recorded := wantNoTable[test.id]; !recorded {
					t.Fatalf("%s has no table and no recorded reason", test.id)
				}
				return
			}
			for _, c := range columns {
				value, present := valueAtObjectPath(result, c.field)
				if !present {
					t.Errorf("%s column %q names %q, which the emitted result does not carry", test.id, c.header, c.field)
					continue
				}
				switch value.(type) {
				case map[string]any, []any:
					t.Errorf("%s column %q resolves to a container at %q", test.id, c.header, c.field)
				}
			}
		})
	}
}

// TestRenderPathsExistInThePublicSchema catches schemas that are too broad to
// describe a column or that drift from the adapter result. Known properties are
// declared while additional product fields remain allowed for compatibility.
func TestRenderPathsExistInThePublicSchema(t *testing.T) {
	for _, command := range Catalogue() {
		if !reflect.DeepEqual(command.RawOutputSchema, command.PublicOutputSchema) {
			t.Errorf("%s raw and public output schemas differ", command.ID)
		}
		var schema map[string]any
		if err := json.Unmarshal(command.PublicOutputSchema, &schema); err != nil {
			t.Fatalf("%s public output schema is not JSON: %v", command.ID, err)
		}
		wantRoot := "object"
		if command.Pagination.Supported {
			wantRoot = "array"
		}
		if schema["type"] != wantRoot {
			t.Errorf("%s public output schema root=%v, want %q", command.ID, schema["type"], wantRoot)
		}
		if command.Render.Table == nil {
			continue
		}
		for _, column := range command.Render.Table.Columns {
			node, ok := schemaAtObjectPath(schema, column.Field)
			if !ok {
				t.Errorf("%s public schema does not declare column path %q", command.ID, column.Field)
				continue
			}
			if node["type"] == "object" || node["type"] == "array" {
				t.Errorf("%s public schema declares column path %q as %v", command.ID, column.Field, node["type"])
			}
		}
		if command.OutputMediaType != "application/json" {
			t.Errorf("%s declares table columns for media type %q", command.ID, command.OutputMediaType)
		}
	}
}

type renderCase struct {
	id     string
	args   []string
	flags  []sdk.FlagOccurrence
	status int32
}

const (
	triggerFixture = `{"id":"trg-1","name":"payments-in","event":"payments.saved","workflowID":"wf-1",` +
		`"version":"v1","filter":"event.type == \"PAYMENT\"","vars":{"account":"event.account"},` +
		`"createdAt":"2026-01-01T00:00:00Z"}`
	workflowFixture = `{"id":"wf-1","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z",` +
		`"config":{"name":"nightly","stages":[{"send":{}}]}}`
	instanceFixture = `{"id":"inst-1","workflowID":"wf-1","createdAt":"2026-01-01T00:00:00Z",` +
		`"updatedAt":"2026-01-02T00:00:00Z","terminated":true,"terminatedAt":"2026-01-02T00:00:00Z",` +
		`"error":"stage failed","status":[{"stage":0,"instanceID":"inst-1","startedAt":"2026-01-01T00:00:00Z"}],` +
		`"workflow":` + workflowFixture + `}`
	occurrenceFixture = `{"date":"2026-01-01T00:00:00Z","triggerID":"trg-1","workflowInstanceID":"inst-1",` +
		`"workflowInstance":` + instanceFixture + `,"error":"filter did not match","event":{"type":"payments.saved"}}`
	triggerTestFixture = `{"filter":{"match":true},"variables":{"account":{"value":"acc-1"}}}`
	historyFixture     = `{"name":"stage-one","input":{"duration":"1s"},"terminated":true,` +
		`"startedAt":"2026-01-01T00:00:00Z","terminatedAt":"2026-01-01T00:01:00Z"}`
)

func renderEvidenceCases() []renderCase {
	return []renderCase{
		{id: "flows.v2.triggers.list", status: 200},
		{id: "flows.v2.triggers.show", args: []string{"trg-1"}, status: 200},
		{id: "flows.v2.triggers.create", args: []string{"payments.saved", "wf-1"}, status: 201},
		{id: "flows.v2.triggers.delete", args: []string{"trg-1"}, status: 204},
		{id: "flows.v2.triggers.test", args: []string{"trg-1", `{"type":"payments.saved"}`}, status: 200},
		{id: "flows.v2.triggers.occurrences.list", args: []string{"trg-1"}, status: 200},
		{id: "flows.v2.workflows.list", status: 200},
		{id: "flows.v2.workflows.show", args: []string{"wf-1"}, status: 200},
		{id: "flows.v2.workflows.create", args: []string{`{"name":"wf","stages":[]}`}, status: 201},
		{id: "flows.v2.workflows.delete", args: []string{"wf-1"}, status: 204},
		{id: "flows.v2.workflows.run", args: []string{"wf-1"}, status: 201},
		{id: "flows.v2.instances.list", status: 200},
		{id: "flows.v2.instances.show", args: []string{"inst-1"}, status: 200},
		{id: "flows.v2.instances.describe", args: []string{"inst-1"}, status: 200},
		{id: "flows.v2.instances.send-event", args: []string{"inst-1", "approved"}, status: 204},
		{id: "flows.v2.instances.stop", args: []string{"inst-1"}, status: 204},
	}
}

func renderFixture(operation string) (int32, string) {
	page := func(item string) string {
		return `{"cursor":{"pageSize":1,"hasMore":false,"data":[` + item + `]}}`
	}
	switch operation {
	case "v2ListTriggers":
		return 200, page(triggerFixture)
	case "v2ListTriggersOccurrences":
		return 200, page(occurrenceFixture)
	case "v2ListWorkflows":
		return 200, page(workflowFixture)
	case "v2ListInstances":
		return 200, page(instanceFixture)
	case "v2ReadTrigger":
		return 200, `{"data":` + triggerFixture + `}`
	case "v2CreateTrigger":
		return 201, `{"data":` + triggerFixture + `}`
	case "testTrigger":
		return 200, `{"data":` + triggerTestFixture + `}`
	case "v2GetWorkflow":
		return 200, `{"data":` + workflowFixture + `}`
	case "v2CreateWorkflow":
		return 201, `{"data":` + workflowFixture + `}`
	case "v2GetInstance":
		return 200, `{"data":` + instanceFixture + `}`
	case "v2RunWorkflow":
		return 201, `{"data":` + instanceFixture + `}`
	case "v2GetInstanceHistory":
		return 200, `{"data":[` + historyFixture + `]}`
	case "v2GetInstanceStageHistory":
		return 200, `{"data":[]}`
	default:
		return 204, ""
	}
}

func executeForRender(t *testing.T, test renderCase) []byte {
	t.Helper()
	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		status, body := renderFixture(request.Operation)
		if status == 204 {
			return sdk.NewResponseStream(sdk.Response{Status: 204}), nil
		}
		return sdk.NewResponseStream(sdk.Response{Status: status, ContentType: "application/json", Body: []byte(body)}), nil
	})
	if err := (Plugin{}).Execute(context.Background(), executeRequest(test.id, test.args, test.flags...), host); err != nil {
		t.Fatalf("execute %s: %v", test.id, err)
	}
	events := host.Events()
	if len(events) != 1 || events[0].Result == nil {
		t.Fatalf("%s emitted %#v", test.id, events)
	}
	return events[0].Result.Data
}

func decodeResultRow(t *testing.T, data []byte) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if items, isArray := value.([]any); isArray {
		if len(items) == 0 {
			t.Fatalf("collection result carries no row to describe")
		}
		value = items[0]
	}
	row, isObject := value.(map[string]any)
	if !isObject {
		t.Fatalf("result row is %T, want an object", value)
	}
	return row
}

func valueAtObjectPath(value map[string]any, path string) (any, bool) {
	var current any = value
	for _, segment := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[segment]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func schemaAtObjectPath(schema map[string]any, path string) (map[string]any, bool) {
	current := schema
	if current["type"] == "array" {
		var ok bool
		current, ok = current["items"].(map[string]any)
		if !ok {
			return nil, false
		}
	}
	for _, segment := range strings.Split(path, ".") {
		properties, ok := current["properties"].(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := properties[segment].(map[string]any)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}
