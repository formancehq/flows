package cmd

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/go-libs/v5/pkg/fx/authnfx"
	"github.com/formancehq/go-libs/v5/pkg/fx/messagingfx"
	"github.com/formancehq/go-libs/v5/pkg/fx/observefx"
	"github.com/formancehq/go-libs/v5/pkg/fx/storagefx"
	"github.com/formancehq/go-libs/v5/pkg/fx/workflowfx"
	otlp "github.com/formancehq/go-libs/v5/pkg/observe"
	otlptraces "github.com/formancehq/go-libs/v5/pkg/observe/traces"
	"github.com/formancehq/go-libs/v5/pkg/service"
	bunconnect "github.com/formancehq/go-libs/v5/pkg/storage/bun/connect"
	bunmigrate "github.com/formancehq/go-libs/v5/pkg/storage/bun/migrate"
	"github.com/formancehq/go-libs/v5/pkg/workflow/temporal"
	"github.com/formancehq/orchestration/internal/storage"
	"github.com/formancehq/orchestration/internal/temporalworker"
	"github.com/formancehq/orchestration/internal/tracer"
	"github.com/formancehq/orchestration/internal/triggers"
	"github.com/formancehq/orchestration/internal/workflow"
	_ "github.com/formancehq/orchestration/internal/workflow/stages/all"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

var (
	ServiceName = "orchestration"
	Version     = "develop"
	BuildDate   = "-"
	Commit      = "-"
)

const (
	stackFlag             = "stack"
	stackURLFlag          = "stack-url"
	stackClientIDFlag     = "stack-client-id"
	stackClientSecretFlag = "stack-client-secret"
	topicsFlag            = "topics"
	listenFlag            = "listen"
	workerFlag            = "worker"
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{}

	cobra.EnableTraverseRunHooks = true

	cmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	cmd.AddCommand(
		newServeCommand(),
		newVersionCommand(),
		newWorkerCommand(),
		bunmigrate.NewDefaultCommand(func(cmd *cobra.Command, args []string, db *bun.DB) error {
			return storage.Migrate(cmd.Context(), db)
		}),
	)
	otlp.AddFlags(cmd.PersistentFlags())
	otlptraces.AddFlags(cmd.PersistentFlags())

	return cmd
}

func Execute() {
	service.Execute(NewRootCommand())
}

func commonOptions(cmd *cobra.Command) (fx.Option, error) {
	connectionOptions, err := bunconnect.ConnectionOptionsFromFlags(cmd.Flags(), cmd.Context())
	if err != nil {
		return nil, err
	}

	stack, _ := cmd.Flags().GetString(stackFlag)
	temporalTaskQueue, _ := cmd.Flags().GetString(temporal.TemporalTaskQueueFlag)

	return fx.Options(
		observefx.ResourceModuleFromFlags(cmd),
		observefx.TracesModuleFromFlags(cmd),
		workflowfx.TemporalClientModuleFromFlags(
			cmd,
			tracer.Tracer,
			temporal.SearchAttributes{
				SearchAttributes: temporalworker.MergeSearchAttributes(
					workflow.SearchAttributes,
					triggers.SearchAttributes,
				),
			},
		),
		observefx.MetricsModuleFromFlags(cmd),
		storagefx.BunConnectModule(*connectionOptions, service.IsDebug(cmd)),
		messagingfx.PublishModuleFromFlags(cmd, service.IsDebug(cmd)),
		authnfx.JWTModuleFromFlags(cmd),
		authnfx.LicenceModuleFromFlags(cmd, ServiceName),
		workflow.NewModule(stack, temporalTaskQueue),
		triggers.NewModule(stack, temporalTaskQueue),
		fx.Provide(func() *bunconnect.ConnectionOptions {
			return connectionOptions
		}),
		fx.Provide(func() *http.Client {
			httpClient := &http.Client{
				Transport: otlp.NewRoundTripper(http.DefaultTransport, service.IsDebug(cmd)),
			}

			stackClientID, _ := cmd.Flags().GetString(stackClientIDFlag)
			stackClientSecret, _ := cmd.Flags().GetString(stackClientSecretFlag)
			stackURL, _ := cmd.Flags().GetString(stackURLFlag)

			if stackClientID == "" {
				return httpClient
			}
			oauthConfig := clientcredentials.Config{
				ClientID:     stackClientID,
				ClientSecret: stackClientSecret,
				TokenURL:     fmt.Sprintf("%s/api/auth/oauth/token", stackURL),
				Scopes:       []string{"openid", "ledger:read", "ledger:write", "wallets:read", "wallets:write", "payments:read", "payments:write"},
			}
			return oauthConfig.Client(context.WithValue(context.Background(),
				oauth2.HTTPClient, httpClient))
		}),
	), nil
}
