package compiler

// Regression coverage for the reviewed Evidence Descriptor fail-open defect.
//
// Every case below goes through the public v1alpha5 Check and Build entries with
// the existing trusted dependency fixture and CapabilityContracts bundle. The
// untouched fixture is the positive control; each malformed descriptor must be
// rejected with EVIDENCE_GUARANTEE_LINK_INVALID and a failed Build must not emit
// any deliverable artifact.

import "testing"

func evidenceRegressionRequest(t *testing.T, mutate func(*wp3aFixture)) BuildRequest {
	t.Helper()
	fixture := wp3aBaseFixture(t)
	if mutate != nil {
		mutate(&fixture)
	}
	fixture.refreshSnapshotDigests(t)
	req := wp3cRequest(t, fixture, wp3cBaseBundle(t))
	req.APIVersion = BuildRequestAPIVersion
	req.LanguageTarget = V1Alpha5Target
	return req
}

func assertEvidenceRejectedThroughPublicBuild(t *testing.T, req BuildRequest, wantMessage string) {
	t.Helper()
	check := Check(req)
	found := false
	for _, diagnostic := range check {
		if diagnostic.Code == "EVIDENCE_GUARANTEE_LINK_INVALID" {
			found = true
			if wantMessage != "" && !containsText(diagnostic.Message, wantMessage) {
				t.Fatalf("diagnostic %q does not contain %q", diagnostic.Message, wantMessage)
			}
		}
	}
	if !found {
		t.Fatalf("malformed Evidence Descriptor was accepted by Check: %#v", check)
	}
	result := Build(req)
	if result.Success {
		t.Fatalf("malformed Evidence Descriptor produced a successful Build")
	}
	if len(result.OutputFiles) != 0 {
		t.Fatalf("failed Build emitted deliverable output files: %v", len(result.OutputFiles))
	}
	if len(result.Diagnostics) == 0 {
		t.Fatalf("failed Build must report the descriptor diagnostic")
	}
}

func containsText(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func TestEvidenceDescriptorBaselinePositiveControlBuilds(t *testing.T) {
	req := evidenceRegressionRequest(t, nil)
	for _, diagnostic := range Check(req) {
		if diagnostic.Severity == "error" {
			t.Fatalf("baseline fixture must pass Check: %+v", diagnostic)
		}
	}
	result := Build(req)
	if !result.Success {
		t.Fatalf("baseline fixture must Build: %+v", result.Diagnostics)
	}
	if len(result.OutputFiles) == 0 {
		t.Fatal("baseline successful Build emitted no artifact files")
	}
}

func TestEvidenceDescriptorMalformedBindingRejected(t *testing.T) {
	req := evidenceRegressionRequest(t, func(fixture *wp3aFixture) {
		fixture.replaceSnapshotFile("ca", "evidence-guarantees.yaml",
			"invocation: {record: invocationId, equals: execution.invocationId}",
			"invocation: broken")
	})
	assertEvidenceRejectedThroughPublicBuild(t, req, "correlation.invocation must be a mapping")
}

func TestEvidenceDescriptorMalformedStructuresRejected(t *testing.T) {
	const descriptorPath = "ca"
	const descriptorFile = "evidence-guarantees.yaml"
	cases := []struct {
		name       string
		old, new   string
		wantPhrase string
	}{
		{
			name:       "correlation container not a mapping",
			old:        "  correlation:\n    invocation: {record: invocationId, equals: execution.invocationId}\n    producerInvocation: {record: producerInvocationId, equals: execution.producerInvocationId}\n",
			new:        "  correlation: broken\n",
			wantPhrase: "correlation must be a mapping",
		},
		{
			name:       "binding missing equals",
			old:        "    invocation: {record: invocationId, equals: execution.invocationId}\n",
			new:        "    invocation: {record: invocationId}\n",
			wantPhrase: "correlation.invocation.equals must be a non-empty string",
		},
		{
			name:       "binding equals wrong type",
			old:        "    invocation: {record: invocationId, equals: execution.invocationId}\n",
			new:        "    invocation: {record: invocationId, equals: 42}\n",
			wantPhrase: "correlation.invocation.equals must be a non-empty string",
		},
		{
			name:       "binding record not a static path",
			old:        "    invocation: {record: invocationId, equals: execution.invocationId}\n",
			new:        "    invocation: {record: 'not a path!', equals: execution.invocationId}\n",
			wantPhrase: "correlation.invocation.record must be a static record-relative field path",
		},
		{
			name:       "producer entry not a mapping",
			old:        "  producers:\n    - owner: example-ue-owner\n      kind: observation\n      id: ue.registration.current\n      outcome: SUCCEEDED\n      path: output\n",
			new:        "  producers:\n    - broken\n",
			wantPhrase: "producers[0] must be a mapping",
		},
		{
			name:       "selection not a mapping",
			old:        "  selection: {mode: latest, orderBy: observedAt}\n",
			new:        "  selection: broken\n",
			wantPhrase: "selection must be a mapping",
		},
		{
			name:       "freshness maxAgeMs not an integer",
			old:        "  freshness: {timestamp: observedAt, maxAgeMs: 10000}\n",
			new:        "  freshness: {timestamp: observedAt, maxAgeMs: \"10000\"}\n",
			wantPhrase: "freshness.maxAgeMs must be a positive integer",
		},
		{
			name:       "freshness container not a mapping",
			old:        "  freshness: {timestamp: observedAt, maxAgeMs: 10000}\n",
			new:        "  freshness: broken\n",
			wantPhrase: "freshness must be a mapping",
		},
		{
			name:       "factValue not a static path",
			old:        "  factValue: value\n",
			new:        "  factValue: 42\n",
			wantPhrase: "factValue must be a string",
		},
		{
			name:       "authority missing",
			old:        "  authority: example-ue-owner\n",
			new:        "",
			wantPhrase: "authority is required",
		},
		{
			name:       "sourceIdentity not a mapping",
			old:        "  freshness: {timestamp: observedAt, maxAgeMs: 10000}\n",
			new:        "  freshness: {timestamp: observedAt, maxAgeMs: 10000}\n  sourceIdentity: broken\n",
			wantPhrase: "sourceIdentity must be a mapping",
		},
		{
			name:       "factIdentity not a mapping",
			old:        "  freshness: {timestamp: observedAt, maxAgeMs: 10000}\n",
			new:        "  freshness: {timestamp: observedAt, maxAgeMs: 10000}\n  factIdentity: broken\n",
			wantPhrase: "factIdentity must be a mapping",
		},
		{
			name:       "factIdentity scopePaths wrong type",
			old:        "  freshness: {timestamp: observedAt, maxAgeMs: 10000}\n",
			new:        "  freshness: {timestamp: observedAt, maxAgeMs: 10000}\n  factIdentity:\n    namespace: urn:example:shared-fact:ue-registration\n    fact: ue.registration\n    scopePaths: broken\n    valuePath: record.value\n",
			wantPhrase: "factIdentity.scopePaths must be a mapping",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			req := evidenceRegressionRequest(t, func(fixture *wp3aFixture) {
				fixture.replaceSnapshotFile(descriptorPath, descriptorFile, testCase.old, testCase.new)
			})
			assertEvidenceRejectedThroughPublicBuild(t, req, testCase.wantPhrase)
		})
	}
}

func TestEvidenceDescriptorMalformedEntryRejected(t *testing.T) {
	// Add a malformed descriptor entry alongside the valid one: the trusted
	// document must be rejected instead of silently dropping the bad entry.
	req := evidenceRegressionRequest(t, func(fixture *wp3aFixture) {
		fixture.replaceSnapshotFile("ca", "evidence-guarantees.yaml",
			"ue.registration-state:\n", "other.guarantee: broken\nue.registration-state:\n")
	})
	assertEvidenceRejectedThroughPublicBuild(t, req, "Evidence Descriptor entry must be a mapping")
}
