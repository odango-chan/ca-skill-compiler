package spike

import (
	"fmt"
	"strings"
)

type SecurityResult struct {
	Candidate string `json:"candidate"`
	ID        string `json:"id"`
	Reference string `json:"reference"`
	Passed    bool   `json:"passed"`
	Error     string `json:"error,omitempty"`
	Trace     *Trace `json:"trace,omitempty"`
}

func RunSecurity(c Candidate) []SecurityResult {
	refs := []struct{ id, ref string }{
		{"http", "http://remote.invalid/schema.json"},
		{"https", "https://remote.invalid/schema.json"},
		{"absolute-path", "/etc/passwd"},
		{"file-uri", "file:///etc/passwd"},
		{"path-traversal", "../../outside.json"},
		{"unknown-scheme", "gopher://remote.invalid/schema.json"},
		{"custom-scheme", "custom://remote.invalid/schema.json"},
		{"missing-resource", "mem:///project/missing.json"},
	}
	results := make([]SecurityResult, 0, len(refs))
	for _, test := range refs {
		root := "https://spike.invalid/project/security/" + test.id + ".json"
		documents := map[string]any{root: map[string]any{
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"$ref":    test.ref,
		}}
		compiled, err := c.Compile(documents, root, false, "")
		passed := err != nil && compiled.Trace != nil && contains(compiled.Trace.RegistryLookups, root)
		if strings.HasPrefix(test.ref, "http:") || strings.HasPrefix(test.ref, "https:") || strings.HasPrefix(test.ref, "file:") {
			passed = passed && len(compiled.Trace.DeniedExternal) > 0
		} else {
			passed = passed && len(compiled.Trace.DeniedReferences) > 0
		}
		if err == nil {
			passed = false
		}
		result := SecurityResult{Candidate: c.Name(), ID: test.id, Reference: test.ref, Passed: passed, Trace: compiled.Trace}
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Error = "unsafe reference unexpectedly compiled"
		}
		if !passed && result.Error == "" {
			result.Error = fmt.Sprintf("reference was not denied through instrumented registry: trace=%+v", compiled.Trace)
		}
		results = append(results, result)
	}
	return results
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
