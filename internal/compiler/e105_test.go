package compiler

// E105 conformance (identifier / qualified reference / package path variants).
// Variant 5 (LocalScriptPath) is covered by the Script path tests; this fixture
// set covers the PredicateId, Alias, ExportId and QualifiedSchemaId variants.

import "testing"

func e105Fixture(t *testing.T, mutate func(*testing.T, *wp3aFixture)) BuildRequest {
	t.Helper()
	fixture := wp3aBaseFixture(t)
	mutate(t, &fixture)
	return fixture.request(t)
}

func TestE105IdentifierAndQualifiedReferenceVariants(t *testing.T) {
	cases := []struct {
		name     string
		expected string
		mutate   func(*testing.T, *wp3aFixture)
	}{
		{
			name:     "PredicateId foo..bar",
			expected: "IDENTIFIER_INVALID",
			mutate: func(t *testing.T, fixture *wp3aFixture) {
				fixture.replaceFile(wp3aPredicatesPath, "  example.ue.registered:", "  foo..bar:")
			},
		},
		{
			name:     "Alias ca:demo",
			expected: "IDENTIFIER_INVALID",
			mutate: func(t *testing.T, fixture *wp3aFixture) {
				fixture.replaceFile(wp3aContractPath, "  ca: urn:example:ca:tools", "  ca:demo: urn:example:ca:tools")
			},
		},
		{
			name:     "ExportId foo..bar inside a qualified callable reference",
			expected: "QUALIFIED_REFERENCE_INVALID",
			mutate: func(t *testing.T, fixture *wp3aFixture) {
				fixture.replaceFile(wp3aProcedurePath, "ca::ue.registration.current", "ca::foo..bar")
			},
		},
		{
			name:     "QualifiedSchemaId ca::relative/path",
			expected: "QUALIFIED_REFERENCE_INVALID",
			mutate: func(t *testing.T, fixture *wp3aFixture) {
				fixture.replaceFile(wp3aContractPath, "ca-context::urn:example:ca:invocation-context:v1", "ca-context::relative/path")
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			diagnostics := Check(e105Fixture(t, testCase.mutate))
			if !hasCode(diagnostics, testCase.expected) {
				t.Fatalf("expected %s for %s, got %+v", testCase.expected, testCase.name, diagnostics)
			}
		})
	}
}
