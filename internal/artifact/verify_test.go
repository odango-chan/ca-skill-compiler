package artifact

// Artifact integrity verification: synthetic tree acceptance and tampering.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gowebpki/jcs"
)

func verifiedTree(t *testing.T) []File {
	t.Helper()
	skill := "demo"
	schema, err := ScriptOutputSchemaFile(skill, "scripts/compute.py", map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           map[string]any{},
		"required":             []any{},
	})
	if err != nil {
		t.Fatalf("generated schema: %v", err)
	}
	manifestValue := map[string]any{
		"apiVersion":        APIVersion,
		"languageTarget":    LanguageTarget,
		"skill":             map[string]any{"name": skill, "goal": "demo goal"},
		"generator":         map[string]any{"name": GeneratorName, "version": "test"},
		"sourceFiles":       []any{"skills/demo/SKILL.md"},
		"generatedFiles":    []any{schema.Path},
		"dependencyClosure": []any{},
	}
	encoded, err := json.Marshal(manifestValue)
	if err != nil {
		t.Fatalf("manifest marshal: %v", err)
	}
	canonical, err := jcs.Transform(encoded)
	if err != nil {
		t.Fatalf("manifest canonicalize: %v", err)
	}
	return []File{
		{Path: "skills/demo/SKILL.md", Bytes: []byte("# demo\n")},
		schema,
		{Path: ManifestPath(skill), Bytes: canonical},
	}
}

func TestVerifyAcceptsConsistentTree(t *testing.T) {
	if err := Verify(verifiedTree(t)); err != nil {
		t.Fatalf("consistent artifact tree rejected: %v", err)
	}
}

func TestVerifyRejectsTamperedGeneratedSchema(t *testing.T) {
	files := verifiedTree(t)
	for index := range files {
		if strings.Contains(files[index].Path, "/schemas/generated/") {
			files[index].Bytes = []byte(strings.Replace(string(files[index].Bytes), "urn:connectionagent:skill:", "urn:connectionagent:evil:", 1))
		}
	}
	if err := Verify(files); err == nil {
		t.Fatal("tampered generated Schema identity accepted")
	}
}

func TestVerifyRejectsMissingManifestEntry(t *testing.T) {
	files := verifiedTree(t)
	files = append(files, File{Path: "skills/demo/LICENSE", Bytes: []byte("x")})
	if err := Verify(files); err == nil {
		t.Fatal("unlisted artifact file accepted")
	}
}
