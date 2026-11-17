package compiler

// WP8 slice 3 conformance: retry family rules.

import "testing"

func TestRetryFamilyViolations(t *testing.T) {
	edges := []retryEdge{
		{nodeID: "retry", target: "mutate"},
	}
	effects := map[string]string{"mutate": "MUTATION", "retry": "MUTATION"}
	if violations := retryFamilyViolations(edges, effects); len(violations) != 0 {
		t.Fatalf("valid retry family rejected: %+v", violations)
	}

	effects = map[string]string{"mutate": "READ", "retry": "MUTATION"}
	if violations := retryFamilyViolations(edges, effects); len(violations) != 1 {
		t.Fatalf("retry of READ callable accepted: %+v", violations)
	}

	effects = map[string]string{"mutate": "MUTATION", "retry": "READ"}
	if violations := retryFamilyViolations(edges, effects); len(violations) != 1 {
		t.Fatalf("README retry Node accepted: %+v", violations)
	}

	effects = map[string]string{"mutate": "MUTATION", "retry": "MUTATION"}
	if violations := retryFamilyViolations([]retryEdge{
		{nodeID: "retryA", target: "mutate"},
		{nodeID: "retryB", target: "mutate"},
	}, effects); len(violations) != 1 {
		t.Fatalf("second retry dispatch of one Mutation family accepted: %+v", violations)
	}

	if violations := retryFamilyViolations([]retryEdge{
		{nodeID: "retry", target: "mutate", reconcileTarget: "example.session.reconcile"},
	}, effects); len(violations) != 1 {
		t.Fatalf("retry Node using the Reconcile READ callable accepted: %+v", violations)
	}
}
