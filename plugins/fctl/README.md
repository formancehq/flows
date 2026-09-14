# fctl Flows plugin

This directory contains the product-owned Flows command provider for `fctl`.
It exposes the 16 established command leaves through the Flows v2 API. Service
discovery, target resolution, authentication, request admission, deadlines and
rendering remain owned by the host.

## Command surface

- `triggers list|show|create|delete|test`
- `triggers occurrences list`
- `workflows list|show|create|delete|run`
- `instances list|show|describe|send-event|stop`

The four v2 list operations support host-controlled pagination. Without
`--all`, one page and its opaque continuation are returned. With `--all`, the
plugin follows opaque cursors under the descriptor's page limit and the host's
item and byte budgets. Only the first request carries filters or `pageSize`;
each following request carries the opaque cursor alone.
Cursor envelopes fail closed: `hasMore=true` requires a non-empty `next`, and
`hasMore=false` forbids `next`, in both single-page and `--all` modes.

Every emitted result identifies the selected command ID, as required by the
host composition contract. Lists emit JSON arrays, ordinary reads and writes
emit JSON objects, and no-content mutations emit the canonical empty result.
Neither `cursor` nor a numeric page selector is exposed as a product flag.

`workflows run`, `instances show` and `instances describe` retain their
multi-request presentation reads. Instance history is bounded to the portable
host-request ceiling. `workflows run --wait` is still bounded by the host's
execution deadline even though the product endpoint has no server-side
deadline.

Arbitrary trigger `vars` and test-event JSON is decoded with `UseNumber`, so
integers larger than JavaScript's 53-bit exact range traverse the generated
client without float64 rounding.

## Table render hints

Eleven commands declare a compact, ordered table the host may render. Each
column names one scalar field of the result the adapter emits. Nested scalar
leaves use the SDK's dotted object paths. The paths are proved against both a
real emitted result and the command's public output schema:

| Command | Columns |
|---|---|
| `triggers list`, `show`, `create` | ID, Name, Event, Workflow ID, Created At |
| `triggers occurrences list` | Date, Trigger ID, Instance ID |
| `triggers test` | Match |
| `workflows list`, `show`, `create` | ID, Created At, Updated At |
| `workflows run`, `instances show` | ID, Workflow ID, Workflow Name, Terminated |
| `instances list` | ID, Workflow ID, Created At, Updated At, Terminated |

`Name`, `Instance ID`, `Match` and the composite commands' `Workflow Name` name
optional product fields, so a row where the product omits them has no value to
render. The absence fixtures and public schemas keep those paths optional.
`Workflow Name` is nevertheless useful and stable: both composite commands
already fetch the workflow for presentation, and the hint reads its optional
`workflow.config.name` leaf without exposing the workflow definition.

The other five commands declare no table, and nothing is invented for them. The
four no-content mutations (`triggers delete`, `workflows delete`,
`instances send-event`, `instances stop`) emit the canonical empty result.
`instances describe` emits only the `history` and `stages` arrays; object-path
columns cannot select one stable scalar row from those collections.

Nested, unbounded and low-signal fields are left out on purpose: trigger `vars`,
`filter` and `version`, the occurrence and instance `error` reasons, the
instance `terminatedAt`, workflow definitions, trigger-test variable values and
the composite reads' remaining members.

Render hints select what is displayed; they do not narrow the declared contract.
Each command now declares its known result properties (and collection item
shape) while allowing additional product fields where the API can evolve.
`PublicOutputSchema` stays byte-equal to `RawOutputSchema`, which is what the
pinned SDK requires for an ordinary JSON result. The host owns rendering; this
repository declares the hints and changes no renderer.

## Layout

| Path | Purpose |
|---|---|
| `core/catalogue.go` | Exact command, operation, scope, risk and budget descriptors. |
| `core/adapter_v2.go` | Generated Flows v2 client mapped onto host-owned `producthttp`. |
| `component/descriptor.go` | Immutable portable descriptor. |
| `entrypoints/flows` | Portable lifecycle exports. |
| `wit/plugin.wit` | Public lifecycle WIT. |
| `wit/imports.allowlist` | Reviewed exact component import set enforced by the build. |
| `scripts/build-component.sh` | Reproducible two-lane component build and 16 MiB admission gate. |
| `audit` and `docs` | Generated API inventory and compatibility evidence. |

The adapter imports the product's generated `pkg/client` as a nested module and
supplies `producthttp` as its HTTP client. It does not configure generated
security or retries: endpoint, credentials, request admission and retry policy
remain host-owned. A non-2xx product response is classified by that bridge and
the adapter forwards the resulting typed failure without reinterpreting it. Only
its code crosses the portable boundary: the pinned SDK's terminal frame carries
a failure code and nothing else, so a host sees `product_http_error` where it
used to see the opaque `product_response_failed`, while the failure's message,
its `httpStatus` details and its retryability verdict stay adapter-local. A host
retry policy cannot be built on them today. Nix pins the exact
Speakeasy CLI version recorded in the client lock. `just generate-client`
regenerates with that pin and checks the nested module;
`just generated-client-regeneration-check` regenerates in an isolated copy and
requires a byte-for-byte null diff plus `go list` and `go vet`.
The plugin SDK contract is recorded in `fctl-sdk.lock.json` at fctl revision
`e9b1395f46f3100b381dbe00f5213de28e6df0e1` of
`https://github.com/formancehq/fctl-v2-poc.git`, with exact SDK content and
canonical WIT hashes. That revision is reachable from the SDK's
`codex/mvp5-integration` branch and not from its default branch, and the lock
carries no reason field recording why it was selected.

The lock is the only source of that contract. Four other tracked files restate
part of it at eight anchored sites: the Nix tool pin, the wrapper contract test,
this file and `docs/command-inventory.md`.
`TestEveryRestatedSDKFactMatchesTheLock` fails when an anchored site disagrees
with the lock. `TestNoSupersededSDKValueSurvivesInATrackedFile` is its
complement: it fails when any of those four files, or the lock itself, carries a
revision, canonical WIT hash, SDK content hash, SDK remote owner or SDK module
path the lock does not pin, in any phrasing and at any position. The few
unrelated revisions and content hashes those files legitimately carry are listed
with their reason in that test.

## Validate

From this directory, enter the repository's pinned development shell:

```sh
nix develop ../..
just test
```

The SDK wrapper materializes the exact locked commit in the user cache when no
checkout is supplied. Set `FCTL_SDK_ROOT=/path/to/fctl-v2-poc` to use an
existing checkout instead. In both cases it validates the module path and
locked content hashes. When the source includes Git metadata, it also requires
the locked commit and origin, then projects the SDK and WIT paths from that
exact commit. Ignored or modified working-tree files therefore cannot affect
the command. It creates an ephemeral Go workspace, runs the requested command
against the validated projection, and removes the whole projection afterward.
No workstation path or Nix store path is tracked.

The component build needs the fctl authoring tools plus Binaryen. They are
declared by the repository development shell used above:

```sh
just build-component
```

The build compiles twice from the same staging path, validates both components,
compares the component, WIT, import list and digest receipts byte-for-byte,
enforces the reviewed import allowlist, and rejects an artifact larger than
16 MiB. Before authoring, it requires exact runtime versions:
`componentize-go 0.4.1`, `wasi-virt 0.2.0`, `wasm-tools 1.239.0` and
`wasm-opt 124`. The staging guard rejects any `.claude-flow` path both before
and after component authoring, before artifacts can be copied. Generated
`build/`, `bindings/` and `dist/` content is not committed; the source lifecycle
under `entrypoints/` remains tracked.
