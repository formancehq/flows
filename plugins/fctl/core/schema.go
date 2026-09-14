package core

import (
	"sort"
	"strconv"
	"strings"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

var objectSchema = []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object"}`)
var collectionSchema = []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"array"}`)

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
