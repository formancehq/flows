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
