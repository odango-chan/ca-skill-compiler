package compiler

// WP6 slice 4 conformance: Tool argument key/required checking.

import "testing"

func TestWP6ArgumentKeysPositive(t *testing.T) {
	request := wp3aBaseFixture(t).request(t)
	diagnostics := Check(request)
	if hasCode(diagnostics, "INPUT_REQUIRED_MISSING") || hasCode(diagnostics, "INPUT_TYPE_MISMATCH") {
		t.Fatalf("valid Tool arguments rejected: %+v", diagnostics)
	}
}

func TestWP6ArgumentRequiredMissing(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "      arguments:\n        ueRef: context.ueRef", "      arguments: {}")
	diagnostics := Check(fixture.request(t))
	if !hasCode(diagnostics, "INPUT_REQUIRED_MISSING") {
		t.Fatalf("missing required Tool argument accepted: %+v", diagnostics)
	}
}

func TestWP6ArgumentUnknownKey(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "        ueRef: context.ueRef", "        ueRef: context.ueRef\n        unexpected: 'x'")
	diagnostics := Check(fixture.request(t))
	if !hasCode(diagnostics, "INPUT_TYPE_MISMATCH") {
		t.Fatalf("unknown Tool argument accepted: %+v", diagnostics)
	}
}
