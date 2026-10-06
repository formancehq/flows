# Activity Retries

Every step Flows runs against another service or its own database is executed as a Temporal activity. Retryable stage operations use the workflow's configured attempt budget. With pause mode disabled, exhausting that budget fails the stage and workflow instance; with pause mode enabled, the activity pauses instead. Non-retryable errors fail immediately. Trigger and bookkeeping activities retain their bounded failure policy.

## Retry Policy

The default stage activity policy is shown below. The maximum attempts comes
from [`activityMaxAttempts`](#per-workflow-attempt-budget), stored in each new
workflow configuration. Trigger and bookkeeping activities keep their fixed
15-attempt policy.

| Setting | Value |
|---------|-------|
| Initial interval | 2s |
| Backoff coefficient | ×2 |
| Maximum interval | 200s |
| Maximum attempts | **15** by default; configurable per workflow for stage activities |

With the default 15-attempt budget, the 14 waits between attempts add up to about 28 minutes. Counting how long each attempt may run, an activity reaches its budget after roughly **30 to 43 minutes**.

Per-attempt timeouts:

| Activities | Timeout per attempt |
|------------|---------------------|
| Stage operations: ledger transactions, wallet debit/credit, payment reads, PSP transfer initiation | 60s |
| Trigger activities (`ListTriggers`, `EvalTriggerVariables`, `InsertTriggerOccurrence`, `SendEventForTriggerTermination`) | 10s |
| Instance and stage bookkeeping (`InsertNewInstance`, `UpdateInstance`, `InsertNewStage`, …) | 10s |

## Non-Retryable Errors

Some errors fail the same way on every attempt. Flows does not retry these; the activity fails on the first attempt:

| Activities | Non-retryable error codes |
|------------|---------------------------|
| Ledger, wallet and payment read operations (`send`, `update` stages) | `VALIDATION`, `CONFLICT`, `NO_SCRIPT`, `COMPILATION_FAILED`, `INSUFFICIENT_FUND` |
| PSP transfer initiation (`CreateTransferInitiation`, `StripeTransfer`) | `VALIDATION`, `CONFLICT` |

With pause mode disabled, Flows retries every other stage error up to the
workflow's configured attempt budget. Non-retryable errors still fail on the
first attempt even when the budget is larger. Trigger and bookkeeping activities
retain their 15-attempt limit. Pause mode changes `INSUFFICIENT_FUND` handling
as described below.

## When Retries Run Out

### Stage Operations

With pause mode disabled, the stage fails with the last activity error when its
configured attempt budget runs out, and the workflow instance ends as failed.
Non-retryable errors can fail the stage earlier. With pause mode enabled,
exhausted retryable stage activities pause as described below.

Flows does not roll back ledger or wallet operations that succeeded earlier in the same stage. For example, a cross-ledger `send` may have committed its first transaction and failed on the second. Check the ledger and wallet state before you re-run the workflow.

Retries of one activity reuse the same idempotency key (`RunID-ActivityID`), so they can never post twice. A manual re-run gets new keys, whether you start a new instance or reset the Temporal workflow. If an earlier attempt may have committed before failing, a re-run can post the operation again.

### Trigger Activities

- `EvalTriggerVariables`: Flows records the occurrence with the error and does not start the workflow, as it does for any variable evaluation error.
- `ListTriggers`, `InsertTriggerOccurrence`, `SendEventForTriggerTermination`: the trigger workflow fails. The event may have no recorded occurrence, or no termination event may have been published.

### Bookkeeping Activities

The workflow fails, and the database may not reflect what actually happened. For example, if `UpdateInstance` never succeeds, an instance can still show as running although its Temporal workflow has ended. Use the Temporal workflow history as the source of truth in that case.

## Activities Scheduled Before a Deploy

Temporal stores an activity's retry policy when it schedules the activity. A change to the policy therefore only applies to activities scheduled after the deploy.

Before this bound was introduced, most activities retried without limit. Any such activity that was still retrying at the upgrade keeps its unlimited policy and never stops on its own. Find these in Temporal (running workflows with a pending activity at a high attempt count), then terminate or reset them by hand.

## Pausing Exhausted Stage Activities

Workers and `serve --worker` accept two options:

| Option | Default | Effect |
|--------|---------|--------|
| `--pause-stage-activities` | `false` | Pause retryable stage operations at their attempt limit instead of terminating the stage. Requires Temporal's native `PauseActivity` API. |
| `--stage-activity-attempts` | `15` | Fallback attempt budget for legacy or synthetic workflows without `activityMaxAttempts`. Must be a positive 32-bit integer. |

With pause mode enabled, `INSUFFICIENT_FUND` becomes retryable until the configured limit. Validation, conflict and compilation errors remain terminal. Trigger and database/publisher bookkeeping activities retain their bounded failure policy.

The workflow stays unfinished while its activity is paused. Workers do not wait for an operator: Temporal stops dispatching that activity. Earlier successful operations stay completed. This uses the original activity execution, preserving `RunID-ActivityID` and heartbeat details. It does not reset or restart the workflow.

The limit is recorded in workflow history and the activity header when the activity is scheduled. A configuration change does not alter the limit for an already scheduled activity. If a timeout or worker crash consumes the final attempt, the next worker invocation pauses without calling the business operation again. If the pause control RPC fails, the activity returns a non-retryable `ACTIVITY_PAUSE_FAILED` error instead of executing the business operation beyond the limit.

### Per-Workflow Attempt Budget

Set `activityMaxAttempts` in the workflow configuration when creating a workflow
through either API version. It counts the initial attempt and accepts integers
from `1` through `2147483647`. Omitting it stores `15` in the new workflow's
configuration; explicit `0`, negative values and larger values are rejected.
Creating and reading the workflow return the stored value in `config`.

For example, create a workflow with a three-attempt budget:

```http
POST /v2/workflows
Content-Type: application/json
Authorization: Bearer TOKEN

{
  "name": "three-attempt-workflow",
  "activityMaxAttempts": 3,
  "stages": []
}
```

The same configuration is accepted at `POST /workflows` in v1. This budget is
per stage activity, not a shared counter across the workflow. It applies whether
pause mode is enabled or disabled, independently of the worker fallback value.
Pausing still requires `--pause-stage-activities`. With pause mode
enabled, an exhausted retryable stage activity pauses after its third attempt.
The worker flag supplies a fallback only for legacy or synthetic workflows
whose configuration has no value; it does not override a newly created
workflow's stored budget. Already scheduled activities keep their captured
limit. Trigger and bookkeeping activities keep their own bounded policies.

### Visibility

`GET /instances/{instanceID}` includes an optional `pendingActivities` array on unfinished stage statuses. It contains only paused activities, with their identity, actual Temporal run ID, last failure, attempt, reason and configured maximum where available. Stage history also exposes `paused`, `pauseReason`, `activityID`, `temporalRunID` and `lastFailureType`; a paused activity has no `nextExecution`. Reads derive these fields from Temporal rather than storing a second pause state in PostgreSQL.

Instance lists retain their current `running` semantics and do not fetch Temporal for every row. API consumers can display the pause and its cause. No external notification is sent. A paused instance remains nonterminal, so `wait=true` still waits until completion or cancellation.

### Explicit Resume

After resolving the blocking condition, read `GET /instances/{instanceID}`
(`/v2/instances/{instanceID}` for v2). Copy the stage number and the activity's
`activityID`, `temporalRunID` and `pausedAt` from `pendingActivities`, then call:

```http
POST /v2/instances/{instanceID}/stages/{number}/activities/{activityID}/resume
Content-Type: application/json
Authorization: Bearer TOKEN

{"temporalRunID":"OBSERVED_CHILD_RUN_ID","pausedAt":"OBSERVED_PAUSE_TIMESTAMP"}
```

The same route without `/v2` is available in v1. It requires the existing
orchestration write authorization and returns `204` on success. It grants a
fresh attempt budget without restarting the workflow, clearing heartbeat
checkpoints or replaying completed operations. Repeated requests while the
activity remains active return `204` without resetting its attempts again.

`pausedAt` identifies the observed pause: after another pause, an old request
returns `409` rather than granting another budget. Flows serializes resume
requests per stage across API replicas. A stale run, terminal instance/stage,
canceling activity or an attempt still settling after a pause returns `409`.
Refresh the instance before deciding whether to retry. Missing resources return
`404`, malformed requests `400`, and unavailable dependencies `500`.
An activity that has already completed is not pending and returns `404`;
a completed stage or instance returns `409`. The endpoint does not promise a
replayed HTTP response after completion. A lost response can be reconciled with
the instance detail; never restart the workflow to retry the resume request.

Existing manually paused activities can also be resumed through this endpoint,
provided Temporal exposes their pause timestamp. They retain their captured
retry policy: legacy unlimited retries do not become bounded by this operation.
Pause and resume operations performed directly in Temporal bypass Flows' stage
lock; do not operate on the same activity concurrently through both paths.

Operators can alternatively resume that activity directly in Temporal:

```sh
temporal activity reset \
  --namespace YOUR_NAMESPACE \
  --workflow-id INSTANCE_ID-STAGE_NUMBER \
  --run-id OBSERVED_TEMPORAL_RUN_ID \
  --activity-id OBSERVED_ACTIVITY_ID
```

Resetting the activity attempt counter grants another configured attempt budget while retaining the activity identity and its idempotency key. Do not use a workflow reset or start a new instance to resume a partially completed stage. Resuming a non-idempotent operation still requires checking the downstream service's behavior.

### Rollout

Deploy the new interceptor to **every worker polling the task queue before accepting workflows with custom activity budgets or enabling pause mode**. Older workers ignore the activity header and cannot enforce the pause budget. Keep workers that understand the header when disabling the option: existing activities still carry their captured limit. Use the same worker version for replay and continuation after activation; history contains the pause feature version and captured limits.

Verify `PauseActivity` support and permissions on the target Temporal deployment first. A finite schedule-to-close timeout can still terminate a paused activity. Existing activities scheduled before activation keep their original policy and are not automatically converted. The paused LTK backlog can remain paused while new activity protection is rolled out.
