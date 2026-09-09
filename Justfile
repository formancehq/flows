set dotenv-load

default:
  @just --list

# fctl-audit-check runs last so it validates the freshly regenerated openapi.yaml
pre-commit: generate tidy lint openapi fctl-audit-check
pc: pre-commit

lint:
  @golangci-lint run --fix --build-tags it --timeout 5m
  @cd plugins/fctl && golangci-lint run --fix --timeout 5m

tidy:
  @go mod tidy
  @cd plugins/fctl && go mod tidy

generate:
  @go generate ./...

tests:
  @go test -race -covermode=atomic \
    -coverprofile coverage.txt \
    -tags it \
    ./...
  @cd plugins/fctl && go test -race -covermode=atomic -coverprofile ../../coverage-fctl.txt ./...
  @tail -n +2 coverage-fctl.txt >> coverage.txt && rm coverage-fctl.txt

openapi:
  @yq eval-all '. as $item ireduce ({}; . * $item)' openapi/v1.yaml openapi/v2.yaml openapi/overlay.yaml > openapi.yaml

# These artefacts are the plugin's source of truth for the operation set, the
# legacy command mapping and the recorded blockers: always review the diff.
# Regenerates the committed fctl Flows plugin operation inventory
fctl-audit:
  @cd plugins/fctl && go run ./cmd/specaudit -spec ../../openapi.yaml -out .

# Fails when the committed inventory no longer matches openapi.yaml.
fctl-audit-check:
  @cd plugins/fctl && go run ./cmd/specaudit -spec ../../openapi.yaml -out . -check

generate-client:
  @speakeasy generate sdk -s openapi.yaml -o ./pkg/client -l go

release-local:
  @goreleaser release --nightly --skip=publish --clean

release-ci:
  @goreleaser release --nightly --clean

release:
  @goreleaser release --clean
