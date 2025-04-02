package workflow

import (
	"context"
	"testing"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/testing/docker"
	"github.com/formancehq/go-libs/v2/testing/platform/pgtesting"
	"github.com/formancehq/go-libs/v2/testing/utils"
	"github.com/formancehq/orchestration/internal/temporalclient"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

var (
	srv       *pgtesting.PostgresServer
	devServer *testsuite.DevServer
)

func TestMain(m *testing.M) {
	utils.WithTestMain(func(t *utils.TestingTForMain) int {
		srv = pgtesting.CreatePostgresServer(t, docker.NewPool(t, logging.Testing()))

		var err error
		devServer, err = testsuite.StartDevServer(context.Background(), testsuite.DevServerOptions{})
		require.NoError(t, err)

		err = temporalclient.CreateSearchAttributes(logging.TestingContext(), devServer.Client(), "default", SearchAttributes)
		require.NoError(t, err)

		t.Cleanup(func() {
			require.NoError(t, devServer.Stop())
		})

		return m.Run()
	})
}
