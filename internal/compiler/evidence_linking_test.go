package compiler

// WP7 slice 1 conformance: Evidence requirement parameter binding.

import "testing"

func TestWP7EvidenceRequirementPositive(t *testing.T) {
	request := wp3aBaseFixture(t).request(t)
	if hasCode(Check(request), "PREDICATE_EVIDENCE_REQUIRED") {
		t.Fatalf("valid Evidence requirement rejected: %+v", Check(request))
	}
}

func TestWP7EvidenceRequirementUndeclaredParameter(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aPredicatesPath, "        registration: [ca::ue.registration-state]", "        registraton: [ca::ue.registration-state]")
	diagnostics := Check(fixture.request(t))
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "PREDICATE_EVIDENCE_REQUIRED" && diagnostic.Path == wp3aPredicatesPath && diagnostic.Line > 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("Evidence requirement binding to undeclared parameter accepted: %+v", diagnostics)
	}
}

// WP7 slice 2: guarantee references must satisfy the frozen qualified-reference
// syntax (Alias::GuaranteeId). Descriptor content stays fail-closed.
func TestWP7GuaranteeReferenceSyntax(t *testing.T) {
	valid := `predicates:
  test.ready:
    parameters: [{ue: string}]
    evidence:
      requires:
        ue: [ca::ue.registration-state]
    evaluator: {type: cel, expression: 'ue != ""'}
  test.complete:
    parameters: [{result: ca::urn:example:result:v1}]
    evaluator: {type: cel, expression: "result.ready == true"}
`
	request := testRequest(testContract, testProcedure, valid, testSchema)
	for _, diagnostic := range Check(request) {
		if diagnostic.Code == "QUALIFIED_REFERENCE_INVALID" {
			t.Fatalf("valid guarantee reference rejected: %+v", diagnostic)
		}
	}

	invalid := replaceAll(valid, "ca::ue.registration-state", "ue.registration-state")
	request = testRequest(testContract, testProcedure, invalid, testSchema)
	if !hasCode(Check(request), "QUALIFIED_REFERENCE_INVALID") {
		t.Fatalf("unqualified guarantee reference accepted")
	}
}
