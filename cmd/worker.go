package cmd

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	sdk "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/go-libs/v5/pkg/authn/licence"
	"github.com/formancehq/go-libs/v5/pkg/cloud/aws/iam"
	"github.com/formancehq/go-libs/v5/pkg/messaging/publish"
	otlpmetrics "github.com/formancehq/go-libs/v5/pkg/observe/metrics"
	"github.com/formancehq/go-libs/v5/pkg/service"
	bunconnect "github.com/formancehq/go-libs/v5/pkg/storage/bun/connect"
	"github.com/formancehq/go-libs/v5/pkg/workflow/temporal"
	"github.com/formancehq/orchestration/internal/temporalworker"
	"github.com/formancehq/orchestration/internal/triggers"
	"github.com/spf13/cobra"
	"go.temporal.io/sdk/worker"
	"go.uber.org/fx"
)

func stackClientModule(cmd *cobra.Command) fx.Option {
	stackURL, _ := cmd.Flags().GetString(stackURLFlag)

	return fx.Options(
		fx.Provide(fx.Annotate(func(httpClient *http.Client) *sdk.Formance {
			return sdk.New(
				sdk.WithClient(httpClient),
				sdk.WithServerURL(stackURL),
			)
		}, fx.ParamTags(`name:"stack"`))),
	)
}

func workerOptions(cmd *cobra.Command) (fx.Option, error) {

	stack, _ := cmd.Flags().GetString(stackFlag)
	temporalTaskQueue, _ := cmd.Flags().GetString(temporal.TemporalTaskQueueFlag)
	// The flag is registered as a float64 in go-libs; reading it with GetInt
	// silently fails and yields 0, so the configured limit was never applied.
	temporalMaxParallelActivities, err := cmd.Flags().GetFloat64(temporal.TemporalMaxParallelActivitiesFlag)
	if err != nil {
		return nil, err
	}
	maxIntExclusive := math.Exp2(float64(strconv.IntSize - 1))
	if temporalMaxParallelActivities <= 0 ||
		math.Trunc(temporalMaxParallelActivities) != temporalMaxParallelActivities ||
		temporalMaxParallelActivities >= maxIntExclusive {
		return nil, fmt.Errorf("%s must be a positive whole number", temporal.TemporalMaxParallelActivitiesFlag)
	}
	topics, _ := cmd.Flags().GetStringSlice(topicsFlag)

	return fx.Options(
		stackClientModule(cmd),
		temporalworker.NewWorkerModule(temporalTaskQueue, worker.Options{
			// "max parallel activities" caps concurrency, which maps to
			// MaxConcurrentActivityExecutionSize, not the queue-wide rate limit
			// TaskQueueActivitiesPerSecond it was previously wired to.
			MaxConcurrentActivityExecutionSize: int(temporalMaxParallelActivities),
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

	publish.AddFlags(ServiceName, ret.Flags())
	bunconnect.AddFlags(ret.Flags())
	iam.AddFlags(ret.Flags())
	service.AddFlags(ret.Flags())
	licence.AddFlags(ret.Flags())
	temporal.AddFlags(ret.Flags())
	otlpmetrics.AddFlags(ret.Flags())

	return ret
}
