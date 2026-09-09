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
| fctl-v2 programme tree | `8de8c4539ea6664351762dd8dd0e865292e3f216` |

The repository's root module is `github.com/formancehq/orchestration` while its
VCS name is `formancehq/flows`. Both names appear below and they are not
interchangeable; §6 records what that costs.

## 2. What this directory is, and is not

This is the Flows-owned preparation for the fctl Flows plugin (fctl-v2
programme Task 10B). It contains **no plugin**: no runtime, no component entry
point, no HTTP client, no generated bindings, no ABI, no catalogue, no adapter.

The programme gates every product plugin implementation behind its MVP4
contract freeze (portable component lifecycle, host-owned access and
capabilities, exact per-operation authorisation scopes). Writing a catalogue or
an adapter against an unfrozen ABI produces work that has to be thrown away.
Establishing *which operations exist, which ones the legacy CLI covered, what
each one requires, and what is genuinely blocked* does not depend on that
freeze, and it is the input the later implementation needs.

For Flows there is a second, product-local reason to stop here: the generated
client this plugin is required to use does not build. See §6, blocker B1.

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

## 7. Blockers

Recorded in `audit/blockers.go`, separated from the facts above. **All 35
operations are blocked**, because B1 is service-wide.

### B1 — the generated client does not build and cannot be imported

This is the one that stops Task 10B for Flows. The task requires each product
core to route every operation through its own generated `pkg/client` and
explicitly forbids hand-written DTOs, endpoints and transports. At the pinned
revision that client is not usable:

- `pkg/client/go.mod` declares `module openapi` with `go 1.20`, against
  `module github.com/formancehq/orchestration` and `go 1.25.10` in the root
  `go.mod`.
- Of the 221 Go files under `pkg/client`, **94 import `openapi/...`** and
  `pkg/client/formance.go` imports
  **`github.com/formancehq/flows/pkg/client/...`**. The two halves of the
  module disagree about its own path, and neither matches the declaration.
- `pkg/client/go.sum` holds exactly three lines, all `/go.mod` hashes, with
  **no `h1:` module-content hash** for any of its three dependencies.
- The root `go.mod` neither requires nor replaces the client, so nothing in
  this repository compiles it.

Reproduce:

```sh
cd pkg/client && go build ./...
# openapi imports github.com/formancehq/flows/pkg/client/internal/hooks:
#   module declares its path as: openapi
#           but was required as: github.com/formancehq/flows/pkg/client

cd pkg/client && GOFLAGS=-mod=readonly go build ./internal/utils
# missing go.sum entry for module providing package github.com/cenkalti/backoff/v4
```

**Not repaired here, deliberately.** The client is regenerated by
`just generate-client` (speakeasy), so its module path is a
generation-configuration decision, not a hand-editable typo — and the decision
is a real one, between the root module namespace
(`github.com/formancehq/orchestration/pkg/client`) and the VCS namespace
(`github.com/formancehq/flows/pkg/client`, which is what the generated sources
already assume). Editing the generated tree by hand would be overwritten by the
next generation and would hide the choice. This module therefore declares
`github.com/formancehq/orchestration/plugins/fctl` — a subdirectory of the
authoritative root module path — and does not import `pkg/client` at all.

### B2 — unbounded collection (7 operations)

See §5. Blocks `listWorkflows`, `listInstances`, `listTriggersOccurrences`,
`getInstanceHistory`, `getInstanceStageHistory`, `v2GetInstanceHistory`,
`v2GetInstanceStageHistory` until the boundary either declares pagination or
declares a ceiling.

### B3 — silent truncation (1 operation)

See §5. Blocks `listTriggers`. The v2 form is unaffected and is the correct
target for a `flows triggers list` command.

### B4 — unbounded blocking wait (2 operations)

See §5. Blocks the *waiting* form of `runWorkflow` and `v2RunWorkflow` until
the host's deadline and cancellation semantics are frozen. The non-waiting form
is not blocked by B4.

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

## 9. What is prepared, and what remains gated

Prepared and committed here:

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
  exists, a reproducing command.

**Nothing else is claimed.** In particular the following Task 10B acceptance
items are **not** met and are **not** ticked:

- [ ] no catalogue exists, and none can be written while B1 stands;
- [ ] no `producthttp` adapter, no `core/execute.go`, no per-major adapter;
- [ ] no portable component entry point, no WASM component, no local artifact
      recipe;
- [ ] no OCI installation, no dual-host validation, no browser-host coverage;
- [ ] no integration scenario, real read or real mutation, against a running
      service;
- [ ] no `/_info` preflight against a live Stack 3.2 service, so the product
      major returned at runtime is unverified here.

Remaining gates before a portable Flows component, in order:

1. **B1** — decide the generated client's module path, fix it in the speakeasy
   generation configuration, regenerate, and commit a `pkg/client` that builds
   with a complete `go.sum`. Until then no plugin code in this repository can
   compile against it.
2. **fctl-v2 MVP4 4B / 4C / 4D** — the portable ABI freeze, the component
   lifecycle, and the old-path retirement. Task 10B implementation waits on all
   three by the programme's own dependency statement.
3. **B2 / B3** — a bounding decision for the seven unbounded collections and
   the truncating listing, or an explicit catalogue choice to expose only the
   v2 listings.
4. **B4** — the host deadline and cancellation contract, before `--wait` can be
   offered.
5. **D7** — prove the `name` filter before exposing it.
