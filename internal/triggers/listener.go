package triggers

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"

	"go.temporal.io/api/enums/v1"

	collectionutils "github.com/formancehq/go-libs/v5/pkg/types/collections"
	"github.com/formancehq/orchestration/internal/tracer"
	"github.com/formancehq/orchestration/internal/workflow"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"go.temporal.io/api/serviceerror"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/formancehq/go-libs/v5/pkg/messaging/publish"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/client"
)

// Quick hack to filter already processed events
func getWorkflowIDFromEvent(event publish.EventMessage) *string {
	switch event.Type {
	case "SAVED_PAYMENT", "SAVED_ACCOUNT":
		data, err := json.Marshal(event.Payload)
		if err != nil {
			panic(err)
		}

		type object struct {
			ID string `json:"id"`
		}
		o := &object{}
		if err := json.Unmarshal(data, o); err != nil {
			panic(err)
		}

		return pointer.For(o.ID)
	default:
		return nil
	}
}

func listMatchingTriggers(ctx context.Context, db *bun.DB, evaluator *expressionEvaluator, event publish.EventMessage) ([]Trigger, error) {
	triggers := make([]Trigger, 0)
	if err := db.NewSelect().
		Model(&triggers).
		Relation("Workflow").
		Where("trigger.deleted_at is null").
		Where("event = ?", event.Type).
		Where("CASE WHEN trigger.version IS NULL THEN true ELSE trigger.version = ? END", event.Version).
		Scan(ctx); err != nil {
		return nil, err
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Int("triggers-found", len(triggers)),
		attribute.String("triggers-found-ids", strings.Join(collectionutils.Map(triggers, Trigger.GetID), ", ")),
	)

	matched := make([]Trigger, 0, len(triggers))
	for _, trigger := range triggers {
		if trigger.Filter != nil && *trigger.Filter != "" {
			ok, err := evaluator.evalFilter(event.Payload, *trigger.Filter)
			if err != nil {
				logging.FromContext(ctx).Errorf("Error evaluating filter for trigger %s: %s", trigger.ID, err)
				span.SetAttributes(attribute.String("filter-error-"+trigger.ID, err.Error()))
				continue
			}
			if !ok {
				continue
			}
		}
		matched = append(matched, trigger)
	}

	span.SetAttributes(attribute.Int("triggers-matched", len(matched)))

	return matched, nil
}

func handleMessage(
	temporalClient client.Client,
	db *bun.DB,
	evaluator *expressionEvaluator,
	stack, taskIDPrefix, taskQueue string,
	includeSearchAttributes bool,
	msg *message.Message,
) error {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println(e)
			debug.PrintStack()
		}
	}()

	var event *publish.EventMessage
	span, event, err := publish.UnmarshalMessage(msg)
	if err != nil {
		logging.FromContext(msg.Context()).Error(err.Error())
		return err
	}

	ctx, span := tracer.Tracer.Start(msg.Context(), "Trigger:HandleEvent",
		trace.WithLinks(trace.Link{
			SpanContext: span.SpanContext(),
		}),
		trace.WithAttributes(
			attribute.String("event-id", msg.UUID),
			attribute.String("event-type", event.Type),
			attribute.String("event-version", event.Version),
			attribute.String("event-payload", string(msg.Payload)),
		),
	)
	defer span.End()
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
	}()

	matched, err := listMatchingTriggers(ctx, db, evaluator, *event)
	if err != nil {
		return errors.Wrap(err, "listing matching triggers")
	}

	if len(matched) == 0 {
		return nil
	}

	objectID := getWorkflowIDFromEvent(*event)

	for _, trigger := range matched {
		searchAttributes := map[string]interface{}{
			workflow.SearchAttributeStack: stack,
		}
		if includeSearchAttributes {
			searchAttributes[workflow.SearchAttributeTriggerID] = trigger.ID
		}

		options := client.StartWorkflowOptions{
			TaskQueue:        taskQueue,
			SearchAttributes: searchAttributes,
		}
		if objectID != nil {
			options.ID = taskIDPrefix + "-" + trigger.ID + "-" + *objectID
			options.WorkflowIDReusePolicy = enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE
			options.WorkflowExecutionErrorWhenAlreadyStarted = true
		}

		_, execErr := temporalClient.ExecuteWorkflow(ctx, options, ExecuteTrigger, ProcessEventRequest{
			Event: *event,
		}, trigger)
		if execErr != nil {
			var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
			if errors.As(execErr, &alreadyStarted) {
				span.SetAttributes(attribute.Bool("duplicate-"+trigger.ID, true))
				continue
			}
			logging.FromContext(ctx).Errorf("Error executing workflow for trigger %s: %s", trigger.ID, execErr)
			err = execErr
			return errors.Wrap(err, "executing workflow")
		}
	}

	return nil
}

func registerListener(
	r *message.Router,
	s message.Subscriber,
	temporalClient client.Client,
	db *bun.DB,
	evaluator *expressionEvaluator,
	stack, taskIDPrefix, taskQueue string,
	includeSearchAttributes bool,
	topics []string,
) {
	for _, topic := range topics {
		r.AddConsumerHandler(fmt.Sprintf("listen-%s-events", topic), topic, s, func(msg *message.Message) error {
			if err := handleMessage(temporalClient, db, evaluator, stack, taskIDPrefix, taskQueue, includeSearchAttributes, msg); err != nil {
				logging.Errorf("Error executing workflow: %s", err)
				return err
			}
			return nil
		})
	}
}
