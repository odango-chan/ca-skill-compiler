package compiler

// Context Schema root $id must match the Contract contextSchema identity even
// when the root is reached through a $ref chain.

import "testing"

func TestContextSchemaRootIdentityMismatch(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceSnapshotFile("ca-context", "invocation-context.schema.yaml", "type: object", "$ref: urn:example:ca:registration:v1")
	fixture.refreshSnapshotDigests(t)
	diagnostics := Check(fixture.request(t))
	if !v4HasCode(diagnostics, "CONTEXT_SCHEMA_MISMATCH") {
		t.Fatalf("context Schema identity mismatch not reported: %+v", diagnostics)
	}
}

func TestContextSchemaRootIdentityPositive(t *testing.T) {
	diagnostics := Check(wp3aBaseFixture(t).request(t))
	if v4HasCode(diagnostics, "CONTEXT_SCHEMA_MISMATCH") {
		t.Fatalf("valid context Schema identity rejected: %+v", diagnostics)
	}
}
