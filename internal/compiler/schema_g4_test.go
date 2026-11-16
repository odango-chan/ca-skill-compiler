package compiler

// WP4A conformance fixtures: CA-SL G.4 Schema dialect allowlist and Request
// Schema root shape. Derived from the frozen G.4 / G.6 / E110 requirements.

import (
	"strings"
	"testing"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func g4Issues(t *testing.T, source string) []g4Issue {
	t.Helper()
	root, err := yaml.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse schema fixture: %v", err)
	}
	return validateG4Schema(root)
}

func TestWP4AG4AllowlistUnit(t *testing.T) {
	valid := []string{
		`$schema: https://json-schema.org/draft/2020-12/schema
type: object
additionalProperties: false
required: [name]
properties:
  name: {type: string, minLength: 1, maxLength: 64}
`,
		`$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:schema:v1
type: [string, "null"]
minLength: 0
maxLength: 10
`,
		`$schema: https://json-schema.org/draft/2020-12/schema
type: array
items: {type: integer, minimum: 0, maximum: 100, exclusiveMaximum: 101}
minItems: 1
maxItems: 5
`,
		`$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:schema:v1
$ref: urn:example:target:v1
`,
		`$schema: https://json-schema.org/draft/2020-12/schema
type: string
enum: [ACTIVE, IDLE]
const: ACTIVE
`,
	}
	for index, source := range valid {
		issues := g4Issues(t, source)
		if len(issues) != 0 {
			t.Fatalf("valid G.4 schema #%d rejected: %+v", index, issues)
		}
	}

	invalid := []struct {
		name   string
		source string
	}{
		{"missing dialect", `type: object
properties: {}
additionalProperties: false
`},
		{"wrong dialect", `$schema: http://json-schema.org/draft-07/schema
type: object
`},
		{"composition keyword", `$schema: https://json-schema.org/draft/2020-12/schema
anyOf:
  - {type: string}
  - {type: integer}
`},
		{"pattern", `$schema: https://json-schema.org/draft/2020-12/schema
type: string
pattern: "^[a-z]+$"
`},
		{"defs", `$schema: https://json-schema.org/draft/2020-12/schema
$defs:
  a: {type: string}
type: object
`},
		{"fragment ref", `$schema: https://json-schema.org/draft/2020-12/schema
$ref: "#/$defs/x"
`},
		{"ref with sibling assertion", `$schema: https://json-schema.org/draft/2020-12/schema
$ref: urn:example:target:v1
type: object
`},
		{"general union", `$schema: https://json-schema.org/draft/2020-12/schema
type: [string, integer]
`},
		{"single-item union", `$schema: https://json-schema.org/draft/2020-12/schema
type: [string]
`},
		{"duplicate union member", `$schema: https://json-schema.org/draft/2020-12/schema
type: [string, string]
`},
		{"required not declared", `$schema: https://json-schema.org/draft/2020-12/schema
type: object
additionalProperties: false
properties:
  a: {type: string}
required: [b]
`},
		{"additionalProperties not boolean", `$schema: https://json-schema.org/draft/2020-12/schema
type: object
additionalProperties: {type: string}
`},
		{"tuple items", `$schema: https://json-schema.org/draft/2020-12/schema
type: array
items: [{type: string}]
`},
		{"minItems greater than maxItems", `$schema: https://json-schema.org/draft/2020-12/schema
type: array
items: {type: string}
minItems: 5
maxItems: 2
`},
		{"length constraint on integer", `$schema: https://json-schema.org/draft/2020-12/schema
type: integer
minLength: 1
`},
		{"enum duplicate", `$schema: https://json-schema.org/draft/2020-12/schema
type: string
enum: [A, A]
`},
		{"enum type mismatch", `$schema: https://json-schema.org/draft/2020-12/schema
type: integer
enum: ["not-an-integer"]
`},
		{"const type mismatch", `$schema: https://json-schema.org/draft/2020-12/schema
type: boolean
const: "true"
`},
		{"minimum greater than maximum", `$schema: https://json-schema.org/draft/2020-12/schema
type: number
minimum: 10
maximum: 1
`},
		{"exclusive bounds crossed", `$schema: https://json-schema.org/draft/2020-12/schema
type: number
exclusiveMinimum: 5
exclusiveMaximum: 5
`},
	}
	for _, testCase := range invalid {
		if issues := g4Issues(t, testCase.source); len(issues) == 0 {
			t.Fatalf("invalid G.4 schema accepted: %s", testCase.name)
		}
	}
}

func TestWP4ARequestSchemaRootShape(t *testing.T) {
	valid := `$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:request:v1alpha4
type: object
additionalProperties: false
required: [dnn]
properties:
  dnn: {type: string, minLength: 1}
`
	if got := g4Issues(t, valid); len(got) != 0 {
		t.Fatalf("valid request Schema rejected: %+v", got)
	}
	rootRef := `$schema: https://json-schema.org/draft/2020-12/schema
$ref: urn:example:request:target:v1
`
	if got := g4Issues(t, rootRef); len(got) != 0 {
		t.Fatalf("valid root-$ref Schema rejected: %+v", got)
	}

	openRoot := strings.Replace(valid, "additionalProperties: false", "additionalProperties: true", 1)
	if got := g4Issues(t, openRoot); len(got) != 0 {
		t.Fatalf("G.4 structural check must leave root-closure policy to the workflow: %+v", got)
	}
}

func TestWP4AValidationIntegratedIntoBuild(t *testing.T) {
	validSchema := `$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:request:v1alpha4
type: object
additionalProperties: false
required: [dnn]
properties:
  dnn: {type: string, minLength: 1}
`
	validReq := v4TestRequest(v4TestContract, v4TestProcedure, v4TestPredicates, validSchema)
	if diagnostics := Check(validReq); v4HasCode(diagnostics, "SCHEMA_INVALID") {
		t.Fatalf("valid request Schema rejected in pipeline: %+v", diagnostics)
	}

	cases := map[string]struct {
		schema   string
		wantCode string
	}{
		"unsupported keyword": {
			schema:   strings.Replace(validSchema, "minLength: 1", "pattern: '^[a-z]+$'", 1),
			wantCode: "SCHEMA_INVALID",
		},
		"composition keyword": {
			schema: `$schema: https://json-schema.org/draft/2020-12/schema
type: object
anyOf: [{type: object}]
additionalProperties: false
properties: {dnn: {type: string}}
`,
			wantCode: "SCHEMA_INVALID",
		},
		"open root": {
			schema:   strings.Replace(validSchema, "additionalProperties: false", "additionalProperties: true", 1),
			wantCode: "SCHEMA_INVALID",
		},
		"non-object root": {
			schema: `$schema: https://json-schema.org/draft/2020-12/schema
type: string
`,
			wantCode: "SCHEMA_INVALID",
		},
		"missing properties": {
			schema: `$schema: https://json-schema.org/draft/2020-12/schema
type: object
additionalProperties: false
`,
			wantCode: "SCHEMA_INVALID",
		},
		"missing schema file": {
			schema:   "",
			wantCode: "REQUEST_SCHEMA_MISSING",
		},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			req := v4TestRequest(v4TestContract, v4TestProcedure, v4TestPredicates, testCase.schema)
			diagnostics := Check(req)
			if !v4HasCode(diagnostics, testCase.wantCode) {
				t.Fatalf("missing %s: %+v", testCase.wantCode, diagnostics)
			}
			found := false
			for _, diagnostic := range diagnostics {
				if diagnostic.Code == testCase.wantCode && diagnostic.Path != "" && diagnostic.Line > 0 && diagnostic.Column > 0 {
					found = true
				}
			}
			if !found {
				t.Fatalf("%s diagnostic must retain schema source location: %+v", testCase.wantCode, diagnostics)
			}
			buildDiagnostics := Build(req).Diagnostics
			if len(filterDiagnosticsByCode(diagnostics, testCase.wantCode)) != len(filterDiagnosticsByCode(buildDiagnostics, testCase.wantCode)) {
				t.Fatalf("Check/Build disagree on %s", testCase.wantCode)
			}
		})
	}
}

func TestWP4AValidatesEverySkillOwnedSchema(t *testing.T) {
	req := v4TestRequest(v4TestContract, v4TestProcedure, v4TestPredicates, `$schema: https://json-schema.org/draft/2020-12/schema
type: object
additionalProperties: false
properties: {dnn: {type: string}}
`)
	req.Files = append(req.Files, File{
		Path: "skills/pdu/schemas/helper.schema.yaml",
		Bytes: []byte(`$schema: https://json-schema.org/draft/2020-12/schema
type: object
allOf: [{type: object}]
`),
	})
	diagnostics := Check(req)
	if !v4HasCode(diagnostics, "SCHEMA_INVALID") {
		t.Fatalf("invalid unused Skill-owned Schema escaped G.4 validation: %+v", diagnostics)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "SCHEMA_INVALID" && diagnostic.Path == "skills/pdu/schemas/helper.schema.yaml" {
			return
		}
	}
	t.Fatalf("schema diagnostic did not name the offending file: %+v", diagnostics)
}

func filterDiagnosticsByCode(diagnostics []Diagnostic, code string) []Diagnostic {
	filtered := make([]Diagnostic, 0)
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			filtered = append(filtered, diagnostic)
		}
	}
	return filtered
}
