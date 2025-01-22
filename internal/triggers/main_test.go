package triggers

import (
	"testing"

	"github.com/formancehq/go-libs/v2/testing/docker"
	"github.com/formancehq/go-libs/v2/testing/utils"
	"github.com/stretchr/testify/require"

	"github.com/formancehq/go-libs/v2/logging"
	"go.temporal.io/sdk/testsuite"

	"github.com/formancehq/go-libs/v2/testing/platform/pgtesting"
)

var (
	srv       *pgtesting.PostgresServer
	devServer *testsuite.DevServer
)

func TestMain(m *testing.M) {
	utils.WithTestMain(func(t *utils.TestingTForMain) int {
		srv = pgtesting.CreatePostgresServer(t, docker.NewPool(t, logging.Testing()))

		var err error
		devServer, err = testsuite.StartDevServer(logging.TestingContext(), testsuite.DevServerOptions{})
		require.NoError(t, err)

		t.Cleanup(func() {
			require.NoError(t, devServer.Stop())
		})

		return m.Run()
	})
}
