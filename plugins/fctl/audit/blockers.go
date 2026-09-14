package audit

import "sort"

// ProductRevision is the orchestration commit every fact in this package was
// read from. It is the commit this preparation is committed on top of.
const ProductRevision = "9dc85b316cbfc2485dd8e176b7caa17b73e04e0d"

// GeneratedClientModulePath is the module path pkg/client/go.mod declares.
// The plugin imports this nested module through a local replacement.
const GeneratedClientModulePath = "openapi"

// ModulePath is this module's own path. It is a subdirectory of the repository
// root module path declared in the root go.mod, which is the authoritative
// contract for this repository regardless of the VCS name.
const ModulePath = "github.com/formancehq/orchestration/plugins/fctl"

// Blocker is a current reason an operation cannot be admitted into the plugin
// catalogue, separate from source defects that the adapter safely contains.
type Blocker struct {
	// OperationIDs are the operations the blocker applies to.
	OperationIDs []string
	// ID is a stable short handle used in the inventory document.
	ID string
	// Summary states the blocker in one sentence.
	Summary string
	// Evidence names the exact sources the blocker was read from.
	Evidence string
	// Reproduce is the exact command that demonstrates the blocker, when one
	// exists. Empty when the blocker is established by reading source only.
	Reproduce string
}

// SourceRisk records a product-source constraint independently from whether
// the current adapter contains it. It intentionally has the same evidence
// shape as an admission blocker.
type SourceRisk Blocker

// SourceRisks are facts about the pinned product source. The current adapter
// resolves or contains them through the generated client over producthttp, v2
// route selection, response limits, request ceilings and the host execution
// deadline. They therefore do not make an admitted command unavailable.
var SourceRisks = []SourceRisk{
	{
		ID: "B2-unbounded-collection",
		OperationIDs: []string{
			"getInstanceHistory",
			"getInstanceStageHistory",
			"listInstances",
			"listTriggersOccurrences",
			"listWorkflows",
			"v2GetInstanceHistory",
			"v2GetInstanceStageHistory",
		},
		Summary: "These operations return a collection the caller has no " +
			"declared way to bound, and the server applies no limit either, so " +
			"the response size is a function of stored state. The adapter " +
			"contains this with host response limits, paginated v2 listings and " +
			"a bounded number of stage-history requests.",
		Evidence: "openapi.yaml declares no cursor/pageSize parameter on any of " +
			"the seven. For the three v1 listings the server passes a " +
			"zero-valued OffsetPaginatedQuery " +
			"(internal/api/v1/handler_list_workflows.go, handler_list_instances.go, " +
			"handler_list_triggers_occurrences.go) and go-libs v3.6.0 " +
			"bun/bunpaginate/pagination_offset.go usingOffset applies " +
			"`sb.Limit(...)` only `if query.PageSize > 0`, so no SQL LIMIT is " +
			"emitted. For the four history reads both majors call " +
			"backend.ReadInstanceHistory / backend.ReadStageHistory with no " +
			"query object at all (internal/api/v1/handler_read_instance_history.go, " +
			"handler_read_stage_history.go and their identical v2 counterparts) " +
			"and render the whole array. The v2 listings are not in this list: " +
			"they declare cursor + pageSize and render the cursor back through " +
			"sharedapi.RenderCursor.",
	},
	{
		ID:           "B3-silent-truncation",
		OperationIDs: []string{"listTriggers"},
		Summary: "v1 GET /triggers returns at most 15 triggers and says nothing " +
			"about it. The server defaults the page size, computes the " +
			"next-page cursor, then drops it from the response, and the " +
			"document declares neither parameter. A caller reading the document " +
			"cannot tell a truncated page from a complete list, which is a " +
			"wrong answer rather than an error.",
		Evidence: "internal/api/v1/handler_list_triggers.go calls " +
			"bunpaginate.GetPageSize(r), which returns " +
			"bunpaginate.QueryDefaultPageSize = 15 when no pageSize is supplied " +
			"(go-libs v3.6.0 bun/bunpaginate/pagination.go), and then responds " +
			"`sharedapi.Ok(w, triggers.Data)` — dropping the Cursor's HasMore, " +
			"Next and Previous fields that usingOffset populated. openapi.yaml " +
			"declares exactly one parameter for listTriggers: the `name` query " +
			"string. The v2 form is unaffected: " +
			"internal/api/v2/handler_list_triggers.go responds " +
			"sharedapi.RenderCursor(w, *triggers).",
	},
	{
		ID:           "B4-unbounded-blocking-wait",
		OperationIDs: []string{"runWorkflow", "v2RunWorkflow"},
		Summary: "With `wait=true` these operations block until the Temporal " +
			"workflow terminates, with no server-side deadline. The legacy " +
			"command exposed exactly this as `--wait`. The portable component " +
			"contains the source behavior through the host-owned execution " +
			"deadline and cancellation contract.",
		Evidence: "openapi.yaml declares `wait` (query, boolean) on POST " +
			"/workflows/{workflowID}/instances and its /v2 form. " +
			"internal/api/v1/handler_run_workflow.go and " +
			"internal/api/v2/handler_run_workflow.go call " +
			"backend.Wait(r.Context(), instance.ID) before responding when the " +
			"parameter is true or 1, with no timeout applied. At legacy fctl " +
			BaselineRevision + ", cmd/orchestration/workflows/run.go declares " +
			"the `--wait` bool flag and forwards it as RunWorkflowRequest.Wait. " +
			"The non-waiting form of both operations is not blocked.",
	},
}

// Blockers contains current catalogue-admission blockers. Every command
// admitted by the v2 catalogue has a bounded, host-mediated execution path.
var Blockers = []Blocker{}

// BlockedOperationIDs returns the sorted, de-duplicated set of operationIds
// carrying at least one blocker.
func BlockedOperationIDs() []string {
	seen := map[string]struct{}{}
	for _, b := range Blockers {
		for _, id := range b.OperationIDs {
			seen[id] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// SourceRiskOperationIDs returns operations affected by at least one recorded
// product-source risk, independently from current catalogue admission.
func SourceRiskOperationIDs() []string {
	seen := map[string]struct{}{}
	for _, risk := range SourceRisks {
		for _, id := range risk.OperationIDs {
			seen[id] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Divergence is a recorded mismatch between the document and the server, or
// between the document and itself, that is not an admission blocker on its own
// but changes what fctl may assume.
type Divergence struct {
	ID       string
	Summary  string
	Evidence string
}

// Divergences are the recorded spec-versus-server and spec-internal mismatches.
var Divergences = []Divergence{
	{
		ID: "D1-info-probe-auth",
		Summary: "The document declares `getServerInfo` (GET /_info) under " +
			"`Authorization: [orchestration:read]`, but the server registers it " +
			"on the root router outside every authenticated group, so it " +
			"answers unauthenticated. fctl needs the unauthenticated read to " +
			"learn the major before it can pick a provider, so the server " +
			"behaviour is the one to rely on — and the document is the one to " +
			"fix.",
		Evidence: "openapi/v1.yaml GET /_info declares the security block; " +
			"internal/api/router.go NewRouter calls r.Get(\"/_info\", " +
			"getInfo(info)) on the root router, while auth.Middleware(authenticator) " +
			"is applied only inside the per-version builders " +
			"(internal/api/v1/router.go, internal/api/v2/router.go).",
	},
	{
		ID: "D2-v2-info-not-routed",
		Summary: "`v2GetServerInfo` (GET /v2/_info) is declared in the document " +
			"and generated into the client, but the v2 router registers no " +
			"`_info` route, so the path resolves to the v2 mux and 404s. The " +
			"version probe is only ever available unprefixed, which is what " +
			"fctl must rely on for every major.",
		Evidence: "openapi/v2.yaml declares GET /v2/_info as v2GetServerInfo and " +
			"pkg/client/v2.go declares func (s *V2) GetServerInfo. " +
			"internal/api/v2/router.go registers only /triggers, /workflows and " +
			"/instances. internal/api/router.go mounts each non-first version at " +
			"`/v<n>/*` through http.StripPrefix, so GET /v2/_info reaches the v2 " +
			"mux as /_info and matches nothing there; the root router's own " +
			"/_info route is not reachable through the /v2 prefix.",
	},
	{
		ID: "D3-undeclared-scheme-scopes",
		Summary: "Every operation references `orchestration:read` or " +
			"`orchestration:write`, but the `Authorization` security scheme " +
			"declares an empty scope dictionary, so the scope names the " +
			"operations use are not defined in the scheme they are attached to. " +
			"The scope strings are still readable per operation; the document " +
			"just does not define them anywhere.",
		Evidence: "openapi/overlay.yaml declares " +
			"components.securitySchemes.Authorization as `type: oauth2` with " +
			"`flows.clientCredentials.scopes: { }`, and it merges into " +
			"openapi.yaml unchanged, while all operations declare " +
			"`security: [{Authorization: [orchestration:read|write]}]`. " +
			"Asserted by TestSecuritySchemeDeclaresNoScopes.",
	},
	{
		ID: "D4-scope-check-is-method-derived-and-off-by-default",
		Summary: "The per-operation scope arrays are a declared contract, not a " +
			"per-operation server check. When scope checking is enabled at all, " +
			"the service derives the requirement from the HTTP method, not from " +
			"the operation; and it is disabled by default. An exact-scope " +
			"catalogue is therefore provable from the document and consistent " +
			"with the method-derived rule as a minimum, but it is not validated " +
			"by this service.",
		Evidence: "internal/api/v1/router.go and internal/api/v2/router.go use " +
			"auth.Middleware(authenticator); go-libs v3.6.0 auth/middleware.go " +
			"only calls Authenticate. auth/auth.go JWTAuth.Authenticate checks " +
			"scopes solely when checkScopes is set, and then allows " +
			"`<service>:read` OR `<service>:write` for GET/HEAD/OPTIONS/TRACE " +
			"and requires `<service>:write` for every other method — the " +
			"operationId is never consulted. auth/cli.go defaults " +
			"--auth-check-scopes to false and --auth-service to the empty " +
			"string. The document's own read/write split is consistent with " +
			"that rule for all operations: asserted by " +
			"TestDeclaredScopesMatchMethodDerivedRule.",
	},
	{
		ID: "D5-test-trigger-operation-id",
		Summary: "`testTrigger` is a /v2-only operation whose operationId " +
			"carries no `v2` prefix, unlike all seventeen of its siblings. Its " +
			"`x-speakeasy-name-override` keeps the generated method correct " +
			"(V2.TestTrigger), so the hazard is not in the client but in any " +
			"adapter that derives the API major from the operationId. That is " +
			"exactly the mapping this baseline needs, because " +
			"`orchestration triggers test` is the one legacy command that calls " +
			"/v2.",
		Evidence: "openapi/v2.yaml declares operationId testTrigger under POST " +
			"/v2/triggers/{triggerID}/test with tag orchestration.v2; every " +
			"other orchestration.v2 operationId begins with `v2`. " +
			"pkg/client/v2.go declares func (s *V2) TestTrigger. At legacy fctl " +
			BaselineRevision + ", cmd/orchestration/triggers/test.go calls " +
			"stackClient.Orchestration.V2.TestTrigger. Asserted by " +
			"TestOnlyTestTriggerLacksVersionPrefix.",
	},
	{
		ID: "D6-v1-accepts-undeclared-pagination",
		Summary: "v1 GET /triggers reads `pageSize` and `cursor` from the query " +
			"although the document declares neither, and then omits the cursor " +
			"from its response. Both parameters are accepted but unusable by " +
			"construction: a caller can shrink the page and can never obtain " +
			"the token needed to advance it.",
		Evidence: "internal/api/v1/handler_list_triggers.go wraps its query " +
			"builder in bunpaginate.Extract, which decodes " +
			"bunpaginate.QueryKeyCursor (\"cursor\") when present, and calls " +
			"bunpaginate.GetPageSize, which reads " +
			"bunpaginate.QueryKeyPageSize (\"pageSize\"); the response is " +
			"sharedapi.Ok(w, triggers.Data). openapi.yaml declares only the " +
			"`name` query parameter for listTriggers. Related to Blocker B3.",
	},
	{
		ID: "D7-trigger-name-filter-sql",
		Summary: "The `name` filter of the trigger listings reaches a WHERE " +
			"fragment that puts the bun placeholder inside a quoted literal and " +
			"terminates the statement mid-fragment, so the filter cannot behave " +
			"as the document describes. `orchestration triggers list --name` is " +
			"the one baseline command that exercises it, so the plugin must " +
			"prove the filter's behaviour before exposing it.",
		Evidence: "internal/triggers/manager.go ListTriggers builds " +
			"`query.Where(\"Name ILIKE '%?%';\", paramsQuery.Options.Name)`: the " +
			"`?` sits inside a single-quoted literal and the fragment ends in " +
			"`;`. openapi.yaml documents `name` as \"search by name\" on both " +
			"listTriggers and v2ListTriggers. origin carries an unmerged branch " +
			"`origin/fix/triggers-name-filter-sql` at ProductRevision.",
	},
	{
		ID: "D8-abort-named-cancel-event",
		Summary: "`cancelEvent` / `v2CancelEvent` (PUT /instances/{id}/abort) " +
			"terminate a running workflow instance; they do not cancel an " +
			"event. The legacy command named it correctly " +
			"(`orchestration instances stop`). Any catalogue that derives a " +
			"command name from the operationId will misname a destructive " +
			"operation as an event operation.",
		Evidence: "openapi.yaml declares operationId cancelEvent on PUT " +
			"/instances/{instanceID}/abort with summary \"Cancel a running " +
			"workflow\"; internal/api/v1/router.go maps it to " +
			"abortWorkflowInstance(backend). At legacy fctl " + BaselineRevision +
			", cmd/orchestration/instances/stop.go exposes it as " +
			"`stop <instance-id>`.",
	},
}
