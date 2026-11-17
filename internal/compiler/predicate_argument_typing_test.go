package compiler

// WP6 slice 6 conformance: PredicateCall positional argument F.5 typing.

import "testing"

func TestWP6PredicateArgumentTypesPositive(t *testing.T) {
	request := wp3aBaseFixture(t).request(t)
	diagnostics := Check(request)
	if v4HasCode(diagnostics, "PREDICATE_TYPE_MISMATCH") || v4HasCode(diagnostics, "INPUT_NULLABILITY_MISMATCH") {
		t.Fatalf("assignable Predicate argument rejected: %+v", diagnostics)
	}
}

func TestWP6PredicateArgumentTypeMismatch(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aPredicatesPath, "ca::urn:example:ca:registration:v1", "ca::urn:example:ca:registration-query:v1")
	diagnostics := Check(fixture.request(t))
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "PREDICATE_TYPE_MISMATCH" && diagnostic.Path == wp3aProcedurePath {
			found = true
		}
	}
	if !found {
		t.Fatalf("incompatible Predicate argument accepted: %+v", diagnostics)
	}
}

func TestWP6PredicateArgumentPrimitiveMismatch(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "            - inspect-registration.observation.output", "            - context.ueRef")
	diagnostics := Check(fixture.request(t))
	if !v4HasCode(diagnostics, "PREDICATE_TYPE_MISMATCH") {
		t.Fatalf("string argument accepted for object parameter: %+v", diagnostics)
	}
}
