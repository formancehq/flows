#!/usr/bin/env bash
set -euo pipefail

readonly script_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
readonly test_root="$(mktemp -d "${TMPDIR:-/tmp}/flows-coverage-test.XXXXXXXX")"
trap 'rm -rf "$test_root"' EXIT

mkdir -p "$test_root/bin"
cat >"$test_root/bin/go" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" == "test" ]]; then
  exit 0
fi
if [[ "$1" == "tool" && "$2" == "cover" ]]; then
  printf 'total:\t\t(statements)\t21204\t%s%%\n' "${FAKE_COVERAGE:-80.2}"
  exit 0
fi
printf 'unexpected go arguments: %s\n' "$*" >&2
exit 97
EOF
chmod +x "$test_root/bin/go"

output="$(PATH="$test_root/bin:$PATH" bash "$script_root/test-with-coverage.sh" "$test_root/coverage.out")"
[[ "$output" == "fctl plugin coverage: 80.2% (minimum 80%)" ]] || {
  printf 'unexpected passing output: %s\n' "$output" >&2
  exit 1
}

if PATH="$test_root/bin:$PATH" bash "$script_root/test-with-coverage.sh" "$test_root/coverage.out" 81 \
  >"$test_root/stdout" 2>"$test_root/stderr"; then
  printf 'coverage below the threshold unexpectedly passed\n' >&2
  exit 1
fi
grep -F 'fctl plugin coverage is 80.2%; expected at least 81%' "$test_root/stderr" >/dev/null || {
  printf 'missing below-threshold diagnostic\n' >&2
  exit 1
}

printf 'fctl plugin coverage gate: ok\n'
