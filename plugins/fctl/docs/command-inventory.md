# fctl Flows plugin — command inventory and admission record

Source of truth for the fctl Flows plugin's operation surface. Hand-written:
the reasoning, the evidence and the gates live here. The tables live in
[`operations.generated.md`](operations.generated.md), which is regenerated from
`openapi.yaml` by `just fctl-audit` and gated by `just fctl-audit-check`.

Every count quoted below is derived by `plugins/fctl/audit` and pinned by
`TestDocumentedTotals`. None of them is transcribed by hand.

## 1. Pinned revisions

| What | Revision |
|---|---|
| Orchestration (this repository, `origin/main`) | `9dc85b316cbfc2485dd8e176b7caa17b73e04e0d` |
| Legacy fctl baseline | `693c58e27865f83332e6c3199d61fed81b742f41` |
| fctl plugin SDK | `e9b1395f46f3100b381dbe00f5213de28e6df0e1` (`../fctl-sdk.lock.json`) |

The pinned fctl SDK revision is reachable from the SDK's
`codex/mvp5-integration` branch, not from its default branch, and the lock has
no field recording why it was selected. Its predecessor was off the default
branch too, so this is continuity rather than a change; it is written down here
because a force-push or rebase of that branch would make the pinned revision
unfetchable.

`../../nix/fctl-component-tools.nix` copies four tool derivations and one patch
from the SDK's authoring toolchain at that revision. Every copied definition was
compared by hand against the SDK and is identical; nothing gates that equality,
which is why it is recorded as an open gate below rather than as a guarantee.

The repository's root module is `github.com/formancehq/orchestration` while its
VCS name is `formancehq/flows`. Both names appear below and they are not
interchangeable; §6 records what that costs.

## 2. Implementation boundary

This directory contains the Flows-owned command provider, v2 adapter, portable
component lifecycle and deterministic build. The host owns Stack target
resolution, credentials, request admission, deadlines and rendering. The
catalogue admits the 16 established command leaves and maps them to the current
v2 operations; the server probes remain host-owned. Its module path is
`github.com/formancehq/orchestration/plugins/fctl`.

The adapter imports the generated `pkg/client` nested module and supplies the
public `producthttp` bridge as its HTTP client, without configuring generated
security or retries. Nix pins the Speakeasy CLI version recorded in the client
lock, and the isolated regeneration gate requires a null diff. The fctl SDK
lock pins the source repository, commit, module path, SDK NAR hash and canonical
WIT hash without embedding a developer checkout path.

## 3. Operation surface

The merged `openapi.yaml` declares **35 operations** with **35 unique
operationIds**: **17** under `orchestration.v1` and **18** under
`orchestration.v2`. **No operation is marked deprecated** in either major.

The two majors are near-mirrors. v2 has one operation v1 does not:
`testTrigger` (`POST /v2/triggers/{triggerID}/test`). Everything else exists in
both, with the v2 form prefixed `v2` in both its path and its operationId.

Functional grouping (frozen in `audit/classify.go`, explicit rather than
prefix-derived, so a new operation fails the completeness test instead of being
absorbed silently):

| Family | Operations | In first tranche |
|---|---:|---|
| `triggers` | 9 | yes |
| `workflows` | 8 | yes |
| `instances` | 10 | yes |
| `instance-history` | 4 | yes |
| `trigger-occurrences` | 2 | yes |
| `server-probe` | 2 | no — host-owned |

`server-probe` (`getServerInfo`, `v2GetServerInfo`) is excluded from the plugin
surface because fctl-v2 owns version discovery: the host probes `/_info` before
it selects a provider. §7 D1 and D2 record what the probe actually does.

The whole surface is behind a single security scheme with exactly two scopes,
`orchestration:read` and `orchestration:write`. The split is mechanical: every
GET declares `read`, every other method declares `write`. That is verified, not
assumed — `TestDeclaredScopesMatchMethodDerivedRule`.

## 4. Legacy baseline mapping

The legacy fctl tree exposes **16 executable `orchestration` commands** at the
pinned baseline revision. Grouping-only cobra commands (`orchestration`,
`orchestration triggers`, `orchestration triggers occurrences`,
`orchestration workflows`, `orchestration instances`) and the shared render
helper are not commands and are not counted.

**All 16 map onto a current operation. There is no exclusion and no
deprecation to justify** — pinned by `TestBaselineHasNoExclusions`. This is
the cleanest baseline of the product set: the orchestration command tree was
never re-cut across API majors, so nothing was orphaned.

The 16 commands reach **17 distinct operations**. The count exceeds the command
count because three commands issue more than one request:

- `workflows run <id>` calls `runWorkflow`, then `getWorkflow` to label the
  stages it prints.
- `instances show <instance-id>` calls `getInstance`, then `getWorkflow` for
  the same reason.
- `instances describe <instance-id>` calls `getInstanceHistory`, then
  `getInstanceStageHistory` once per stage — its request count is proportional
  to the instance's stage count, not fixed.

Those secondary reads are recorded because they are part of the observable
request sequence a replacement has to reproduce or consciously drop, not
because the command "is" two commands.

**The baseline is v1 except for one command.** `orchestration triggers test`
calls `Orchestration.V2.TestTrigger`, because v1 declares and serves no
trigger-test operation at all. Pinned by
`TestBaselineIsOnlyV1PlusTestTrigger`. Any adapter that assumes "legacy fctl
was v1-only" gets this one wrong, and §7 D5 records why the operationId itself
will not warn it.

**18 operations have no legacy precedent**: the two server probes and the 16
v2 forms the legacy tree never called. They are recorded, not admitted. Where
a v2 form is strictly better than the v1 form the legacy command used — which
is the case for all four listings, see §5 — the eventual plugin should prefer
it, and that is a decision for the catalogue, not for this inventory.

## 5. Pagination, bounding and streaming

This is the sharpest split in the surface and the one that most affects a
portable component with a bounded result payload.

**4 operations declare cursor pagination**, all in v2, all listings:
`v2ListTriggers`, `v2ListTriggersOccurrences`, `v2ListWorkflows`,
`v2ListInstances`. They declare the shared `cursor` and `pageSize` parameters
and the server renders the cursor back through `sharedapi.RenderCursor`. Pinned
by `TestOnlyV2ListingsArePaginated`.

The adapter preserves cursors as opaque strings and rejects incoherent
envelopes before emitting a result: `hasMore=true` requires non-empty `next`,
while `hasMore=false` forbids it. The same check applies to single-page and
all-pages execution.

**8 operations return a collection with no declared way to bound it**, and
they fail in two different ways:

- **Unbounded** (blocker B2, 7 operations): the three v1 listings
  `listWorkflows`, `listInstances`, `listTriggersOccurrences`, and all four
  history reads in both majors. The v1 listings pass a zero-valued
  `OffsetPaginatedQuery` and `bunpaginate.usingOffset` emits a SQL `LIMIT` only
  when `PageSize > 0`, so there is none. The history reads take no query object
  at all and render the whole array; instance history is Temporal workflow
  history, so its size is a function of how long the instance ran.
- **Silently truncated** (blocker B3, 1 operation): `listTriggers`. The server
  defaults the page size to 15, computes the next-page cursor, and then drops
  it — `sharedapi.Ok(w, triggers.Data)`. The document declares neither
  parameter. A caller cannot distinguish a truncated page from a complete list.
  That is a wrong answer, not an error, and it is the failure mode the legacy
  `orchestration triggers list` has today.

No operation in this surface streams. The nearest thing is the blocking wait
described next.

**2 operations can block for an unbounded wall-clock duration** (blocker B4):
`runWorkflow` and `v2RunWorkflow` with `wait=true` call
`backend.Wait(r.Context(), instance.ID)`, which returns when the Temporal
workflow terminates. There is no server-side deadline. The legacy command
exposed exactly this as `--wait`.

## 6. Risks

**Destructive — 6 operations.** `deleteTrigger`, `deleteWorkflow`,
`cancelEvent` and their v2 forms. Two of them are not named as such: see §7 D8.
`deleteWorkflow` is a soft delete (`deleted_at IS NULL` filtering in
`internal/workflow/manager.go`); the legacy command said so
(`Soft delete a workflow`) and a replacement should keep saying so.

**Secrets — none.** No schema in the document declares a credential-shaped
property at any nesting depth, verified by
`TestNoCredentialShapedSchemaProperty` over every property name under
`components.schemas`. Orchestration never holds provider credentials: it
addresses Ledger, Payments and Wallets through the stack-internal client built
in `cmd/root.go`, whose OAuth client credentials come from process flags and
never traverse this API.

**Display-once — none.** Every success body in this surface is a re-readable
resource. Pinned by `TestNoDisplayOnceOperation`.

**User expressions — 13 operations.** The sensitive surface here is different
in kind from a credential. Trigger `filter` and `vars` are user-authored
expressions and workflow definitions are user-authored stage configurations,
and every read operation returns both verbatim. `testTrigger` goes furthest: it
echoes the filter's match result and every evaluated variable value back to the
caller. This is user-data disclosure risk in rendered output, and it is tracked
as `Risk.EchoesUserExpressions` rather than as a secret, because calling it a
secret would imply a redaction contract that does not exist and should not be
invented here.

Trigger `vars` and trigger-test events are arbitrary JSON. The adapter uses
lossless JSON numbers before handing their typed maps to the generated client,
so integer tokens above 2^53 are not rounded through `float64`.

**Idempotence — no inbound key anywhere.** The document declares no
`Idempotency-Key` parameter or header on any operation in either major, and the
word does not appear in it at all (`TestNoIdempotencyKeyIsDeclared`). The
repository does use idempotency keys, but strictly outbound: the Temporal
activities under `internal/workflow/activities/` pass `IdempotencyKey` to
Ledger, Wallets and Payments. Consequence for the plugin: a retried
`createWorkflow`, `createTrigger`, `runWorkflow` or `sendEvent` creates a
duplicate. **15 operations use a state-changing method**; the 9 POSTs among
them are not replay-safe and a host-level retry must not be applied to them
blindly.

One of those 9 is a false positive worth carrying into the catalogue:
`testTrigger` is a POST that mutates nothing — it evaluates a trigger against a
sample event and returns the result. It is nonetheless gated behind
`orchestration:write`, both in the document and on the server, because the
server derives the scope from the HTTP method (§7 D4). A read-only command
behind a write scope is a fact to surface to the operator, not one to correct
in the plugin.

## 7. Recorded source risks

Recorded in `audit/blockers.go`, separated from current catalogue admission.
**10 operations carry a recorded product-behavior source risk**. **35 operations have no current admission blocker** and the generated-client build and
regeneration gates are green.
The current plugin contains the command-facing cases by selecting v2
listings, enforcing host response limits, bounding multi-request history
traversal and relying on the host execution deadline.

### B2 — source-side unbounded collection (7 operations)

See §5. This affects `listWorkflows`, `listInstances`,
`listTriggersOccurrences`, `getInstanceHistory`, `getInstanceStageHistory`,
`v2GetInstanceHistory`, and `v2GetInstanceStageHistory`. The adapter selects
the paginated v2 listings and places response and request-count ceilings around
the two v2 history reads.

### B3 — v1 silent truncation (1 operation)

See §5. This affects only `listTriggers`; the catalogue uses the cursor-bearing
v2 operation instead.

### B4 — source-side unbounded blocking wait (2 operations)

See §5. The product provides no deadline for the waiting form of `runWorkflow`
or `v2RunWorkflow`; the portable invocation remains bounded by the host-owned
execution deadline and cancellation contract.

## 8. Spec-versus-server divergences

Recorded in full, with evidence, in
[`operations.generated.md`](operations.generated.md#spec-versus-server-divergences).
Summary:

| ID | What |
|---|---|
| D1 | `/_info` is served unauthenticated although declared under `orchestration:read`. fctl should rely on the server behaviour; the document is what needs fixing. |
| D2 | `GET /v2/_info` is declared and generated but not routed — it 404s. The probe is only ever available unprefixed. |
| D3 | The `Authorization` scheme declares an empty scope dictionary while every operation references two scopes. |
| D4 | Scope checking is method-derived, not per-operation, and is off by default (`--auth-check-scopes=false`, `--auth-service=""`). |
| D5 | `testTrigger` is the only v2 operationId without a `v2` prefix. The SDK name override hides it from the client; an adapter deriving the major from the operationId will not be so lucky. |
| D6 | v1 `GET /triggers` accepts `pageSize` and `cursor` the document does not declare, then omits the cursor from the response — accepted but unusable by construction. |
| D7 | The `name` trigger filter reaches `Where("Name ILIKE '%?%';", …)`, which cannot behave as documented. `origin/fix/triggers-name-filter-sql` is unmerged. |
| D8 | `cancelEvent` aborts a running instance; it does not cancel an event. A catalogue deriving names from operationIds will misname a destructive operation. |

D1 and D2 together are the actionable pair for fctl-v2: **the version probe is
`GET /_info`, unauthenticated, for every major**, and no `/v2` variant exists at
runtime regardless of what the document and the client say.

## 9. Implemented surface and remaining external gates

Implemented locally here:

- the exhaustive operation inventory, derived from source, regenerated by
  command and gated for determinism;
- the exact legacy command mapping, with the multi-request commands and the
  single v2 call called out;
- the functional grouping, frozen and completeness-tested;
- method, path, request/response shape and declared scopes per operation, read
  from the document and never inferred;
- the risk profile per operation: destructive, secret, display-once,
  user-expression, replay-safety, pagination, unbounded, long-running;
- four blockers and eight divergences, each with named evidence and, where one
  exists, a reproducing command;
- a 16-command v2 catalogue with exact operations and scopes;
- a generated-client adapter over `producthttp`, portable lifecycle, WIT and deterministic
  two-lane build recipe;
- focused catalogue, adapter, pagination, lock-agreement, typed
  product-HTTP-failure and lifecycle tests.

Release acceptance also still needs a built artifact receipt, OCI installation,
dual-host execution and live-service read/mutation evidence. Nothing yet gates
`../../nix/fctl-component-tools.nix` against the SDK's authoring toolchain at
the locked revision, so that copy can drift silently; closing it needs the lock
to project the toolchain file and its patch, which is a lock-schema decision.
The `name` filter divergence remains server-owned and must be considered before
relying on it in production.
