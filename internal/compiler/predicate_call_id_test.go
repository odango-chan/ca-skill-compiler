package compiler

// PredicateCall PredicateId (DottedId) conformance.

import (
	"strings"
	"testing"
)

func TestPredicateCallIdGrammar(t *testing.T) {
	valid := testRequest(testContract, testProcedure, testPredicates, testSchema)
	if hasCode(Check(valid), "IDENTIFIER_INVALID") {
		t.Fatalf("valid PredicateId rejected: %+v", Check(valid))
	}
	invalid := strings.Replace(testProcedure, "predicate: test.ready", "predicate: \"bad predicate\"", 1)
	diagnostics := Check(testRequest(testContract, invalid, testPredicates, testSchema))
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "IDENTIFIER_INVALID" && diagnostic.Phase == "C_STATIC" && diagnostic.Line > 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("invalid PredicateId accepted: %+v", diagnostics)
	}
}
