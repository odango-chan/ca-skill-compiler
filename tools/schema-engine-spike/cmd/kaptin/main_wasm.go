//go:build js && wasm

package main

import "syscall/js"

func main() {
	b, err := result()
	if err != nil {
		js.Global().Set("__schemaSpikeError", err.Error())
		return
	}
	js.Global().Set("__schemaSpikeResult", string(b))
}
