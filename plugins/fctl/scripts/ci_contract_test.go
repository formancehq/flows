package main

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

const workflowPath = "../../../.github/workflows/main.yml"

type defaultWorkflow struct {
	Jobs map[string]struct {
		Secrets map[string]string `yaml:"secrets"`
	} `yaml:"jobs"`
}

// TestDefaultPipelineGrantsSDKCredentialsToEveryPluginGate catches the
// production break where the SDK wrapper can materialize its immutable pin,
// but both reusable jobs lack credentials for the private source repository.
func TestDefaultPipelineGrantsSDKCredentialsToEveryPluginGate(t *testing.T) {
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}
	var workflow defaultWorkflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatalf("parse workflow: %v", err)
	}

	for _, name := range []string{"Dirty", "Tests"} {
		job, ok := workflow.Jobs[name]
		if !ok {
			t.Errorf("workflow declares no %s job", name)
			continue
		}
		if job.Secrets["GIT_PRIVATE_TOKEN"] == "" {
			t.Errorf("%s job cannot materialize the private fctl SDK: GIT_PRIVATE_TOKEN is absent", name)
		}
	}
}
