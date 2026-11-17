package compiler

// WP6 slice 2 conformance: context/request SourcePath typing and availability.

import "testing"

var wp6SchemaDiagnosticCodes = map[string]bool{
	"SOURCE_PATH_INVALID":       true,
	"SOURCE_NOT_AVAILABLE":      true,
	"CONTEXT_SCHEMA_UNDECLARED": true,
	"REQUEST_SCHEMA_MISSING":    true,
}

func wp6SchemaDiagnostics(diagnostics []Diagnostic) []Diagnostic {
	filtered := make([]Diagnostic, 0)
	for _, diagnostic := range diagnostics {
		if wp6SchemaDiagnosticCodes[diagnostic.Code] {
			filtered = append(filtered, diagnostic)
		}
	}
	sortDiagnostics(filtered)
	return filtered
}

func TestWP6ContextSourcePathPositive(t *testing.T) {
	request := wp3aBaseFixture(t).request(t)
	if diagnostics := wp6SchemaDiagnostics(Check(request)); len(diagnostics) != 0 {
		t.Fatalf("valid context SourcePath rejected: %+v", diagnostics)
	}
}

func TestWP6ContextSourcePathUnknownField(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "ueRef: context.ueRef", "ueRef: context.unknownField")
	diagnostics := wp6SchemaDiagnostics(Check(fixture.request(t)))
	if len(diagnostics) != 1 || diagnostics[0].Code != "SOURCE_PATH_INVALID" || diagnostics[0].Path != wp3aProcedurePath {
		t.Fatalf("unknown context field diagnostics = %+v, want one SOURCE_PATH_INVALID at procedure.yaml", diagnostics)
	}
}

func TestWP6ContextSourcePathOptionalField(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceSnapshotFile("ca-context", "invocation-context.schema.yaml", "required: [ueRef]", "required: []")
	fixture.refreshSnapshotDigests(t)
	request := fixture.request(t)
	diagnostics := wp6SchemaDiagnostics(Check(request))
	if len(diagnostics) != 1 || diagnostics[0].Code != "SOURCE_NOT_AVAILABLE" || diagnostics[0].Path != wp3aProcedurePath {
		t.Fatalf("optional context field diagnostics = %+v, want one SOURCE_NOT_AVAILABLE at procedure.yaml", diagnostics)
	}
}

func TestWP6ContextSchemaUndeclared(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aContractPath, "contextSchema: ca-context::urn:example:ca:invocation-context:v1\n", "")
	diagnostics := wp6SchemaDiagnostics(Check(fixture.request(t)))
	if len(diagnostics) != 1 || diagnostics[0].Code != "CONTEXT_SCHEMA_UNDECLARED" {
		t.Fatalf("undeclared contextSchema diagnostics = %+v, want one CONTEXT_SCHEMA_UNDECLARED", diagnostics)
	}
}

func TestWP6RequestSourcePath(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aContractPath, "contextSchema:", "requestSchema: schemas/request.schema.yaml\ncontextSchema:")
	fixture.files["skills/verify-ue-registration/schemas/request.schema.yaml"] = []byte(`$schema: https://json-schema.org/draft/2020-12/schema
type: object
additionalProperties: false
required: [dnn]
properties:
  dnn: {type: string}
`)
	fixture.replaceFile(wp3aProcedurePath, "ueRef: context.ueRef", "ueRef: context.ueRef\n        dnn: context.request.dnn")
	request := fixture.request(t)
	if diagnostics := wp6SchemaDiagnostics(Check(request)); len(diagnostics) != 0 {
		t.Fatalf("valid context.request SourcePath rejected: %+v", diagnostics)
	}

	missing := wp3aBaseFixture(t)
	missing.replaceFile(wp3aProcedurePath, "ueRef: context.ueRef", "ueRef: context.ueRef\n        dnn: context.request.dnn")
	diagnostics := wp6SchemaDiagnostics(Check(missing.request(t)))
	if len(diagnostics) != 1 || diagnostics[0].Code != "REQUEST_SCHEMA_MISSING" {
		t.Fatalf("context.request without requestSchema diagnostics = %+v, want one REQUEST_SCHEMA_MISSING", diagnostics)
	}
}
