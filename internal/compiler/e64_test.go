package compiler

// E64 conformance subset: a reachable legal final Postcondition PASS->SUCCEEDED
// edge exists, but another reachable branch performs a local Postcondition PASS
// and then requests success from a later business Node. The local PASS cannot
// carry Goal proof, so SUCCESS_BYPASSES_FINAL_POSTCONDITION is the primary
// diagnosis (spec H.0 row 2).

import (
	"strings"
	"testing"
)

func TestE64LocalPostconditionPassThenMutationCannotSucceed(t *testing.T) {
	procedure := `start: split
nodes:
  split:
    precondition:
      predicates:
        - predicate: example.ue.registered
          arguments: [context.ueRef]
      routes:
        PASS: verify-legal
        FAIL: verify-local
        UNKNOWN: UNRESOLVED
  verify-legal:
    postcondition:
      predicates:
        - predicate: example.ue.registered
          arguments: [context.ueRef]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
  verify-local:
    postcondition:
      predicates:
        - predicate: example.ue.registered
          arguments: [context.ueRef]
      routes:
        PASS: mutate
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
  mutate:
    operation:
      tool: ca::example.session.mutate
      arguments: {}
      routes:
        SUCCEEDED: SUCCEEDED
        FAILED: FAILED
        NOT_COMMITTED: SAFE_STOPPED
        OUTCOME_UNKNOWN: UNRESOLVED
        COMMITTED: SAFE_STOPPED
`
	request := eRouteFixture(t, reconcileCatalog, procedure)
	for index := range request.Files {
		if strings.HasSuffix(request.Files[index].Path, "assurance/contract.yaml") {
			request.Files[index].Bytes = append(request.Files[index].Bytes,
				[]byte("\nprecondition: split.precondition\n")...)
		}
	}
	diagnostics := Check(request)
	if !v4HasCode(diagnostics, "SUCCESS_BYPASSES_FINAL_POSTCONDITION") {
		t.Fatalf("local Postcondition PASS then direct success not reported: %+v", diagnostics)
	}
	if v4HasCode(diagnostics, "FINAL_POSTCONDITION_MISSING") {
		t.Fatalf("reachable legal final edge present, FINAL_POSTCONDITION_MISSING must not be the diagnosis: %+v", diagnostics)
	}
}
