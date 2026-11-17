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
		var req compiler.BuildRequest
		if err := json.Unmarshal([]byte(raw), &req); err != nil {
			return marshalResult(command, wireErrorResult(command, "REQUEST_INVALID", "invalid JSON request"))
		}
		var result any
		if command == "inspect" {
			result = compiler.InspectLibraries(req)
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
	callbacks = append(callbacks, check, build, inspect)
	js.Global().Set("skillcCheck", check)
	js.Global().Set("skillcBuild", build)
	js.Global().Set("skillcInspectLibraries", inspect)
	select {}
}
