package compiler

// WP7 slice 3 conformance: static Predicate Evidence satisfiability.

import "testing"

func TestPredicateEvidenceSatisfiedByProducerGuarantee(t *testing.T) {
	for _, diagnostic := range Check(wp3aBaseFixture(t).request(t)) {
		if diagnostic.Code == "PREDICATE_EVIDENCE_UNSATISFIABLE" {
			t.Fatalf("producer with published guarantee rejected: %+v", diagnostic)
		}
	}
}

func TestPredicateEvidenceUnsatisfiedWithoutPublishedGuarantee(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceSnapshotFile("ca", "tool-catalog.yaml", "      guarantees:\n        evidence:\n          provides:\n            - guarantee: ue.registration-state\n              path: output\n", "")
	fixture.refreshSnapshotDigests(t)
	request := fixture.request(t)
	if !hasCode(Check(request), "PREDICATE_EVIDENCE_UNSATISFIABLE") {
		t.Fatalf("producer without published guarantee accepted: %+v", Check(request))
	}
}

func TestPredicateEvidenceRequestDataUnsatisfiable(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "            - inspect-registration.observation.output", "            - context.ueRef")
	request := fixture.request(t)
	if !hasCode(Check(request), "PREDICATE_EVIDENCE_UNSATISFIABLE") {
		t.Fatalf("Request/Host data satisfying a declared guarantee accepted: %+v", Check(request))
	}
}
