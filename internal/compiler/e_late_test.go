package compiler

// Late E-case subset: E11, E35, E52, E60, E65 static variants.

import "testing"

func TestE11NoSuccessRequestsLegalStops(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.files[wp3aProcedurePath] = []byte(`start: inspect-registration
nodes:
  inspect-registration:
    observation:
      tool: ca::ue.registration.current
      arguments:
        ueRef: context.ueRef
      routes:
        SUCCEEDED: final
        FAILED: FAILED
  final:
    postcondition:
      predicates:
        - predicate: example.ue.registered
          arguments: [inspect-registration.observation.output]
      routes:
        PASS: SAFE_STOPPED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`)
	diagnostics := Check(fixture.request(t))
	if v4HasCode(diagnostics, "FINAL_POSTCONDITION_MISSING") {
		t.Fatalf("no-SUCCEEDED procedure with legal stop routes rejected: %+v", diagnostics)
	}

	missingFinal := wp3aBaseFixture(t)
	missingFinal.files[wp3aProcedurePath] = []byte(`start: inspect-registration
nodes:
  inspect-registration:
    observation:
      tool: ca::ue.registration.current
      arguments:
        ueRef: context.ueRef
      routes:
        SUCCEEDED: SUCCEEDED
        FAILED: FAILED
`)
	if got := Check(missingFinal.request(t)); !v4HasCode(got, "FINAL_POSTCONDITION_MISSING") {
		t.Fatalf("direct SUCCEEDED without final Postcondition accepted: %+v", got)
	}
}

func TestE35DependencyRequirementObjectInvalid(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aContractPath, "  ca: urn:example:ca:tools", "  ca: {id: urn:example:ca:tools}")
	if got := Check(fixture.request(t)); !v4HasCode(got, "DEPENDENCY_REQUIREMENT_INVALID") {
		t.Fatalf("object dependency requirement not rejected: %+v", got)
	}
}

func TestE52EntryPreconditionReferenceVariants(t *testing.T) {
	cases := []string{
		"precondition: inspect-registration.postcondition",
		"precondition: no-such-node.precondition",
		"precondition: [inspect-registration.precondition]",
		"precondition: ''",
	}
	for _, declaration := range cases {
		fixture := wp3aBaseFixture(t)
		fixture.replaceFile(wp3aContractPath, "goal: ue.registration_verified", "goal: ue.registration_verified\n"+declaration)
		if got := Check(fixture.request(t)); !v4HasCode(got, "ENTRY_PRECONDITION_REF_INVALID") {
			t.Fatalf("invalid entry precondition reference %q accepted: %+v", declaration, got)
		}
	}
}

func TestE60ReconcileBindMustBeOriginalInvocation(t *testing.T) {
	req, _, _, _ := reconcileFixture(t, replaceAll(reconcileCatalog, "          bind: originalInvocationId", "          bind: otherInvocationId"))
	if got := Check(req); !v4HasCode(got, "RECONCILE_CONTRACT_INVALID") {
		t.Fatalf("Reconcile bind other than originalInvocationId not rejected: %+v", got)
	}
}

func TestE65UnreachableNodeWarningOnly(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "  inspect-registration:", "  unused:\n    postcondition:\n      predicates:\n        - predicate: example.ue.registered\n          arguments: [context.ueRef]\n      routes:\n        PASS: SUCCEEDED\n        FAIL: FAILED\n        UNKNOWN: UNRESOLVED\n  inspect-registration:")
	diagnostics := Check(fixture.request(t))
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "UNREACHABLE_NODE" && diagnostic.Severity == "warning" {
			found = true
		}
	}
	if !found {
		t.Fatalf("unreachable Node not reported as UNREACHABLE_NODE warning: %+v", diagnostics)
	}
}
