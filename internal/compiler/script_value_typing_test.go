package compiler

// WP9 slice 3 conformance: the derived Script input Schema is a real F.5 target.

import (
	"strings"
	"testing"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

const scriptValueInputABI = `from typing import NotRequired, TypedDict

class In(TypedDict):
    count: int
    note: NotRequired[str]

class Out(TypedDict):
    result: str

def main(input: In) -> Out:
    return {"result": str(input["count"])}
`

func TestScriptInputSchemaF5Assignability(t *testing.T) {
	abi, err := deriveProcedureScriptABI([]byte(scriptValueInputABI))
	if err != nil {
		t.Fatalf("derive ABI: %v", err)
	}
	target := schemaNode(abi.InputSchema)
	if target == nil {
		t.Fatalf("derived Script input Schema did not convert to YAML")
	}
	countProperty := child(child(unalias(target), "properties"), "count")
	if countProperty == nil {
		t.Fatalf("derived Schema is missing count: %+v", abi.InputSchema)
	}

	stringSource := mustNode(t, `{type: string}`)
	if _, ok := f5Assignable(stringSource, unalias(countProperty)); ok {
		t.Fatalf("string Source accepted for integer Script input property")
	}
	integerSource := mustNode(t, `{type: integer}`)
	if _, ok := f5Assignable(integerSource, unalias(countProperty)); !ok {
		t.Fatalf("integer Source rejected for integer Script input property")
	}
}

func TestScriptInputSchemaKeepsNotRequiredOptional(t *testing.T) {
	abi, err := deriveProcedureScriptABI([]byte(scriptValueInputABI))
	if err != nil {
		t.Fatalf("derive ABI: %v", err)
	}
	required, _ := abi.InputSchema["required"].([]string)
	if len(required) != 1 || required[0] != "count" {
		t.Fatalf("NotRequired key must not be required: %v", required)
	}
}

func mustNode(t *testing.T, source string) *yaml.Node {
	t.Helper()
	node, err := yaml.ParseValue([]byte(strings.TrimSpace(source)))
	if err != nil {
		t.Fatalf("parse fixture node %q: %v", source, err)
	}
	return unalias(node)
}
