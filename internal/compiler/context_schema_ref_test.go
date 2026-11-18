package compiler

// Contract.contextSchema QualifiedSchemaId conformance.

import (
	"strings"
	"testing"
)

func TestContextSchemaQualifiedReference(t *testing.T) {
	valid := wp3aBaseFixture(t).request(t)
	if hasCode(Check(valid), "QUALIFIED_REFERENCE_INVALID") {
		t.Fatalf("valid contextSchema rejected: %+v", Check(valid))
	}
	cases := map[string]string{
		"no alias":     "contextSchema: urn:example:ca:invocation-context:v1",
		"single colon": "contextSchema: ca:urn:example:ca:invocation-context:v1",
		"empty alias":  "contextSchema: \"::urn:example:ca:invocation-context:v1\"",
		"bad alias":    "contextSchema: 1ca::urn:example:ca:invocation-context:v1",
		"empty uri":    "contextSchema: \"ca::\"",
	}
	for name, replacement := range cases {
		fixture := wp3aBaseFixture(t)
		original := "contextSchema: ca-context::urn:example:ca:invocation-context:v1"
		fixture.replaceFile(wp3aContractPath, original, replacement)
		diagnostics := Check(fixture.request(t))
		found := false
		for _, diagnostic := range diagnostics {
			if diagnostic.Code == "QUALIFIED_REFERENCE_INVALID" && diagnostic.Phase == "C_STATIC" {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s: expected QUALIFIED_REFERENCE_INVALID: %+v", name, diagnostics)
		}
		_ = strings.TrimSpace(name)
	}
}
