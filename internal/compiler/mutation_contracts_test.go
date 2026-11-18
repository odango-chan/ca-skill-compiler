package compiler

// WP8 slice 1 conformance: Mutation contract identity requirements.

import "testing"

func TestWP8ValidMutationContract(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.addSnapshotFile("ca", "session-catalog.yaml", []byte(`apiVersion: connectionagent.io/toolcatalog/v1alpha4
kind: ToolCatalog
metadata:
  owner: example-ue-owner
spec:
  contracts:
    - kind: operation
      id: example.session.create
      effect: MUTATION
      ownerOutcomes: [SUCCEEDED, NOT_COMMITTED]
      inputSchema: {$ref: urn:example:ca:registration-query:v1}
      outputSchema: {$ref: urn:example:ca:registration:v1}
      receiptSchema: {$ref: urn:example:ca:registration:v1}
      mutation:
        commitProof: {schema: urn:example:ca:registration:v1, path: state, bind: originalInvocationId}
`))
	fixture.refreshSnapshotDigests(t)
	diagnostics := Check(fixture.request(t))
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "RECONCILE_CONTRACT_INVALID" {
			t.Fatalf("valid Mutation contract rejected: %+v", diagnostic)
		}
	}
}

func TestWP8IncompleteMutationContract(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.addSnapshotFile("ca", "broken-mutation-catalog.yaml", []byte(`apiVersion: connectionagent.io/toolcatalog/v1alpha4
kind: ToolCatalog
metadata:
  owner: example-ue-owner
spec:
  contracts:
    - kind: operation
      id: example.session.broken
      effect: MUTATION
      ownerOutcomes: [SUCCEEDED]
      inputSchema: {$ref: urn:example:ca:registration-query:v1}
      outputSchema: {$ref: urn:example:ca:registration:v1}
`))
	fixture.refreshSnapshotDigests(t)
	diagnostics := Check(fixture.request(t))
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "RECONCILE_CONTRACT_INVALID" && diagnostic.Phase == "C_STATIC" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Mutation without receiptSchema/commitProof accepted: %+v", diagnostics)
	}
}
