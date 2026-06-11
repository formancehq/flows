package wait_event

import (
	internalWorkflow "github.com/formancehq/orchestration/internal/workflow"
	"go.temporal.io/sdk/workflow"
)

func RunWaitEvent(ctx workflow.Context, waitEvent WaitEvent) error {
	channel := workflow.GetSignalChannel(ctx, internalWorkflow.EventSignalName)
	// Drain the signal channel one signal at a time until the expected event
	// arrives. Using a blocking Receive loop (rather than ReceiveAsync inside
	// an Await predicate) guarantees no buffered signal is consumed and
	// dropped: an Await predicate is only evaluated once per workflow-task
	// wakeup, so two signals delivered in the same task would leave the second
	// one buffered with nothing left to re-wake the coroutine, blocking forever.
	for {
		var signal internalWorkflow.Event
		channel.Receive(ctx, &signal)
		if signal.Name == waitEvent.Event {
			return nil
		}
		workflow.GetLogger(ctx).Debug("received unexpected event, still waiting", "event", signal.Name)
	}
}
