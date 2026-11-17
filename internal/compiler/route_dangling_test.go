package compiler

// WP5 slice 5: a route value that is neither a terminal nor an existing Node is
// the frozen CONTROL_FLOW_DANGLING identity (no invented route-target code).

import "testing"

func TestDanglingRouteTargetUsesFrozenDiagnostic(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "        FAILED: FAILED", "        FAILED: FAILED_TYPO")
	request := fixture.request(t)
	diagnostics := Check(request)
	if !v4HasCode(diagnostics, "CONTROL_FLOW_DANGLING") {
		t.Fatalf("dangling route target not reported: %+v", diagnostics)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "ROUTE_TARGET_INVALID" {
			t.Fatalf("non-frozen route-target diagnostic emitted: %+v", diagnostic)
		}
	}
}
