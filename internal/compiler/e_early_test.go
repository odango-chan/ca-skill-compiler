package compiler

// E02/E08/E09/E12/E18 conformance subset: exact Appendix E variants whose
// expected diagnostics are implemented in the static linking pipeline.

import "testing"

func TestE02ProcedureGoalFieldRootInvalid(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "start: inspect-registration", "start: inspect-registration\ngoal: ue.registration_verified")
	diagnostics := Check(fixture.request(t))
	if !hasCode(diagnostics, "SOURCE_ROOT_INVALID") {
		t.Fatalf("procedure.goal not classified as SOURCE_ROOT_INVALID: %+v", diagnostics)
	}
}

func TestE08PredicateArityMismatch(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aPredicatesPath, "      - registration: ca::urn:example:ca:registration:v1", "      - registration: ca::urn:example:ca:registration:v1\n      - second: string")
	diagnostics := Check(fixture.request(t))
	if !hasCode(diagnostics, "PREDICATE_ARITY_MISMATCH") {
		t.Fatalf("extra predicate parameter not reported as arity mismatch: %+v", diagnostics)
	}
}

func TestE09PredicateTypeMismatch(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "            - inspect-registration.observation.output", "            - context.ueRef")
	diagnostics := Check(fixture.request(t))
	if !hasCode(diagnostics, "PREDICATE_TYPE_MISMATCH") {
		t.Fatalf("context string argument not reported as predicate type mismatch: %+v", diagnostics)
	}
}

func TestE12ContextSchemaUndeclared(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aContractPath, "contextSchema: ca-context::urn:example:ca:invocation-context:v1\n", "")
	diagnostics := Check(fixture.request(t))
	if !hasCode(diagnostics, "CONTEXT_SCHEMA_UNDECLARED") {
		t.Fatalf("missing contextSchema not reported: %+v", diagnostics)
	}
}

func TestE18CommittedRouteCannotPublishSucceeded(t *testing.T) {
	req, _, _, _ := reconcileFixture(t, reconcileCatalog)
	fixture := wp3aBaseFixture(t)
	fixture.addSnapshotFile("ca", "mutation-catalog.yaml", []byte(reconcileCatalog))
	fixture.addSnapshotFile("ca", "proof.schema.yaml", []byte(reconcileProofSchema))
	fixture.addSnapshotFile("ca", "receipt.schema.yaml", []byte(reconcileReceiptSchema))
	fixture.addSnapshotFile("ca", "recoverable.schema.yaml", []byte(reconcileRecoverableSchema))
	fixture.refreshSnapshotDigests(t)
	_ = req
	fixture.files[wp3aProcedurePath] = []byte(`start: mutate
nodes:
  mutate:
    operation:
      tool: ca::example.session.mutate
      arguments: {}
      routes:
        SUCCEEDED: final
        FAILED: FAILED
        NOT_COMMITTED: SAFE_STOPPED
        COMMITTED: SUCCEEDED
  final:
    postcondition:
      predicates:
        - predicate: example.ue.registered
          arguments: [mutate.operation.output]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`)
	diagnostics := Check(fixture.request(t))
	if !hasCode(diagnostics, "SUCCESS_BYPASSES_FINAL_POSTCONDITION") {
		t.Fatalf("COMMITTED route to SUCCEEDED not reported: %+v", diagnostics)
	}
}
