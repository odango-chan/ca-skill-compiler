package compiler

// E119 conformance subset: Pre/Postcondition structural negatives.

import "testing"

func TestE119PredicateStageStructuralNegatives(t *testing.T) {
	emptyList := replaceOnceString(v4TestProcedure,
		"      predicates:\n        - predicate: test.complete\n          arguments: [read.observation.output]",
		"      predicates: []")
	if got := Check(v4TestRequest(v4TestContract, emptyList, v4TestPredicates, v4TestSchema)); !v4HasCode(got, "PREDICATE_GROUP_INVALID") {
		t.Fatalf("empty postcondition predicates list accepted: %+v", got)
	}

	missingArguments := replaceOnceString(v4TestProcedure,
		"        - predicate: test.complete\n          arguments: [read.observation.output]",
		"        - predicate: test.complete")
	if got := Check(v4TestRequest(v4TestContract, missingArguments, v4TestPredicates, v4TestSchema)); !v4HasCode(got, "PREDICATE_GROUP_INVALID") {
		t.Fatalf("predicate member without arguments not classified structurally: %+v", got)
	}

	memberRoutes := replaceOnceString(v4TestProcedure,
		"        - predicate: test.complete\n          arguments: [read.observation.output]",
		"        - predicate: test.complete\n          arguments: [read.observation.output]\n          routes: {PASS: SUCCEEDED}")
	if got := Check(v4TestRequest(v4TestContract, memberRoutes, v4TestPredicates, v4TestSchema)); !v4HasCode(got, "UNKNOWN_FIELD") {
		t.Fatalf("predicate member carrying routes not reported: %+v", got)
	}

	postconditionGoal := replaceOnceString(v4TestProcedure,
		"    postcondition:\n      predicates:",
		"    postcondition:\n      goal: pdu_session.initial_request_fulfilled\n      predicates:")
	if got := Check(v4TestRequest(v4TestContract, postconditionGoal, v4TestPredicates, v4TestSchema)); !v4HasCode(got, "PREDICATE_GROUP_INVALID") {
		t.Fatalf("removed Postcondition goal field not classified structurally: %+v", got)
	}
}
