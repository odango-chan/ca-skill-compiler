package compiler

// E119 conformance subset: Pre/Postcondition structural negatives.

import "testing"

func TestE119PredicateStageStructuralNegatives(t *testing.T) {
	emptyList := replaceOnceString(testProcedure,
		"      predicates:\n        - predicate: test.complete\n          arguments: [read.observation.output]",
		"      predicates: []")
	if got := Check(testRequest(testContract, emptyList, testPredicates, testSchema)); !hasCode(got, "PREDICATE_GROUP_INVALID") {
		t.Fatalf("empty postcondition predicates list accepted: %+v", got)
	}

	missingArguments := replaceOnceString(testProcedure,
		"        - predicate: test.complete\n          arguments: [read.observation.output]",
		"        - predicate: test.complete")
	if got := Check(testRequest(testContract, missingArguments, testPredicates, testSchema)); !hasCode(got, "PREDICATE_GROUP_INVALID") {
		t.Fatalf("predicate member without arguments not classified structurally: %+v", got)
	}

	memberRoutes := replaceOnceString(testProcedure,
		"        - predicate: test.complete\n          arguments: [read.observation.output]",
		"        - predicate: test.complete\n          arguments: [read.observation.output]\n          routes: {PASS: SUCCEEDED}")
	if got := Check(testRequest(testContract, memberRoutes, testPredicates, testSchema)); !hasCode(got, "UNKNOWN_FIELD") {
		t.Fatalf("predicate member carrying routes not reported: %+v", got)
	}

	postconditionGoal := replaceOnceString(testProcedure,
		"    postcondition:\n      predicates:",
		"    postcondition:\n      goal: pdu_session.initial_request_fulfilled\n      predicates:")
	if got := Check(testRequest(testContract, postconditionGoal, testPredicates, testSchema)); !hasCode(got, "PREDICATE_GROUP_INVALID") {
		t.Fatalf("removed Postcondition goal field not classified structurally: %+v", got)
	}
}
