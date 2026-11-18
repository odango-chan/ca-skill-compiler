package compiler

// WP6 per-path availability: a producer output read on a non-SUCCEEDED path.

import "testing"

func pathAvailabilityFixture(t *testing.T) BuildRequest {
	t.Helper()
	fixture := wp3aBaseFixture(t)
	addSpecMutationProofSchemas(&fixture)
	fixture.addSnapshotFile("ca", "mutation-catalog.yaml", []byte(`apiVersion: connectionagent.io/toolcatalog/v1alpha4
kind: ToolCatalog
metadata:
  owner: example-ue-owner
spec:
  contracts:
    - kind: operation
      id: example.session.create
      effect: MUTATION
      ownerOutcomes: [SUCCEEDED, FAILED, NOT_COMMITTED]
      inputSchema: {$ref: urn:example:ca:registration-query:v1}
      outputSchema: {$ref: urn:example:ca:registration:v1}
`+specMutationContractBlock))
	fixture.refreshSnapshotDigests(t)
	fixture.files[wp3aProcedurePath] = []byte(`start: mutate
nodes:
  mutate:
    operation:
      tool: ca::example.session.create
      arguments: {}
      routes:
        SUCCEEDED: final
        FAILED: FAILED
        NOT_COMMITTED: final
        OUTCOME_UNKNOWN: UNRESOLVED
  final:
    postcondition:
      predicates:
        - predicate: example.ue.registered
          arguments: [mutate.operation.output]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`)
	return fixture.request(t)
}

func TestProducerOutputNotAvailableOnNonSucceededPath(t *testing.T) {
	diagnostics := Check(pathAvailabilityFixture(t))
	if !hasCode(diagnostics, "SOURCE_NOT_AVAILABLE") {
		t.Fatalf("output read on NOT_COMMITTED path accepted: %+v", diagnostics)
	}
}
