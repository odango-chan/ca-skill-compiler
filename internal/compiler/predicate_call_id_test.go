package compiler

// PredicateCall PredicateId (DottedId) conformance.

import (
	"strings"
	"testing"
)

func TestPredicateCallIdGrammar(t *testing.T) {
	valid := v4TestRequest(v4TestContract, v4TestProcedure, v4TestPredicates, v4TestSchema)
	if v4HasCode(Check(valid), "IDENTIFIER_INVALID") {
		t.Fatalf("valid PredicateId rejected: %+v", Check(valid))
	}
	invalid := strings.Replace(v4TestProcedure, "predicate: test.ready", "predicate: \"bad predicate\"", 1)
	diagnostics := Check(v4TestRequest(v4TestContract, invalid, v4TestPredicates, v4TestSchema))
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
