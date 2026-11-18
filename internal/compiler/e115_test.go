package compiler

// E115 conformance subset: Pre/Postcondition single-member and two-member
// `predicates` lists share the same frozen author shape and link structurally.

import "testing"

func TestE115PredicateStageMemberCounts(t *testing.T) {
	single := testRequest(testContract, testProcedure, testPredicates, testSchema)
	if hasCode(Check(single), "PREDICATE_GROUP_INVALID") {
		t.Fatalf("single-member predicate stage rejected: %+v", Check(single))
	}

	twoMembers := replaceAll(testProcedure,
		"        - predicate: test.complete\n          arguments: [read.observation.output]",
		"        - predicate: test.complete\n          arguments: [read.observation.output]\n        - predicate: test.ready\n          arguments: [context.ueRef]")
	request := testRequest(testContract, twoMembers, testPredicates, testSchema)
	diagnostics := Check(request)
	if hasCode(diagnostics, "PREDICATE_GROUP_INVALID") {
		t.Fatalf("two-member predicate stage rejected: %+v", diagnostics)
	}
	if hasCode(diagnostics, "PREDICATE_CALL_ARGUMENT_INVALID") {
		t.Fatalf("two-member predicate stage arguments rejected: %+v", diagnostics)
	}
}
