package audit

import "sort"

// Family is a frozen operation family of the orchestration surface. The table
// is deliberately version-agnostic: the same family holds the v1 and v2 form of
// the same behaviour, so the plugin's eventual command grouping does not have
// to be re-derived per major.
type Family string

const (
	FamilyServerProbe        Family = "server-probe"
	FamilyTriggers           Family = "triggers"
	FamilyTriggerOccurrences Family = "trigger-occurrences"
	FamilyWorkflows          Family = "workflows"
	FamilyInstances          Family = "instances"
	FamilyInstanceHistory    Family = "instance-history"
)

// FrozenFamilies are the families in scope for the first plugin tranche: every
// family the legacy baseline reached. FamilyServerProbe is excluded because the
// version probe is host-owned in fctl-v2, not a product command.
var FrozenFamilies = []Family{
	FamilyTriggers,
	FamilyTriggerOccurrences,
	FamilyWorkflows,
	FamilyInstances,
	FamilyInstanceHistory,
}

// familyOf assigns every operationId to exactly one family. The table is
// explicit rather than prefix-derived so that a new spec operation fails the
// completeness test instead of being silently absorbed by a prefix rule.
var familyOf = map[string]Family{
	// server probe; host-owned in fctl-v2, recorded for completeness
	"getServerInfo":   FamilyServerProbe,
	"v2GetServerInfo": FamilyServerProbe,

	// triggers
	"listTriggers":    FamilyTriggers,
	"createTrigger":   FamilyTriggers,
	"readTrigger":     FamilyTriggers,
	"deleteTrigger":   FamilyTriggers,
	"v2ListTriggers":  FamilyTriggers,
	"v2CreateTrigger": FamilyTriggers,
	"v2ReadTrigger":   FamilyTriggers,
	"v2DeleteTrigger": FamilyTriggers,
	// testTrigger carries no v2 prefix although it is a /v2-only operation:
	// see Divergence D5-test-trigger-operation-id.
	"testTrigger": FamilyTriggers,

	// trigger occurrences
	"listTriggersOccurrences":   FamilyTriggerOccurrences,
	"v2ListTriggersOccurrences": FamilyTriggerOccurrences,

	// workflows
	"listWorkflows":    FamilyWorkflows,
	"createWorkflow":   FamilyWorkflows,
	"getWorkflow":      FamilyWorkflows,
	"deleteWorkflow":   FamilyWorkflows,
	"v2ListWorkflows":  FamilyWorkflows,
	"v2CreateWorkflow": FamilyWorkflows,
	"v2GetWorkflow":    FamilyWorkflows,
	"v2DeleteWorkflow": FamilyWorkflows,

	// instances (lifecycle: run, list, read, event, abort)
	"runWorkflow":     FamilyInstances,
	"listInstances":   FamilyInstances,
	"getInstance":     FamilyInstances,
	"sendEvent":       FamilyInstances,
	"cancelEvent":     FamilyInstances,
	"v2RunWorkflow":   FamilyInstances,
	"v2ListInstances": FamilyInstances,
	"v2GetInstance":   FamilyInstances,
	"v2SendEvent":     FamilyInstances,
	"v2CancelEvent":   FamilyInstances,

	// instance history
	"getInstanceHistory":        FamilyInstanceHistory,
	"getInstanceStageHistory":   FamilyInstanceHistory,
	"v2GetInstanceHistory":      FamilyInstanceHistory,
	"v2GetInstanceStageHistory": FamilyInstanceHistory,
}

// FamilyOf returns the frozen family of an operationId, and whether it is
// classified at all.
func FamilyOf(operationID string) (Family, bool) {
	f, ok := familyOf[operationID]
	return f, ok
}

// ClassifiedOperationIDs returns every classified operationId, sorted.
func ClassifiedOperationIDs() []string {
	out := make([]string, 0, len(familyOf))
	for id := range familyOf {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// SecretDirection says which side of an operation carries credentials.
type SecretDirection string

const (
	SecretNone     SecretDirection = ""
	SecretInbound  SecretDirection = "request"
	SecretOutbound SecretDirection = "response"
)

// secretBearing is empty, and that is a verified statement rather than an
// unexamined default.
//
// Evidence: no schema in the pinned openapi.yaml declares a credential-shaped
// property (asserted by TestNoCredentialShapedSchemaProperty, which greps the
// document for apiKey/apiSecret/clientSecret/password/passphrase/privateKey/
// accessKey/token/secret property names). Orchestration never holds provider
// credentials: it addresses Ledger, Payments and Wallets through the
// stack-internal client built in cmd/root.go, whose OAuth client credentials
// come from process flags and never traverse this API.
//
// The sensitive surface here is different in kind: trigger `filter` and `vars`
// are user-authored expressions and workflow definitions are user-authored
// stage configurations, and every read operation returns both verbatim. That is
// user data disclosure risk, not a credential in the payload, and it is
// recorded as Risk.EchoesUserExpressions rather than as a secret.
var secretBearing = map[string]SecretDirection{}

// displayOnce is empty: no operation in this surface mints a value obtainable
// exactly once. Asserted by TestNoDisplayOnceOperation, which pins the empty
// table against the document's success bodies.
var displayOnce = map[string]struct{}{}

// echoesUserExpressions lists the operations whose request or success body
// carries a user-authored trigger expression (`filter`, `vars`) or workflow
// stage configuration back verbatim.
//
// Evidence: components.schemas.TriggerData / V2TriggerData declare `filter`
// (expression evaluated against the event) and `vars`
// (additionalProperties: true); V2TriggerTest echoes both the filter match and
// every evaluated variable value; CreateWorkflowRequest / V2CreateWorkflowRequest
// carry the stage list, and the workflow read operations return it unchanged.
var echoesUserExpressions = map[string]struct{}{
	"createTrigger":    {},
	"readTrigger":      {},
	"listTriggers":     {},
	"createWorkflow":   {},
	"getWorkflow":      {},
	"listWorkflows":    {},
	"v2CreateTrigger":  {},
	"v2ReadTrigger":    {},
	"v2ListTriggers":   {},
	"v2CreateWorkflow": {},
	"v2GetWorkflow":    {},
	"v2ListWorkflows":  {},
	"testTrigger":      {},
}

// destructiveNonDelete lists the operations that remove or terminate server
// state without using the DELETE method. DELETE-method operations are derived.
//
// cancelEvent / v2CancelEvent (PUT /instances/{id}/abort) terminate a running
// workflow instance. The name is a misnomer inherited from the document: the
// operation aborts the instance, it does not cancel an event.
var destructiveNonDelete = map[string]struct{}{
	"cancelEvent":   {},
	"v2CancelEvent": {},
}

// longRunning lists the operations that can block for an unbounded wall-clock
// duration by contract.
//
// Evidence: runWorkflow and v2RunWorkflow declare a `wait` query parameter;
// internal/api/v1/handler_run_workflow.go and internal/api/v2/handler_run_workflow.go
// call backend.Wait(r.Context(), instance.ID) before responding when it is set,
// which blocks until the Temporal workflow terminates. There is no server-side
// deadline on that wait.
var longRunning = map[string]struct{}{
	"runWorkflow":   {},
	"v2RunWorkflow": {},
}

// Risk is the per-operation risk profile, derived from spec facts plus the
// explicit tables above.
type Risk struct {
	// Destructive is true for operations that remove or terminate server state.
	Destructive bool `json:"destructive"`
	// Secret says whether credentials cross the boundary, and in which
	// direction. Always SecretNone on this surface: see secretBearing.
	Secret SecretDirection `json:"secret"`
	// DisplayOnce is true when the success body carries a one-shot value.
	// Always false on this surface: see displayOnce.
	DisplayOnce bool `json:"displayOnce"`
	// EchoesUserExpressions is true when the operation carries user-authored
	// trigger expressions or workflow stage configuration verbatim.
	EchoesUserExpressions bool `json:"echoesUserExpressions"`
	// ReplaySafe is true for HTTP-idempotent methods (GET, PUT, PATCH, DELETE).
	// The document declares no idempotency key on any operation, so POST
	// operations are never replay-safe: see NoIdempotencyKey.
	ReplaySafe bool `json:"replaySafe"`
	// Paginated is true when the document declares cursor + pageSize.
	Paginated bool `json:"paginated"`
	// UnboundedCollection is true when the operation returns a collection the
	// document gives the caller no way to bound. Every v1 listing and every
	// history read is in this state: see Blocker B2.
	UnboundedCollection bool `json:"unboundedCollection"`
	// LongRunning is true when the operation can block for an unbounded
	// wall-clock duration by contract.
	LongRunning bool `json:"longRunning"`
}

// NoIdempotencyKey records that the pinned orchestration document declares no
// Idempotency-Key (or equivalent) parameter or header on any operation, in
// either major. Asserted by TestNoIdempotencyKeyIsDeclared.
//
// The repository does use idempotency keys, but only outbound: the Temporal
// activities in internal/workflow/activities/ pass IdempotencyKey to Ledger,
// Wallets and Payments. Nothing inbound on this API accepts one, so a retried
// createWorkflow, createTrigger, runWorkflow or sendEvent creates a duplicate.
const NoIdempotencyKey = true

// collectionReturning lists the operations whose success body is a collection.
// It is the input to the UnboundedCollection derivation: a collection operation
// that declares no pagination gives the caller no way to bound the response.
var collectionReturning = map[string]struct{}{
	"listTriggers":              {},
	"listTriggersOccurrences":   {},
	"listWorkflows":             {},
	"listInstances":             {},
	"getInstanceHistory":        {},
	"getInstanceStageHistory":   {},
	"v2ListTriggers":            {},
	"v2ListTriggersOccurrences": {},
	"v2ListWorkflows":           {},
	"v2ListInstances":           {},
	"v2GetInstanceHistory":      {},
	"v2GetInstanceStageHistory": {},
}

// RiskOf derives the risk profile of an operation.
func RiskOf(op Operation) Risk {
	_, abortLike := destructiveNonDelete[op.OperationID]
	_, once := displayOnce[op.OperationID]
	_, echoes := echoesUserExpressions[op.OperationID]
	_, blocking := longRunning[op.OperationID]
	_, collection := collectionReturning[op.OperationID]
	paginated := op.Paginated()

	return Risk{
		Destructive:           op.Method == "DELETE" || abortLike,
		Secret:                secretBearing[op.OperationID],
		DisplayOnce:           once,
		EchoesUserExpressions: echoes,
		ReplaySafe:            op.Method == "GET" || op.Method == "PUT" || op.Method == "PATCH" || op.Method == "DELETE",
		Paginated:             paginated,
		UnboundedCollection:   collection && !paginated,
		LongRunning:           blocking,
	}
}
