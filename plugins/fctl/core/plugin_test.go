package core

import (
	"testing"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

func TestPluginContractMatchesCatalogue(t *testing.T) {
	plugin := Plugin{}
	metadata := plugin.Metadata()
	if metadata.Name != Name || metadata.Version != Version {
		t.Fatalf("metadata=%#v, want name=%q version=%q", metadata, Name, Version)
	}
	if len(metadata.Facets) != 1 || metadata.Facets[0].Kind != sdk.FacetCommandProvider {
		t.Fatalf("facets=%#v, want one command-provider facet", metadata.Facets)
	}
	if got := metadata.Facets[0].RequiredHostCapabilities; len(got) != 1 || got[0] != sdk.HostCapabilityGeneratedClientV1 {
		t.Fatalf("required host capabilities=%#v", got)
	}
	if got := plugin.Commands(); len(got) != 16 {
		t.Fatalf("commands=%d, want 16", len(got))
	}
	if resources := plugin.DocumentationResources(); resources != nil {
		t.Fatalf("documentation resources=%#v, want nil", resources)
	}
}
