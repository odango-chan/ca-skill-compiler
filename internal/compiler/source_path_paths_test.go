package compiler

// WP6 per-path availability: a producer output read on a non-SUCCEEDED path.

import (
	"strings"
	"testing"
)

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

func TestProducerOutputAfterPostcondition(t *testing.T) {
	const downstream = `  verify:
    postcondition:
      predicates:
        - predicate: example.ue.registered
          arguments: [inspect-registration.observation.output]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`
	for _, test := range []struct {
		name   string
		bypass string
	}{
		{name: "postcondition PASS preserves output"},
		{name: "observation failure bypass still rejects output", bypass: "observation"},
		{name: "precondition failure bypass still rejects output", bypass: "precondition"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := wp3aBaseFixture(t)
			procedure := string(fixture.files[wp3aProcedurePath])
			procedure = strings.Replace(procedure, "PASS: SUCCEEDED", "PASS: verify", 1)
			switch test.bypass {
			case "observation":
				procedure = strings.Replace(procedure, "FAILED: FAILED", "FAILED: verify", 1)
			case "precondition":
				procedure = strings.Replace(procedure, "    observation:", `    precondition:
      predicates:
        - predicate: example.request.valid
          arguments: [context.ueRef]
      routes:
        FAIL: verify
        UNKNOWN: UNRESOLVED
    observation:`, 1)
				fixture.files[wp3aPredicatesPath] = append(fixture.files[wp3aPredicatesPath], []byte(`
  example.request.valid:
    parameters:
      - ueRef: string
    evaluator:
      type: cel
      expression: ueRef != ""
`)...)
			}
			fixture.files[wp3aProcedurePath] = []byte(procedure + downstream)
			result := Build(wp3cRequest(t, fixture, wp3cBaseBundle(t)))
			if test.bypass != "" {
				assertFailureAtomic(t, result)
				if !hasCode(result.Diagnostics, "SOURCE_NOT_AVAILABLE") {
					t.Fatalf("failed producer route was not diagnosed: %+v", result.Diagnostics)
				}
			} else if !result.Success || len(result.OutputFiles) == 0 {
				t.Fatalf("postcondition PASS did not carry producer output: %+v", result.Diagnostics)
			}
		})
	}
}
