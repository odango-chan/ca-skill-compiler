package compiler

// WP9 NotRequired guard-proof conformance (CA-SL §32.1).

import (
	"strings"
	"testing"
)

func optionalFieldScript(body string) string {
	return `from typing import NotRequired, TypedDict

class In(TypedDict):
    value: str
    note: NotRequired[str]

class Out(TypedDict):
    result: str

def main(input: In) -> Out:
` + body
}

func optionalFieldRequest(t *testing.T, source string) BuildRequest {
	t.Helper()
	procedure := strings.Replace(scriptProcedure("scripts/compute.py"), "      arguments: {}\n", "      arguments:\n        value: context.ueRef\n", 1)
	request := testRequest(testContract, procedure, testPredicates, testSchema)
	request.Files = append(request.Files, File{Path: "skills/pdu/scripts/compute.py", Bytes: []byte(source)})
	return request
}

func TestOptionalInputKeyRequiresGuard(t *testing.T) {
	unsafe := optionalFieldScript(`    return {"result": input["note"]}
`)
	if !hasCode(Check(optionalFieldRequest(t, unsafe)), "SCRIPT_CONTRACT_INVALID") {
		t.Fatalf("unguarded optional key read accepted")
	}

	guarded := optionalFieldScript(`    if "note" in input:
        result = input["note"]
    else:
        result = ""
    return {"result": result}
`)
	for _, diagnostic := range Check(optionalFieldRequest(t, guarded)) {
		if diagnostic.Code == "SCRIPT_CONTRACT_INVALID" {
			t.Fatalf("guarded optional key read rejected: %+v", diagnostic)
		}
	}

	safeGet := optionalFieldScript(`    return {"result": input.get("note", "")}
`)
	for _, diagnostic := range Check(optionalFieldRequest(t, safeGet)) {
		if diagnostic.Code == "SCRIPT_CONTRACT_INVALID" {
			t.Fatalf("presence-tolerant get() rejected: %+v", diagnostic)
		}
	}
}
