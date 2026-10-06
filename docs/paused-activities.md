# Paused activity visibility and manual recovery

The instance detail API exposes `status[].pendingActivities` for paused activities
in active stages. Each entry includes the activity ID, the described stage's
Temporal run ID, activity name and attempt counter. Failure details and pause
reason and `pausedAt` are included when Temporal provides them. A reason of
`ACTIVITY_ATTEMPT_LIMIT:<limit>` also exposes a positive `maxAttempts` value.
The default limit is 15 total attempts, including the initial attempt.

Stage activity history additionally exposes `activityID`, `temporalRunID`,
`paused`, `pauseReason` and `lastFailureType`. `nextExecution` is absent while
paused or when Temporal has no scheduled next attempt. A pause is nonterminal;
it does not mark the instance or stage as failed.

These fields are read-only projections, not stored database state. Instance
lists do not perform Temporal lookups. A missing legacy stage execution is
ignored on detail reads. Other Temporal errors leave the instance readable and
set `status[].pauseStateUnavailable` to `true`, with no internal error details
or stale `pendingActivities`. API consumers must treat the pause state as unknown for that stage
before interpreting the absence of paused activities. Successful reads omit
the flag (false). Only the
stage execution itself is described: nested child activities and activities
outside recorded stages are not included in `status[].pendingActivities`.
The data reflects a live snapshot, not an atomic snapshot across all stages.

## Recovery documentation

For worker configuration and explicit manual recovery, see
[Activity Retries](retries.md#explicit-resume). Flows provides an authenticated
resume operation for one paused activity; it never restarts the instance.
