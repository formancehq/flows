package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

// trackedPaths are every tracked file that repeats part of the fctl SDK lock,
// relative to this directory. The lock itself is scanned too, so a value edited
// into it out of band is still held to its own fields.
var trackedPaths = []string{
	"../../../nix/fctl-component-tools.nix",
	"test-fctl-sdk-contract.sh",
	"../README.md",
	"../docs/command-inventory.md",
	"../fctl-sdk.lock.json",
}

// restatement is one tracked place that repeats part of the fctl SDK lock.
type restatement struct {
	// path is relative to this directory.
	path string
	// anchor captures exactly one value in its first group. It must be
	// specific enough that a superseded value cannot survive beside the
	// current one, because a repin edits these files by hand. Anchors tolerate
	// any whitespace between their literal words so that reflowing a paragraph
	// or a table does not read as a stale pin.
	anchor *regexp.Regexp
	// want reads the authoritative value out of the lock.
	want func(sdkLock) string
}

// The lock is the single source of truth for the fctl SDK contract, but four
// other tracked files restate part of it across eight anchored sites: the Nix
// tool pin, the wrapper contract test's four fixtures, the README's revision
// and remote, and the inventory's revision. Repinning the SDK used to mean
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
		anchor: regexp.MustCompile("at\\s+fctl\\s+revision\\s+`([0-9a-f]{40})`"),
		want:   func(l sdkLock) string { return l.Commit },
	},
	{
		path:   "../README.md",
		anchor: regexp.MustCompile("at\\s+fctl\\s+revision\\s+`[0-9a-f]{40}`\\s+of\\s+`(\\S+?)`"),
		want:   func(l sdkLock) string { return l.Repository },
	},
	{
		path:   "../docs/command-inventory.md",
		anchor: regexp.MustCompile("\\|\\s*fctl\\s+plugin\\s+SDK\\s*\\|\\s*`([0-9a-f]{40})`"),
		want:   func(l sdkLock) string { return l.Commit },
	},
}

// unrelatedRevisions are the 40-hex revisions a tracked file legitimately
// carries that are not the fctl SDK pin. Anything else is treated as a stale
// SDK revision, so adding one is a decision, not an oversight.
var unrelatedRevisions = map[string]map[string]string{
	"test-fctl-sdk-contract.sh": {
		"0000000000000000000000000000000000000000": "negative fixture: a checkout revision the wrapper must reject",
	},
	"../../../nix/fctl-component-tools.nix": {
		"448f6df8f688cee5d6995e96b1ffc31f9bf00742": "WASI-Virt upstream revision, copied from the fctl authoring toolchain",
	},
	"../docs/command-inventory.md": {
		"9dc85b316cbfc2485dd8e176b7caa17b73e04e0d": "this repository's pinned origin/main",
		"693c58e27865f83332e6c3199d61fed81b742f41": "legacy fctl command-surface baseline",
	},
}

// unrelatedContentHashes are the SRI content hashes a tracked file legitimately
// carries that are not the SDK content hash.
var unrelatedContentHashes = map[string]map[string]string{
	"test-fctl-sdk-contract.sh": {
		"sha256-wrong": "negative fixture: a content hash the wrapper must reject",
	},
	"../../../nix/fctl-component-tools.nix": {
		"sha256-9wJSNC/clO8M7E840i1lRWJQT7AdRQX468swmZ4O1rg=": "wasm-tools source",
		"sha256-xIfTYJMVP47timzquEYEb9M8BHsj83NjgD44lbzgd+Y=": "wasm-tools cargo vendor",
		"sha256-Kgh17i2vnCAfGaBTNmVafYsG+WJ4OatEdOmrCwrRUs4=": "componentize-go source",
		"sha256-DoeHrw1+LI3aLhSGMFrUH347nZV2ftykRQ/JUSaERlA=": "componentize-go cargo vendor",
		"sha256-g6g1gT7cj8U740ew6c+GT3ZVOENU4wqFf10oHni8MOs=": "WASI-Virt source",
		"sha256-P39bkgjlqy8/L7iSA7/hQDOfa3WUF+YV3DlcoGZ73jo=": "WASI-Virt cargo vendor",
	},
}

var (
	// alphanumericWord matches a maximal run of letters and digits, so a hex
	// value is only classified when the whole word is that value. This keeps
	// the 40-hex rule from firing on the first 40 characters of a 64-hex hash
	// or on a hex-looking run inside a base64 payload.
	alphanumericWord = regexp.MustCompile(`[0-9A-Za-z]+`)
	revisionWord     = regexp.MustCompile(`^[0-9a-f]{40}$`)
	witHashWord      = regexp.MustCompile(`^[0-9a-f]{64}$`)
	contentHash      = regexp.MustCompile(`sha256-[A-Za-z0-9+/]+=*`)
	sdkReference     = regexp.MustCompile(`github\.com/([A-Za-z0-9_.-]+)/fctl-v2-poc([A-Za-z0-9_./-]*)`)
)

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

// TestNoSupersededSDKValueSurvivesInATrackedFile is the anchors' complement.
// An anchor constrains one phrasing at one site; this walks every SDK-shaped
// value in every tracked file regardless of phrasing or position, so a stale
// revision, WIT hash, content hash, remote owner or module path cannot survive
// in prose, in a comment or beside a fresh value.
func TestNoSupersededSDKValueSurvivesInATrackedFile(t *testing.T) {
	lock := readLock(t)
	owner, err := repositoryOwner(lock.Repository)
	if err != nil {
		t.Fatalf("lock repository %q: %v", lock.Repository, err)
	}
	for _, path := range trackedPaths {
		t.Run(path, func(t *testing.T) {
			content := read(t, path)

			for _, word := range alphanumericWord.FindAllString(content, -1) {
				switch {
				case revisionWord.MatchString(word):
					if word == lock.Commit {
						continue
					}
					if reason, allowed := unrelatedRevisions[path][word]; allowed {
						t.Logf("%s carries unrelated revision %s (%s)", path, word, reason)
						continue
					}
					t.Errorf("%s carries revision %s, but the lock pins %s and records no reason for it",
						path, word, lock.Commit)
				case witHashWord.MatchString(word):
					if word != lock.WITSHA256 {
						t.Errorf("%s carries WIT hash %s, but the lock pins %s", path, word, lock.WITSHA256)
					}
				}
			}

			for _, hash := range contentHash.FindAllString(content, -1) {
				if hash == lock.SDKNarHash {
					continue
				}
				if reason, allowed := unrelatedContentHashes[path][hash]; allowed {
					t.Logf("%s carries unrelated content hash %s (%s)", path, hash, reason)
					continue
				}
				t.Errorf("%s carries content hash %s, but the lock pins %s and records no reason for it",
					path, hash, lock.SDKNarHash)
			}

			for _, reference := range sdkReference.FindAllStringSubmatch(content, -1) {
				if reference[1] != owner {
					t.Errorf("%s names the SDK under %q, but the lock pins the remote %q",
						path, reference[0], lock.Repository)
					continue
				}
				if reference[2] == "/pkg/plugin" && reference[0] != lock.ModulePath {
					t.Errorf("%s names the SDK module %q, but the lock pins %q",
						path, reference[0], lock.ModulePath)
				}
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

// repositoryOwner is the account segment of an https remote, which is the field
// the superseded pin actually got wrong.
func repositoryOwner(repository string) (string, error) {
	rest, ok := strings.CutPrefix(repository, "https://github.com/")
	if !ok {
		return "", fmt.Errorf("not an https github remote")
	}
	owner, _, ok := strings.Cut(rest, "/")
	if !ok || owner == "" {
		return "", fmt.Errorf("has no owner segment")
	}
	return owner, nil
}

func readLock(t *testing.T) sdkLock {
	t.Helper()
	f, err := os.Open("../fctl-sdk.lock.json")
	if err != nil {
		t.Fatalf("open lock: %v", err)
	}
	lock, err := decodeLock(f)
	if err != nil {
		t.Fatalf("decode lock: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close lock: %v", err)
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
