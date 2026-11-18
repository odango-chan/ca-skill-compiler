package compiler

// Predicate parameter name grammar (G.1 Parameter name = Name = Identifier).

import (
	"strings"
	"testing"
)

func TestPredicateParameterNameGrammar(t *testing.T) {
	valid := testRequest(testContract, testProcedure, testPredicates, testSchema)
	if hasCode(Check(valid), "IDENTIFIER_INVALID") {
		t.Fatalf("valid Predicate parameter names rejected: %+v", Check(valid))
	}
	invalid := strings.Replace(testPredicates, "- predicate: test.ready", "- predicate: test.ready", 1)
	invalid = strings.Replace(invalid, "{ue: string}", "{1bad: string}", 1)
	diagnostics := Check(testRequest(testContract, testProcedure, invalid, testSchema))
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "IDENTIFIER_INVALID" && diagnostic.Path != "" && diagnostic.Line > 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("invalid Predicate parameter name accepted: %+v", diagnostics)
	}
}
