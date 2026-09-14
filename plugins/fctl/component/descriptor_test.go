package component

import (
	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
	"testing"
)

func TestDescriptorIsValidFlowsCommandProvider(t *testing.T) {
	d := Descriptor()
	if d.Metadata.Name != "flows" || len(d.Commands) != 16 {
		t.Fatalf("descriptor=%q/%d", d.Metadata.Name, len(d.Commands))
	}
	if err := sdk.ValidateCatalogue(d.Commands, d.DocumentationResources); err != nil {
		t.Fatal(err)
	}
}
