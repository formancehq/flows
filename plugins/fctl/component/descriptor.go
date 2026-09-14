package component

import (
	pb "github.com/formancehq/fctl-v2-poc/pkg/plugin/protocol/componentbridgev1alpha1"
	"github.com/formancehq/orchestration/plugins/fctl/core"
)

func Descriptor() pb.Descriptor {
	plugin := core.Plugin{}
	return pb.Descriptor{Metadata: plugin.Metadata(), Commands: plugin.Commands(), DocumentationResources: plugin.DocumentationResources()}
}
