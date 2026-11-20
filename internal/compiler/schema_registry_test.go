package compiler

// WP4B conformance fixtures: exact Schema registry, external $ref closure and
// San v6.0.3 adapter, plus Request/Context root shape through a root $ref chain.

import (
	"reflect"
	"testing"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

var wp4bDiagnosticCodes = map[string]bool{
	"SCHEMA_INVALID":            true,
	"SCHEMA_MISSING":            true,
	"EXPORT_ID_AMBIGUOUS":       true,
	"CONTEXT_REQUEST_COLLISION": true,
}

func wp4bDiagnostics(diagnostics []Diagnostic) []Diagnostic {
	filtered := make([]Diagnostic, 0)
	for _, diagnostic := range diagnostics {
		if wp4bDiagnosticCodes[diagnostic.Code] {
			filtered = append(filtered, diagnostic)
		}
	}
	sortDiagnostics(filtered)
	return filtered
}

func wp4bClosure(t *testing.T, fixture wp3aFixture) []Diagnostic {
	t.Helper()
	req := fixture.request(t)
	identity := wp3aIdentity(t, req)
	ast := wp3aParseSkillAST(t, req, wp3aSkillName)
	ownerLinking := validateOwnerLinking(req, ast, identity)
	return wp4bDiagnostics(validateSchemaClosure(req, ast, identity, ownerLinking, nil))
}

func wp4bCheckBuildAgreement(t *testing.T, fixture wp3aFixture) []Diagnostic {
	t.Helper()
	req := fixture.request(t)
	check := wp4bDiagnostics(Check(req))
	build := wp4bDiagnostics(Build(req).Diagnostics)
	if !reflect.DeepEqual(check, build) {
		t.Fatalf("Check and Build disagree on WP4B diagnostics:\ncheck=%+v\nbuild=%+v", check, build)
	}
	return check
}

// WP4B-001: positive exact closure, including the San compile path.
func TestWP4B001PositiveSchemaClosure(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aContractPath, "contextSchema:", "requestSchema: schemas/request.schema.yaml\ncontextSchema:")
	fixture.files["skills/verify-ue-registration/schemas/request.schema.yaml"] = []byte(`$schema: https://json-schema.org/draft/2020-12/schema
type: object
additionalProperties: false
required: [ueRef]
properties:
  ueRef: {type: string, minLength: 1}
`)
	if diagnostics := wp4bClosure(t, fixture); len(diagnostics) != 0 {
		t.Fatalf("positive Schema closure emitted diagnostics: %+v", diagnostics)
	}
	if diagnostics := wp4bCheckBuildAgreement(t, fixture); len(diagnostics) != 0 {
		t.Fatalf("positive Schema closure emitted diagnostics through Check/Build: %+v", diagnostics)
	}
}

// WP4B-002: an external Schema $ref that does not resolve is SCHEMA_MISSING.
func TestWP4B002ExternalRefMissing(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceSnapshotFile("ca-context", "invocation-context.schema.yaml", `  ueRef:
    type: string
    minLength: 1`, `  ueRef: {$ref: urn:example:ca:missing-schema:v1}`)
	fixture.refreshSnapshotDigests(t)
	diagnostics := wp4bClosure(t, fixture)
	if len(diagnostics) != 1 || diagnostics[0].Code != "SCHEMA_MISSING" || diagnostics[0].Path != "" {
		t.Fatalf("external missing $ref diagnostics = %+v, want one SCHEMA_MISSING with empty path", diagnostics)
	}
	if diagnostics := wp4bCheckBuildAgreement(t, fixture); len(diagnostics) != 1 {
		t.Fatalf("Check/Build should agree on one SCHEMA_MISSING: %+v", diagnostics)
	}
}

// WP4B-003: a cyclic external Schema $ref chain is rejected.
func TestWP4B003ExternalRefCycle(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceSnapshotFile("ca-context", "invocation-context.schema.yaml", "type: object", "$ref: urn:example:cycle:a")
	fixture.addSnapshotFile("ca-context", "cycle-a.schema.yaml", []byte(`$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:cycle:a
$ref: urn:example:cycle:b
`))
	fixture.addSnapshotFile("ca-context", "cycle-b.schema.yaml", []byte(`$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:cycle:b
$ref: urn:example:cycle:a
`))
	fixture.refreshSnapshotDigests(t)
	diagnostics := wp4bClosure(t, fixture)
	if !hasDiagnosticCode(diagnostics, "SCHEMA_INVALID") {
		t.Fatalf("cyclic Schema closure was not rejected: %+v", diagnostics)
	}
}

// WP4B-004: a Host Context root that is not a closed object collides.
func TestWP4B004ContextRootCollision(t *testing.T) {
	open := wp3aBaseFixture(t)
	open.replaceSnapshotFile("ca-context", "invocation-context.schema.yaml", "additionalProperties: false", "additionalProperties: true")
	open.refreshSnapshotDigests(t)
	diagnostics := wp4bClosure(t, open)
	if len(diagnostics) != 1 || diagnostics[0].Code != "CONTEXT_REQUEST_COLLISION" {
		t.Fatalf("open Host Context root diagnostics = %+v, want one CONTEXT_REQUEST_COLLISION", diagnostics)
	}

	reserved := wp3aBaseFixture(t)
	reserved.replaceSnapshotFile("ca-context", "invocation-context.schema.yaml", `  ueRef:
    type: string
    minLength: 1`, `  ueRef:
    type: string
    minLength: 1
  request: {type: object}`)
	reserved.refreshSnapshotDigests(t)
	diagnostics = wp4bClosure(t, reserved)
	if len(diagnostics) != 1 || diagnostics[0].Code != "CONTEXT_REQUEST_COLLISION" {
		t.Fatalf("reserved Host Context property diagnostics = %+v, want one CONTEXT_REQUEST_COLLISION", diagnostics)
	}
}

// WP4B-005: a missing external Context Schema is diagnosed once (WP3B owns the
// reference identity; WP4B must not duplicate it).
func TestWP4B005MissingContextSchemaNotDuplicated(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aContractPath, "ca-context::urn:example:ca:invocation-context:v1", "ca-context::urn:example:ca:missing-context:v1")
	req := fixture.request(t)
	count := 0
	for _, diagnostic := range wp4bDiagnostics(Check(req)) {
		if diagnostic.Code == "SCHEMA_MISSING" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("SCHEMA_MISSING count = %d, want 1 (no duplicated closure diagnostic): %+v", count, wp4bDiagnostics(Check(req)))
	}
}

// WP4B-006: San is invoked only over the already-G.4-validated external closure.
func TestWP4B006SanAdapterAcceptsValidClosure(t *testing.T) {
	registry := schemaRegistry{external: map[string][]schemaResource{}, local: map[string]schemaResource{}}
	root, err := parseTestYAML(`$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:san:root:v1
type: object
additionalProperties: false
properties:
  value: {$ref: urn:example:san:leaf:v1}
`)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := parseTestYAML(`$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:san:leaf:v1
type: string
`)
	if err != nil {
		t.Fatal(err)
	}
	rootResource := schemaResource{identity: "urn:example:san:root:v1", external: true, alias: "ca", path: "root.schema.yaml", node: root}
	leafResource := schemaResource{identity: "urn:example:san:leaf:v1", external: true, alias: "ca", path: "leaf.schema.yaml", node: leaf}
	registry.external[rootResource.identity] = []schemaResource{rootResource}
	registry.external[leafResource.identity] = []schemaResource{leafResource}
	validated := map[string]bool{schemaResourceKey(rootResource): true, schemaResourceKey(leafResource): true}
	if diagnostics := compileSchemaClosure(registry, validated); len(diagnostics) != 0 {
		t.Fatalf("San rejected the valid external closure: %+v", diagnostics)
	}
}

func parseTestYAML(source string) (*yaml.Node, error) {
	return yaml.Parse([]byte(source))
}

func hasDiagnosticCode(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
