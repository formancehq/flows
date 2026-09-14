package audit

import "sort"

// BaselineRevision is the legacy fctl commit this baseline was transcribed
// from. Every SourceFile below is a path in that tree.
const BaselineRevision = "693c58e27865f83332e6c3199d61fed81b742f41"

// BaselineNamespace is the command namespace the legacy tree used. The fctl-v2
// programme exposes the same surface under the public `flows` namespace; the
// baseline is recorded under its own historical name so the mapping stays
// checkable against BaselineRevision.
const BaselineNamespace = "orchestration"

// BaselineCommand is one executable legacy fctl `orchestration` command.
//
// Grouping-only cobra commands (`orchestration`, `orchestration triggers`,
// `orchestration triggers occurrences`, `orchestration workflows`,
// `orchestration instances`) and the shared render helper
// (`cmd/orchestration/internal/print.go`) are not commands and are not listed:
// the baseline counts executable leaves only.
type BaselineCommand struct {
	// Path is the canonical invocation, without the `fctl` prefix.
	Path string
	// SourceFile is the declaring file at BaselineRevision.
	SourceFile string
	// Ops are the operationIds the legacy command reaches, in the order the
	// legacy code issues them. Secondary reads the command performs purely to
	// render its output are listed too, because they are part of the observable
	// request sequence a replacement has to reproduce or consciously drop.
	Ops []string
	// SDKCalls are the generated-client calls the legacy command makes, in the
	// same order as Ops. They pin which API major the legacy command actually
	// used, which is not always the one the command name suggests.
	SDKCalls []string
	// Exclusion is the evidence-backed reason the command is not carried over.
	// Non-empty exactly when Ops is empty.
	Exclusion string
}

// Baseline is the complete set of executable legacy fctl `orchestration`
// commands at BaselineRevision, with each one's mapping onto the current
// operation surface.
//
// Every entry maps: there is no excluded command in this namespace. That is a
// finding, not an omission — it is asserted by TestBaselineHasNoExclusions.
var Baseline = []BaselineCommand{
	// triggers
	{
		Path:       "orchestration triggers list",
		SourceFile: "cmd/orchestration/triggers/list.go",
		Ops:        []string{"listTriggers"},
		SDKCalls:   []string{"Orchestration.V1.ListTriggers"},
	},
	{
		Path:       "orchestration triggers show <trigger-id>",
		SourceFile: "cmd/orchestration/triggers/show.go",
		Ops:        []string{"readTrigger"},
		SDKCalls:   []string{"Orchestration.V1.ReadTrigger"},
	},
	{
		Path:       "orchestration triggers create <event> <workflow-id>",
		SourceFile: "cmd/orchestration/triggers/create.go",
		Ops:        []string{"createTrigger"},
		SDKCalls:   []string{"Orchestration.V1.CreateTrigger"},
	},
	{
		Path:       "orchestration triggers delete <trigger-id>",
		SourceFile: "cmd/orchestration/triggers/delete.go",
		Ops:        []string{"deleteTrigger"},
		SDKCalls:   []string{"Orchestration.V1.DeleteTrigger"},
	},
	{
		// The only legacy command in this namespace that calls /v2: v1 declares
		// and serves no trigger-test operation at all.
		Path:       "orchestration triggers test <trigger-id> <event>",
		SourceFile: "cmd/orchestration/triggers/test.go",
		Ops:        []string{"testTrigger"},
		SDKCalls:   []string{"Orchestration.V2.TestTrigger"},
	},
	{
		Path:       "orchestration triggers occurrences list",
		SourceFile: "cmd/orchestration/triggers/occurrences/list.go",
		Ops:        []string{"listTriggersOccurrences"},
		SDKCalls:   []string{"Orchestration.V1.ListTriggersOccurrences"},
	},

	// workflows
	{
		Path:       "orchestration workflows list",
		SourceFile: "cmd/orchestration/workflows/list.go",
		Ops:        []string{"listWorkflows"},
		SDKCalls:   []string{"Orchestration.V1.ListWorkflows"},
	},
	{
		Path:       "orchestration workflows show <id>",
		SourceFile: "cmd/orchestration/workflows/show.go",
		Ops:        []string{"getWorkflow"},
		SDKCalls:   []string{"Orchestration.V1.GetWorkflow"},
	},
	{
		Path:       "orchestration workflows create <file>|-",
		SourceFile: "cmd/orchestration/workflows/create.go",
		Ops:        []string{"createWorkflow"},
		SDKCalls:   []string{"Orchestration.V1.CreateWorkflow"},
	},
	{
		Path:       "orchestration workflows delete <workflow-id>",
		SourceFile: "cmd/orchestration/workflows/delete.go",
		Ops:        []string{"deleteWorkflow"},
		SDKCalls:   []string{"Orchestration.V1.DeleteWorkflow"},
	},
	{
		// The trailing getWorkflow is a render-time read: after running the
		// workflow the controller fetches the workflow definition to label the
		// stages it prints.
		Path:       "orchestration workflows run <id>",
		SourceFile: "cmd/orchestration/workflows/run.go",
		Ops:        []string{"runWorkflow", "getWorkflow"},
		SDKCalls:   []string{"Orchestration.V1.RunWorkflow", "Orchestration.V1.GetWorkflow"},
	},

	// instances
	{
		Path:       "orchestration instances list",
		SourceFile: "cmd/orchestration/instances/list.go",
		Ops:        []string{"listInstances"},
		SDKCalls:   []string{"Orchestration.V1.ListInstances"},
	},
	{
		// The trailing getWorkflow is a render-time read for the stage labels.
		Path:       "orchestration instances show <instance-id>",
		SourceFile: "cmd/orchestration/instances/show.go",
		Ops:        []string{"getInstance", "getWorkflow"},
		SDKCalls:   []string{"Orchestration.V1.GetInstance", "Orchestration.V1.GetWorkflow"},
	},
	{
		// describe issues one history read and then one stage-history read per
		// stage, so its request count is proportional to the instance's stage
		// count rather than fixed.
		Path:       "orchestration instances describe <instance-id>",
		SourceFile: "cmd/orchestration/instances/describe.go",
		Ops:        []string{"getInstanceHistory", "getInstanceStageHistory"},
		SDKCalls:   []string{"Orchestration.V1.GetInstanceHistory", "Orchestration.V1.GetInstanceStageHistory"},
	},
	{
		Path:       "orchestration instances send-event <instance-id> <event>",
		SourceFile: "cmd/orchestration/instances/send_event.go",
		Ops:        []string{"sendEvent"},
		SDKCalls:   []string{"Orchestration.V1.SendEvent"},
	},
	{
		Path:       "orchestration instances stop <instance-id>",
		SourceFile: "cmd/orchestration/instances/stop.go",
		Ops:        []string{"cancelEvent"},
		SDKCalls:   []string{"Orchestration.V1.CancelEvent"},
	},
}

// MappedBaseline returns the baseline commands that map onto an operation.
func MappedBaseline() []BaselineCommand {
	var out []BaselineCommand
	for _, c := range Baseline {
		if len(c.Ops) > 0 {
			out = append(out, c)
		}
	}
	return out
}

// ExcludedBaseline returns the baseline commands recorded as not carried over.
func ExcludedBaseline() []BaselineCommand {
	var out []BaselineCommand
	for _, c := range Baseline {
		if len(c.Ops) == 0 {
			out = append(out, c)
		}
	}
	return out
}

// BaselineTargets returns the sorted, de-duplicated operationIds the baseline
// reaches.
func BaselineTargets() []string {
	seen := map[string]struct{}{}
	for _, c := range Baseline {
		for _, op := range c.Ops {
			seen[op] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for op := range seen {
		out = append(out, op)
	}
	sort.Strings(out)
	return out
}
