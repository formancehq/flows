set dotenv-load

default:
  @just --list

# fctl-audit-check runs last so it validates the freshly regenerated openapi.yaml
pre-commit: generate tidy lint openapi generated-client-regeneration-check fctl-sdk-check fctl-plugin-test fctl-audit-check
pc: pre-commit

lint:
  @golangci-lint run --fix --build-tags it --timeout 5m
  @cd plugins/fctl && ./scripts/with-fctl-sdk.sh golangci-lint run --fix --timeout 5m

tidy: && fctl-plugin-tidy
  @go mod tidy

fctl-plugin-tidy:
  @cd plugins/fctl && ./scripts/with-fctl-sdk.sh ./scripts/tidy-with-fctl-sdk.sh

fctl-plugin-tidy-check:
  @cd plugins/fctl && ./scripts/with-fctl-sdk.sh ./scripts/tidy-with-fctl-sdk.sh --check

generate:
  @go generate ./...

tests:
  @go test -race -covermode=atomic \
    -coverprofile coverage.txt \
    -tags it \
    ./...
  @just _fctl-plugin-test "{{justfile_directory()}}/coverage-fctl.txt"
  @tail -n +2 coverage-fctl.txt >> coverage.txt && rm coverage-fctl.txt

# The plugin is a nested Go module, so root tests do not load it. Keep its race
# suite and the repository-wide 80% module threshold in every pre-commit run.
fctl-plugin-test:
  @coverage_profile="$(mktemp "${TMPDIR:-/tmp}/flows-fctl-coverage.XXXXXXXX")"; \
    trap 'rm -f "$coverage_profile"' EXIT; \
    just _fctl-plugin-test "$coverage_profile"

[private]
_fctl-plugin-test coverage_profile:
  @bash plugins/fctl/scripts/test-with-coverage_test.sh
  @cd plugins/fctl && ./scripts/with-fctl-sdk.sh bash ./scripts/test-with-coverage.sh "{{coverage_profile}}"

openapi:
  @yq eval-all '. as $item ireduce ({}; . * $item)' openapi/v1.yaml openapi/v2.yaml openapi/overlay.yaml > openapi.yaml

# These artefacts are the plugin's source of truth for the operation set, the
# legacy command mapping and the recorded blockers: always review the diff.
# Regenerates the committed fctl Flows plugin operation inventory
fctl-audit:
  @cd plugins/fctl && ./scripts/with-fctl-sdk.sh go run ./cmd/specaudit -spec ../../openapi.yaml -out .

# Fails when the committed inventory no longer matches openapi.yaml.
fctl-audit-check:
  @cd plugins/fctl && ./scripts/with-fctl-sdk.sh go run ./cmd/specaudit -spec ../../openapi.yaml -out . -check

fctl-sdk-check:
  @cd plugins/fctl && just validate-sdk

generate-client:
  @nix run .#speakeasy -- generate sdk -y -s openapi.yaml -o ./pkg/client -l go
  @cd pkg/client && go mod tidy
  @just generated-client-check

# The generated client is a nested module, so the root Go commands do not load it.
# Keep an explicit gate here and after every regeneration.
generated-client-check:
  @cd pkg/client && go mod tidy -diff
  @cd pkg/client && go list ./...
  @cd pkg/client && go vet ./...

# Re-run the exact Nix-pinned generator in isolation and require byte-for-byte
# equality with the checked-in client, including generated docs and lock data.
generated-client-regeneration-check:
  @client_project_dir="$(pwd)"; \
    client_check_dir="$(mktemp -d)"; \
    trap 'rm -rf "$client_check_dir"' EXIT; \
    cp -R pkg/client "$client_check_dir/client"; \
    nix run .#speakeasy -- generate sdk -y -s "$client_project_dir/openapi.yaml" -o "$client_check_dir/client" -l go; \
    cd "$client_check_dir/client"; \
    go mod tidy; \
    cd "$client_project_dir"; \
    diff -ru pkg/client "$client_check_dir/client"
  @just generated-client-check

release-local:
  @goreleaser release --nightly --skip=publish --clean

release-ci:
  @goreleaser release --nightly --clean

release:
  @goreleaser release --clean
