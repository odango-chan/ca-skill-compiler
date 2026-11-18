package compiler

// WP9 Predicate Script input schema conformance: PredicateInput keys come from the
// owning Predicate.parameters; script reads are checked against that projection.

import (
	"strings"
	"testing"
)

const predicateScriptWithParameterRead = `from __future__ import annotations
from typing import Literal, TypedDict

class PredicateEvaluationResult(TypedDict):
    result: Literal["PASS", "FAIL", "UNKNOWN"]
    reasonCode: str
    evidenceUsed: list[str]

def main(input: PredicateInput) -> PredicateEvaluationResult:
    seen = input["ue"]
    return {"result": "PASS", "reasonCode": str(seen), "evidenceUsed": []}
`

func predicateScriptRequest(t *testing.T, script string) BuildRequest {
	t.Helper()
	predicates := `predicates:
  test.ready:
    parameters: [{ue: string}]
    evaluator: {type: script, script: scripts/ready.py}
  test.complete:
    parameters: [{result: ca::urn:example:result:v1}]
    evaluator: {type: cel, expression: "result.ready == true"}
`
	request := testRequest(testContract, testProcedure, predicates, testSchema)
	request.Files = append(request.Files, File{Path: "skills/pdu/scripts/ready.py", Bytes: []byte(script)})
	return request
}

func TestPredicateScriptInputParametersAccepted(t *testing.T) {
	request := predicateScriptRequest(t, predicateScriptWithParameterRead)
	for _, diagnostic := range Check(request) {
		if diagnostic.Code == "SCRIPT_CONTRACT_INVALID" {
			t.Fatalf("declared PredicateInput parameter read rejected: %+v", diagnostic)
		}
	}
}

func TestPredicateScriptUnknownParameterRejected(t *testing.T) {
	source := strings.Replace(predicateScriptWithParameterRead, `input["ue"]`, `input["unknown"]`, 1)
	request := predicateScriptRequest(t, source)
	if !hasCode(Check(request), "SCRIPT_CONTRACT_INVALID") {
		t.Fatalf("undeclared PredicateInput key accepted: %+v", Check(request))
	}
}
