package compiler

// E01/E69/E98 static conformance subset.

import (
	"strings"
	"testing"
)

func TestE01FullBaselineStaticLink(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	req := wp3cRequest(t, fixture, wp3cBaseBundle(t))
	for _, diagnostic := range Check(req) {
		if diagnostic.Severity == "error" && strings.HasPrefix(diagnostic.Code, "DEPENDENCY_") {
			t.Fatalf("E01 baseline produced a dependency linking error: %+v", diagnostic)
		}
		if diagnostic.Severity == "error" && (diagnostic.Code == "EFFECT_MISMATCH" || diagnostic.Code == "CAPABILITY_NOT_GRANTED") {
			t.Fatalf("E01 baseline produced an authorization error: %+v", diagnostic)
		}
	}
}

func TestE69MutationMustRouteOutcomeUnknown(t *testing.T) {
	procedure := replaceAll(eRouteMutationProcedure, "        OUTCOME_UNKNOWN: UNRESOLVED\n", "")
	req := eRouteFixture(t, reconcileCatalog, procedure)
	if got := Check(req); !v4HasCode(got, "ROUTE_OUTCOME_MISSING") {
		t.Fatalf("Mutation without OUTCOME_UNKNOWN route accepted: %+v", got)
	}
}

func TestE98GuaranteePathMustCoverReferencedValue(t *testing.T) {
	// provides at output.result does not cover the whole output value.
	fixture := wp3aBaseFixture(t)
	fixture.replaceSnapshotFile("ca", "tool-catalog.yaml", "              path: output", "              path: output.result")
	fixture.refreshSnapshotDigests(t)
	if got := Check(fixture.request(t)); !v4HasCode(got, "PREDICATE_EVIDENCE_UNSATISFIABLE") {
		t.Fatalf("narrow guarantee path bound to a wider value accepted: %+v", got)
	}
}
