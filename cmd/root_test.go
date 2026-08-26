package cmd

import (
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestCommonOptionsWithAuthEnabled(t *testing.T) {
	cmd := newServeCommand()
	require.NoError(t, cmd.Flags().Set("auth-enabled", "true"))
	require.NoError(t, cmd.Flags().Set("auth-issuer", "https://auth.example.com"))
	require.NoError(t, cmd.Flags().Set("postgres-uri", "postgres://localhost/orchestration"))

	options, err := commonOptions(cmd)
	require.NoError(t, err)
	require.NoError(t, fx.ValidateApp(
		options,
		fx.NopLogger,
		fx.Supply(fx.Annotate(logging.Testing(), fx.As(new(logging.Logger)))),
	))
}
