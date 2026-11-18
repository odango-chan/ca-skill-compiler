package compiler

// WP5 slice 2 conformance: stage route obligations.

import (
	"strings"
	"testing"
)

func TestWP5StageRoutesPositive(t *testing.T) {
	request := testRequest(testContract, testProcedure, testPredicates, testSchema)
	diagnostics := Check(request)
	for _, code := range []string{"ROUTE_NORMAL_OUTCOME_NOT_ALLOWED", "ROUTE_OUTCOME_MISSING", "ROUTE_OUTCOME_UNKNOWN"} {
		if hasCode(diagnostics, code) {
			t.Fatalf("positive Procedure emitted %s: %+v", code, diagnostics)
		}
	}
}

func TestWP5NonLastNormalOutcomeRejected(t *testing.T) {
	procedure := strings.Replace(testProcedure, `  read:
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
	request := testRequest(testContract, procedure, testPredicates, testSchema)
	if !hasCode(Check(request), "ROUTE_NORMAL_OUTCOME_NOT_ALLOWED") {
		t.Fatalf("non-last normal Outcome route accepted: %+v", Check(request))
	}
}

func TestWP5LastStageMissingNormalRoute(t *testing.T) {
	procedure := strings.Replace(testProcedure, `        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED`, `        FAIL: FAILED
        UNKNOWN: UNRESOLVED`, 1)
	request := testRequest(testContract, procedure, testPredicates, testSchema)
	if !hasCode(Check(request), "ROUTE_OUTCOME_MISSING") {
		t.Fatalf("last stage without normal Outcome route accepted: %+v", Check(request))
	}
}

func TestWP5PredicateStageInvalidRouteKey(t *testing.T) {
	procedure := strings.Replace(testProcedure, `        PASS: read
        FAIL: FAILED
        UNKNOWN: UNRESOLVED`, `        PASS: read
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
        RESULT: FAILED`, 1)
	request := testRequest(testContract, procedure, testPredicates, testSchema)
	if !hasCode(Check(request), "ROUTE_OUTCOME_UNKNOWN") {
		t.Fatalf("ineligible predicate route key accepted: %+v", Check(request))
	}
}
