// Package audit extracts a deterministic, auditable inventory of the Flows
// (orchestration) HTTP surface from this repository's own merged OpenAPI
// document, and pins the legacy fctl command baseline it has to be measured
// against.
//
// The package is deliberately read-only and dependency-light: it is the
// evidence layer for the fctl Flows plugin and stays independent from the
// runtime, component ABI and transport implementation.
//
// It also does not import the generated client: the audit remains an
// independent OpenAPI oracle for adapter/catalogue contract tests.
package audit

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Tag values used by the merged orchestration OpenAPI document. Every
// operation carries exactly one of them, which is how the document separates
// the unprefixed v1 API from the /v2 API.
const (
	TagV1 = "orchestration.v1"
	TagV2 = "orchestration.v2"
)

// Version is the API major an operation belongs to, derived from its tag.
type Version string

const (
	V1 Version = "v1"
	V2 Version = "v2"
)

// versionOfTag maps a document tag onto an API major. Any other tag is an
// error rather than a default, so a new tag fails extraction instead of being
// silently absorbed.
func versionOfTag(tag string) (Version, error) {
	switch tag {
	case TagV1:
		return V1, nil
	case TagV2:
		return V2, nil
	default:
		return "", fmt.Errorf("unknown tag %q", tag)
	}
}

// methodsOf returns the path item's declared operations in a fixed method
// order, so extraction output is stable for a given document.
func methodsOf(item yamlPathItem) []struct {
	method string
	op     *yamlOperation
} {
	return []struct {
		method string
		op     *yamlOperation
	}{
		{"GET", item.Get},
		{"PUT", item.Put},
		{"POST", item.Post},
		{"DELETE", item.Delete},
		{"PATCH", item.Patch},
		{"HEAD", item.Head},
		{"OPTIONS", item.Options},
		{"TRACE", item.Trace},
	}
}

// Parameter is one resolved OpenAPI parameter of an operation.
type Parameter struct {
	Name     string `json:"name" yaml:"name"`
	In       string `json:"in" yaml:"in"`
	Required bool   `json:"required" yaml:"required"`
}

// Operation is one (method, path) OpenAPI operation of the orchestration API.
//
// Every field is read from the document; nothing is inferred. In particular
// Scopes is nil when the operation declares no security block at all, which is
// a different, weaker fact than an empty declared scope array.
type Operation struct {
	OperationID string      `json:"operationId"`
	Method      string      `json:"method"`
	Path        string      `json:"path"`
	Tag         string      `json:"tag"`
	Version     Version     `json:"version"`
	SDKMethod   string      `json:"sdkMethod"`
	Deprecated  bool        `json:"deprecated"`
	HasSecurity bool        `json:"hasSecurity"`
	Scopes      []string    `json:"scopes"`
	Parameters  []Parameter `json:"parameters"`
	RequestBody string      `json:"requestBody"`
	SuccessCode string      `json:"successCode"`
	SuccessBody string      `json:"successBody"`
}

// Paginated reports whether the operation declares the shared cursor
// pagination parameters. Both are always declared together in this document;
// the check requires both so a single stray name cannot fake pagination.
//
// This is a statement about the document only. Whether the server honours it,
// and whether the response carries the cursor back, is recorded separately in
// blockers.go.
func (o Operation) Paginated() bool {
	var cursor, pageSize bool
	for _, p := range o.Parameters {
		switch p.Name {
		case "cursor":
			cursor = true
		case "pageSize":
			pageSize = true
		}
	}
	return cursor && pageSize
}

// HasRequestBody reports whether the operation declares a JSON request body.
func (o Operation) HasRequestBody() bool { return o.RequestBody != "" }

// HasQueryParam reports whether the operation declares the named query
// parameter.
func (o Operation) HasQueryParam(name string) bool {
	for _, p := range o.Parameters {
		if p.In == "query" && p.Name == name {
			return true
		}
	}
	return false
}

// Mutating reports whether the operation uses a state-changing HTTP method.
func (o Operation) Mutating() bool {
	switch o.Method {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

// yaml shapes: only the fields this audit reads are modelled.

type yamlSchema struct {
	Ref string `yaml:"$ref"`
}

type yamlMediaType struct {
	Schema yamlSchema `yaml:"schema"`
}

type yamlBody struct {
	Ref     string                   `yaml:"$ref"`
	Content map[string]yamlMediaType `yaml:"content"`
}

type yamlResponse struct {
	Ref     string                   `yaml:"$ref"`
	Content map[string]yamlMediaType `yaml:"content"`
}

type yamlParameter struct {
	Ref      string `yaml:"$ref"`
	Name     string `yaml:"name"`
	In       string `yaml:"in"`
	Required bool   `yaml:"required"`
}

type yamlOperation struct {
	OperationID  string                  `yaml:"operationId"`
	Tags         []string                `yaml:"tags"`
	Deprecated   bool                    `yaml:"deprecated"`
	NameOverride string                  `yaml:"x-speakeasy-name-override"`
	Security     []map[string][]string   `yaml:"security"`
	Parameters   []yamlParameter         `yaml:"parameters"`
	RequestBody  *yamlBody               `yaml:"requestBody"`
	Responses    map[string]yamlResponse `yaml:"responses"`
}

type yamlPathItem struct {
	Parameters []yamlParameter `yaml:"parameters"`
	Get        *yamlOperation  `yaml:"get"`
	Put        *yamlOperation  `yaml:"put"`
	Post       *yamlOperation  `yaml:"post"`
	Delete     *yamlOperation  `yaml:"delete"`
	Patch      *yamlOperation  `yaml:"patch"`
	Head       *yamlOperation  `yaml:"head"`
	Options    *yamlOperation  `yaml:"options"`
	Trace      *yamlOperation  `yaml:"trace"`
}

type yamlSecurityScheme struct {
	Type  string `yaml:"type"`
	Flows map[string]struct {
		TokenURL string            `yaml:"tokenUrl"`
		Scopes   map[string]string `yaml:"scopes"`
	} `yaml:"flows"`
}

type yamlDocument struct {
	Paths      map[string]yamlPathItem `yaml:"paths"`
	Components struct {
		Parameters      map[string]yamlParameter      `yaml:"parameters"`
		RequestBodies   map[string]yamlBody           `yaml:"requestBodies"`
		Responses       map[string]yamlResponse       `yaml:"responses"`
		SecuritySchemes map[string]yamlSecurityScheme `yaml:"securitySchemes"`
		Schemas         map[string]any                `yaml:"schemas"`
	} `yaml:"components"`
}

// LoadOperations parses the merged orchestration OpenAPI document at specPath
// and returns every operation it declares, sorted by (version, operationId) so
// the result is byte-stable for a given document.
func LoadOperations(specPath string) ([]Operation, error) {
	doc, err := loadDocument(specPath)
	if err != nil {
		return nil, err
	}

	var ops []Operation
	for path, item := range doc.Paths {
		for _, entry := range methodsOf(item) {
			if entry.op == nil || entry.op.OperationID == "" {
				continue
			}
			op, err := convert(doc, path, entry.method, item.Parameters, *entry.op)
			if err != nil {
				return nil, err
			}
			ops = append(ops, op)
		}
	}

	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Version != ops[j].Version {
			return ops[i].Version < ops[j].Version
		}
		return ops[i].OperationID < ops[j].OperationID
	})
	return ops, nil
}

// DeclaredSecuritySchemeScopes returns the scope names the named security
// scheme declares in its clientCredentials flow, sorted.
//
// It exists so the audit can state, rather than assume, that the scheme
// declares no scopes while the operations reference two: see Divergence
// D3-undeclared-scheme-scopes.
func DeclaredSecuritySchemeScopes(specPath, scheme string) ([]string, error) {
	doc, err := loadDocument(specPath)
	if err != nil {
		return nil, err
	}
	target, ok := doc.Components.SecuritySchemes[scheme]
	if !ok {
		return nil, fmt.Errorf("security scheme %q is not declared", scheme)
	}
	flow, ok := target.Flows["clientCredentials"]
	if !ok {
		return nil, fmt.Errorf("security scheme %q declares no clientCredentials flow", scheme)
	}
	out := make([]string, 0, len(flow.Scopes))
	for name := range flow.Scopes {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

// SchemaPropertyNames returns every property name declared anywhere under
// components.schemas, at any nesting depth, sorted and de-duplicated.
//
// It exists so the audit can state, rather than assume, that no schema in this
// document declares a credential-shaped property: see the empty secretBearing
// table in classify.go.
func SchemaPropertyNames(specPath string) ([]string, error) {
	doc, err := loadDocument(specPath)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	collectPropertyNames(doc.Components.Schemas, seen)

	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

// collectPropertyNames walks a decoded YAML value and records the keys of every
// mapping that appears under a "properties" key.
func collectPropertyNames(node any, into map[string]struct{}) {
	switch v := node.(type) {
	case map[string]any:
		for key, child := range v {
			if key == "properties" {
				if props, ok := child.(map[string]any); ok {
					for name := range props {
						into[name] = struct{}{}
					}
				}
			}
			collectPropertyNames(child, into)
		}
	case []any:
		for _, child := range v {
			collectPropertyNames(child, into)
		}
	}
}

func loadDocument(specPath string) (*yamlDocument, error) {
	raw, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("read openapi document: %w", err)
	}
	var doc yamlDocument
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse openapi document: %w", err)
	}
	return &doc, nil
}

func convert(doc *yamlDocument, path, method string, shared []yamlParameter, raw yamlOperation) (Operation, error) {
	op := Operation{
		OperationID: raw.OperationID,
		Method:      method,
		Path:        path,
		Deprecated:  raw.Deprecated,
		SDKMethod:   raw.NameOverride,
	}
	if op.SDKMethod == "" {
		op.SDKMethod = raw.OperationID
	}
	if len(raw.Tags) != 1 {
		return Operation{}, fmt.Errorf("operation %s: expected exactly one tag, got %v", raw.OperationID, raw.Tags)
	}
	op.Tag = raw.Tags[0]
	version, err := versionOfTag(op.Tag)
	if err != nil {
		return Operation{}, fmt.Errorf("operation %s: %w", raw.OperationID, err)
	}
	op.Version = version

	if raw.Security != nil {
		op.HasSecurity = true
		op.Scopes = []string{}
		for _, scheme := range raw.Security {
			for _, scopes := range scheme {
				op.Scopes = append(op.Scopes, scopes...)
			}
		}
		sort.Strings(op.Scopes)
	}

	for _, group := range [][]yamlParameter{shared, raw.Parameters} {
		for _, p := range group {
			resolved, err := resolveParameter(doc, p)
			if err != nil {
				return Operation{}, fmt.Errorf("operation %s: %w", raw.OperationID, err)
			}
			op.Parameters = append(op.Parameters, resolved)
		}
	}
	sort.Slice(op.Parameters, func(i, j int) bool {
		if op.Parameters[i].In != op.Parameters[j].In {
			return op.Parameters[i].In < op.Parameters[j].In
		}
		return op.Parameters[i].Name < op.Parameters[j].Name
	})

	if raw.RequestBody != nil {
		body, err := resolveBody(doc, *raw.RequestBody)
		if err != nil {
			return Operation{}, fmt.Errorf("operation %s: %w", raw.OperationID, err)
		}
		op.RequestBody = schemaName(body.Content)
	}

	code, response, err := successResponse(doc, raw.Responses)
	if err != nil {
		return Operation{}, fmt.Errorf("operation %s: %w", raw.OperationID, err)
	}
	op.SuccessCode = code
	op.SuccessBody = schemaName(response.Content)

	return op, nil
}

func resolveParameter(doc *yamlDocument, p yamlParameter) (Parameter, error) {
	if p.Ref != "" {
		name, err := refName(p.Ref, "parameters")
		if err != nil {
			return Parameter{}, err
		}
		target, ok := doc.Components.Parameters[name]
		if !ok {
			return Parameter{}, fmt.Errorf("unresolved parameter ref %q", p.Ref)
		}
		p = target
	}
	return Parameter{Name: p.Name, In: p.In, Required: p.Required}, nil
}

func resolveBody(doc *yamlDocument, b yamlBody) (yamlBody, error) {
	if b.Ref == "" {
		return b, nil
	}
	name, err := refName(b.Ref, "requestBodies")
	if err != nil {
		return yamlBody{}, err
	}
	target, ok := doc.Components.RequestBodies[name]
	if !ok {
		return yamlBody{}, fmt.Errorf("unresolved requestBody ref %q", b.Ref)
	}
	return target, nil
}

func resolveResponse(doc *yamlDocument, r yamlResponse) (yamlResponse, error) {
	if r.Ref == "" {
		return r, nil
	}
	name, err := refName(r.Ref, "responses")
	if err != nil {
		return yamlResponse{}, err
	}
	target, ok := doc.Components.Responses[name]
	if !ok {
		return yamlResponse{}, fmt.Errorf("unresolved response ref %q", r.Ref)
	}
	return target, nil
}

// successResponse returns the single declared 2xx response. The orchestration
// document declares exactly one success code per operation plus "default";
// anything else is a shape this audit refuses to guess about.
func successResponse(doc *yamlDocument, responses map[string]yamlResponse) (string, yamlResponse, error) {
	var codes []string
	for code := range responses {
		if strings.HasPrefix(code, "2") {
			codes = append(codes, code)
		}
	}
	if len(codes) != 1 {
		return "", yamlResponse{}, fmt.Errorf("expected exactly one 2xx response, got %v", codes)
	}
	resolved, err := resolveResponse(doc, responses[codes[0]])
	if err != nil {
		return "", yamlResponse{}, err
	}
	return codes[0], resolved, nil
}

// InlineSchema is the placeholder recorded for a declared JSON payload that is
// spelled out in place rather than referenced from components.schemas. It is
// distinct from the empty string, which means no JSON payload is declared at
// all: sendEvent declares an inline body, deleteTrigger declares none.
const InlineSchema = "(inline object)"

func schemaName(content map[string]yamlMediaType) string {
	media, ok := content["application/json"]
	if !ok {
		return ""
	}
	if media.Schema.Ref == "" {
		return InlineSchema
	}
	name, err := refName(media.Schema.Ref, "schemas")
	if err != nil {
		return InlineSchema
	}
	return name
}

func refName(ref, kind string) (string, error) {
	prefix := "#/components/" + kind + "/"
	if !strings.HasPrefix(ref, prefix) {
		return "", fmt.Errorf("ref %q is not a local %s ref", ref, kind)
	}
	return strings.TrimPrefix(ref, prefix), nil
}

// ByVersion returns the operations of one API major, preserving input order.
func ByVersion(ops []Operation, version Version) []Operation {
	var out []Operation
	for _, op := range ops {
		if op.Version == version {
			out = append(out, op)
		}
	}
	return out
}

// Index returns the operations keyed by operationId.
func Index(ops []Operation) map[string]Operation {
	out := make(map[string]Operation, len(ops))
	for _, op := range ops {
		out[op.OperationID] = op
	}
	return out
}
