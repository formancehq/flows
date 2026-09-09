package audit_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/formancehq/orchestration/plugins/fctl/audit"
)

// specPath is the merged document at the repository root, from this package's
// directory.
const specPath = "../../../openapi.yaml"

func build(t *testing.T) *audit.Report {
	t.Helper()
	report, err := audit.Build(specPath)
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	return report
}

// TestGoldenReportIsUpToDate is the determinism gate: the committed report must
// be exactly what the current document produces.
func TestGoldenReportIsUpToDate(t *testing.T) {
	report := build(t)

	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("encode report: %v", err)
	}
	encoded = append(encoded, '\n')

	golden, err := os.ReadFile(filepath.Join("testdata", "report.json"))
	if err != nil {
		t.Fatalf("read golden report: %v", err)
	}

	if string(golden) != string(encoded) {
		t.Fatalf("testdata/report.json is out of date; run `just fctl-audit`")
	}
}

// TestBuildIsDeterministic proves repeated builds of the same document are
// byte-identical, which is what makes the golden gate meaningful.
func TestBuildIsDeterministic(t *testing.T) {
	first, err := json.Marshal(build(t))
	if err != nil {
		t.Fatalf("encode first: %v", err)
	}
	for i := 0; i < 5; i++ {
		next, err := json.Marshal(build(t))
		if err != nil {
			t.Fatalf("encode run %d: %v", i, err)
		}
		if string(first) != string(next) {
			t.Fatalf("run %d differs from the first build", i)
		}
	}
}

// TestEveryOperationIsClassified fails when the document gains an operation the
// family table does not know about. This is the mechanism that stops a new
// operation from silently escaping the service-wide blocker.
func TestEveryOperationIsClassified(t *testing.T) {
	report := build(t)

	classified := map[string]struct{}{}
	for _, id := range audit.ClassifiedOperationIDs() {
		classified[id] = struct{}{}
	}

	for _, rec := range report.Operations {
		if _, ok := classified[rec.OperationID]; !ok {
			t.Errorf("operation %s is in the document but not in the family table", rec.OperationID)
		}
		delete(classified, rec.OperationID)
	}
	for id := range classified {
		t.Errorf("family table classifies %s, which the document does not declare", id)
	}
}

// TestBaselineTargetsExist proves every legacy command maps onto an operation
// the current document actually declares.
func TestBaselineTargetsExist(t *testing.T) {
	report := build(t)
	if missing := report.UnknownBaselineTargets(); len(missing) > 0 {
		t.Fatalf("baseline references operations absent from the document: %v", missing)
	}
}

// TestBaselineHasNoExclusions pins the finding that every executable legacy
// `orchestration` command maps onto a current operation. If a future document
// drops one, this fails and forces an evidenced exclusion instead.
func TestBaselineHasNoExclusions(t *testing.T) {
	if excluded := audit.ExcludedBaseline(); len(excluded) != 0 {
		t.Fatalf("expected no excluded baseline command, got %d", len(excluded))
	}
	if got, want := len(audit.MappedBaseline()), len(audit.Baseline); got != want {
		t.Fatalf("mapped baseline commands = %d, want all %d", got, want)
	}
}

// TestBaselineIsOnlyV1PlusTestTrigger pins the mapping's central fact: the
// legacy tree used the v1 API for everything except the trigger test, which has
// no v1 form at all.
func TestBaselineIsOnlyV1PlusTestTrigger(t *testing.T) {
	report := build(t)
	byID := map[string]audit.Record{}
	for _, rec := range report.Operations {
		byID[rec.OperationID] = rec
	}

	var v2Targets []string
	for _, id := range audit.BaselineTargets() {
		rec, ok := byID[id]
		if !ok {
			t.Fatalf("baseline target %s absent from the document", id)
		}
		if rec.Version == audit.V2 {
			v2Targets = append(v2Targets, id)
		}
	}
	sort.Strings(v2Targets)

	if len(v2Targets) != 1 || v2Targets[0] != "testTrigger" {
		t.Fatalf("baseline v2 targets = %v, want exactly [testTrigger]", v2Targets)
	}
	if _, ok := byID["v2TestTrigger"]; ok {
		t.Fatal("document declares v2TestTrigger; the D5 divergence needs revisiting")
	}
}

// TestBlockedOperationsExist proves no blocker references an operation the
// document does not declare.
func TestBlockedOperationsExist(t *testing.T) {
	report := build(t)
	present := map[string]struct{}{}
	for _, rec := range report.Operations {
		present[rec.OperationID] = struct{}{}
	}
	for _, id := range audit.BlockedOperationIDs() {
		if _, ok := present[id]; !ok {
			t.Errorf("blocker references %s, which the document does not declare", id)
		}
	}
}

// TestGeneratedClientBlockerCoversEverything pins the consequence of B1: while
// the generated client is unusable, no operation is admissible.
func TestGeneratedClientBlockerCoversEverything(t *testing.T) {
	report := build(t)
	if report.Totals.Blocked != report.Totals.SpecOperations {
		t.Fatalf("blocked = %d, want all %d operations while B1 stands",
			report.Totals.Blocked, report.Totals.SpecOperations)
	}
	if report.Totals.Unblocked != 0 {
		t.Fatalf("unblocked = %d, want 0 while B1 stands", report.Totals.Unblocked)
	}
}

// TestNoIdempotencyKeyIsDeclared backs the NoIdempotencyKey constant against
// the document: no operation declares an idempotency parameter of any kind.
func TestNoIdempotencyKeyIsDeclared(t *testing.T) {
	if !audit.NoIdempotencyKey {
		t.Fatal("NoIdempotencyKey is false; this test pins the true case")
	}
	report := build(t)
	for _, rec := range report.Operations {
		for _, p := range rec.Parameters {
			if strings.Contains(strings.ToLower(p.Name), "idempot") {
				t.Errorf("%s declares idempotency parameter %q", rec.OperationID, p.Name)
			}
		}
	}

	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read document: %v", err)
	}
	if strings.Contains(strings.ToLower(string(raw)), "idempot") {
		t.Error("the document mentions idempotency; NoIdempotencyKey needs revisiting")
	}
}

// TestNoCredentialShapedSchemaProperty backs the empty secret table: no schema
// in the document declares a credential-shaped property, at any nesting depth.
//
// The scan is over declared property names only. The OAuth2 security scheme's
// tokenUrl/refreshUrl and its clientCredentials flow name are outside
// components.schemas and are not payload fields, so they are correctly not in
// scope here.
func TestNoCredentialShapedSchemaProperty(t *testing.T) {
	names, err := audit.SchemaPropertyNames(specPath)
	if err != nil {
		t.Fatalf("read schema properties: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("no schema properties found; the scan is not reaching the document")
	}

	for _, name := range names {
		lower := strings.ToLower(name)
		for _, needle := range []string{
			"apikey", "apisecret", "clientsecret", "passphrase", "privatekey",
			"accesskey", "password", "secret", "token", "credential",
		} {
			if strings.Contains(lower, needle) {
				t.Errorf("schema property %q is credential-shaped (%q); the secret table needs revisiting", name, needle)
			}
		}
	}
}

// TestNoDisplayOnceOperation pins the empty display-once table: every success
// body in the document is a re-readable resource, so no value is obtainable
// exactly once.
func TestNoDisplayOnceOperation(t *testing.T) {
	report := build(t)
	for _, rec := range report.Operations {
		if rec.Risk.DisplayOnce {
			t.Errorf("%s is marked display-once but the table is meant to be empty", rec.OperationID)
		}
		if rec.Risk.Secret != audit.SecretNone {
			t.Errorf("%s is marked secret-bearing but the table is meant to be empty", rec.OperationID)
		}
	}
}

// TestDeclaredScopesMatchMethodDerivedRule proves the document's read/write
// split is consistent with the server's method-derived rule (D4): GET declares
// read, every other method declares write.
func TestDeclaredScopesMatchMethodDerivedRule(t *testing.T) {
	report := build(t)
	for _, rec := range report.Operations {
		if !rec.HasSecurity {
			t.Errorf("%s declares no security block", rec.OperationID)
			continue
		}
		want := "orchestration:write"
		if rec.Method == "GET" {
			want = "orchestration:read"
		}
		if len(rec.Scopes) != 1 || rec.Scopes[0] != want {
			t.Errorf("%s (%s) declares scopes %v, want [%s]",
				rec.OperationID, rec.Method, rec.Scopes, want)
		}
	}
}

// TestSecuritySchemeDeclaresNoScopes backs divergence D3.
func TestSecuritySchemeDeclaresNoScopes(t *testing.T) {
	scopes, err := audit.DeclaredSecuritySchemeScopes(specPath, "Authorization")
	if err != nil {
		t.Fatalf("read security scheme: %v", err)
	}
	if len(scopes) != 0 {
		t.Fatalf("Authorization declares scopes %v, want none (D3 assumes none)", scopes)
	}
}

// TestOnlyTestTriggerLacksVersionPrefix backs divergence D5.
func TestOnlyTestTriggerLacksVersionPrefix(t *testing.T) {
	report := build(t)
	var unprefixed []string
	for _, rec := range report.Operations {
		if rec.Version == audit.V2 && !strings.HasPrefix(rec.OperationID, "v2") {
			unprefixed = append(unprefixed, rec.OperationID)
		}
	}
	sort.Strings(unprefixed)
	if len(unprefixed) != 1 || unprefixed[0] != "testTrigger" {
		t.Fatalf("v2 operations without a v2 prefix = %v, want exactly [testTrigger]", unprefixed)
	}
}

// TestOnlyV2ListingsArePaginated pins the pagination split the inventory
// quotes: pagination exists only in v2, and only on the three listings.
func TestOnlyV2ListingsArePaginated(t *testing.T) {
	report := build(t)
	var paginated []string
	for _, rec := range report.Operations {
		if !rec.Risk.Paginated {
			continue
		}
		if rec.Version != audit.V2 {
			t.Errorf("%s is paginated but belongs to %s", rec.OperationID, rec.Version)
		}
		paginated = append(paginated, rec.OperationID)
	}
	sort.Strings(paginated)

	want := []string{"v2ListInstances", "v2ListTriggers", "v2ListTriggersOccurrences", "v2ListWorkflows"}
	if strings.Join(paginated, ",") != strings.Join(want, ",") {
		t.Fatalf("paginated operations = %v, want %v", paginated, want)
	}
}

// TestUnboundedCollectionsAreBlocked proves the risk derivation and the blocker
// table agree: every operation flagged unbounded carries B2 or B3.
func TestUnboundedCollectionsAreBlocked(t *testing.T) {
	report := build(t)
	for _, rec := range report.Operations {
		if !rec.Risk.UnboundedCollection {
			continue
		}
		var covered bool
		for _, id := range rec.Blockers {
			if id == "B2-unbounded-collection" || id == "B3-silent-truncation" {
				covered = true
			}
		}
		if !covered {
			t.Errorf("%s returns an unbounded collection but carries no bounding blocker (%v)",
				rec.OperationID, rec.Blockers)
		}
	}
}

// TestDestructiveOperationsAreExactlyTheKnownSet pins the destructive set,
// including the two abort operations the operationId does not name as such.
func TestDestructiveOperationsAreExactlyTheKnownSet(t *testing.T) {
	report := build(t)
	var destructive []string
	for _, rec := range report.Operations {
		if rec.Risk.Destructive {
			destructive = append(destructive, rec.OperationID)
		}
	}
	sort.Strings(destructive)

	want := []string{
		"cancelEvent", "deleteTrigger", "deleteWorkflow",
		"v2CancelEvent", "v2DeleteTrigger", "v2DeleteWorkflow",
	}
	if strings.Join(destructive, ",") != strings.Join(want, ",") {
		t.Fatalf("destructive operations = %v, want %v", destructive, want)
	}
}

// TestLongRunningOperationsDeclareWait proves the long-running flag is derived
// from something the document actually declares.
func TestLongRunningOperationsDeclareWait(t *testing.T) {
	report := build(t)
	for _, rec := range report.Operations {
		if !rec.Risk.LongRunning {
			continue
		}
		if !rec.HasQueryParam("wait") {
			t.Errorf("%s is marked long-running but declares no `wait` query parameter", rec.OperationID)
		}
	}
}

// TestServerInfoIsTheOnlyUnreferencedFamily records that the version probe is
// the only family the legacy baseline never reached, because fctl owns it.
func TestServerInfoIsTheOnlyUnreferencedFamily(t *testing.T) {
	report := build(t)
	reached := map[audit.Family]bool{}
	for _, rec := range report.Operations {
		if len(rec.BaselineCommands) > 0 {
			reached[rec.Family] = true
		}
	}
	for _, f := range audit.FrozenFamilies {
		if !reached[f] {
			t.Errorf("frozen family %s is never reached by the legacy baseline", f)
		}
	}
	if reached[audit.FamilyServerProbe] {
		t.Error("the legacy baseline reaches the server probe; it is meant to be host-owned")
	}
}

// TestMarkdownCoversEveryOperation proves the generated document renders the
// whole surface rather than a subset.
func TestMarkdownCoversEveryOperation(t *testing.T) {
	report := build(t)
	rendered := report.Markdown()

	for _, rec := range report.Operations {
		if !strings.Contains(rendered, "`"+rec.OperationID+"`") {
			t.Errorf("generated markdown omits %s", rec.OperationID)
		}
	}
	for _, b := range audit.Blockers {
		if !strings.Contains(rendered, b.ID) {
			t.Errorf("generated markdown omits blocker %s", b.ID)
		}
	}
	for _, d := range audit.Divergences {
		if !strings.Contains(rendered, d.ID) {
			t.Errorf("generated markdown omits divergence %s", d.ID)
		}
	}
}

// TestGeneratedMarkdownIsUpToDate is the second half of the determinism gate.
func TestGeneratedMarkdownIsUpToDate(t *testing.T) {
	report := build(t)
	committed, err := os.ReadFile(filepath.Join("..", "docs", "operations.generated.md"))
	if err != nil {
		t.Fatalf("read generated document: %v", err)
	}
	if string(committed) != report.Markdown() {
		t.Fatal("docs/operations.generated.md is out of date; run `just fctl-audit`")
	}
}

// TestInlineRequestBodiesAreRecorded proves a declared-but-unreferenced JSON
// payload is distinguished from no payload at all, so HasRequestBody cannot
// under-report. sendEvent and testTrigger are the operations with inline
// bodies in this document.
func TestInlineRequestBodiesAreRecorded(t *testing.T) {
	report := build(t)
	var inline []string
	for _, rec := range report.Operations {
		if rec.RequestBody == audit.InlineSchema {
			inline = append(inline, rec.OperationID)
		}
		if rec.RequestBody != "" && !rec.HasRequestBody() {
			t.Errorf("%s records a body but HasRequestBody is false", rec.OperationID)
		}
	}
	sort.Strings(inline)

	want := []string{"sendEvent", "testTrigger", "v2SendEvent"}
	if strings.Join(inline, ",") != strings.Join(want, ",") {
		t.Fatalf("inline-body operations = %v, want %v", inline, want)
	}
}

// TestLoadOperationsRejectsMissingDocument covers the error path.
func TestLoadOperationsRejectsMissingDocument(t *testing.T) {
	if _, err := audit.LoadOperations(filepath.Join(t.TempDir(), "absent.yaml")); err == nil {
		t.Fatal("expected an error for a missing document")
	}
}

// TestLoadOperationsRejectsMalformedDocument covers the parse error path.
func TestLoadOperationsRejectsMalformedDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.yaml")
	if err := os.WriteFile(path, []byte("paths: [this is not a mapping"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, err := audit.LoadOperations(path); err == nil {
		t.Fatal("expected a parse error")
	}
}

// TestBuildRejectsUnclassifiedOperation proves the completeness guard fires on
// a document that declares an operation the family table does not know.
func TestBuildRejectsUnclassifiedOperation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "extra.yaml")
	doc := `
paths:
  /surprise:
    get:
      operationId: surpriseOperation
      tags: [orchestration.v1]
      responses:
        '200':
          description: ok
      security:
        - Authorization: [orchestration:read]
`
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	_, err := audit.Build(path)
	if err == nil || !strings.Contains(err.Error(), "surpriseOperation") {
		t.Fatalf("expected a classification error naming the operation, got %v", err)
	}
}

// TestBuildRejectsUnknownTag proves an unrecognised tag is an error rather than
// a silent default.
func TestBuildRejectsUnknownTag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tag.yaml")
	doc := `
paths:
  /surprise:
    get:
      operationId: getServerInfo
      tags: [orchestration.v9]
      responses:
        '200':
          description: ok
`
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	_, err := audit.Build(path)
	if err == nil || !strings.Contains(err.Error(), "orchestration.v9") {
		t.Fatalf("expected an unknown-tag error, got %v", err)
	}
}

// TestBuildRejectsAmbiguousSuccessResponse proves the audit refuses to guess
// which 2xx response is the success shape.
func TestBuildRejectsAmbiguousSuccessResponse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ambiguous.yaml")
	doc := `
paths:
  /_info:
    get:
      operationId: getServerInfo
      tags: [orchestration.v1]
      responses:
        '200':
          description: ok
        '204':
          description: also ok
`
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, err := audit.Build(path); err == nil {
		t.Fatal("expected an ambiguous-success error")
	}
}

// TestDeclaredSecuritySchemeScopesRejectsUnknownScheme covers the error path.
func TestDeclaredSecuritySchemeScopesRejectsUnknownScheme(t *testing.T) {
	if _, err := audit.DeclaredSecuritySchemeScopes(specPath, "Absent"); err == nil {
		t.Fatal("expected an error for an undeclared scheme")
	}
}

// TestByVersionPartitionsTheSurface covers the version helper.
func TestByVersionPartitionsTheSurface(t *testing.T) {
	ops, err := audit.LoadOperations(specPath)
	if err != nil {
		t.Fatalf("load operations: %v", err)
	}
	v1 := audit.ByVersion(ops, audit.V1)
	v2 := audit.ByVersion(ops, audit.V2)
	if len(v1)+len(v2) != len(ops) {
		t.Fatalf("v1(%d) + v2(%d) != total(%d)", len(v1), len(v2), len(ops))
	}
	if len(v1) == 0 || len(v2) == 0 {
		t.Fatal("expected both majors to be non-empty")
	}
}
