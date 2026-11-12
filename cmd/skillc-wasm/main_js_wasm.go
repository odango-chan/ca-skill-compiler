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
			return `{"success":false,"diagnostics":[{"severity":"error","code":"REQUEST_INVALID","path":"","message":"expected JSON string"}],"outputFiles":[]}`
		}
		var req compiler.BuildRequest
		if err := json.Unmarshal([]byte(args[0].String()), &req); err != nil {
			return `{"success":false,"diagnostics":[{"severity":"error","code":"REQUEST_INVALID","path":"","message":"invalid JSON request"}],"outputFiles":[]}`
		}
		var result any
		if command == "inspect" {
			result = compiler.InspectLibraries(req)
		} else if command == "check" {
			diagnostics := compiler.Check(req)
			success := true
			for _, d := range diagnostics {
				if d.Severity == "error" {
					success = false
					break
				}
			}
			result = compiler.BuildResult{Diagnostics: diagnostics, Success: success, OutputFiles: []compiler.File{}}
		} else {
			result = compiler.Build(req)
		}
		out, err := json.Marshal(result)
		if err != nil {
			return `{"success":false,"diagnostics":[{"severity":"error","code":"INTERNAL","path":"","message":"marshal failed"}],"outputFiles":[]}`
		}
		return string(out)
	})
}

func main() {
	check, build, inspect := respond("check"), respond("build"), respond("inspect")
	callbacks = append(callbacks, check, build, inspect)
	js.Global().Set("skillcCheck", check)
	js.Global().Set("skillcBuild", build)
	js.Global().Set("skillcInspectLibraries", inspect)
	select {}
}
