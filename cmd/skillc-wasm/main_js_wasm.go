//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/odango-chan/ca-skill-compiler/internal/compiler"
)

var callbacks []js.Func

func respond(command string) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 1 || args[0].Type() != js.TypeString {
			return marshalResult(command, wireErrorResult(command, "REQUEST_INVALID", "expected JSON string"))
		}
		raw := args[0].String()
		if wireRequestTooLarge(raw) {
			return marshalResult(command, wireErrorResult(command, "INPUT_LIMIT", "BuildRequest JSON exceeds wire size limit"))
		}
		req, err := compiler.DecodeBuildRequestJSON([]byte(raw))
		if err != nil {
			return marshalResult(command, wireErrorResult(command, "REQUEST_INVALID", "invalid BuildRequest JSON or base64 byte field"))
		}
		var result any
		if command == "inspect" {
			result = compiler.InspectDependencies(req)
		} else if command == "check" {
			result = compiler.Check(req)
		} else {
			result = compiler.Build(req)
		}
		return marshalResult(command, result)
	})
}

func marshalResult(command string, result any) string {
	out, err := json.Marshal(result)
	if err != nil {
		out, _ = json.Marshal(wireErrorResult(command, "INTERNAL", "marshal failed"))
	}
	return string(out)
}

func main() {
	check, build, inspect := respond("check"), respond("build"), respond("inspect")
	verify := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 1 || args[0].Type() != js.TypeString {
			return "expected JSON string"
		}
		if err := compiler.VerifyBuildResultJSON([]byte(args[0].String())); err != nil {
			return err.Error()
		}
		return ""
	})
	buildDevelopment := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 1 || args[0].Type() != js.TypeString {
			return string(developmentWireError("expected JSON string"))
		}
		raw := args[0].String()
		if wireRequestTooLarge(raw) {
			return string(developmentWireError("Development BuildRequest JSON exceeds wire size limit"))
		}
		request, err := compiler.DecodeDevelopmentBuildRequestJSON([]byte(raw))
		if err != nil {
			return string(developmentWireError("invalid Development BuildRequest JSON or base64 byte field: " + err.Error()))
		}
		out, err := json.Marshal(compiler.BuildDevelopment(request))
		if err != nil {
			return string(developmentWireError("marshal failed"))
		}
		return string(out)
	})
	verifyDevelopment := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 2 || args[0].Type() != js.TypeString || args[1].Type() != js.TypeString {
			return "expected two JSON strings"
		}
		if wireRequestTooLarge(args[0].String()) || wireRequestTooLarge(args[1].String()) {
			return "Development request/result JSON exceeds wire size limit"
		}
		if err := compiler.VerifyDevelopmentJSON([]byte(args[0].String()), []byte(args[1].String())); err != nil {
			return err.Error()
		}
		return ""
	})
	callbacks = append(callbacks, check, build, inspect, verify, buildDevelopment, verifyDevelopment)
	js.Global().Set("skillcCheck", check)
	js.Global().Set("skillcBuild", build)
	js.Global().Set("skillcInspectDependencies", inspect)
	js.Global().Set("skillcVerifyArtifact", verify)
	js.Global().Set("skillcBuildDevelopment", buildDevelopment)
	js.Global().Set("skillcVerifyDevelopment", verifyDevelopment)
	select {}
}

// developmentWireError returns a fail-closed Development BuildResult JSON string
// for an undecodable request.
func developmentWireError(message string) []byte {
	result := compiler.DevelopmentBuildResult{
		APIVersion:          compiler.DevelopmentBuildResultAPIVersion,
		Success:             false,
		SourceStatus:        "FAILED",
		InterfaceLinkStatus: "NOT_RUN",
		ArtifactStatus:      "NOT_RUN",
		ExecutionStatus:     "NOT_ADMITTED",
		Diagnostics: []compiler.Diagnostic{{
			Severity: "error",
			Phase:    "C_STATIC",
			Code:     "DEVELOPMENT_REQUEST_INVALID",
			Path:     "",
			Message:  message,
		}},
		OutputFiles: []compiler.File{},
	}
	out, err := json.Marshal(result)
	if err != nil {
		return []byte(`{"apiVersion":"connectionagent.io/compiler-development-build-result/v1alpha1","success":false,"sourceStatus":"FAILED","interfaceLinkStatus":"NOT_RUN","artifactStatus":"NOT_RUN","executionStatus":"NOT_ADMITTED","diagnostics":[],"outputFiles":[]}`)
	}
	return out
}
