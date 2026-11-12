package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/odango-chan/ca-skill-compiler/tools/schema-engine-spike"
)

func TestHTTPAndHTTPSReferencesDeniedBeforeNetworkRequest(t *testing.T) {
	for _, scheme := range []string{"http", "https"} {
		t.Run(scheme, func(t *testing.T) {
			var requests atomic.Int64
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				_, _ = w.Write([]byte(`{"type":"string"}`))
			})
			var server *httptest.Server
			if scheme == "https" {
				server = httptest.NewTLSServer(handler)
			} else {
				server = httptest.NewServer(handler)
			}
			defer server.Close()

			root := "https://spike.invalid/security/root.json"
			ref := server.URL + "/schema.json"
			compiled, err := (Candidate{}).Compile(map[string]any{root: securitySchema(ref)}, root, false, "")
			assertRegistryDenial(t, compiled, err, ref)
			if compiled.Trace.ExternalAccesses != 1 || !containsString(compiled.Trace.DeniedExternal, ref) {
				t.Fatalf("external access trace does not identify the denied URI %q: %+v", ref, compiled.Trace)
			}
			if got := requests.Load(); got != 0 {
				t.Fatalf("server received %d request(s)", got)
			}
		})
	}
}

func TestFileReferenceDeniedWithoutReadingFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "valid-schema.json")
	if err := os.WriteFile(file, []byte(`{"type":"string"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	fileURL := &url.URL{Scheme: "file", Path: file}
	fileURI, err := fileURL.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	ref := string(fileURI)
	root := "https://spike.invalid/security/file-root.json"
	compiled, compileErr := (Candidate{}).Compile(map[string]any{root: securitySchema(ref)}, root, false, "")
	assertRegistryDenial(t, compiled, compileErr, ref)
	if !containsString(compiled.Trace.DeniedExternal, ref) {
		t.Fatalf("file URI is not recorded as denied external resource: %+v", compiled.Trace)
	}
}

func TestRelativeReferenceUsesOnlyRegisteredResource(t *testing.T) {
	root := "https://spike.invalid/project/schemas/root.json"
	child := "https://spike.invalid/project/shared/child.json"
	ref := "../shared/child.json"
	compiled, err := (Candidate{}).Compile(map[string]any{
		root:  securitySchema(ref),
		child: map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string"},
	}, root, false, "")
	if err != nil {
		t.Fatalf("registered relative reference failed to compile: %v (trace=%+v)", err, compiled.Trace)
	}
	if !containsString(compiled.Trace.RegistryLookups, child) {
		t.Fatalf("relative URI did not resolve to the registered child: %+v", compiled.Trace)
	}
	if compiled.Trace.ResourcesPassedInMem != 2 || len(compiled.Trace.RegistryDenials) != 0 {
		t.Fatalf("unexpected resolution path: %+v", compiled.Trace)
	}
	if err := compiled.Validator.Validate("accepted"); err != nil {
		t.Fatalf("child schema was not applied: %v", err)
	}
	if err := compiled.Validator.Validate(42); err == nil {
		t.Fatal("child schema did not reject an invalid value")
	}
}

func TestPathTraversalAndMissingReferencesDeniedByRegistry(t *testing.T) {
	for _, tc := range []struct{ name, ref, resolved string }{
		{"path-traversal", "../../outside.json", "https://spike.invalid/outside.json"},
		{"missing-resource", "missing.json", "https://spike.invalid/project/schemas/missing.json"},
		{"unknown-scheme", "gopher://remote.invalid/schema.json", "gopher://remote.invalid/schema.json"},
		{"custom-scheme", "custom://remote.invalid/schema.json", "custom://remote.invalid/schema.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := "https://spike.invalid/project/schemas/root.json"
			compiled, err := (Candidate{}).Compile(map[string]any{root: securitySchema(tc.ref)}, root, false, "")
			assertRegistryDenial(t, compiled, err, tc.resolved)
		})
	}
}

func TestRunSecurityProvesAllEightAttacksWereDenied(t *testing.T) {
	results := spike.RunSecurity(Candidate{})
	if len(results) != 8 {
		t.Fatalf("security cases=%d, want 8", len(results))
	}
	for _, result := range results {
		if !result.Passed || !result.RegistryDenied || !result.CompileRejectedByRegistry {
			t.Errorf("%s did not prove exact registry denial: %+v", result.ID, result)
		}
		if result.Trace == nil {
			t.Errorf("%s has no registry trace", result.ID)
			continue
		}
		if result.ResolvedReference == "" || !containsString(result.Trace.RegistryLookups, result.ResolvedReference) {
			t.Errorf("%s did not look up its resolved target %q: %+v", result.ID, result.ResolvedReference, result.Trace)
		}
	}
}

func TestExternalSchemeTracingIsCaseInsensitive(t *testing.T) {
	const ref = "HTTP://remote.invalid/schema.json"
	registry := spike.NewRegistry(map[string]any{})
	if _, _, err := registry.Resolve(ref); err == nil {
		t.Fatal("unregistered uppercase HTTP reference unexpectedly resolved")
	}
	if registry.Trace.ExternalAccesses != 1 || !containsString(registry.Trace.DeniedExternal, ref) {
		t.Fatalf("uppercase external scheme was not recorded: %+v", registry.Trace)
	}
}

func securitySchema(reference string) map[string]any {
	return map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$ref":    reference,
	}
}

func assertRegistryDenial(t *testing.T, compiled spike.Compiled, err error, wantURI string) {
	t.Helper()
	if err == nil {
		t.Fatalf("reference %q unexpectedly compiled", wantURI)
	}
	if compiled.Trace == nil {
		t.Fatal("compile failure has no registry trace")
	}
	if !containsString(compiled.Trace.RegistryLookups, wantURI) {
		t.Fatalf("registry did not attempt exact reference %q: %+v", wantURI, compiled.Trace)
	}
	if !containsString(compiled.Trace.DeniedReferences, wantURI) {
		t.Fatalf("exact reference %q was not denied: %+v", wantURI, compiled.Trace)
	}
	if !hasDenial(compiled.Trace, wantURI) {
		t.Fatalf("exact reference %q has no coded registry denial: %+v", wantURI, compiled.Trace)
	}
	if !strings.Contains(err.Error(), spike.RegistryDeniedCode) || !strings.Contains(err.Error(), wantURI) {
		t.Fatalf("compile failed for a reason other than denial of %q: %v", wantURI, err)
	}
}

func hasDenial(trace *spike.Trace, uri string) bool {
	for _, denial := range trace.RegistryDenials {
		if denial.URI == uri && denial.Code == spike.RegistryDeniedCode {
			return true
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
