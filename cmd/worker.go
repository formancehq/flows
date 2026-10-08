package cmd

import (
	"fmt"
	"net/http"

	sdk "github.com/formancehq/formance-sdk-go/v5"
	"github.com/formancehq/go-libs/v3/aws/iam"
	"github.com/formancehq/go-libs/v3/bun/bunconnect"
	"github.com/formancehq/go-libs/v3/licence"
	"github.com/formancehq/go-libs/v3/otlp/otlpmetrics"
	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/go-libs/v3/service"
	"github.com/formancehq/go-libs/v3/temporal"
	"github.com/formancehq/orchestration/internal/temporalworker"
	"github.com/formancehq/orchestration/internal/triggers"
	"github.com/spf13/cobra"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
	"go.uber.org/fx"
)

func stackClientModule(cmd *cobra.Command) fx.Option {
	stackURL, _ := cmd.Flags().GetString(stackURLFlag)

	return fx.Options(
		fx.Provide(func(httpClient *http.Client) *sdk.Formance {
			return sdk.New(
				sdk.WithClient(httpClient),
				sdk.WithServerURL(stackURL),
			)
		}),
	)
}

func workerOptions(cmd *cobra.Command) (fx.Option, error) {

	stack, _ := cmd.Flags().GetString(stackFlag)
	temporalTaskQueue, _ := cmd.Flags().GetString(temporal.TemporalTaskQueueFlag)
	temporalMaxParallelActivities, err := cmd.Flags().GetFloat64(temporal.TemporalMaxParallelActivitiesFlag)
	if err != nil {
		return nil, fmt.Errorf("reading flag --%s: %w", temporal.TemporalMaxParallelActivitiesFlag, err)
	}
	topics, _ := cmd.Flags().GetStringSlice(topicsFlag)
	attempts, _ := cmd.Flags().GetInt(stageActivityAttemptsFlag)
	pause, _ := cmd.Flags().GetBool(pauseStageActivitiesFlag)
	if !pause {
		attempts = 0
	}

	return fx.Options(
		stackClientModule(cmd),
		temporalworker.NewWorkerModule(temporalTaskQueue, worker.Options{
			TaskQueueActivitiesPerSecond: temporalMaxParallelActivities,
			Interceptors:                 []interceptor.WorkerInterceptor{&temporalworker.StagePauseInterceptor{Attempts: attempts}},
		}),
		triggers.NewListenerModule(
			stack,
			stack,
			temporalTaskQueue,
			true,
			topics,
		),
	), nil
}

func newWorkerCommand() *cobra.Command {
	ret := &cobra.Command{
		Use: "worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			commonOptions, err := commonOptions(cmd)
			if err != nil {
				return err
			}

			workerOptions, err := workerOptions(cmd)
			if err != nil {
				return err
			}

			return service.New(cmd.OutOrStdout(), commonOptions, workerOptions).Run(cmd)
		},
	}
	ret.Flags().String(stackURLFlag, "", "Stack url")
	ret.Flags().String(stackClientIDFlag, "", "Stack client ID")
	ret.Flags().String(stackClientSecretFlag, "", "Stack client secret")
	ret.Flags().StringSlice(topicsFlag, []string{}, "Topics to listen")
	ret.Flags().String(stackFlag, "", "Stack")
	ret.Flags().Bool(pauseStageActivitiesFlag, false, "Pause exhausted stage activities instead of failing (requires Temporal activity pause support)")
	ret.Flags().Int(stageActivityAttemptsFlag, 15, "Total stage activity attempts before pausing, including the initial attempt")

	publish.AddFlags(ServiceName, ret.Flags())
	bunconnect.AddFlags(ret.Flags())
	iam.AddFlags(ret.Flags())
	service.AddFlags(ret.Flags())
	licence.AddFlags(ret.Flags())
	temporal.AddFlags(ret.Flags())
	otlpmetrics.AddFlags(ret.Flags())

	return ret
}
