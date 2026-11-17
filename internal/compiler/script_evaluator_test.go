package compiler

// WP9 slice 2 conformance: Predicate script evaluator shape/presence.

import "testing"

func scriptEvaluatorPredicates(script string) string {
	return `predicates:
  test.ready:
    parameters: [{ue: string}]
    evaluator: {type: script, script: ` + script + `}
  test.complete:
    parameters: [{result: ca::urn:example:result:v1}]
    evaluator: {type: cel, expression: "result.ready == true"}
`
}

func TestWP9PredicateScriptEvaluatorPositive(t *testing.T) {
	request := v4TestRequest(v4TestContract, v4TestProcedure, scriptEvaluatorPredicates("scripts/ready.py"), v4TestSchema)
	request.Files = append(request.Files, File{Path: "skills/pdu/scripts/ready.py", Bytes: []byte(predicateScriptABI)})
	if v4HasCode(Check(request), "SCRIPT_CONTRACT_INVALID") {
		t.Fatalf("valid script evaluator rejected: %+v", Check(request))
	}
}

func TestWP9PredicateScriptEvaluatorInvalid(t *testing.T) {
	cases := map[string]struct {
		script    string
		addScript bool
		wantCode  string
	}{
		"missing file": {script: "scripts/missing.py", addScript: false, wantCode: "SCRIPT_CONTRACT_INVALID"},
		"bad path":     {script: "ready.py", addScript: true, wantCode: "SCRIPT_CONTRACT_INVALID"},
	}
	for name, testCase := range cases {
		request := v4TestRequest(v4TestContract, v4TestProcedure, scriptEvaluatorPredicates(testCase.script), v4TestSchema)
		if testCase.addScript {
			request.Files = append(request.Files, File{Path: "skills/pdu/scripts/ready.py", Bytes: []byte("x")})
		}
		diagnostics := Check(request)
		if !v4HasCode(diagnostics, testCase.wantCode) {
			t.Fatalf("%s: expected %s: %+v", name, testCase.wantCode, diagnostics)
		}
	}
}

func TestWP9CelEvaluatorWithScriptField(t *testing.T) {
	predicates := `predicates:
  test.ready:
    parameters: [{ue: string}]
    evaluator: {type: cel, expression: 'ue != ""', script: scripts/ready.py}
  test.complete:
    parameters: [{result: ca::urn:example:result:v1}]
    evaluator: {type: cel, expression: "result.ready == true"}
`
	request := v4TestRequest(v4TestContract, v4TestProcedure, predicates, v4TestSchema)
	if !v4HasCode(Check(request), "CEL_INVALID") {
		t.Fatalf("cel evaluator with script field accepted: %+v", Check(request))
	}
}
