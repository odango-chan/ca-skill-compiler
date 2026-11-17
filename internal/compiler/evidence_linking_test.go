package compiler

// WP7 slice 1 conformance: Evidence requirement parameter binding.

import "testing"

func TestWP7EvidenceRequirementPositive(t *testing.T) {
	request := wp3aBaseFixture(t).request(t)
	if v4HasCode(Check(request), "PREDICATE_EVIDENCE_REQUIRED") {
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
