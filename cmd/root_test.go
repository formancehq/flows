package cmd

import (
	"net/http"
	"testing"

	"github.com/formancehq/go-libs/v5/pkg/fx/authnfx"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestCommonOptionsBuildsWithJWTAndStackHTTPClients(t *testing.T) {
	cmd := newServeCommand()
	app := fx.New(
		fx.NopLogger,
		authnfx.JWTModuleFromFlags(cmd),
		stackHTTPClientModule(cmd),
		fx.Invoke(fx.Annotate(
			func(stackClient *http.Client) {
				require.NotNil(t, stackClient)
			},
			fx.ParamTags(`name:"stack"`),
		)),
	)
	require.NoError(t, app.Err())
}
