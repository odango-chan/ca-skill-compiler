package compiler

// E115 conformance subset: Pre/Postcondition single-member and two-member
// `predicates` lists share the same frozen author shape and link structurally.

import "testing"

func TestE115PredicateStageMemberCounts(t *testing.T) {
	single := v4TestRequest(v4TestContract, v4TestProcedure, v4TestPredicates, v4TestSchema)
	if v4HasCode(Check(single), "PREDICATE_GROUP_INVALID") {
		t.Fatalf("single-member predicate stage rejected: %+v", Check(single))
	}

	twoMembers := replaceAll(v4TestProcedure,
		"        - predicate: test.complete\n          arguments: [read.observation.output]",
		"        - predicate: test.complete\n          arguments: [read.observation.output]\n        - predicate: test.ready\n          arguments: [context.ueRef]")
	request := v4TestRequest(v4TestContract, twoMembers, v4TestPredicates, v4TestSchema)
	diagnostics := Check(request)
	if v4HasCode(diagnostics, "PREDICATE_GROUP_INVALID") {
		t.Fatalf("two-member predicate stage rejected: %+v", diagnostics)
	}
	if v4HasCode(diagnostics, "PREDICATE_CALL_ARGUMENT_INVALID") {
		t.Fatalf("two-member predicate stage arguments rejected: %+v", diagnostics)
	}
}
