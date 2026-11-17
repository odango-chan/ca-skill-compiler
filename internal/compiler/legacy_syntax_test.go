package compiler

// Legacy-syntax diagnostic priority conformance (CA-SL §24.1 errata).

import "testing"

func TestLegacyWaitSyntaxPriority(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "  inspect-registration:\n    observation:", "  inspect-registration:\n    wait: 1000\n    observation:")
	diagnostics := Check(fixture.request(t))
	if !v4HasCode(diagnostics, "UNSUPPORTED_WAIT") {
		t.Fatalf("Wait syntax not classified as UNSUPPORTED_WAIT: %+v", diagnostics)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "UNKNOWN_FIELD" {
			t.Fatalf("Wait syntax must not also be UNKNOWN_FIELD: %+v", diagnostic)
		}
	}
}

func TestLegacyV02SyntaxPriority(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "  inspect-registration:\n    observation:", "  inspect-registration:\n    verify:\n      predicates: []\n    observation:")
	diagnostics := Check(fixture.request(t))
	if !v4HasCode(diagnostics, "UNSUPPORTED_V02_SYNTAX") {
		t.Fatalf("legacy verify field not classified as UNSUPPORTED_V02_SYNTAX: %+v", diagnostics)
	}
}

func TestRemovedV1Alpha3FieldStaysUnknown(t *testing.T) {
	procedure := replaceOnceString(v4TestProcedure, "      tool: ca::example.session.read", "      tool: ca::example.session.read\n      target: ca::example.session.read")
	request := v4TestRequest(v4TestContract, procedure, v4TestPredicates, v4TestSchema)
	diagnostics := Check(request)
	if !v4HasCode(diagnostics, "UNKNOWN_FIELD") {
		t.Fatalf("removed v1alpha3 target field must stay UNKNOWN_FIELD: %+v", diagnostics)
	}
	if v4HasCode(diagnostics, "UNSUPPORTED_V02_SYNTAX") || v4HasCode(diagnostics, "UNSUPPORTED_WAIT") {
		t.Fatalf("removed v1alpha3 field misclassified as legacy syntax: %+v", diagnostics)
	}
}

func replaceOnceString(source, old, replacement string) string {
	for i := 0; i+len(old) <= len(source); i++ {
		if source[i:i+len(old)] == old {
			return source[:i] + replacement + source[i+len(old):]
		}
	}
	return source
}
