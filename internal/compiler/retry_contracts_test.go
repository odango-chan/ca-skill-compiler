package compiler

// WP8 slice 2 conformance: retryOf reference validity.

import (
	"strings"
	"testing"
)

const retryProcedureTemplate = `start: mutate
nodes:
  mutate:
    operation:
      tool: ca::ue.registration.current
      arguments: {}
      routes:
        NOT_COMMITTED: retry
        SUCCEEDED: FAILED
        FAILED: FAILED
  retry:
    operation:
      tool: ca::ue.registration.current
      arguments: {}
      retryOf: %s
      routes:
        SUCCEEDED: SUCCEEDED
        FAILED: FAILED
`

func TestRetryContractPositive(t *testing.T) {
	request := testRequest(testContract, strings.Replace(retryProcedureTemplate, "%s", "mutate", 1), testPredicates, testSchema)
	if hasCode(Check(request), "RETRY_CONTRACT_INVALID") {
		t.Fatalf("valid retryOf rejected: %+v", Check(request))
	}
}

func TestRetryContractInvalid(t *testing.T) {
	cases := map[string]string{
		"undeclared node":    "ghost",
		"self reference":     "retry",
		"non-operation node": "final",
	}
	for name, target := range cases {
		procedure := strings.Replace(retryProcedureTemplate, "%s", target, 1)
		procedure += `  final:
    postcondition:
      predicates:
        - predicate: test.complete
          arguments: [mutate.operation.output]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`
		request := testRequest(testContract, procedure, testPredicates, testSchema)
		if !hasCode(Check(request), "RETRY_CONTRACT_INVALID") {
			t.Fatalf("%s: expected RETRY_CONTRACT_INVALID: %+v", name, Check(request))
		}
	}
}

func TestRetryContractSecondGeneration(t *testing.T) {
	procedure := strings.Replace(retryProcedureTemplate, "%s", "retry", 1) + `  retry2:
    operation:
      tool: ca::ue.registration.current
      arguments: {}
      retryOf: retry
      routes:
        SUCCEEDED: SUCCEEDED
        FAILED: FAILED
`
	request := testRequest(testContract, procedure, testPredicates, testSchema)
	if !hasCode(Check(request), "RETRY_CONTRACT_INVALID") {
		t.Fatalf("second-generation retry chain accepted: %+v", Check(request))
	}
}

func TestRetryReachabilityRequiresNotCommittedEdge(t *testing.T) {
	procedure := strings.Replace(retryProcedureTemplate, "        NOT_COMMITTED: retry\n        SUCCEEDED: FAILED", "        SUCCEEDED: retry", 1)
	request := testRequest(testContract, procedure, testPredicates, testSchema)
	if !hasCode(Check(request), "RETRY_CONTRACT_INVALID") {
		t.Fatalf("retry reached through SUCCEEDED edge accepted: %+v", Check(request))
	}
}
