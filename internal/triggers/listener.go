package triggers

import (
	"encoding/json"
	"fmt"
	"runtime/debug"

	"go.temporal.io/api/enums/v1"

	"github.com/formancehq/orchestration/internal/tracer"
	"github.com/formancehq/orchestration/internal/workflow"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/formancehq/go-libs/v2/pointer"
	"go.temporal.io/api/serviceerror"

	"github.com/formancehq/go-libs/v2/logging"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/formancehq/go-libs/v2/publish"
	"github.com/pkg/errors"
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

func handleMessage(temporalClient client.Client, stack, taskIDPrefix, taskQueue string, msg *message.Message) error {
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
			attribute.Bool("duplicate", false),
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

	options := client.StartWorkflowOptions{
		TaskQueue: taskQueue,
		SearchAttributes: map[string]interface{}{
			workflow.SearchAttributeStack: stack,
		},
	}
	if ik := getWorkflowIDFromEvent(*event); ik != nil {
		options.ID = taskIDPrefix + "-" + *ik
		options.WorkflowIDReusePolicy = enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE
		options.WorkflowExecutionErrorWhenAlreadyStarted = true
	}

	_, err = temporalClient.ExecuteWorkflow(ctx, options, RunTrigger, ProcessEventRequest{
		Event: *event,
	})
	if err != nil {
		_, ok := err.(*serviceerror.WorkflowExecutionAlreadyStarted)
		if ok {
			span.SetAttributes(attribute.Bool("duplicate", true))
			err = nil
			return nil
		}
	}

	return errors.Wrap(err, "executing workflow")
}

func registerListener(r *message.Router, s message.Subscriber, temporalClient client.Client,
	stack, taskIDPrefix, taskQueue string, topics []string) {
	for _, topic := range topics {
		r.AddNoPublisherHandler(fmt.Sprintf("listen-%s-events", topic), topic, s, func(msg *message.Message) error {
			if err := handleMessage(temporalClient, stack, taskIDPrefix, taskQueue, msg); err != nil {
				logging.Errorf("Error executing workflow: %s", err)
				return err
			}
			return nil
		})
	}
}
