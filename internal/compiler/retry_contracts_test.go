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
        SUCCEEDED: retry
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
	request := v4TestRequest(v4TestContract, strings.Replace(retryProcedureTemplate, "%s", "mutate", 1), v4TestPredicates, v4TestSchema)
	if v4HasCode(Check(request), "RETRY_CONTRACT_INVALID") {
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
		request := v4TestRequest(v4TestContract, procedure, v4TestPredicates, v4TestSchema)
		if !v4HasCode(Check(request), "RETRY_CONTRACT_INVALID") {
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
	request := v4TestRequest(v4TestContract, procedure, v4TestPredicates, v4TestSchema)
	if !v4HasCode(Check(request), "RETRY_CONTRACT_INVALID") {
		t.Fatalf("second-generation retry chain accepted: %+v", Check(request))
	}
}
