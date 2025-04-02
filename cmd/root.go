package cmd

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/go-libs/v2/bun/bunmigrate"
	"github.com/formancehq/go-libs/v2/licence"
	"github.com/formancehq/orchestration/internal/storage"
	"github.com/uptrace/bun"

	"github.com/formancehq/go-libs/v2/bun/bunconnect"

	"github.com/formancehq/go-libs/v2/auth"
	"github.com/formancehq/go-libs/v2/otlp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/formancehq/orchestration/internal/triggers"
	"github.com/formancehq/orchestration/internal/workflow"

	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/orchestration/internal/temporalclient"

	"github.com/formancehq/go-libs/v2/otlp/otlptraces"
	"github.com/formancehq/go-libs/v2/service"
	_ "github.com/formancehq/orchestration/internal/workflow/stages/all"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

var (
	ServiceName = "orchestration"
	Version     = "develop"
	BuildDate   = "-"
	Commit      = "-"
)

const (
	stackFlag                         = "stack"
	stackURLFlag                      = "stack-url"
	stackClientIDFlag                 = "stack-client-id"
	stackClientSecretFlag             = "stack-client-secret"
	temporalAddressFlag               = "temporal-address"
	temporalNamespaceFlag             = "temporal-namespace"
	temporalSSLClientKeyFlag          = "temporal-ssl-client-key"
	temporalSSLClientCertFlag         = "temporal-ssl-client-cert"
	temporalTaskQueueFlag             = "temporal-task-queue"
	temporalInitSearchAttributes      = "temporal-init-search-attributes"
	temporalMaxParallelActivitiesFlag = "temporal-max-parallel-activities"
	topicsFlag                        = "topics"
	listenFlag                        = "listen"
	workerFlag                        = "worker"
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
	connectionOptions, err := bunconnect.ConnectionOptionsFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	stack, _ := cmd.Flags().GetString(stackFlag)
	temporalAddress, _ := cmd.Flags().GetString(temporalAddressFlag)
	temporalNamespace, _ := cmd.Flags().GetString(temporalNamespaceFlag)
	temporalSSLClientKey, _ := cmd.Flags().GetString(temporalSSLClientKeyFlag)
	temporalSSLClientCert, _ := cmd.Flags().GetString(temporalSSLClientCertFlag)
	temporalTaskQueue, _ := cmd.Flags().GetString(temporalTaskQueueFlag)
	temporalInitSearchAttributes, _ := cmd.Flags().GetBool(temporalInitSearchAttributes)

	return fx.Options(
		otlp.FXModuleFromFlags(cmd),
		otlptraces.FXModuleFromFlags(cmd),
		temporalclient.NewModule(
			temporalAddress,
			temporalNamespace,
			temporalSSLClientCert,
			temporalSSLClientKey,
			temporalInitSearchAttributes,
			workflow.SearchAttributes,
			triggers.SearchAttributes,
		),
		bunconnect.Module(*connectionOptions, service.IsDebug(cmd)),
		publish.FXModuleFromFlags(cmd, service.IsDebug(cmd)),
		auth.FXModuleFromFlags(cmd),
		licence.FXModuleFromFlags(cmd, ServiceName),
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
