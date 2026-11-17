package compiler

// WP5 conformance: Procedure reachability and global Precondition dominance.

import "testing"

func TestWP5ProcedureGraphPositive(t *testing.T) {
	request := v4TestRequest(v4TestContract, v4TestProcedure, v4TestPredicates, v4TestSchema)
	diagnostics := Check(request)
	if v4HasCode(diagnostics, "UNREACHABLE_NODE") || v4HasCode(diagnostics, "ENTRY_PRECONDITION_BYPASSED") {
		t.Fatalf("positive Procedure graph rejected: %+v", diagnostics)
	}
	if Check(request)[0].Phase != Build(request).Diagnostics[0].Phase {
		t.Fatalf("Check/Build phase disagreement")
	}
}

func TestWP5UnreachableNodeWarning(t *testing.T) {
	procedure := v4TestProcedure + `  orphan:
    operation:
      tool: ca::example.session.read
      arguments: {}
      routes:
        SUCCEEDED: orphan
        FAILED: FAILED
`
	request := v4TestRequest(v4TestContract, procedure, v4TestPredicates, v4TestSchema)
	diagnostics := Check(request)
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "UNREACHABLE_NODE" && diagnostic.Phase == "C_WARNING" && diagnostic.Severity == "warning" && diagnostic.Path != "" {
			found = true
		}
	}
	if !found {
		t.Fatalf("unreachable Node warning missing: %+v", diagnostics)
	}
}

func TestWP5GlobalPreconditionBypass(t *testing.T) {
	contract := `goal: pdu_session.initial_request_fulfilled
precondition: gate.precondition
requestSchema: schemas/request.schema.yaml
contextSchema: ca-context::urn:example:host:v1
dependencies:
  ca-context: urn:example:host
capabilities: [fivegc-session-read]
`
	procedure := `start: entry
nodes:
  entry:
    operation:
      tool: ca::example.session.read
      arguments: {}
      routes:
        SUCCEEDED: read
        FAILED: FAILED
  gate:
    precondition:
      predicates:
        - predicate: test.ready
          arguments: [context.ueRef]
      routes:
        PASS: read
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
  read:
    observation:
      tool: ca::example.session.read
      arguments:
        ueRef: context.ueRef
      routes:
        SUCCEEDED: final
        FAILED: FAILED
        UNKNOWN: UNRESOLVED
  final:
    postcondition:
      predicates:
        - predicate: test.complete
          arguments: [read.observation.output]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`
	request := v4TestRequest(contract, procedure, v4TestPredicates, v4TestSchema)
	diagnostics := Check(request)
	if !v4HasCode(diagnostics, "ENTRY_PRECONDITION_BYPASSED") {
		t.Fatalf("global Precondition bypass not detected: %+v", diagnostics)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "ENTRY_PRECONDITION_BYPASSED" && (diagnostic.Phase != "C_STATIC" || diagnostic.Path == "" || diagnostic.Line == 0) {
			t.Fatalf("bypass diagnostic shape/location invalid: %+v", diagnostic)
		}
	}
}
