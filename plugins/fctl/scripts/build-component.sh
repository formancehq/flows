#!/usr/bin/env bash
set -euo pipefail

readonly component_limit_bytes=$((16 * 1024 * 1024))
plugin_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
entrypoint="$plugin_root/entrypoints/flows/implementation.go"
wit_root="$plugin_root/wit"
module_path="github.com/formancehq/orchestration/plugins/fctl"

for tool in cmp go; do
  command -v "$tool" >/dev/null || { printf 'required build tool is unavailable: %s\n' "$tool" >&2; exit 1; }
done
"$plugin_root/scripts/check-component-toolchain.sh"

export GOFLAGS="${GOFLAGS:-} -trimpath -tags=fctl_component_guest"
mkdir -p "$plugin_root/build" "$plugin_root/dist"
staging="$plugin_root/build/component-staging"
rm -rf "$staging"; mkdir "$staging"
cleanup() { rm -rf "$staging"; }; trap cleanup EXIT

build_lane() {
  local lane="$1"
  local lane_root="$staging/source"
  local lane_result="$staging/result-$lane"
  local bindings_path="$module_path/build/component-staging/source/bindings"
  local implementation_dir="$lane_root/bindings/export_formance_fctl_plugin_lifecycle"
  rm -rf "$lane_root"; mkdir -p "$implementation_dir" "$lane_result"
  cp "$entrypoint" "$implementation_dir/implementation.go"
  componentize-go --ignore-toml-files -d "$wit_root" -w plugin bindings --format --pkg-name "$bindings_path" --output "$lane_root/bindings"
  sed "s|github.com/formancehq/orchestration/plugins/fctl/bindings|$bindings_path|g" "$plugin_root/main.go.in" > "$lane_root/main.go"
  "$plugin_root/scripts/check-component-staging.sh" "$staging"
  (
    cd "$lane_root"
    componentize-go --ignore-toml-files -d "$wit_root" -w plugin build --go "$plugin_root/scripts/go-component-build.sh" --output flows.raw.wasm
    wasi-virt --allow-clocks --allow-random --allow-env --stdio=ignore --out flows.wasm flows.raw.wasm
    wasm-tools strip --all --output flows.stripped.wasm flows.wasm
    mv flows.stripped.wasm flows.wasm
    wasm-tools validate flows.wasm
    wasm-tools component wit flows.wasm > flows.wit
    sed -n -E 's/^[[:space:]]*import ([^[:space:];]+).*/\1/p' flows.wit > imports.txt
    cmp "$wit_root/imports.allowlist" imports.txt || {
      printf 'component imports differ from the reviewed allowlist\n' >&2
      exit 1
    }
    size="$(wc -c < flows.wasm | tr -d ' ')"
    [[ "$size" -le "$component_limit_bytes" ]] || { printf 'component exceeds %d-byte admission limit: %d\n' "$component_limit_bytes" "$size" >&2; exit 1; }
    shasum -a 256 flows.wasm flows.wit imports.txt > artifact.sha256
  )
  "$plugin_root/scripts/check-component-staging.sh" "$staging"
  cp "$lane_root/flows.wasm" "$lane_root/flows.wit" "$lane_root/imports.txt" "$lane_root/artifact.sha256" "$lane_result/"
}

build_lane one; build_lane two
for artifact in flows.wasm flows.wit imports.txt artifact.sha256; do cmp "$staging/result-one/$artifact" "$staging/result-two/$artifact"; done
destination="$plugin_root/dist/flows"; rm -rf "$destination"; mkdir "$destination"
install -m 0444 "$staging/result-one/flows.wasm" "$staging/result-one/flows.wit" "$staging/result-one/imports.txt" "$staging/result-one/artifact.sha256" "$destination/"
printf '%s\n' "$destination/flows.wasm"
