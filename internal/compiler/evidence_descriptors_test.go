package compiler

// Evidence Descriptor positive-path linking tests (F.3 / §17 / G.3.1).

import (
	"strings"
	"testing"
)

func evidenceFixtureRequest(t *testing.T) BuildRequest {
	t.Helper()
	return wp3aBaseFixture(t).request(t)
}

func expectEvidenceLinkDiagnostic(t *testing.T, req BuildRequest) {
	t.Helper()
	if !hasCode(Check(req), "EVIDENCE_GUARANTEE_LINK_INVALID") {
		t.Fatalf("expected EVIDENCE_GUARANTEE_LINK_INVALID, got %+v", Check(req))
	}
}

func withoutSnapshotFile(fixture *wp3aFixture, dependencyID, path string) {
	for index := range fixture.dependencies {
		if fixture.dependencies[index].DependencyID != dependencyID {
			continue
		}
		kept := make([]File, 0, len(fixture.dependencies[index].Files))
		for _, file := range fixture.dependencies[index].Files {
			if file.Path != path {
				kept = append(kept, file)
			}
		}
		fixture.dependencies[index].Files = kept
	}
}

const secondEvidenceDescriptor = `ue.registration-freshness:
  authority: example-ue-owner
  fact: ue.registration-freshness
  producers:
    - owner: example-ue-owner
      kind: observation
      id: ue.registration.current
      outcome: SUCCEEDED
      path: output
  scope:
    subject: {record: scope.subject, equals: query.ueRef}
  correlation:
    invocation: {record: invocationId, equals: execution.invocationId}
  factValue: observedAt
  selection: {mode: latest, orderBy: observedAt}
  freshness: {timestamp: observedAt, maxAgeMs: 10000}
  sourceIdentity: {domain: "urn:example:record-domain:smf-events", path: record.sourceRecordId}
`

func independencePredicates(sourceDomain string) string {
	base := `predicates:
  example.ue.registered:
    parameters:
      - registration: ca::urn:example:ca:registration:v1
    evidence:
      requires:
        registration: [ca::ue.registration-state, ca::ue.registration-freshness]
      independence:
        - members:
            - parameter: registration
              guarantee: ca::ue.registration-state
            - parameter: registration
              guarantee: ca::ue.registration-freshness
    evaluator:
      type: cel
      expression: >-
        registration.state == "REGISTERED" &&
        registration.accessState == "ACTIVE"
`
	if sourceDomain == "" {
		return base
	}
	return strings.Replace(base,
		"  sourceIdentity: {domain: \"urn:example:record-domain:smf-events\", path: record.sourceRecordId}\n",
		"  sourceIdentity: {domain: \""+sourceDomain+"\", path: record.sourceRecordId}\n", 1)
}

// TestEvidenceDescriptorPositiveLink proves the base trusted fixture links its
// guarantee descriptor and producer without a link diagnostic.
func TestEvidenceDescriptorPositiveLink(t *testing.T) {
	diagnostics := Check(evidenceFixtureRequest(t))
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "EVIDENCE_GUARANTEE_LINK_INVALID" {
			t.Fatalf("valid Evidence Descriptor rejected: %+v", diagnostic)
		}
	}
}

// TestEvidenceDescriptorMissingRejected proves a referenced guarantee without a
// descriptor export fails closed.
func TestEvidenceDescriptorMissingRejected(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	withoutSnapshotFile(&fixture, "ca", "evidence-guarantees.yaml")
	fixture.refreshSnapshotDigests(t)
	expectEvidenceLinkDiagnostic(t, fixture.request(t))
}

// TestEvidenceDescriptorProducerMismatchRejected proves the descriptor Producer
// must match an exact trusted ToolCatalog callable and provides.path.
func TestEvidenceDescriptorProducerMismatchRejected(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceSnapshotFile("ca", "evidence-guarantees.yaml",
		"id: ue.registration.current", "id: ue.registration.other")
	fixture.refreshSnapshotDigests(t)
	expectEvidenceLinkDiagnostic(t, fixture.request(t))
}

// TestEvidenceIndependenceCrossDomainWithoutTrustRejected proves independence
// across distinct source domains requires the trusted IndependentDomains input.
func TestEvidenceIndependenceCrossDomainWithoutTrustRejected(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.addSnapshotFile("ca", "freshness-guarantees.yaml", []byte(secondEvidenceDescriptor))
	fixture.replaceSnapshotFile("ca", "evidence-guarantees.yaml",
		"  freshness: {timestamp: observedAt, maxAgeMs: 10000}\n",
		"  freshness: {timestamp: observedAt, maxAgeMs: 10000}\n  sourceIdentity: {domain: \"urn:example:record-domain:ran\", path: record.sourceRecordId}\n")
	fixture.files[wp3aPredicatesPath] = []byte(independencePredicates(""))
	fixture.refreshSnapshotDigests(t)
	expectEvidenceLinkDiagnostic(t, fixture.request(t))
}
