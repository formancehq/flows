package workflow

import (
	"fmt"
	"time"

	"github.com/formancehq/orchestration/internal/schema"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// terminationContext returns a context safe for running terminal bookkeeping
// activities (status updates, termination events). When the workflow has been
// cancelled, the supplied context is already cancelled and any activity started
// on it fails immediately -- which would leave the instance/stage rows stuck
// "running" and skip the termination event. In that case a disconnected context
// is returned so the bookkeeping still runs.
func terminationContext(ctx workflow.Context) workflow.Context {
	if ctx.Err() == nil {
		return ctx
	}
	disconnected, _ := workflow.NewDisconnectedContext(ctx)
	return disconnected
}

type RawStage map[string]map[string]any

type Config struct {
	Name   string     `json:"name"`
	Stages []RawStage `json:"stages"`
}

func (c *Config) runStage(ctx workflow.Context, s Stage, stage RawStage, variables map[string]string) (err error) {
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
		err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Second,
		}), InsertNewStageActivity, instance, ind).Get(ctx, &stage)
		if err != nil {
			return err
		}

		err = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Second,
		}), SendWorkflowStageStartedEventActivity, instance, stage).Get(ctx, nil)
		if err != nil {
			return err
		}

		runError := c.runStage(ctx, stage, rawStage, variables)
		if runError != nil {
			logger.Debug("error running stage", "error", runError)
		}
		stage.SetTerminated(runError, workflow.Now(ctx).Round(time.Nanosecond))

		// Record the stage termination on a context that survives cancellation,
		// otherwise a cancelled stage would never be marked terminated.
		cleanupCtx := terminationContext(ctx)

		err = workflow.ExecuteActivity(workflow.WithActivityOptions(cleanupCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Second,
		}), UpdateStageActivity, stage).Get(cleanupCtx, nil)
		if err != nil {
			return err
		}

		err = workflow.ExecuteActivity(workflow.WithActivityOptions(cleanupCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Second,
		}), SendWorkflowStageTerminationEventActivity, instance, stage).Get(cleanupCtx, nil)
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
