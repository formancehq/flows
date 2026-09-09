// The module path is a subdirectory of the repository root module path
// declared in the root go.mod (github.com/formancehq/orchestration), which is
// this repository's authoritative module namespace regardless of its VCS name.
// It is deliberately not github.com/formancehq/flows/plugins/fctl: see
// plugins/fctl/audit/blockers.go B1-generated-client-unusable, which records
// the generated client's conflicting paths without repairing them here.
module github.com/formancehq/orchestration/plugins/fctl

go 1.25.10

require gopkg.in/yaml.v3 v3.0.1
