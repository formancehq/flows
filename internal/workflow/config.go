package workflow

import (
	"fmt"
	"math"
	"time"

	"github.com/formancehq/orchestration/internal/retry"
	"github.com/formancehq/orchestration/internal/schema"
	"github.com/formancehq/orchestration/internal/temporalworker"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type RawStage map[string]map[string]any

type Config struct {
	Name                string     `json:"name"`
	Stages              []RawStage `json:"stages"`
	ActivityMaxAttempts *int       `json:"activityMaxAttempts,omitempty"`
}

func (c *Config) runStage(ctx workflow.Context, s Stage, stage RawStage, variables map[string]string) (err error) {
	if c.ActivityMaxAttempts != nil {
		ctx = temporalworker.WithStageActivityAttempts(ctx, *c.ActivityMaxAttempts)
	}
	var (
		name  string
		value map[string]any
	)
	for name, value = range stage {
	}

	stageSchema, err := schema.Resolve(schema.Context{
		Variables: variables,
	}, value, name)
	if err != nil {
		return err
	}

	if err := schema.ValidateRequirements(stageSchema); err != nil {
		return err
	}

	err = workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID: s.TemporalWorkflowID(),
		}),
		stageSchema.GetWorkflow(),
		stageSchema,
	).Get(ctx, nil)
	if err != nil {
		var appError *temporal.ApplicationError
		if errors.As(err, &appError) {
			return errors.New(appError.Message())
		}
		var canceledError *temporal.CanceledError
		if errors.As(err, &canceledError) {
			return canceledError
		}
		return err
	}

	return nil
}

func (c *Config) run(ctx workflow.Context, instance Instance, variables map[string]string) (err error) {

	logger := workflow.GetLogger(ctx)
	for ind, rawStage := range c.Stages {
		logger.Info("run stage", "index", ind, "workflowID", instance.ID)

		stage := Stage{}
		err := workflow.ExecuteActivity(retry.ShortActivityContext(ctx), InsertNewStageActivity, instance, ind).Get(ctx, &stage)
		if err != nil {
			return err
		}

		err = workflow.ExecuteActivity(retry.ShortActivityContext(ctx), SendWorkflowStageStartedEventActivity, instance, stage).Get(ctx, nil)
		if err != nil {
			return err
		}

		runError := c.runStage(ctx, stage, rawStage, variables)
		if runError != nil {
			logger.Debug("error running stage", "error", runError)
		}
		stage.SetTerminated(runError, workflow.Now(ctx).Round(time.Nanosecond))

		err = workflow.ExecuteActivity(retry.ShortActivityContext(ctx), UpdateStageActivity, stage).Get(ctx, nil)
		if err != nil {
			return err
		}

		err = workflow.ExecuteActivity(retry.ShortActivityContext(ctx), SendWorkflowStageTerminationEventActivity, instance, stage).Get(ctx, nil)
		if err != nil {
			return err
		}

		logger.Info("stage terminated", "index", ind, "workflowID", stage.InstanceID)

		if runError != nil {
			return runError
		}
	}

	return nil
}

func (c *Config) Validate() error {
	if c.ActivityMaxAttempts != nil && (*c.ActivityMaxAttempts < 1 || *c.ActivityMaxAttempts > math.MaxInt32) {
		return fmt.Errorf("activityMaxAttempts must be between 1 and %d", math.MaxInt32)
	}
	for _, rawStage := range c.Stages {
		if len(rawStage) == 0 {
			return fmt.Errorf("empty specification")
		}
		if len(rawStage) > 1 {
			return fmt.Errorf("a specification should have only one name")
		}
		var (
			name  string
			value map[string]any
		)
		for name, value = range rawStage {
		}

		_, err := schema.Resolve(schema.Context{}, value, name)
		if err != nil {
			return err
		}
	}
	return nil
}
