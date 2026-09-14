package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"regexp"
	"testing"
)

// restatement is one tracked place that repeats part of the fctl SDK lock.
type restatement struct {
	// path is relative to this directory.
	path string
	// anchor captures exactly one value in its first group. It must be
	// specific enough that a superseded value cannot survive beside the
	// current one, because a repin edits these files by hand.
	anchor *regexp.Regexp
	// want reads the authoritative value out of the lock.
	want func(sdkLock) string
}

// The lock is the single source of truth for the fctl SDK contract, but four
// other tracked files restate part of it: the Nix tool pin, the wrapper
// contract test's fixtures, and two documents. Repinning the SDK used to mean
// editing all of them by hand with nothing to catch a miss or a leftover, so
// every restatement is pinned here instead.
var restatements = []restatement{
	{
		path:   "../../../nix/fctl-component-tools.nix",
		anchor: regexp.MustCompile(`fctlSDKRevision = "([0-9a-f]{40})";`),
		want:   func(l sdkLock) string { return l.Commit },
	},
	{
		path:   "test-fctl-sdk-contract.sh",
		anchor: regexp.MustCompile(`readonly expected_commit='([0-9a-f]{40})'`),
		want:   func(l sdkLock) string { return l.Commit },
	},
	{
		path:   "test-fctl-sdk-contract.sh",
		anchor: regexp.MustCompile(`readonly expected_repository='(\S+)'`),
		want:   func(l sdkLock) string { return l.Repository },
	},
	{
		path:   "test-fctl-sdk-contract.sh",
		anchor: regexp.MustCompile(`readonly expected_nar_hash='(\S+)'`),
		want:   func(l sdkLock) string { return l.SDKNarHash },
	},
	{
		path:   "test-fctl-sdk-contract.sh",
		anchor: regexp.MustCompile(`readonly expected_wit_hash='([0-9a-f]{64})'`),
		want:   func(l sdkLock) string { return l.WITSHA256 },
	},
	{
		path:   "../README.md",
		anchor: regexp.MustCompile("at fctl revision\n`([0-9a-f]{40})`"),
		want:   func(l sdkLock) string { return l.Commit },
	},
	{
		path:   "../docs/command-inventory.md",
		anchor: regexp.MustCompile("\\| fctl plugin SDK \\| `([0-9a-f]{40})`"),
		want:   func(l sdkLock) string { return l.Commit },
	},
}

func TestEveryRestatedSDKFactMatchesTheLock(t *testing.T) {
	lock := readLock(t)
	for _, r := range restatements {
		t.Run(r.path+"/"+r.anchor.String(), func(t *testing.T) {
			matches := r.anchor.FindAllStringSubmatch(read(t, r.path), -1)
			if len(matches) != 1 {
				t.Fatalf("%s states %d values for %s; want exactly one",
					r.path, len(matches), r.anchor)
			}
			if got, want := matches[0][1], r.want(lock); got != want {
				t.Errorf("%s restates %q, but the lock pins %q", r.path, got, want)
			}
		})
	}
}

// TestVendoredWITMatchesLock proves the WIT copied into this repository is the
// exact SDK interface the lock pins, without needing an SDK checkout.
func TestVendoredWITMatchesLock(t *testing.T) {
	lock := readLock(t)
	sum := sha256.Sum256([]byte(read(t, "../wit/plugin.wit")))
	if got := hex.EncodeToString(sum[:]); got != lock.WITSHA256 {
		t.Fatalf("vendored wit/plugin.wit hashes to %s, but the lock pins %s", got, lock.WITSHA256)
	}
}

func readLock(t *testing.T) sdkLock {
	t.Helper()
	f, err := os.Open("../fctl-sdk.lock.json")
	if err != nil {
		t.Fatalf("open lock: %v", err)
	}
	defer f.Close()
	lock, err := decodeLock(f)
	if err != nil {
		t.Fatalf("decode lock: %v", err)
	}
	return lock
}

func read(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}
