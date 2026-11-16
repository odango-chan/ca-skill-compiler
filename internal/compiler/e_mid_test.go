package compiler

// E34/E36/E37/E41/E42/E47 conformance subset: exact Appendix E variants whose
// expected diagnostics are decidable by the static linking pipeline.

import "testing"

func TestE34RemovedProcedureRootAndReservedBindings(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "start: inspect-registration", "start: inspect-registration\ninputs: {ueRef: string}")
	if got := Check(fixture.request(t)); !v4HasCode(got, "SOURCE_ROOT_INVALID") {
		t.Fatalf("removed procedure root inputs field not reported: %+v", got)
	}

	fixture = wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "        ueRef: context.ueRef", "        ueRef: input.ueRef")
	if got := Check(fixture.request(t)); !v4HasCode(got, "SOURCE_PATH_INVALID") {
		t.Fatalf("input.ueRef binding not rejected: %+v", got)
	}

	fixture = wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "  inspect-registration:", "  context:")
	if got := Check(fixture.request(t)); !v4HasCode(got, "NODE_ID_RESERVED") {
		t.Fatalf("reserved context Node ID not rejected: %+v", got)
	}
}

func TestE36BareStringArgument(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "        ueRef: context.ueRef", "        ueRef: ue-1")
	if got := Check(fixture.request(t)); !v4HasCode(got, "SOURCE_PATH_INVALID") {
		t.Fatalf("bare string argument not rejected: %+v", got)
	}
}

func TestE37OptionalProducerFieldNotAvailable(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceSnapshotFile("ca", "registration.schema.yaml", "  accessState:\n    type: string\n    enum: [ACTIVE, IDLE]", "  accessState:\n    type: string\n    enum: [ACTIVE, IDLE]\n  reason:\n    type: string")
	fixture.refreshSnapshotDigests(t)
	fixture.replaceFile(wp3aProcedurePath, "            - inspect-registration.observation.output", "            - inspect-registration.observation.output.reason")
	if got := Check(fixture.request(t)); !v4HasCode(got, "SOURCE_NOT_AVAILABLE") {
		t.Fatalf("optional producer field read not reported: %+v", got)
	}
}

func TestE47OwnNodePreconditionReadsObservationOutput(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.files[wp3aProcedurePath] = []byte(`start: inspect-registration
nodes:
  inspect-registration:
    precondition:
      predicates:
        - predicate: example.ue.registered
          arguments: [inspect-registration.observation.output]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
    observation:
      tool: ca::ue.registration.current
      arguments:
        ueRef: context.ueRef
      routes:
        SUCCEEDED: SUCCEEDED
        FAILED: FAILED
`)
	if got := Check(fixture.request(t)); !v4HasCode(got, "SOURCE_NOT_AVAILABLE") {
		t.Fatalf("Precondition reading its own Observation output not rejected: %+v", got)
	}
}

func TestE41PostconditionOnlyStartEstablishesGoal(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.files[wp3aProcedurePath] = []byte(`start: verify
nodes:
  verify:
    postcondition:
      predicates:
        - predicate: example.ue.registered
          arguments: [context.ueRef]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`)
	diagnostics := Check(fixture.request(t))
	if v4HasCode(diagnostics, "FINAL_POSTCONDITION_MISSING") {
		t.Fatalf("Postcondition-only start with PASS->SUCCEEDED rejected: %+v", diagnostics)
	}
}

func TestE42FinalSuccessMustComeFromLastPostcondition(t *testing.T) {
	invalid := wp3aBaseFixture(t)
	invalid.files[wp3aProcedurePath] = []byte(`start: verify-local
nodes:
  verify-local:
    precondition:
      predicates:
        - predicate: example.ue.registered
          arguments: [context.ueRef]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`)
	if got := Check(invalid.request(t)); !v4HasCode(got, "FINAL_POSTCONDITION_MISSING") {
		t.Fatalf("Precondition PASS to SUCCEEDED without a final Postcondition edge accepted: %+v", got)
	}

	valid := wp3aBaseFixture(t)
	valid.files[wp3aProcedurePath] = []byte(`start: verify
nodes:
  verify:
    postcondition:
      predicates:
        - predicate: example.ue.registered
          arguments: [context.ueRef]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`)
	if got := Check(valid.request(t)); v4HasCode(got, "FINAL_POSTCONDITION_MISSING") {
		t.Fatalf("final Postcondition PASS->SUCCEEDED rejected: %+v", got)
	}
}
