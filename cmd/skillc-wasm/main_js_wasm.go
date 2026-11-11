//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/odango-chan/ca-skill-compiler/internal/compiler"
)

var callbacks []js.Func

func respond(checkOnly bool) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 1 || args[0].Type() != js.TypeString {
			return `{"success":false,"diagnostics":[{"severity":"error","code":"REQUEST_INVALID","path":"","message":"expected JSON string"}],"outputFiles":[]}`
		}
		var req compiler.Request
		if err := json.Unmarshal([]byte(args[0].String()), &req); err != nil {
			return `{"success":false,"diagnostics":[{"severity":"error","code":"REQUEST_INVALID","path":"","message":"invalid JSON request"}],"outputFiles":[]}`
		}
		var result compiler.Result
		if checkOnly {
			result.Diagnostics = compiler.Check(req)
			result.Success = true
			result.OutputFiles = []compiler.File{}
			for _, d := range result.Diagnostics {
				if d.Severity == "error" { result.Success = false; break }
			}
		} else {
			result = compiler.Build(req)
		}
		out, err := json.Marshal(result)
		if err != nil { return `{"success":false,"diagnostics":[{"severity":"error","code":"INTERNAL","path":"","message":"marshal failed"}],"outputFiles":[]}` }
		return string(out)
	})
}

func main() {
	check, build := respond(true), respond(false)
	callbacks = append(callbacks, check, build)
	js.Global().Set("skillcCheck", check)
	js.Global().Set("skillcBuild", build)
	select {}
}
