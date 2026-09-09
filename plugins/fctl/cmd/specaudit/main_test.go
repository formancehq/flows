package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// specPath is the merged document at the repository root, from this package's
// directory.
const specPath = "../../../../openapi.yaml"

// stage creates an empty artefact tree and returns its root.
func stage(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{filepath.Join("audit", "testdata"), "docs"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("stage %s: %v", dir, err)
		}
	}
	return root
}

// TestRunWritesBothArtefacts proves the generator writes exactly the two
// committed artefacts.
func TestRunWritesBothArtefacts(t *testing.T) {
	root := stage(t)

	if err := run(specPath, root, false); err != nil {
		t.Fatalf("run: %v", err)
	}

	for _, rel := range []string{
		filepath.Join("audit", "testdata", "report.json"),
		filepath.Join("docs", "operations.generated.md"),
	} {
		info, err := os.Stat(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("stat %s: %v", rel, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", rel)
		}
	}
}

// TestRunCheckPassesOnFreshOutput proves -check accepts what the generator just
// wrote, which is the property `just fctl-audit-check` relies on.
func TestRunCheckPassesOnFreshOutput(t *testing.T) {
	root := stage(t)

	if err := run(specPath, root, false); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := run(specPath, root, true); err != nil {
		t.Fatalf("check after write: %v", err)
	}
}

// TestRunCheckFailsOnStaleOutput proves the gate actually fails when an
// artefact drifts.
func TestRunCheckFailsOnStaleOutput(t *testing.T) {
	root := stage(t)

	if err := run(specPath, root, false); err != nil {
		t.Fatalf("write: %v", err)
	}
	stale := filepath.Join(root, "docs", "operations.generated.md")
	if err := os.WriteFile(stale, []byte("stale\n"), 0o600); err != nil {
		t.Fatalf("corrupt artefact: %v", err)
	}

	err := run(specPath, root, true)
	if err == nil {
		t.Fatal("expected -check to fail on a stale artefact")
	}
	if !strings.Contains(err.Error(), "out of date") {
		t.Fatalf("expected an out-of-date error, got %v", err)
	}
}

// TestRunCheckFailsOnMissingOutput proves -check does not treat an absent
// artefact as up to date.
func TestRunCheckFailsOnMissingOutput(t *testing.T) {
	root := stage(t)

	if err := run(specPath, root, true); err == nil {
		t.Fatal("expected -check to fail when the artefacts do not exist")
	}
}

// TestRunRejectsMissingSpec covers the load error path.
func TestRunRejectsMissingSpec(t *testing.T) {
	root := stage(t)

	if err := run(filepath.Join(root, "absent.yaml"), root, false); err == nil {
		t.Fatal("expected an error for a missing document")
	}
}

// TestRunRejectsUnwritableOutput covers the write error path.
func TestRunRejectsUnwritableOutput(t *testing.T) {
	root := t.TempDir() // no audit/testdata or docs subtree

	if err := run(specPath, root, false); err == nil {
		t.Fatal("expected an error when the artefact directories do not exist")
	}
}
