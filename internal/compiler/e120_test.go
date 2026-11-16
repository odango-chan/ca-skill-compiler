package compiler

// E120 conformance subset: the singular global Precondition gate must dominate
// every path to a business Mutation or a SUCCEEDED request; a legal
// Postcondition PASS -> SUCCEEDED edge still exists in both variants.

import "testing"

const e120BypassProcedure = `start: entry
nodes:
  entry:
    observation:
      tool: ca::example.session.read
      arguments: {ueRef: context.ueRef}
      routes:
        SUCCEEDED: final
        FAILED: FAILED
  check:
    precondition:
      predicates:
        - predicate: test.ready
          arguments: [context.ueRef]
      routes:
        PASS: final
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
  final:
    postcondition:
      predicates:
        - predicate: test.complete
          arguments: [entry.observation.output]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`

const e120GatedProcedure = `start: check
nodes:
  check:
    precondition:
      predicates:
        - predicate: test.ready
          arguments: [context.ueRef]
      routes:
        PASS: final
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
  final:
    postcondition:
      predicates:
        - predicate: test.complete
          arguments: [context.ueRef]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`

func TestE120GlobalPreconditionDominance(t *testing.T) {
	bypass := v4TestRequest(v4TestContract, e120BypassProcedure, v4TestPredicates, v4TestSchema)
	if !v4HasCode(Check(bypass), "ENTRY_PRECONDITION_BYPASSED") {
		t.Fatalf("path reaching SUCCEEDED without the global Precondition was accepted: %+v", Check(bypass))
	}

	gated := v4TestRequest(v4TestContract, e120GatedProcedure, v4TestPredicates, v4TestSchema)
	if v4HasCode(Check(gated), "ENTRY_PRECONDITION_BYPASSED") {
		t.Fatalf("gated procedure reported a false bypass: %+v", Check(gated))
	}
}
