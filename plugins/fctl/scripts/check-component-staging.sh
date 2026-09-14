#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 1 || ! -d "$1" ]]; then
  printf 'component staging guard requires one existing directory\n' >&2
  exit 1
fi

staging_root="$1"
forbidden="$(find "$staging_root" -name .claude-flow -print -quit)"
if [[ -n "$forbidden" ]]; then
  printf 'forbidden staging path: %s\n' "$forbidden" >&2
  exit 1
fi
