package compiler

// WP5 slice 4: Mutation OUTCOME_UNKNOWN may only request a safe terminal.

import "testing"

func mutationRouteFixture(t *testing.T, routes string) BuildRequest {
	t.Helper()
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
	procedure := "start: mutate\nnodes:\n  mutate:\n    operation:\n      tool: ca::example.session.create\n      arguments: {}\n      routes:\n" + routes + `  final:
    postcondition:
      predicates:
        - predicate: example.ue.registered
          arguments: [context.ueRef]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`
	fixture.files[wp3aProcedurePath] = []byte(procedure)
	return fixture.request(t)
}

func TestMutationOutcomeUnknownSafeTerminal(t *testing.T) {
	valid := "        SUCCEEDED: SAFE_STOPPED\n        NOT_COMMITTED: SAFE_STOPPED\n        OUTCOME_UNKNOWN: UNRESOLVED\n"
	request := mutationRouteFixture(t, valid)
	if v4HasCode(Check(request), "RECONCILE_CONTRACT_INVALID") {
		t.Fatalf("valid OUTCOME_UNKNOWN safe terminal rejected: %+v", Check(request))
	}
	invalid := "        SUCCEEDED: SAFE_STOPPED\n        NOT_COMMITTED: SAFE_STOPPED\n        OUTCOME_UNKNOWN: final\n"
	request = mutationRouteFixture(t, invalid)
	if !v4HasCode(Check(request), "RECONCILE_CONTRACT_INVALID") {
		t.Fatalf("OUTCOME_UNKNOWN routed to a business node accepted: %+v", Check(request))
	}
}
