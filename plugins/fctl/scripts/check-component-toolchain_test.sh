#!/usr/bin/env bash
set -euo pipefail

script_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
guard="$script_root/check-component-toolchain.sh"
fixture_root="$(mktemp -d)"
trap 'rm -rf "$fixture_root"' EXIT
fixture_bin="$fixture_root/bin"
mkdir -p "$fixture_bin"

write_tool() {
  local name="$1"
  local version="$2"
  cat > "$fixture_bin/$name" <<EOF
#!/usr/bin/env bash
printf '%s\n' '$version'
EOF
  chmod +x "$fixture_bin/$name"
}

write_expected_tools() {
  write_tool componentize-go 'componentize-go 0.4.1'
  write_tool wasi-virt 'wasi-virt 0.2.0'
  write_tool wasm-tools 'wasm-tools 1.239.0'
  write_tool wasm-opt 'wasm-opt version 124'
}

write_expected_tools
PATH="$fixture_bin:$PATH" "$guard"

for tool in componentize-go wasi-virt wasm-tools wasm-opt; do
  write_expected_tools
  write_tool "$tool" "$tool unexpected-version"
  if PATH="$fixture_bin:$PATH" "$guard" 2>"$fixture_root/$tool.err"; then
    printf '%s version drift was accepted\n' "$tool" >&2
    exit 1
  fi
  grep -q 'version mismatch' "$fixture_root/$tool.err"
done
