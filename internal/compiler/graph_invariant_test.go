package compiler

// E99-style internal graph invariant: an inconsistent analysis graph whose
// reachable non-entry stage has no reachable incoming edge is detected as
// REACHABLE_STAGE_NO_INCOMING_EDGE. Author sources cannot construct this state.

import "testing"

func TestReachableStageWithoutIncomingEdgeInvariant(t *testing.T) {
	graph := procedureGraph{
		edges:    map[string][]string{"start": {"second"}, "second": {}},
		nodeByID: map[string]*V1Alpha4ProcedureNodeAST{},
		order:    []string{"start", "second", "orphan"},
	}
	reachable := map[string]bool{"start": true, "second": true, "orphan": true}
	missing := procedureReachableStagesWithoutIncomingEdge(graph, "start", reachable)
	if len(missing) != 1 || missing[0] != "orphan" {
		t.Fatalf("orphan reachable stage not detected: %v", missing)
	}

	consistent := map[string]bool{"start": true, "second": true}
	if missing := procedureReachableStagesWithoutIncomingEdge(graph, "start", consistent); len(missing) != 0 {
		t.Fatalf("consistent graph flagged: %v", missing)
	}

	// The normal pipeline must never emit the internal-invariant diagnostic.
	request := wp3aBaseFixture(t).request(t)
	for _, diagnostic := range Check(request) {
		if diagnostic.Code == "REACHABLE_STAGE_NO_INCOMING_EDGE" {
			t.Fatalf("valid Procedure triggered an internal invariant diagnostic: %+v", diagnostic)
		}
	}
}
