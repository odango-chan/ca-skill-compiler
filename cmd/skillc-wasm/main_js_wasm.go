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
	callbacks = append(callbacks, check, build, inspect, verify)
	js.Global().Set("skillcCheck", check)
	js.Global().Set("skillcBuild", build)
	js.Global().Set("skillcInspectDependencies", inspect)
	js.Global().Set("skillcVerifyArtifact", verify)
	select {}
}
