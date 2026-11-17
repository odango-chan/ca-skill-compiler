package compiler

// Qualified callable reference conformance (Alias::ExportId).

import (
	"strings"
	"testing"
)

func TestQualifiedCallableReference(t *testing.T) {
	valid := v4TestRequest(v4TestContract, v4TestProcedure, v4TestPredicates, v4TestSchema)
	if v4HasCode(Check(valid), "QUALIFIED_REFERENCE_INVALID") {
		t.Fatalf("valid tool reference rejected: %+v", Check(valid))
	}
	cases := map[string]string{
		"no alias":     "tool: ue.registration.current",
		"empty export": "tool: \"ca::\"",
		"bad alias":    "tool: 1ca::ue.registration.current",
		"single colon": "tool: ca:ue.registration.current",
	}
	for name, replacement := range cases {
		procedure := strings.Replace(v4TestProcedure, "tool: ca::example.session.read", replacement, 1)
		request := v4TestRequest(v4TestContract, procedure, v4TestPredicates, v4TestSchema)
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
