//go:build fctl_component_guest

package export_formance_fctl_plugin_lifecycle

import (
	"fmt"
	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk/portable/component"
	flowscomponent "github.com/formancehq/orchestration/plugins/fctl/component"
	"github.com/formancehq/orchestration/plugins/fctl/core"
)

var lifecycle = mustLifecycle()

func Describe() []byte                        { return lifecycle.Describe() }
func Start(id string, input []byte) [][]byte  { return lifecycle.Start(id, input) }
func Resume(id string, input []byte) [][]byte { return lifecycle.Resume(id, input) }
func Cancel(id string) [][]byte               { return lifecycle.Cancel(id) }
func Close(id string)                         { lifecycle.Close(id) }
func mustLifecycle() *component.Component {
	value, err := component.New(component.Providers{Command: core.Plugin{}}, flowscomponent.Descriptor())
	if err != nil {
		panic(fmt.Sprintf("configure flows portable component: %v", err))
	}
	return value
}
