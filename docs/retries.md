# Activity Retries

Every step Flows runs against another service or its own database is executed as a Temporal activity. If an activity fails, Flows retries it with a bounded policy. If the activity still fails after the last attempt, the workflow that scheduled it fails. No activity retries forever.

## Retry Policy

All activities share the same policy:

| Setting | Value |
|---------|-------|
| Initial interval | 2s |
| Backoff coefficient | ×2 |
| Maximum interval | 200s |
| Maximum attempts | **15** |

The 14 waits between attempts add up to about 28 minutes. Counting how long each attempt may run, an activity gives up after roughly **30 to 43 minutes**.

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

Flows retries every other error, up to the 15-attempt limit.

## When Retries Run Out

### Stage Operations

The stage fails with the last activity error, and the workflow instance ends as failed. Before this policy existed, the instance stayed "running" forever.

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
| `--stage-activity-attempts` | `15` | Total attempts per stage activity before pausing, including the initial attempt. Must be a positive 32-bit integer. |

With pause mode enabled, `INSUFFICIENT_FUND` becomes retryable until the configured limit. Validation, conflict and compilation errors remain terminal. Trigger and database/publisher bookkeeping activities retain their bounded failure policy.

The workflow stays unfinished while its activity is paused. Workers do not wait for an operator: Temporal stops dispatching that activity. Earlier successful operations stay completed. This uses the original activity execution, preserving `RunID-ActivityID` and heartbeat details. It does not reset or restart the workflow.

The limit is recorded in workflow history and the activity header when the activity is scheduled. A configuration change does not alter the limit for an already scheduled activity. If a timeout or worker crash consumes the final attempt, the next worker invocation pauses without calling the business operation again. If the pause control RPC fails, the activity returns a non-retryable `ACTIVITY_PAUSE_FAILED` error instead of executing the business operation beyond the limit.

### Visibility

`GET /instances/{instanceID}` includes an optional `pendingActivities` array on unfinished stage statuses. It contains only paused activities, with their identity, actual Temporal run ID, last failure, attempt, reason and configured maximum where available. Stage history also exposes `paused`, `pauseReason`, `activityID`, `temporalRunID` and `lastFailureType`; a paused activity has no `nextExecution`. Reads derive these fields from Temporal rather than storing a second pause state in PostgreSQL.

Instance lists retain their current `running` semantics and do not fetch Temporal for every row. The Console instance detail can display the pause and its cause. No Slack or external notification is sent. A paused instance remains nonterminal, so `wait=true` still waits until completion or cancellation.

### Explicit Resume

After resolving the blocking condition, inspect the current paused activity and resume that activity alone:

```sh
temporal activity reset \
  --namespace YOUR_NAMESPACE \
  --workflow-id INSTANCE_ID-STAGE_NUMBER \
  --run-id OBSERVED_TEMPORAL_RUN_ID \
  --activity-id OBSERVED_ACTIVITY_ID
```

Resetting the activity attempt counter grants another configured attempt budget while retaining the activity identity and its idempotency key. Do not use a workflow reset or start a new instance to resume a partially completed stage. Resuming a non-idempotent operation still requires checking the downstream service's behavior.

### Rollout

Deploy the new interceptor to **every worker polling the task queue before enabling pause mode**. Older workers ignore the activity header and cannot enforce the pause budget. Keep workers that understand the header when disabling the option: existing activities still carry their captured limit. Use the same worker version for replay and continuation after activation; history contains the pause feature version and captured limits.

Verify `PauseActivity` support and permissions on the target Temporal deployment first. A finite schedule-to-close timeout can still terminate a paused activity. Existing activities scheduled before activation keep their original policy and are not automatically converted. The paused LTK backlog can remain paused while new activity protection is rolled out.
