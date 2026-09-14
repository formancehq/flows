// The module path is a subdirectory of the repository root module path
// declared in the root go.mod (github.com/formancehq/orchestration), which is
// this repository's authoritative module namespace regardless of its VCS name.
// It is deliberately not github.com/formancehq/flows/plugins/fctl. The product
// client remains its own generated nested module and is imported as `openapi`
// through the local replacement below.
module github.com/formancehq/orchestration/plugins/fctl

go 1.25.10

require gopkg.in/yaml.v3 v3.0.1

require (
	github.com/formancehq/fctl-v2-poc/pkg/plugin v0.0.0
	go.bytecodealliance.org/pkg v0.2.2
	openapi v0.0.0
)

require (
	github.com/cenkalti/backoff/v4 v4.2.0 // indirect
	github.com/ericlagergren/decimal v0.0.0-20221120152707-495c53812d05 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace openapi => ../../pkg/client
