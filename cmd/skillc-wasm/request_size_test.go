package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/odango-chan/ca-skill-compiler/internal/compiler"
)

func TestWireRequestSizeGuard(t *testing.T) {
	for _, tc := range []struct {
		name string
		size int
		want bool
	}{
		{name: "below limit", size: maxWireRequestBytes - 1},
		{name: "at limit", size: maxWireRequestBytes},
		{name: "above limit", size: maxWireRequestBytes + 1, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := strings.Repeat("x", tc.size)
			got := wireRequestTooLarge(raw)
			if got != tc.want {
				t.Fatalf("oversized = %v, want %v", got, tc.want)
			}
		})
	}
	for _, command := range []string{"check", "build", "inspect"} {
		t.Run(command+" error response shape", func(t *testing.T) {
			wire, err := json.Marshal(wireErrorResult(command, "INPUT_LIMIT", "too large"))
			if err != nil {
				t.Fatal(err)
			}
			switch command {
			case "check":
				var diagnostics []compiler.Diagnostic
				if err := json.Unmarshal(wire, &diagnostics); err != nil || len(diagnostics) != 1 {
					t.Fatalf("check error response must be Diagnostic[]: %s (%v)", wire, err)
				}
			case "build":
				var result compiler.BuildResult
				if err := json.Unmarshal(wire, &result); err != nil || result.Success || len(result.OutputFiles) != 0 {
					t.Fatalf("build error response must be an empty failed BuildResult: %s (%v)", wire, err)
				}
			case "inspect":
				var result compiler.DependencyInspection
				if err := json.Unmarshal(wire, &result); err != nil || len(result.Diagnostics) != 1 || len(result.Callables) != 0 || len(result.Schemas) != 0 {
					t.Fatalf("inspect error response must be a LibraryResult: %s (%v)", wire, err)
				}
			}
		})
	}
}
