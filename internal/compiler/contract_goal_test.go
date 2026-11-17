package compiler

// Contract goal identifier conformance (G.1 GoalId = DottedId).

import (
	"strings"
	"testing"
)

func TestContractGoalIdentifier(t *testing.T) {
	valid := strings.Replace(v4TestContract, "goal: pdu_session.initial_request_fulfilled", "goal: pdu.session_ok", 1)
	if v4HasCode(Check(v4TestRequest(valid, v4TestProcedure, v4TestPredicates, v4TestSchema)), "IDENTIFIER_INVALID") {
		t.Fatalf("valid GoalId rejected")
	}
	invalid := strings.Replace(v4TestContract, "goal: pdu_session.initial_request_fulfilled", "goal: \"bad goal\"", 1)
	diagnostics := Check(v4TestRequest(invalid, v4TestProcedure, v4TestPredicates, v4TestSchema))
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "IDENTIFIER_INVALID" && diagnostic.Path != "" && diagnostic.Line > 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("invalid GoalId accepted: %+v", diagnostics)
	}
}
