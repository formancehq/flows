package cmd

import (
	"context"

	auth "github.com/formancehq/go-libs/v5/pkg/authn/jwt"
	"github.com/formancehq/go-libs/v5/pkg/authn/licence"
	"github.com/formancehq/go-libs/v5/pkg/cloud/aws/iam"
	"github.com/formancehq/go-libs/v5/pkg/fx/servicefx"
	"github.com/formancehq/go-libs/v5/pkg/fx/transportfx"
	"github.com/formancehq/go-libs/v5/pkg/messaging/publish"
	otlpmetrics "github.com/formancehq/go-libs/v5/pkg/observe/metrics"
	"github.com/formancehq/go-libs/v5/pkg/service"
	"github.com/formancehq/go-libs/v5/pkg/service/health"
	bunconnect "github.com/formancehq/go-libs/v5/pkg/storage/bun/connect"
	"github.com/formancehq/go-libs/v5/pkg/transport/httpserver"
	"github.com/formancehq/go-libs/v5/pkg/workflow/temporal"
	"github.com/formancehq/orchestration/internal/api"
	v1 "github.com/formancehq/orchestration/internal/api/v1"
	v2 "github.com/formancehq/orchestration/internal/api/v2"
	"github.com/formancehq/orchestration/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

func healthCheckModule() fx.Option {
	return fx.Options(
		servicefx.HealthModule(),
		servicefx.ProvideHealthCheck(func() health.NamedCheck {
			return health.NewNamedCheck("default", health.CheckFn(func(ctx context.Context) error {
				return nil
			}))
		}),
	)
}

func newServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "serve",
		RunE: func(cmd *cobra.Command, args []string) error {
			commonOptions, err := commonOptions(cmd)
			if err != nil {
				return err
			}

			listen, _ := cmd.Flags().GetString(listenFlag)

			options := []fx.Option{
				commonOptions,
				healthCheckModule(),
				fx.Provide(func() api.ServiceInfo {
					return api.ServiceInfo{
						Version: Version,
					}
				}),
				v1.NewModule(),
				v2.NewModule(),
				fx.Invoke(func(lifecycle fx.Lifecycle, db *bun.DB) {
					lifecycle.Append(fx.Hook{
						OnStart: func(ctx context.Context) error {
							return storage.Migrate(ctx, db)
						},
					})
				}),
				api.NewModule(service.IsDebug(cmd)),
				fx.Invoke(func(lc fx.Lifecycle, router *chi.Mux) {
					lc.Append(transportfx.FXHook(httpserver.NewHook(router, httpserver.WithAddress(listen))))
				}),
			}
			worker, _ := cmd.Flags().GetBool(workerFlag)
			if worker {
				options = append(options, workerOptions(cmd))
			}

			return service.New(cmd.OutOrStdout(), options...).Run(cmd)
		},
	}

	cmd.Flags().Bool(workerFlag, false, "Enable worker mode")
	cmd.Flags().String(listenFlag, ":8080", "Listening address")
	cmd.Flags().String(stackURLFlag, "", "Stack url")
	cmd.Flags().String(stackClientIDFlag, "", "Stack client ID")
	cmd.Flags().String(stackClientSecretFlag, "", "Stack client secret")
	cmd.Flags().StringSlice(topicsFlag, []string{}, "Topics to listen")
	cmd.Flags().String(stackFlag, "", "Stack")

	service.AddFlags(cmd.Flags())
	publish.AddFlags(ServiceName, cmd.Flags())
	auth.AddFlags(cmd.Flags())
	bunconnect.AddFlags(cmd.Flags())
	iam.AddFlags(cmd.Flags())
	licence.AddFlags(cmd.Flags())
	otlpmetrics.AddFlags(cmd.Flags())
	temporal.AddFlags(cmd.Flags())

	return cmd
}
