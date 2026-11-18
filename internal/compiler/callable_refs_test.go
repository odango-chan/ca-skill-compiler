package compiler

// Qualified callable reference conformance (Alias::ExportId).

import (
	"strings"
	"testing"
)

func TestQualifiedCallableReference(t *testing.T) {
	valid := testRequest(testContract, testProcedure, testPredicates, testSchema)
	if hasCode(Check(valid), "QUALIFIED_REFERENCE_INVALID") {
		t.Fatalf("valid tool reference rejected: %+v", Check(valid))
	}
	cases := map[string]string{
		"no alias":     "tool: ue.registration.current",
		"empty export": "tool: \"ca::\"",
		"bad alias":    "tool: 1ca::ue.registration.current",
		"single colon": "tool: ca:ue.registration.current",
	}
	for name, replacement := range cases {
		procedure := strings.Replace(testProcedure, "tool: ca::example.session.read", replacement, 1)
		request := testRequest(testContract, procedure, testPredicates, testSchema)
		diagnostics := Check(request)
		found := false
		for _, diagnostic := range diagnostics {
			if diagnostic.Code == "QUALIFIED_REFERENCE_INVALID" && diagnostic.Phase == "C_STATIC" {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s: expected QUALIFIED_REFERENCE_INVALID: %+v", name, diagnostics)
		}
	}
}
