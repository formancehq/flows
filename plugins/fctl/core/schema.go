package core

import (
	"sort"
	"strconv"
	"strings"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

const schemaHeader = `"$schema":"https://json-schema.org/draft/2020-12/schema",`

const triggerResultSchema = `{"type":"object","properties":{` +
	`"id":{"type":"string"},"name":{"type":"string"},"event":{"type":"string"},` +
	`"workflowID":{"type":"string"},"version":{"type":"string"},"filter":{"type":"string"},` +
	`"vars":{"type":"object"},"createdAt":{"type":"string"}},"additionalProperties":true}`

const workflowResultSchema = `{"type":"object","properties":{` +
	`"id":{"type":"string"},"createdAt":{"type":"string"},"updatedAt":{"type":"string"},` +
	`"config":{"type":"object","properties":{"name":{"type":"string"},"stages":{"type":"array"}},"additionalProperties":true}` +
	`},"additionalProperties":true}`

const instanceResultSchema = `{"type":"object","properties":{` +
	`"id":{"type":"string"},"workflowID":{"type":"string"},"createdAt":{"type":"string"},` +
	`"updatedAt":{"type":"string"},"terminated":{"type":"boolean"},"terminatedAt":{"type":"string"},` +
	`"error":{"type":"string"},"status":{"type":"array"},"workflow":` + workflowResultSchema +
	`},"additionalProperties":true}`

const occurrenceResultSchema = `{"type":"object","properties":{` +
	`"date":{"type":"string"},"triggerID":{"type":"string"},"workflowInstanceID":{"type":"string"},` +
	`"workflowInstance":` + instanceResultSchema + `,"error":{"type":"string"},"event":{"type":"object"}` +
	`},"additionalProperties":true}`

const triggerTestResultSchema = `{"type":"object","properties":{` +
	`"filter":{"type":"object","properties":{"match":{"type":"boolean"}},"additionalProperties":true},` +
	`"variables":{"type":"object","additionalProperties":{"type":"object","properties":{"value":{"type":"string"}},"additionalProperties":true}}` +
	`},"additionalProperties":true}`

const compositeInstanceResultSchema = `{"type":"object","properties":{"instance":` + instanceResultSchema +
	`,"workflow":` + workflowResultSchema + `},"additionalProperties":true}`

const describeResultSchema = `{"type":"object","properties":{` +
	`"history":{"type":"array","items":{"type":"object","additionalProperties":true}},` +
	`"stages":{"type":"array","items":{"type":"array","items":{"type":"object","additionalProperties":true}}}` +
	`},"additionalProperties":true}`

const emptyResultSchema = `{"type":"object","maxProperties":0,"additionalProperties":false}`

func outputSchema(path string) []byte {
	var body string
	switch path {
	case "triggers.list":
		body = `{"type":"array","items":` + triggerResultSchema + `}`
	case "triggers.show", "triggers.create":
		body = triggerResultSchema
	case "triggers.test":
		body = triggerTestResultSchema
	case "triggers.occurrences.list":
		body = `{"type":"array","items":` + occurrenceResultSchema + `}`
	case "workflows.list":
		body = `{"type":"array","items":` + workflowResultSchema + `}`
	case "workflows.show", "workflows.create":
		body = workflowResultSchema
	case "workflows.run", "instances.show":
		body = compositeInstanceResultSchema
	case "instances.list":
		body = `{"type":"array","items":` + instanceResultSchema + `}`
	case "instances.describe":
		body = describeResultSchema
	default:
		body = emptyResultSchema
	}
	return []byte(`{` + schemaHeader + strings.TrimPrefix(body, `{`))
}

func inputSchema(arguments []sdk.Argument, flags []sdk.Flag) []byte {
	type field struct {
		typ      string
		required bool
	}
	fields := map[string]field{}
	for _, value := range arguments {
		fields[value.Name] = field{"string", value.Required}
	}
	for _, value := range flags {
		typ := "string"
		if value.Type == sdk.FlagBool {
			typ = "boolean"
		}
		if value.Type == sdk.FlagInt32 {
			typ = "integer"
		}
		fields[value.Name] = field{typ, value.Required}
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	props, required := []string{}, []string{}
	for _, name := range names {
		props = append(props, strconv.Quote(name)+`:{"type":`+strconv.Quote(fields[name].typ)+`}`)
		if fields[name].required {
			required = append(required, strconv.Quote(name))
		}
	}
	out := `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{` + strings.Join(props, ",") + `}`
	if len(required) > 0 {
		out += `,"required":[` + strings.Join(required, ",") + `]`
	}
	return []byte(out + `,"additionalProperties":false}`)
}
