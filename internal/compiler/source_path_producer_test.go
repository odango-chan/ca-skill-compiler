package compiler

// WP6 slice 3 conformance: producer output SourcePath typing/dominance/availability.

import (
	"strings"
	"testing"
)

func TestWP6ProducerOutputPositive(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "inspect-registration.observation.output", "inspect-registration.observation.output.state")
	request := fixture.request(t)
	if diagnostics := wp6SchemaDiagnostics(Check(request)); len(diagnostics) != 0 {
		t.Fatalf("valid producer output SourcePath rejected: %+v", diagnostics)
	}
}

func TestWP6ProducerOutputUnknownField(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "inspect-registration.observation.output", "inspect-registration.observation.output.unknownField")
	diagnostics := wp6SchemaDiagnostics(Check(fixture.request(t)))
	if len(diagnostics) != 1 || diagnostics[0].Code != "SOURCE_PATH_INVALID" || diagnostics[0].Path != wp3aProcedurePath {
		t.Fatalf("unknown output field diagnostics = %+v, want one SOURCE_PATH_INVALID", diagnostics)
	}
}

func TestWP6ProducerOutputOptionalField(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceSnapshotFile("ca", "registration.schema.yaml", "required: [state, accessState]", "required: [accessState]")
	fixture.refreshSnapshotDigests(t)
	fixture.replaceFile(wp3aProcedurePath, "inspect-registration.observation.output", "inspect-registration.observation.output.state")
	diagnostics := wp6SchemaDiagnostics(Check(fixture.request(t)))
	if len(diagnostics) != 1 || diagnostics[0].Code != "SOURCE_NOT_AVAILABLE" {
		t.Fatalf("optional output field diagnostics = %+v, want one SOURCE_NOT_AVAILABLE", diagnostics)
	}
}

func TestWP6ProducerDominanceFailure(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "inspect-registration.observation.output", "extra.observation.output")
	fixture.replaceFile(wp3aProcedurePath, `start: inspect-registration`, `start: inspect-registration
`)
	fixture.files[wp3aProcedurePath] = []byte(string(fixture.files[wp3aProcedurePath]) + `  extra:
    observation:
      tool: ca::ue.registration.current
      arguments:
        ueRef: context.ueRef
      routes:
        SUCCEEDED: SUCCEEDED
        FAILED: FAILED
`)
	diagnostics := wp6SchemaDiagnostics(Check(fixture.request(t)))
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "SOURCE_NOT_AVAILABLE" && strings.Contains(diagnostic.Message, "dominate") {
			found = true
		}
	}
	if !found {
		t.Fatalf("non-dominating producer accepted: %+v", diagnostics)
	}
}
