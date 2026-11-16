package compiler

// WP4C conformance fixtures: F.5 conservative Source/Target assignability.

import "testing"
import yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"

func f5Parse(t *testing.T, source string) *yaml.Node {
	t.Helper()
	node, err := yaml.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse assignability fixture: %v", err)
	}
	return node
}

func TestWP4CAssignability(t *testing.T) {
	object := func(properties, required, additional string) string {
		return `$schema: https://json-schema.org/draft/2020-12/schema
type: object
properties:
` + properties + `
required: [` + required + `]
additionalProperties: ` + additional + `
`
	}
	positive := []struct {
		name           string
		source, target string
	}{
		{"integer to number", `$schema: https://json-schema.org/draft/2020-12/schema
type: integer
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: number
`},
		{"same string type", `$schema: https://json-schema.org/draft/2020-12/schema
type: string
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: string
`},
		{"enum subset", `$schema: https://json-schema.org/draft/2020-12/schema
type: string
enum: [ACTIVE]
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: string
enum: [ACTIVE, IDLE]
`},
		{"const matches enum", `$schema: https://json-schema.org/draft/2020-12/schema
type: string
const: ACTIVE
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: string
enum: [ACTIVE, IDLE]
`},
		{"tighter string length", `$schema: https://json-schema.org/draft/2020-12/schema
type: string
minLength: 2
maxLength: 4
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: string
minLength: 1
maxLength: 8
`},
		{"tighter numeric range", `$schema: https://json-schema.org/draft/2020-12/schema
type: integer
minimum: 2
maximum: 4
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: number
minimum: 1
maximum: 8
`},
		{"required subset and closed keys", object("  a: {type: string}\n  b: {type: integer}", "a", "false"),
			object("  a: {type: string}\n  b: {type: integer}", "a", "false")},
		{"optional target property", object("  a: {type: string}", "a", "false"),
			object("  a: {type: string}\n  b: {type: integer}", "a", "false")},
		{"array items and bounds", `$schema: https://json-schema.org/draft/2020-12/schema
type: array
items: {type: integer}
minItems: 2
maxItems: 3
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: array
items: {type: number}
minItems: 1
maxItems: 5
`},
		{"untyped target accepts typed source", `$schema: https://json-schema.org/draft/2020-12/schema
type: string
`, `$schema: https://json-schema.org/draft/2020-12/schema
`},
	}
	for _, testCase := range positive {
		reason, ok := schemaAssignable(f5Parse(t, testCase.source), f5Parse(t, testCase.target))
		if !ok || reason != assignableOK {
			t.Fatalf("expected assignable %s, got %s", testCase.name, reason)
		}
	}

	negative := []struct {
		name           string
		source, target string
		reason         assignabilityReason
	}{
		{"number to integer", `$schema: https://json-schema.org/draft/2020-12/schema
type: number
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: integer
`, assignableTypeMismatch},
		{"string to integer", `$schema: https://json-schema.org/draft/2020-12/schema
type: string
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: integer
`, assignableTypeMismatch},
		{"nullable source to nonnullable target", `$schema: https://json-schema.org/draft/2020-12/schema
type: [string, "null"]
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: string
`, assignableNullabilityMismatch},
		{"enum not subset", `$schema: https://json-schema.org/draft/2020-12/schema
type: string
enum: [ACTIVE, OTHER]
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: string
enum: [ACTIVE, IDLE]
`, assignableTypeMismatch},
		{"const mismatch", `$schema: https://json-schema.org/draft/2020-12/schema
type: string
const: OTHER
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: string
const: ACTIVE
`, assignableTypeMismatch},
		{"target required missing", object("  a: {type: string}", "", "false"),
			object("  a: {type: string}", "a", "false"), assignableTypeMismatch},
		{"extra key against closed target", object("  a: {type: string}\n  b: {type: string}", "a", "false"),
			object("  a: {type: string}", "a", "false"), assignableTypeMismatch},
		{"looser string length", `$schema: https://json-schema.org/draft/2020-12/schema
type: string
minLength: 0
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: string
minLength: 1
`, assignableTypeMismatch},
		{"looser numeric maximum", `$schema: https://json-schema.org/draft/2020-12/schema
type: integer
maximum: 100
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: integer
maximum: 10
`, assignableTypeMismatch},
		{"array item mismatch", `$schema: https://json-schema.org/draft/2020-12/schema
type: array
items: {type: string}
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: array
items: {type: integer}
`, assignableTypeMismatch},
		{"ref requires resolution", `$schema: https://json-schema.org/draft/2020-12/schema
$ref: urn:example:source:v1
`, `$schema: https://json-schema.org/draft/2020-12/schema
type: string
`, assignableUnresolved},
	}
	for _, testCase := range negative {
		reason, ok := schemaAssignable(f5Parse(t, testCase.source), f5Parse(t, testCase.target))
		if ok || reason != testCase.reason {
			t.Fatalf("expected %s for %s, got %s (ok=%v)", testCase.reason, testCase.name, reason, ok)
		}
	}
}

func TestWP4CAssignabilityRequiresG4Dialect(t *testing.T) {
	source, _ := yaml.Parse([]byte("type: string\n"))
	target := f5Parse(t, `$schema: https://json-schema.org/draft/2020-12/schema
type: string
`)
	if reason, ok := schemaAssignable(source, target); ok || reason != assignableUnresolved {
		t.Fatalf("Schema without the G.4 dialect must not be used for assignability: %s", reason)
	}
}
