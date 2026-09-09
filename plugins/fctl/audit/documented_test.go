package audit_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/formancehq/orchestration/plugins/fctl/audit"
)

// inventoryPath is the hand-written source-of-truth document. Every count it
// quotes is pinned here against the derived report, so prose and code cannot
// drift apart silently.
const inventoryPath = "../docs/command-inventory.md"

func inventory(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatalf("read inventory: %v", err)
	}
	return string(raw)
}

func countRisk(report *audit.Report, match func(audit.Risk) bool) int {
	var n int
	for _, rec := range report.Operations {
		if match(rec.Risk) {
			n++
		}
	}
	return n
}

func countMethod(report *audit.Report, method string) int {
	var n int
	for _, rec := range report.Operations {
		if rec.Method == method {
			n++
		}
	}
	return n
}

// TestDocumentedTotals pins every count the inventory prose quotes.
func TestDocumentedTotals(t *testing.T) {
	report := build(t)
	text := inventory(t)
	t.Logf("totals: %+v", report.Totals)

	for _, c := range []struct {
		what  string
		got   int
		quote string
	}{
		{"operations", report.Totals.SpecOperations, "**35 operations**"},
		{"unique operationIds", report.Totals.UniqueOperationIDs, "**35 unique\noperationIds**"},
		{"v1 operations", report.Totals.V1Operations, "**17** under `orchestration.v1`"},
		{"v2 operations", report.Totals.V2Operations, "**18** under\n`orchestration.v2`"},
		{"baseline commands", report.Totals.BaselineCommands, "**16 executable `orchestration` commands**"},
		{"baseline mapped", report.Totals.BaselineMapped, "**All 16 map onto a current operation"},
		{"baseline distinct operations", report.Totals.BaselineDistinctOperations, "reach **17 distinct operations**"},
		{"without baseline", report.Totals.WithoutBaseline, "**18 operations have no legacy precedent**"},
		{"blocked", report.Totals.Blocked, "**All 35\noperations are blocked**"},
		{"paginated", report.Totals.PaginatedOperations, "**4 operations declare cursor pagination**"},
		{"unbounded", report.Totals.UnboundedCollections, "**8 operations return a collection"},
		{"mutating", report.Totals.MutatingOperations, "**15 operations use a state-changing method**"},
		{"destructive", report.Totals.DestructiveOperations, "**Destructive — 6 operations.**"},
		{"long-running", countRisk(report, func(r audit.Risk) bool { return r.LongRunning }),
			"**2 operations can block for an unbounded wall-clock duration**"},
		{"user expressions", countRisk(report, func(r audit.Risk) bool { return r.EchoesUserExpressions }),
			"**User expressions — 13 operations.**"},
		{"POST operations", countMethod(report, "POST"), "the 9 POSTs among\nthem are not replay-safe"},
	} {
		if !strings.Contains(text, c.quote) {
			t.Errorf("inventory no longer contains the %s claim %q", c.what, c.quote)
			continue
		}
		if !strings.Contains(c.quote, fmt.Sprint(c.got)) {
			t.Errorf("%s: derived %d, but the quoted claim is %q", c.what, c.got, c.quote)
		}
	}
}

// TestDocumentedZeroClaims pins the counts the inventory states as zero.
func TestDocumentedZeroClaims(t *testing.T) {
	report := build(t)
	text := inventory(t)

	if report.Totals.DeprecatedOperations != 0 {
		t.Errorf("document now marks %d operations deprecated; the inventory claims none",
			report.Totals.DeprecatedOperations)
	}
	if !strings.Contains(text, "**No operation is marked deprecated**") {
		t.Error("inventory no longer states that no operation is deprecated")
	}

	if report.Totals.BaselineExcluded != 0 {
		t.Errorf("%d baseline commands are now excluded; the inventory claims none",
			report.Totals.BaselineExcluded)
	}
	if !strings.Contains(text, "There is no exclusion and no\ndeprecation to justify") {
		t.Error("inventory no longer states that nothing is excluded")
	}

	if !strings.Contains(text, "**Secrets — none.**") {
		t.Error("inventory no longer states the empty secret finding")
	}
	if !strings.Contains(text, "**Display-once — none.**") {
		t.Error("inventory no longer states the empty display-once finding")
	}
}

// TestDocumentedFamilyCounts pins the family table in the inventory prose.
func TestDocumentedFamilyCounts(t *testing.T) {
	report := build(t)
	text := inventory(t)

	for _, c := range []struct {
		family audit.Family
		row    string
	}{
		{audit.FamilyTriggers, "| `triggers` | 9 |"},
		{audit.FamilyWorkflows, "| `workflows` | 8 |"},
		{audit.FamilyInstances, "| `instances` | 10 |"},
		{audit.FamilyInstanceHistory, "| `instance-history` | 4 |"},
		{audit.FamilyTriggerOccurrences, "| `trigger-occurrences` | 2 |"},
		{audit.FamilyServerProbe, "| `server-probe` | 2 |"},
	} {
		got := len(report.ByFamily()[c.family])
		if !strings.Contains(text, c.row) {
			t.Errorf("inventory no longer contains the row %q (family holds %d)", c.row, got)
			continue
		}
		if !strings.Contains(c.row, fmt.Sprint(got)) {
			t.Errorf("family %s holds %d operations, but the quoted row is %q", c.family, got, c.row)
		}
	}
}

// TestDocumentedRevisions pins the revision table against the constants.
func TestDocumentedRevisions(t *testing.T) {
	text := inventory(t)
	for _, rev := range []string{audit.ProductRevision, audit.BaselineRevision} {
		if !strings.Contains(text, rev) {
			t.Errorf("inventory does not pin revision %s", rev)
		}
	}
}

// TestDocumentedBlockersAndDivergences proves the inventory names every
// recorded blocker and divergence, so none can be dropped from the prose while
// remaining in the code.
func TestDocumentedBlockersAndDivergences(t *testing.T) {
	text := inventory(t)

	for _, b := range audit.Blockers {
		short := strings.SplitN(b.ID, "-", 2)[0]
		if !strings.Contains(text, "### "+short+" —") {
			t.Errorf("inventory has no section for blocker %s", b.ID)
		}
	}
	for _, d := range audit.Divergences {
		short := strings.SplitN(d.ID, "-", 2)[0]
		if !strings.Contains(text, "| "+short+" |") {
			t.Errorf("inventory has no row for divergence %s", d.ID)
		}
	}
}

// TestDocumentedModulePath proves the recorded module path is the one this
// module actually declares, so B1's fallback claim stays true.
func TestDocumentedModulePath(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	var declared string
	for _, line := range strings.Split(string(raw), "\n") {
		if after, ok := strings.CutPrefix(line, "module "); ok {
			declared = strings.TrimSpace(after)
			break
		}
	}
	if declared != audit.ModulePath {
		t.Fatalf("go.mod declares module %q, want %q", declared, audit.ModulePath)
	}
	if !strings.Contains(inventory(t), audit.ModulePath) {
		t.Errorf("inventory does not state the module path %s", audit.ModulePath)
	}
}

// TestNoAcceptanceBoxIsTicked proves the inventory ticks no runtime, component,
// installation or dual-host acceptance item.
func TestNoAcceptanceBoxIsTicked(t *testing.T) {
	text := inventory(t)
	if strings.Contains(text, "- [x]") {
		t.Fatal("inventory ticks an acceptance box; no runtime gate is proven here")
	}
	if !strings.Contains(text, "- [ ] no catalogue exists") {
		t.Error("inventory no longer records the unticked catalogue item")
	}
}
