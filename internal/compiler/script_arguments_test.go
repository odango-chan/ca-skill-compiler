package compiler

// WP9 slice 2 conformance: Procedure Script call-site arguments and the resolved
// inline-object/array Expression rule.

import (
	"strings"
	"testing"
)

func scriptCallProcedure(arguments string) string {
	return strings.Replace(scriptProcedure("scripts/compute.py"), "      arguments: {}\n", arguments, 1)
}

func TestScriptCallSiteRequiredAndUnknownKeys(t *testing.T) {
	valid := scriptCallProcedure("      arguments:\n        value: context.ueRef\n")
	request := testRequest(testContract, valid, testPredicates, testSchema)
	request.Files = append(request.Files, File{Path: "skills/pdu/scripts/compute.py", Bytes: []byte(procedureScriptABI)})
	for _, diagnostic := range Check(request) {
		if diagnostic.Code == "INPUT_REQUIRED_MISSING" || strings.Contains(diagnostic.Message, "not declared") {
			t.Fatalf("valid Script arguments rejected: %+v", diagnostic)
		}
	}

	missing := scriptCallProcedure("      arguments: {}\n")
	request = testRequest(testContract, missing, testPredicates, testSchema)
	request.Files = append(request.Files, File{Path: "skills/pdu/scripts/compute.py", Bytes: []byte(procedureScriptABI)})
	if !hasCode(Check(request), "INPUT_REQUIRED_MISSING") {
		t.Fatalf("missing required Script input accepted: %+v", Check(request))
	}

	unknown := scriptCallProcedure("      arguments:\n        value: context.ueRef\n        extra: context.ueRef\n")
	request = testRequest(testContract, unknown, testPredicates, testSchema)
	request.Files = append(request.Files, File{Path: "skills/pdu/scripts/compute.py", Bytes: []byte(procedureScriptABI)})
	if !hasCode(Check(request), "INPUT_TYPE_MISMATCH") {
		t.Fatalf("unknown Script argument accepted: %+v", Check(request))
	}
}

func TestInlineArgumentExpressionRejected(t *testing.T) {
	cases := map[string]string{
		"inline mapping":  strings.Replace(testProcedure, "        ueRef: context.ueRef", "        ueRef: {nested: value}", 1),
		"inline sequence": strings.Replace(testProcedure, "        ueRef: context.ueRef", "        ueRef: [context.ueRef]", 1),
	}
	for name, procedure := range cases {
		request := testRequest(testContract, procedure, testPredicates, testSchema)
		if !hasCode(Check(request), "SOURCE_PATH_INVALID") {
			t.Fatalf("%s: inline Expression accepted: %+v", name, Check(request))
		}
	}
}

func TestEmptyOuterArgumentsAllowed(t *testing.T) {
	procedure := strings.Replace(testProcedure, "      arguments:\n        ueRef: context.ueRef", "      arguments: {}", 1)
	request := testRequest(testContract, procedure, testPredicates, testSchema)
	if hasCode(Check(request), "SOURCE_PATH_INVALID") {
		t.Fatalf("legal empty outer arguments container rejected: %+v", Check(request))
	}
}
