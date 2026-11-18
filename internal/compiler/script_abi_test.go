package compiler

// WP9 Script ABI conformance: strict TypedDict input/output derivation from the
// Python CST via the internal gositter adapter, without executing Python.

import (
	"strings"
	"testing"
)

const procedureScriptABI = `from typing import TypedDict

class In(TypedDict):
    value: str
    count: NotRequired[int]

class Out(TypedDict):
    result: str

def main(input: In) -> Out:
    return {"result": input["value"]}
`

const predicateScriptABI = `from __future__ import annotations
from typing import Literal, TypedDict

class PredicateEvaluationResult(TypedDict):
    result: Literal["PASS", "FAIL", "UNKNOWN"]
    reasonCode: str
    evidenceUsed: list[str]

def main(input: PredicateInput) -> PredicateEvaluationResult:
    return {"result": "PASS", "reasonCode": "ok", "evidenceUsed": []}
`

func TestScriptABIDerivesTypedSchemas(t *testing.T) {
	source := strings.Replace(procedureScriptABI, "from typing import TypedDict", "from typing import NotRequired, TypedDict", 1)
	abi, err := deriveProcedureScriptABI([]byte(source))
	if err != nil {
		t.Fatalf("valid Procedure Script rejected: %v", err)
	}
	if abi.InputName != "In" || abi.OutputName != "Out" {
		t.Fatalf("unexpected ABI type names: %+v", abi)
	}
	required, _ := abi.InputSchema["required"].([]string)
	if len(required) != 1 || required[0] != "value" {
		t.Fatalf("input required projection wrong: %+v", abi.InputSchema)
	}
	outRequired, _ := abi.OutputSchema["required"].([]string)
	if len(outRequired) != 1 || outRequired[0] != "result" {
		t.Fatalf("output required projection wrong: %+v", abi.OutputSchema)
	}
	if abi.OutputSchema["additionalProperties"] != false {
		t.Fatalf("output Schema must be closed: %+v", abi.OutputSchema)
	}
}

func TestScriptABIRejectsInvalidShapes(t *testing.T) {
	cases := map[string]string{
		"untyped input": `from typing import TypedDict
class Out(TypedDict):
    result: str
def main(input: dict) -> Out:
    return {"result": ""}
`,
		"missing return annotation": `from typing import TypedDict
class In(TypedDict):
    value: str
def main(input: In):
    return {}
`,
		"optional output field": `from typing import TypedDict
class In(TypedDict):
    value: str
class Out(TypedDict):
    result: NotRequired[str]
def main(input: In) -> Out:
    return {}
`,
		"broad type": `from typing import TypedDict
class Out(TypedDict):
    value: Any
def main(input: dict) -> Out:
    return {}
`,
		"unknown type": `from typing import TypedDict
class Out(TypedDict):
    value: Custom
def main(input: dict) -> Out:
    return {}
`,
		"empty output": `from typing import TypedDict
class Out(TypedDict):
    pass
def main(input: dict) -> Out:
    return {}
`,
		"field default": `from typing import TypedDict
class Out(TypedDict):
    value: str = "x"
def main(input: dict) -> Out:
    return {}
`,
		"module executable statement": `from typing import TypedDict
class Out(TypedDict):
    value: str
helper = 1
def main(input: dict) -> Out:
    return {}
`,
		"duplicate main": `from typing import TypedDict
class Out(TypedDict):
    value: str
def main(input: dict) -> Out:
    return {}
def main(input: dict) -> Out:
    return {}
`,
		"two parameters": `from typing import TypedDict
class Out(TypedDict):
    value: str
def main(input: dict, extra: str) -> Out:
    return {}
`,
		"syntax error": "def main(:\n",
		"total false": `from typing import TypedDict
class Out(TypedDict, total=False):
    value: str
def main(input: dict) -> Out:
    return {}
`,
	}
	for name, source := range cases {
		if _, err := deriveProcedureScriptABI([]byte(source)); err == nil {
			t.Fatalf("%s: invalid Script ABI accepted", name)
		}
	}
}

func TestPredicateScriptABIRules(t *testing.T) {
	abi, err := derivePredicateScriptABI([]byte(predicateScriptABI))
	if err != nil {
		t.Fatalf("valid Predicate Script rejected: %v", err)
	}
	if abi.InputName != "PredicateInput" || abi.OutputName != "PredicateEvaluationResult" {
		t.Fatalf("unexpected Predicate ABI: %+v", abi)
	}
	if _, ok := abi.OutputSchema["properties"].(map[string]any)["result"]; !ok {
		t.Fatalf("predicate output Schema missing result: %+v", abi.OutputSchema)
	}

	noFuture := strings.Replace(predicateScriptABI, "from __future__ import annotations\n", "", 1)
	if _, err := derivePredicateScriptABI([]byte(noFuture)); err == nil {
		t.Fatalf("Predicate Script without delayed annotations accepted")
	}
	badInput := strings.Replace(predicateScriptABI, "input: PredicateInput", "input: In", 1)
	if _, err := derivePredicateScriptABI([]byte(badInput)); err == nil {
		t.Fatalf("Predicate Script with a custom input type accepted")
	}
}

func TestScriptABIIntegratedDiagnostic(t *testing.T) {
	procedure := strings.Replace(scriptProcedure("scripts/compute.py"), "      script: scripts/compute.py", "      script: scripts/compute.py", 1)
	request := testRequest(testContract, procedure, testPredicates, testSchema)
	request.Files = append(request.Files, File{Path: "skills/pdu/scripts/compute.py", Bytes: []byte(procedureScriptABI)})
	if hasCode(Check(request), "SCRIPT_CONTRACT_INVALID") {
		t.Fatalf("valid typed Procedure Script rejected: %+v", Check(request))
	}
	request.Files = append(request.Files[:len(request.Files)-1], File{Path: "skills/pdu/scripts/compute.py", Bytes: []byte("def main(input):\n    return {}\n")})
	found := false
	for _, diagnostic := range Check(request) {
		if diagnostic.Code == "SCRIPT_CONTRACT_INVALID" && diagnostic.Phase == "C_STATIC" {
			found = true
		}
	}
	if !found {
		t.Fatalf("invalid Procedure Script ABI accepted")
	}
}
