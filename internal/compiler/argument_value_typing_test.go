package compiler

// WP6 slice 5 conformance: F.5 value typing for Tool arguments.

import "testing"

func TestWP6ArgumentValueTypesPositive(t *testing.T) {
	request := wp3aBaseFixture(t).request(t)
	diagnostics := Check(request)
	if v4HasCode(diagnostics, "INPUT_TYPE_MISMATCH") || v4HasCode(diagnostics, "INPUT_NULLABILITY_MISMATCH") {
		t.Fatalf("assignable argument value rejected: %+v", diagnostics)
	}
}

func TestWP6ArgumentValueTypeMismatch(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceSnapshotFile("ca", "registration-query.schema.yaml", "  ueRef:\n    type: string", "  ueRef:\n    type: integer")
	fixture.refreshSnapshotDigests(t)
	diagnostics := Check(fixture.request(t))
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "INPUT_TYPE_MISMATCH" && diagnostic.Path == wp3aProcedurePath {
			found = true
		}
	}
	if !found {
		t.Fatalf("incompatible argument type accepted: %+v", diagnostics)
	}
}

func TestWP6ArgumentValueNullabilityMismatch(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceSnapshotFile("ca-context", "invocation-context.schema.yaml", "  ueRef:\n    type: string\n    minLength: 1", "  ueRef:\n    type: [string, \"null\"]")
	fixture.refreshSnapshotDigests(t)
	diagnostics := Check(fixture.request(t))
	if !v4HasCode(diagnostics, "INPUT_NULLABILITY_MISMATCH") {
		t.Fatalf("nullable argument accepted by non-nullable target: %+v", diagnostics)
	}
}
