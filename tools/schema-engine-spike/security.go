package spike

import (
	"fmt"
	"net/url"
	"strings"
)

type SecurityResult struct {
	Candidate                 string `json:"candidate"`
	ID                        string `json:"id"`
	Reference                 string `json:"reference"`
	ResolvedReference         string `json:"resolvedReference"`
	RegistryDenied            bool   `json:"registryDenied"`
	CompileRejectedByRegistry bool   `json:"compileRejectedByRegistry"`
	Passed                    bool   `json:"passed"`
	Error                     string `json:"error,omitempty"`
	Trace                     *Trace `json:"trace,omitempty"`
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
		resolvedReference, resolveErr := resolveReference(root, test.ref)
		documents := map[string]any{root: map[string]any{
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"$ref":    test.ref,
		}}
		compiled, err := c.Compile(documents, root, false, "")
		registryDenied := false
		if compiled.Trace != nil {
			registryDenied = hasRegistryDenial(compiled.Trace, resolvedReference)
		}
		compileRejectedByRegistry := err != nil &&
			strings.Contains(err.Error(), RegistryDeniedCode) &&
			strings.Contains(err.Error(), resolvedReference)
		passed := resolveErr == nil && err != nil && compiled.Trace != nil &&
			contains(compiled.Trace.RegistryLookups, root) && registryDenied && compileRejectedByRegistry
		if strings.HasPrefix(test.ref, "http:") || strings.HasPrefix(test.ref, "https:") || strings.HasPrefix(test.ref, "file:") {
			passed = passed && contains(compiled.Trace.DeniedExternal, resolvedReference)
		}
		result := SecurityResult{
			Candidate: c.Name(), ID: test.id, Reference: test.ref,
			ResolvedReference: resolvedReference, RegistryDenied: registryDenied,
			CompileRejectedByRegistry: compileRejectedByRegistry,
			Passed:                    passed, Trace: compiled.Trace,
		}
		if err != nil {
			result.Error = err.Error()
		} else if resolveErr != nil {
			result.Error = "reference resolution failed: " + resolveErr.Error()
		} else {
			result.Error = fmt.Sprintf("reference was not rejected by the closed registry: expected=%s trace=%+v", resolvedReference, compiled.Trace)
		}
		results = append(results, result)
	}
	return results
}

func resolveReference(base, reference string) (string, error) {
	baseURI, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	referenceURI, err := url.Parse(reference)
	if err != nil {
		return "", err
	}
	return baseURI.ResolveReference(referenceURI).String(), nil
}

func hasRegistryDenial(trace *Trace, uri string) bool {
	for _, denial := range trace.RegistryDenials {
		if denial.URI == uri && denial.Code == RegistryDeniedCode {
			return true
		}
	}
	return false
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
