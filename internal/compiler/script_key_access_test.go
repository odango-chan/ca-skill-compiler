package compiler

// WP9 restricted Script input key-access analysis conformance.

import (
	"strings"
	"testing"
)

func TestScriptInputKeyAccess(t *testing.T) {
	base := `from typing import TypedDict

class In(TypedDict):
    value: str

class Out(TypedDict):
    result: str

def main(input: In) -> Out:
%s
`
	valid := strings.Replace(procedureScriptABI, `    return {"result": input["value"]}`, `    return {"result": input["value"]}`, 1)
	if _, err := deriveProcedureScriptABI([]byte(valid)); err != nil {
		t.Fatalf("valid script rejected: %v", err)
	}
	allowed := map[string]bool{"value": true}
	if err := validateScriptInputKeyAccess([]byte(valid), allowed, false); err != nil {
		t.Fatalf("valid literal input access rejected: %v", err)
	}

	unknown := strings.Replace(base, "%s", `    return {"result": input["nope"]}`, 1)
	if err := validateScriptInputKeyAccess([]byte(unknown), allowed, false); err == nil {
		t.Fatalf("undeclared literal input key accepted")
	}

	dynamic := strings.Replace(base, "%s", "    key = \"value\"\n    return {\"result\": input[key]}", 1)
	if err := validateScriptInputKeyAccess([]byte(dynamic), allowed, false); err == nil {
		t.Fatalf("dynamic input key access accepted")
	}

	attribute := strings.Replace(base, "%s", `    return {"result": input.missing}`, 1)
	if err := validateScriptInputKeyAccess([]byte(attribute), allowed, false); err == nil {
		t.Fatalf("undeclared attribute input key accepted")
	}
}

func TestPredicateScriptEvidenceAccessFailsClosed(t *testing.T) {
	source := strings.Replace(predicateScriptABI, `    return {"result": "PASS", "reasonCode": "ok", "evidenceUsed": []}`, `    used = input["evidence"]
    return {"result": "PASS", "reasonCode": "ok", "evidenceUsed": []}`, 1)
	if _, err := derivePredicateScriptABI([]byte(source)); err != nil {
		t.Fatalf("ABI derivation itself must succeed: %v", err)
	}
	if err := validateScriptInputKeyAccess([]byte(source), map[string]bool{}, false); err == nil {
		t.Fatalf("unfrozen Evidence projection access accepted")
	}
}

func TestScriptMainParameterMustBeInput(t *testing.T) {
	source := strings.Replace(procedureScriptABI, "def main(input: In) -> Out:", "def main(payload: In) -> Out:", 1)
	if _, err := deriveProcedureScriptABI([]byte(source)); err == nil {
		t.Fatalf("main parameter not named input accepted")
	}
}
