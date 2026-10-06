package temporalworker

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const pauseAttemptsHeader = "formance-stage-pause-attempts"

// StagePauseInterceptor uses Temporal's native activity pause so a resume keeps
// the original activity ID, heartbeat and payment idempotency key. A zero limit
// retains the bounded failure policy. Deploy to every worker before enabling it.
// Limits are recorded in history and activity headers, not read from live config
// while an activity retries.
type StagePauseInterceptor struct {
	interceptor.WorkerInterceptorBase
	Attempts int
}

func stageActivity(name string) bool {
	switch name {
	case "GetAccount", "AddAccountMetadata", "CreateTransaction", "StripeTransfer",
		"CreateTransferInitiation", "GetPayment", "ConfirmHold", "CreditWallet",
		"DebitWallet", "GetWallet", "ListWallets", "VoidHold":
		return true
	default:
		return false
	}
}

func (i *StagePauseInterceptor) InterceptWorkflow(ctx workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return &pauseWorkflowInbound{WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next}, attempts: i.Attempts}
}

type pauseWorkflowInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	attempts int
}

func (i *pauseWorkflowInbound) Init(next interceptor.WorkflowOutboundInterceptor) error {
	return i.Next.Init(&pauseWorkflowOutbound{WorkflowOutboundInterceptorBase: interceptor.WorkflowOutboundInterceptorBase{Next: next}, attempts: i.attempts})
}

type pauseWorkflowOutbound struct {
	interceptor.WorkflowOutboundInterceptorBase
	attempts int
}

func (i *pauseWorkflowOutbound) ExecuteActivity(ctx workflow.Context, name string, args ...any) workflow.Future {
	if !stageActivity(name) || workflow.GetVersion(ctx, "stage-activity-pause", workflow.DefaultVersion, 1) == workflow.DefaultVersion {
		return i.Next.ExecuteActivity(ctx, name, args...)
	}
	var attempts int
	if err := workflow.SideEffect(ctx, func(workflow.Context) any { return i.attempts }).Get(&attempts); err != nil {
		panic(err)
	}
	if attempts > 0 {
		options := workflow.GetActivityOptions(ctx)
		policy := temporal.RetryPolicy{InitialInterval: 2 * time.Second, BackoffCoefficient: 2, MaximumInterval: 200 * time.Second}
		if options.RetryPolicy != nil {
			policy = *options.RetryPolicy
		}
		// The worker stops dispatch at the limit. A server retry limit would instead
		// complete the activity on its last failure, even if it had been paused.
		policy.MaximumAttempts = 0
		policy.NonRetryableErrorTypes = slices.DeleteFunc(slices.Clone(policy.NonRetryableErrorTypes), func(code string) bool { return code == "INSUFFICIENT_FUND" })
		options.RetryPolicy = &policy
		ctx = workflow.WithActivityOptions(ctx, options)
		payload, err := converter.GetDefaultDataConverter().ToPayload(attempts)
		if err != nil {
			panic(err)
		}
		interceptor.WorkflowHeader(ctx)[pauseAttemptsHeader] = payload
	}
	return i.Next.ExecuteActivity(ctx, name, args...)
}

func (i *StagePauseInterceptor) InterceptActivity(ctx context.Context, next interceptor.ActivityInboundInterceptor) interceptor.ActivityInboundInterceptor {
	return &pauseActivityInbound{ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next}}
}

type pauseActivityInbound struct {
	interceptor.ActivityInboundInterceptorBase
}

func (i *pauseActivityInbound) ExecuteActivity(ctx context.Context, in *interceptor.ExecuteActivityInput) (any, error) {
	payload := interceptor.Header(ctx)[pauseAttemptsHeader]
	if payload == nil {
		return i.Next.ExecuteActivity(ctx, in)
	}
	var limit int
	if err := converter.GetDefaultDataConverter().FromPayload(payload, &limit); err != nil || limit <= 0 || limit > 2147483647 {
		return nil, temporal.NewNonRetryableApplicationError("invalid activity pause limit", "INVALID_PAUSE_LIMIT", err)
	}
	info := activity.GetInfo(ctx)
	// A worker crash or timeout may have consumed the final attempt without
	// returning an error. Never execute the business operation beyond the budget.
	if info.Attempt > int32(limit) {
		err := temporal.NewApplicationError("activity attempt budget exhausted", "ACTIVITY_ATTEMPT_LIMIT")
		return nil, pauseAfterFailure(ctx, info, limit, err)
	}
	result, err := i.Next.ExecuteActivity(ctx, in)
	if err == nil || info.Attempt < int32(limit) || terminalActivityError(err, info.RetryPolicy) {
		return result, err
	}
	return nil, pauseAfterFailure(ctx, info, limit, err)
}

func terminalActivityError(err error, policy *temporal.RetryPolicy) bool {
	var app *temporal.ApplicationError
	if errors.As(err, &app) && (app.NonRetryable() || (policy != nil && slices.Contains(policy.NonRetryableErrorTypes, app.Type()))) {
		return true
	}
	return temporal.IsCanceledError(err) || errors.Is(err, context.Canceled)
}

func pauseAfterFailure(ctx context.Context, info activity.Info, limit int, failure error) error {
	// The operation may have used most of the attempt deadline. Give the control
	// RPC its own bounded context; never hold a worker waiting for human action.
	control, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	_, err := activity.GetClient(ctx).WorkflowService().PauseActivity(control, &workflowservice.PauseActivityRequest{
		Namespace: info.Namespace,
		Execution: &commonpb.WorkflowExecution{WorkflowId: info.WorkflowExecution.ID, RunId: info.WorkflowExecution.RunID},
		Activity:  &workflowservice.PauseActivityRequest_Id{Id: info.ActivityID},
		Reason:    fmt.Sprintf("ACTIVITY_ATTEMPT_LIMIT:%d", limit),
	})
	if err != nil {
		// Fail closed if this server cannot pause; do not replace a bounded retry
		// policy with an unbounded billing loop on unsupported Temporal versions.
		return temporal.NewNonRetryableApplicationError(fmt.Sprintf("could not pause exhausted activity: %v", err), "ACTIVITY_PAUSE_FAILED", failure)
	}
	activity.GetLogger(ctx).Info("Paused exhausted stage activity", "ActivityID", info.ActivityID, "Attempt", info.Attempt, "MaximumAttempts", limit)
	return failure
}
