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
	request := v4TestRequest(v4TestContract, valid, v4TestPredicates, v4TestSchema)
	request.Files = append(request.Files, File{Path: "skills/pdu/scripts/compute.py", Bytes: []byte(procedureScriptABI)})
	for _, diagnostic := range Check(request) {
		if diagnostic.Code == "INPUT_REQUIRED_MISSING" || strings.Contains(diagnostic.Message, "not declared") {
			t.Fatalf("valid Script arguments rejected: %+v", diagnostic)
		}
	}

	missing := scriptCallProcedure("      arguments: {}\n")
	request = v4TestRequest(v4TestContract, missing, v4TestPredicates, v4TestSchema)
	request.Files = append(request.Files, File{Path: "skills/pdu/scripts/compute.py", Bytes: []byte(procedureScriptABI)})
	if !v4HasCode(Check(request), "INPUT_REQUIRED_MISSING") {
		t.Fatalf("missing required Script input accepted: %+v", Check(request))
	}

	unknown := scriptCallProcedure("      arguments:\n        value: context.ueRef\n        extra: context.ueRef\n")
	request = v4TestRequest(v4TestContract, unknown, v4TestPredicates, v4TestSchema)
	request.Files = append(request.Files, File{Path: "skills/pdu/scripts/compute.py", Bytes: []byte(procedureScriptABI)})
	if !v4HasCode(Check(request), "INPUT_TYPE_MISMATCH") {
		t.Fatalf("unknown Script argument accepted: %+v", Check(request))
	}
}

func TestInlineArgumentExpressionRejected(t *testing.T) {
	cases := map[string]string{
		"inline mapping":  strings.Replace(v4TestProcedure, "        ueRef: context.ueRef", "        ueRef: {nested: value}", 1),
		"inline sequence": strings.Replace(v4TestProcedure, "        ueRef: context.ueRef", "        ueRef: [context.ueRef]", 1),
	}
	for name, procedure := range cases {
		request := v4TestRequest(v4TestContract, procedure, v4TestPredicates, v4TestSchema)
		if !v4HasCode(Check(request), "SOURCE_PATH_INVALID") {
			t.Fatalf("%s: inline Expression accepted: %+v", name, Check(request))
		}
	}
}

func TestEmptyOuterArgumentsAllowed(t *testing.T) {
	procedure := strings.Replace(v4TestProcedure, "      arguments:\n        ueRef: context.ueRef", "      arguments: {}", 1)
	request := v4TestRequest(v4TestContract, procedure, v4TestPredicates, v4TestSchema)
	if v4HasCode(Check(request), "SOURCE_PATH_INVALID") {
		t.Fatalf("legal empty outer arguments container rejected: %+v", Check(request))
	}
}
