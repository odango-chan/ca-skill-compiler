package compiler

// WP5 slice 3 conformance: §22.1 route-key eligibility.

import "testing"

func TestRouteEligibilityPositive(t *testing.T) {
	request := wp3aBaseFixture(t).request(t)
	if hasCode(Check(request), "ROUTE_OUTCOME_UNKNOWN") || hasCode(Check(request), "ROUTE_OUTCOME_MISSING") {
		t.Fatalf("valid route keys rejected: %+v", Check(request))
	}
}

func TestRouteEligibilityOwnerFailedRequired(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "        FAILED: FAILED\n        UNKNOWN: UNRESOLVED", "        UNKNOWN: UNRESOLVED")
	request := fixture.request(t)
	if !hasCode(Check(request), "ROUTE_OUTCOME_MISSING") {
		t.Fatalf("missing Owner-declared FAILED route accepted: %+v", Check(request))
	}
}

func TestRouteEligibilityCommittedOnReadRejected(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "        FAILED: FAILED\n        UNKNOWN: UNRESOLVED", "        FAILED: FAILED\n        UNKNOWN: UNRESOLVED\n        COMMITTED: SAFE_STOPPED")
	request := fixture.request(t)
	if !hasCode(Check(request), "ROUTE_OUTCOME_UNKNOWN") {
		t.Fatalf("COMMITTED route on Observation accepted: %+v", Check(request))
	}
}
