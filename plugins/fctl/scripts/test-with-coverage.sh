#!/usr/bin/env bash
set -euo pipefail

readonly coverage_profile="${1:?usage: test-with-coverage.sh COVERAGE_PROFILE [MINIMUM_PERCENT]}"
readonly minimum_coverage="${2:-80}"

go test -race -count=1 -covermode=atomic -coverprofile "$coverage_profile" ./...

coverage="$(go tool cover -func="$coverage_profile" | awk '/^total:/ { gsub("%", "", $NF); print $NF }')"
readonly coverage
[[ -n "$coverage" ]] || { printf 'unable to read fctl plugin coverage\n' >&2; exit 1; }

awk -v coverage="$coverage" -v minimum="$minimum_coverage" 'BEGIN {
  if (coverage + 0 < minimum + 0) {
    printf "fctl plugin coverage is %s%%; expected at least %s%%\n", coverage, minimum > "/dev/stderr"
    exit 1
  }
}'
printf 'fctl plugin coverage: %s%% (minimum %s%%)\n' "$coverage" "$minimum_coverage"
