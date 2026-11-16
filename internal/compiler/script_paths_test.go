package compiler

// WP9 slice 1 conformance: Procedure Script LocalScriptPath grammar/presence.

import "testing"

func scriptProcedure(script string) string {
	return `start: check
nodes:
  check:
    precondition:
      predicates:
        - predicate: test.ready
          arguments: [context.ueRef]
      routes:
        PASS: read
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
  read:
    operation:
      script: ` + script + `
      arguments: {}
      routes:
        SUCCEEDED: final
        FAILED: FAILED
  final:
    postcondition:
      predicates:
        - predicate: test.complete
          arguments: [read.operation.output]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`
}

func TestWP9ScriptPathPositive(t *testing.T) {
	request := v4TestRequest(v4TestContract, scriptProcedure("scripts/compute.py"), v4TestPredicates, v4TestSchema)
	request.Files = append(request.Files, File{Path: "skills/pdu/scripts/compute.py", Bytes: []byte(procedureScriptABI)})
	if v4HasCode(Check(request), "SCRIPT_CONTRACT_INVALID") {
		t.Fatalf("valid LocalScriptPath rejected: %+v", Check(request))
	}
}

func TestWP9ScriptPathInvalidOrMissing(t *testing.T) {
	cases := map[string]string{
		"not under scripts": "compute.py",
		"traversal":         "scripts/../compute.py",
		"wrong suffix":      "scripts/compute.txt",
		"missing file":      "scripts/missing.py",
	}
	for name, script := range cases {
		request := v4TestRequest(v4TestContract, scriptProcedure(script), v4TestPredicates, v4TestSchema)
		request.Files = append(request.Files, File{Path: "skills/pdu/scripts/compute.py", Bytes: []byte("x")})
		diagnostics := Check(request)
		found := false
		for _, diagnostic := range diagnostics {
			if diagnostic.Code == "SCRIPT_CONTRACT_INVALID" && diagnostic.Phase == "C_STATIC" {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s: expected SCRIPT_CONTRACT_INVALID: %+v", name, diagnostics)
		}
	}
}
