set dotenv-load

default:
  @just --list

pre-commit: generate tidy lint openapi
pc: pre-commit

lint:
  @golangci-lint run --fix --build-tags it --timeout 5m

tidy:
  @go mod tidy

generate:
  @go generate ./...

tests:
  @go test -race -covermode=atomic \
    -coverprofile coverage.txt \
    -tags it \
    ./...

openapi:
  @yq eval-all '. as $item ireduce ({}; . * $item)' openapi/v1.yaml openapi/v2.yaml openapi/overlay.yaml > openapi.yaml

generate-client:
  @speakeasy generate sdk -s openapi.yaml -o ./pkg/client -l go

release-local:
  @goreleaser release --nightly --skip=publish --clean

release-ci:
  @goreleaser release --nightly --clean

release:
  @goreleaser release --clean
