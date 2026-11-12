package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/odango-chan/ca-skill-compiler/tools/schema-engine-spike"
)

func TestApprovedCreatedSessionCatalogSchemas(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	repo := filepath.Clean(filepath.Join(filepath.Dir(source), "../../../.."))
	path := filepath.Join(repo, "examples", "ca-agent-demo-project", "lib", "tools", "createdsession.tool.yaml")
	catalog, err := spike.LoadToolCatalogSchemas(path)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Operations != 12 {
		t.Fatalf("approved operation count = %d, want 12", catalog.Operations)
	}
	if catalog.Common == 0 {
		t.Fatal("expected x-common fixture fragments")
	}
	candidate := Candidate{}
	for _, entry := range catalog.Schemas {
		if _, err := candidate.Compile(map[string]any{entry.URI: entry.Schema}, entry.URI, false, ""); err != nil {
			t.Errorf("%s: %v", entry.Name, err)
		}
		if _, err := candidate.Compile(map[string]any{entry.URI: entry.Schema}, entry.URI, false, "regexp2"); err != nil {
			t.Errorf("%s (regexp2): %v", entry.Name, err)
		}
	}
	t.Logf("compiled 12 approved Operations and %d x-common fragments (%d schemas total)", catalog.Common, len(catalog.Schemas))
}
