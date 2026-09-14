//go:build fctl_component_guest

package export_formance_fctl_plugin_lifecycle

import (
	pb "github.com/formancehq/fctl-v2-poc/pkg/plugin/protocol/componentbridgev1alpha1"
	"testing"
)

func TestDescribeExportsFlows(t *testing.T) {
	d, err := pb.DecodeDescriptorEnvelope(Describe())
	if err != nil {
		t.Fatal(err)
	}
	if d.Metadata.Name != "flows" || len(d.Commands) != 16 {
		t.Fatalf("descriptor=%q/%d", d.Metadata.Name, len(d.Commands))
	}
}
