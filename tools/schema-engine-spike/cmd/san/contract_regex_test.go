package main

import (
	"strings"
	"testing"
)

func TestCurrentContractRegexSubset(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	artifact := "csa:v1:" + strings.Repeat("a", 64)
	tests := []struct {
		name, pattern, valid, invalid string
	}{
		{"toolcatalog digest", `^sha256:[0-9a-f]{64}$`, digest, "sha256:" + strings.Repeat("g", 64)},
		{"toolcatalog response artifact", `^csa:v1:[0-9a-f]{64}$`, artifact, "csa:v1:bad"},
		{"skill dnn", `^[a-z0-9][a-z0-9.-]*$`, "internet-1", "Invalid"},
		{"skill mcc", `^[0-9]{3}$`, "001", "01"},
		{"skill mnc", `^[0-9]{2,3}$`, "01", "1"},
		{"script bandwidth", `^[0-9]+(\.[0-9]+)? (bps|Kbps|Mbps|Gbps|Tbps)$`, "1.5 Mbps", "1.5 MBps"},
	}

	for _, engine := range []string{"", "regexp2"} {
		for _, test := range tests {
			t.Run(test.name+"/"+engine, func(t *testing.T) {
				root := "https://spike.invalid/current-patterns/" + strings.ReplaceAll(test.name, " ", "-") + ".json"
				compiled, err := (Candidate{}).Compile(map[string]any{root: map[string]any{
					"$schema": "https://json-schema.org/draft/2020-12/schema",
					"type":    "string",
					"pattern": test.pattern,
				}}, root, false, engine)
				if err != nil {
					t.Fatalf("compile with regex engine %q: %v", engine, err)
				}
				if err := compiled.Validator.Validate(test.valid); err != nil {
					t.Errorf("valid value rejected: %v", err)
				}
				if err := compiled.Validator.Validate(test.invalid); err == nil {
					t.Errorf("invalid value accepted by regex engine %q", engine)
				}
			})
		}
	}
}
