#!/usr/bin/env bash
set -euo pipefail

readonly plugin_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
readonly materializer="$plugin_root/scripts/materialize-fctl-sdk.sh"
readonly test_root="$(mktemp -d)"
trap 'rm -rf -- "$test_root"' EXIT

fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }
expect_failure() {
  local expected="$1"
  shift
  if "$@" >"$test_root/stdout" 2>"$test_root/stderr"; then
    fail "command unexpectedly succeeded: $*"
  fi
  grep -F "$expected" "$test_root/stderr" >/dev/null || fail "missing diagnostic: $expected"
  [[ ! -s "$test_root/stdout" ]] || fail 'failed materialization emitted a result on stdout'
}

[[ -x "$materializer" ]] || fail "materializer is missing or not executable: $materializer"

upstream="$test_root/upstream"
mkdir -p "$upstream/pkg/plugin" "$upstream/wit/formance/fctl/plugin/v1"
git -C "$upstream" init --quiet --initial-branch=main
git -C "$upstream" config user.email fctl@example.invalid
git -C "$upstream" config user.name 'fctl contract'
printf 'module github.com/formancehq/fctl-v2-poc/pkg/plugin\n\ngo 1.25.0\n' >"$upstream/pkg/plugin/go.mod"
printf 'package sdkfixture\n' >"$upstream/pkg/plugin/fixture.go"
printf 'package formance:fctl-plugin@1.0.0;\n' >"$upstream/wit/formance/fctl/plugin/v1/plugin.wit"
git -C "$upstream" add -A
git -C "$upstream" commit --quiet -m pinned
pinned_commit="$(git -C "$upstream" rev-parse HEAD)"
repository="file://$upstream"
cache="$test_root/cache"

expect_failure 'full 40-character object name' "$materializer" "$repository" main "$cache"
resolved="$("$materializer" "$repository" "$pinned_commit" "$cache")"
[[ "$resolved" == "$cache" ]] || fail "resolved $resolved, want $cache"
git -C "$cache" cat-file -e "$pinned_commit^{commit}" || fail 'cache omits pinned commit'
[[ "$(git -C "$cache" remote get-url origin)" == "$repository" ]] || fail 'cache origin drifted'

# CI authenticates private github.com repositories through url.*.insteadOf.
# That rewrite must affect transport without changing the stored repository
# identity or making the materializer reject its own cache.
git_config="$test_root/gitconfig"
git config --file "$git_config" url.https://credential@example.invalid/.insteadOf "$repository"
rewritten="$(GIT_CONFIG_GLOBAL="$git_config" git -C "$cache" remote get-url origin)"
[[ "$rewritten" != "$repository" ]] || fail 'test did not exercise Git URL rewriting'
GIT_CONFIG_GLOBAL="$git_config" "$materializer" "$repository" "$pinned_commit" "$cache" >/dev/null

# Replaying the same intent converges from cache without its source repository.
mv "$upstream" "$test_root/upstream-offline"
[[ "$("$materializer" "$repository" "$pinned_commit" "$cache")" == "$cache" ]] || fail 'cache replay failed'

wrong_host_cache="$test_root/wrong-host-cache"
git init --quiet --bare "$wrong_host_cache"
git -C "$wrong_host_cache" remote add origin https://example.invalid/formancehq/fctl-v2-poc.git
expect_failure 'cache origin mismatch' "$materializer" "$repository" "$pinned_commit" "$wrong_host_cache"

wrong_path_cache="$test_root/wrong-path-cache"
git init --quiet --bare "$wrong_path_cache"
git -C "$wrong_path_cache" remote add origin "file://$test_root/other-upstream"
expect_failure 'cache origin mismatch' "$materializer" "$repository" "$pinned_commit" "$wrong_path_cache"

printf 'fctl SDK materialization contract: ok\n'
