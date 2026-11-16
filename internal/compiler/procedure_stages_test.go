package compiler

// WP5 slice 2 conformance: stage route obligations.

import (
	"strings"
	"testing"
)

func TestWP5StageRoutesPositive(t *testing.T) {
	request := v4TestRequest(v4TestContract, v4TestProcedure, v4TestPredicates, v4TestSchema)
	diagnostics := Check(request)
	for _, code := range []string{"ROUTE_NORMAL_OUTCOME_NOT_ALLOWED", "ROUTE_OUTCOME_MISSING", "ROUTE_OUTCOME_UNKNOWN"} {
		if v4HasCode(diagnostics, code) {
			t.Fatalf("positive Procedure emitted %s: %+v", code, diagnostics)
		}
	}
}

func TestWP5NonLastNormalOutcomeRejected(t *testing.T) {
	procedure := strings.Replace(v4TestProcedure, `  read:
    observation:
      tool: ca::example.session.read
      arguments:
        ueRef: context.ueRef
      routes:
        SUCCEEDED: final
        FAILED: FAILED`,
		`  read:
    observation:
      tool: ca::example.session.read
      arguments:
        ueRef: context.ueRef
      routes:
        SUCCEEDED: final
        FAILED: FAILED
    postcondition:
      predicates:
        - predicate: test.complete
          arguments: [read.observation.output]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED`, 1)
	request := v4TestRequest(v4TestContract, procedure, v4TestPredicates, v4TestSchema)
	if !v4HasCode(Check(request), "ROUTE_NORMAL_OUTCOME_NOT_ALLOWED") {
		t.Fatalf("non-last normal Outcome route accepted: %+v", Check(request))
	}
}

func TestWP5LastStageMissingNormalRoute(t *testing.T) {
	procedure := strings.Replace(v4TestProcedure, `        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED`, `        FAIL: FAILED
        UNKNOWN: UNRESOLVED`, 1)
	request := v4TestRequest(v4TestContract, procedure, v4TestPredicates, v4TestSchema)
	if !v4HasCode(Check(request), "ROUTE_OUTCOME_MISSING") {
		t.Fatalf("last stage without normal Outcome route accepted: %+v", Check(request))
	}
}

func TestWP5PredicateStageInvalidRouteKey(t *testing.T) {
	procedure := strings.Replace(v4TestProcedure, `        PASS: read
        FAIL: FAILED
        UNKNOWN: UNRESOLVED`, `        PASS: read
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
        RESULT: FAILED`, 1)
	request := v4TestRequest(v4TestContract, procedure, v4TestPredicates, v4TestSchema)
	if !v4HasCode(Check(request), "ROUTE_OUTCOME_UNKNOWN") {
		t.Fatalf("ineligible predicate route key accepted: %+v", Check(request))
	}
}
