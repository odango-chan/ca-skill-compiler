package compiler

// E120 conformance subset: the singular global Precondition gate must dominate
// every path to a business Mutation or a SUCCEEDED request; a legal
// Postcondition PASS -> SUCCEEDED edge still exists in both variants.

import (
	"strings"
	"testing"
)

const e120BypassProcedure = `start: split
nodes:
  split:
    observation:
      tool: ca::example.session.read
      arguments: {ueRef: context.ueRef}
      routes:
        SUCCEEDED: check
        FAILED: final
        UNKNOWN: UNRESOLVED
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
          arguments: [split.observation.output]
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

// The gate's FAIL route must not be treated as the aggregate PASS proof, even
// though the FAIL route leaves the gate Node itself.
func TestE120GateFailRouteDoesNotProveGoal(t *testing.T) {
	procedure := `start: check
nodes:
  check:
    precondition:
      predicates:
        - predicate: test.ready
          arguments: [context.ueRef]
      routes:
        PASS: final
        FAIL: SUCCEEDED
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
	request := v4TestRequest(v4TestContract, procedure, v4TestPredicates, v4TestSchema)
	if got := Check(request); !v4HasCode(got, "ENTRY_PRECONDITION_BYPASSED") {
		t.Fatalf("gate FAIL route to SUCCEEDED treated as Goal proof: %+v", got)
	}
}

// Reaching the gate Node is not proof by itself: a FAIL route from the gate may
// still reach a business Mutation, which must be reported as a bypass.
func TestE120GateFailRouteCannotDispatchMutation(t *testing.T) {
	procedure := `start: inspect-registration
nodes:
  inspect-registration:
    observation:
      tool: ca::ue.registration.current
      arguments:
        ueRef: context.ueRef
      routes:
        SUCCEEDED: check
        FAILED: FAILED
        UNKNOWN: UNRESOLVED
  check:
    precondition:
      predicates:
        - predicate: example.ue.registered
          arguments: [context.ueRef]
      routes:
        PASS: SAFE_STOPPED
        FAIL: mutate
        UNKNOWN: UNRESOLVED
  mutate:
    operation:
      tool: ca::example.session.mutate
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
        FAILED: FAILED
        NOT_COMMITTED: SAFE_STOPPED
        OUTCOME_UNKNOWN: UNRESOLVED
        COMMITTED: SAFE_STOPPED
`
	request := eRouteFixture(t, reconcileCatalog, procedure)
	for index := range request.Files {
		if strings.HasSuffix(request.Files[index].Path, "assurance/contract.yaml") {
			request.Files[index].Bytes = append(request.Files[index].Bytes,
				[]byte("\nprecondition: check.precondition\n")...)
		}
	}
	if got := Check(request); !v4HasCode(got, "ENTRY_PRECONDITION_BYPASSED") {
		t.Fatalf("gate FAIL route to a business Mutation not reported as a bypass: %+v", got)
	}
}
