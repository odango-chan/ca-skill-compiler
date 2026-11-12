package main

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestHTTPReferenceDeniedBeforeNetworkRequest(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"type":"string"}`))
	}))
	defer server.Close()
	root := "https://spike.invalid/security/root.json"
	documents := map[string]any{root: map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$ref":    server.URL + "/schema.json",
	}}
	compiled, err := (Candidate{}).Compile(documents, root, false, "")
	if err == nil {
		t.Fatal("remote reference unexpectedly compiled")
	}
	if compiled.Trace == nil || compiled.Trace.ExternalAccesses != 1 {
		t.Fatalf("reference bypassed instrumented registry: %+v", compiled.Trace)
	}
	if requests.Load() != 0 {
		t.Fatalf("server received %d request(s)", requests.Load())
	}
}
