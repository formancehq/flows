package core

import (
	"reflect"
	"strings"
	"testing"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
	"github.com/formancehq/orchestration/plugins/fctl/audit"
)

func TestCatalogueIsTheSixteenCommandV2Surface(t *testing.T) {
	want := []string{"triggers list", "triggers show", "triggers create", "triggers delete", "triggers test", "triggers occurrences list", "workflows list", "workflows show", "workflows create", "workflows delete", "workflows run", "instances list", "instances show", "instances describe", "instances send-event", "instances stop"}
	commands := Catalogue()
	if len(commands) != len(want) {
		t.Fatalf("commands=%d want 16", len(commands))
	}
	if err := sdk.ValidateCatalogue(commands, nil); err != nil {
		t.Fatalf("invalid catalogue: %v", err)
	}
	for i, c := range commands {
		if strings.Join(c.Path, " ") != want[i] {
			t.Errorf("command %d path=%q", i, c.Path)
		}
		if c.ID != "flows.v2."+strings.ReplaceAll(want[i], " ", ".") {
			t.Errorf("id=%q", c.ID)
		}
		if c.Target.Kind != sdk.TargetStack || c.AuthMode != sdk.AuthModeCapability || c.ExecutionKind != sdk.ExecutionKindService {
			t.Errorf("boundary for %s", c.ID)
		}
		if !reflect.DeepEqual(c.Auth, []sdk.AuthRequirement{{Capability: "auth.stack"}}) {
			t.Errorf("auth for %s", c.ID)
		}
		if !reflect.DeepEqual(c.Compatibility, []sdk.ServiceCompatibility{{Service: sdk.ServiceFlows, Majors: []uint32{2}}}) {
			t.Errorf("compatibility for %s", c.ID)
		}
		for _, op := range c.Operations {
			if op.Service != sdk.ServiceFlows || op.HTTP == nil || op.HTTP.GeneratedClient == nil || len(op.Scopes) != 1 {
				t.Errorf("operation for %s: %#v", c.ID, op)
			}
		}
		for _, flag := range c.Flags {
			if flag.Name == "cursor" || flag.Name == "page" {
				t.Errorf("%s exposes host-owned pagination flag %q", c.ID, flag.Name)
			}
		}
	}
}

func TestCataloguePinsSpecialOperationSemantics(t *testing.T) {
	test, _ := commandByID("flows.v2.triggers.test")
	if test.Risk != sdk.RiskRead || !reflect.DeepEqual(test.Operations[0].Scopes, []string{"orchestration:write"}) {
		t.Fatalf("test trigger semantics=%#v", test)
	}
	run, _ := commandByID("flows.v2.workflows.run")
	if len(run.Operations) != 2 || run.ExecutionPolicy.MaxHostRequests != 2 {
		t.Fatalf("run operations=%#v", run.Operations)
	}
	describe, _ := commandByID("flows.v2.instances.describe")
	if len(describe.Operations) != 2 {
		t.Fatalf("describe operations=%#v", describe.Operations)
	}
	for _, id := range []string{"flows.v2.triggers.list", "flows.v2.triggers.occurrences.list", "flows.v2.workflows.list", "flows.v2.instances.list"} {
		c, _ := commandByID(id)
		if !c.Pagination.Supported || c.ExecutionPolicy.MaxHostRequests != sdk.DefaultAllPagesMaxPages {
			t.Errorf("pagination for %s", id)
		}
	}
}

func TestCatalogueMatchesEveryAdmittedOpenAPIOperation(t *testing.T) {
	report, err := audit.Build("../../../openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]audit.Record, len(report.Operations))
	for _, record := range report.Operations {
		byID[record.OperationID] = record
	}
	wantOperations := map[string][]string{
		"flows.v2.triggers.list":             {"v2ListTriggers"},
		"flows.v2.triggers.show":             {"v2ReadTrigger"},
		"flows.v2.triggers.create":           {"v2CreateTrigger"},
		"flows.v2.triggers.delete":           {"v2DeleteTrigger"},
		"flows.v2.triggers.test":             {"testTrigger"},
		"flows.v2.triggers.occurrences.list": {"v2ListTriggersOccurrences"},
		"flows.v2.workflows.list":            {"v2ListWorkflows"},
		"flows.v2.workflows.show":            {"v2GetWorkflow"},
		"flows.v2.workflows.create":          {"v2CreateWorkflow"},
		"flows.v2.workflows.delete":          {"v2DeleteWorkflow"},
		"flows.v2.workflows.run":             {"v2RunWorkflow", "v2GetWorkflow"},
		"flows.v2.instances.list":            {"v2ListInstances"},
		"flows.v2.instances.show":            {"v2GetInstance", "v2GetWorkflow"},
		"flows.v2.instances.describe":        {"v2GetInstanceHistory", "v2GetInstanceStageHistory"},
		"flows.v2.instances.send-event":      {"v2SendEvent"},
		"flows.v2.instances.stop":            {"v2CancelEvent"},
	}

	commands := Catalogue()
	if len(commands) != len(wantOperations) {
		t.Fatalf("commands=%d, want %d", len(commands), len(wantOperations))
	}
	for _, command := range commands {
		wantIDs, declared := wantOperations[command.ID]
		if !declared {
			t.Fatalf("unexpected command %s", command.ID)
		}
		gotIDs := make([]string, 0, len(command.Operations))
		for _, policy := range command.Operations {
			gotIDs = append(gotIDs, policy.ID)
		}
		if !reflect.DeepEqual(gotIDs, wantIDs) {
			t.Fatalf("%s operations=%v, want %v", command.ID, gotIDs, wantIDs)
		}
		if len(command.Operations) == 0 {
			t.Fatalf("%s has no admitted operation", command.ID)
		}
		primary, ok := byID[command.Operations[0].ID]
		if !ok {
			t.Fatalf("%s primary operation %q is absent from OpenAPI", command.ID, command.Operations[0].ID)
		}
		if command.Pagination.Supported != primary.Risk.Paginated {
			t.Errorf("%s pagination=%t, OpenAPI=%t", command.ID, command.Pagination.Supported, primary.Risk.Paginated)
		}

		wantRisk := sdk.RiskRead
		// Testing a trigger sends a POST but does not mutate server state. Every
		// other mutating OpenAPI operation is deliberately surfaced as a mutation.
		if primary.Mutating() && primary.OperationID != "testTrigger" {
			wantRisk = sdk.RiskMutation
		}
		if command.Risk != wantRisk {
			t.Errorf("%s risk=%q, want %q from OpenAPI operation %s", command.ID, command.Risk, wantRisk, primary.OperationID)
		}

		for _, policy := range command.Operations {
			record, exists := byID[policy.ID]
			if !exists {
				t.Errorf("%s operation %q is absent from OpenAPI", command.ID, policy.ID)
				continue
			}
			if policy.HTTP == nil || policy.HTTP.GeneratedClient == nil {
				t.Errorf("%s operation %s has no generated HTTP policy", command.ID, policy.ID)
				continue
			}
			generated := policy.HTTP.GeneratedClient
			if policy.HTTP.Method != record.Method {
				t.Errorf("%s operation %s method=%q, OpenAPI=%q", command.ID, policy.ID, policy.HTTP.Method, record.Method)
			}
			if generated.PathTemplate != record.Path {
				t.Errorf("%s operation %s path=%q, OpenAPI=%q", command.ID, policy.ID, generated.PathTemplate, record.Path)
			}
			hasBody := len(generated.RequestContentTypes) > 0
			if hasBody != record.HasRequestBody() {
				t.Errorf("%s operation %s body=%t, OpenAPI=%t", command.ID, policy.ID, hasBody, record.HasRequestBody())
			}
			if !reflect.DeepEqual(policy.Scopes, record.Scopes) {
				t.Errorf("%s operation %s scopes=%v, OpenAPI=%v", command.ID, policy.ID, policy.Scopes, record.Scopes)
			}
			if policy.Service != sdk.ServiceFlows {
				t.Errorf("%s operation %s service=%q", command.ID, policy.ID, policy.Service)
			}
		}
	}
}

func TestInputSchemasAreStrict(t *testing.T) {
	for _, c := range Catalogue() {
		if !strings.Contains(string(c.InputSchema), `"additionalProperties":false`) {
			t.Errorf("schema for %s", c.ID)
		}
	}
}
