package main

import "github.com/odango-chan/ca-skill-compiler/internal/compiler"

// A valid request can encode at most 16 MiB of file bytes as base64, plus
// bounded path/target JSON overhead. This cap rejects oversized wire payloads
// before encoding/json decodes and allocates every []byte field.
const maxWireRequestBytes = 25 << 20

func wireRequestTooLarge(raw string) bool {
	return len(raw) > maxWireRequestBytes
}

func wireErrorResult(command, code, message string) any {
	diagnostics := []compiler.Diagnostic{{Severity: "error", Code: code, Message: message}}
	switch command {
	case "check":
		return diagnostics
	case "inspect":
		return compiler.LibraryResult{Diagnostics: diagnostics, Tools: []compiler.LibraryTool{}}
	default:
		return compiler.BuildResult{Success: false, Diagnostics: diagnostics, OutputFiles: []compiler.File{}}
	}
}
