#!/usr/bin/env bash
set -euo pipefail

script_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
guard="$script_root/check-component-staging.sh"
fixture_root="$(mktemp -d)"
trap 'rm -rf "$fixture_root"' EXIT

"$guard" "$fixture_root"
mkdir -p "$fixture_root/source/.claude-flow"
if "$guard" "$fixture_root" 2>"$fixture_root/error"; then
  printf '.claude-flow staging content was accepted\n' >&2
  exit 1
fi
grep -q 'forbidden staging path' "$fixture_root/error"
