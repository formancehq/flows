package components

import (
	"encoding/json"
	"testing"
)

func TestWorkflowConfigActivityBudgetRoundTrip(t *testing.T) {
	const payload = `{"name":"custom","stages":[],"activityMaxAttempts":30}`
	for name, model := range map[string]any{"v1": &WorkflowConfig{}, "v2": &V2WorkflowConfig{}} {
		t.Run(name, func(t *testing.T) {
			if err := json.Unmarshal([]byte(payload), model); err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(model)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			if string(fields["activityMaxAttempts"]) != "30" {
				t.Fatalf("workflow budget lost during SDK encoding: %s", encoded)
			}
		})
	}
}
