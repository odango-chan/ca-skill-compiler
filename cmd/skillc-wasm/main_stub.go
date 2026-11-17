//go:build !js || !wasm

package main

import "fmt"

// This binary is intentionally useful only when GOOS=js GOARCH=wasm.
func main() { fmt.Println("Build with GOOS=js GOARCH=wasm to create skillc.wasm") }
