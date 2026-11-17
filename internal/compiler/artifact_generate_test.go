package compiler

// WP10 assembly conformance: deterministic Artifact generation from validated
// v1alpha4 compiler state. The public Build success path is covered separately
// in artifact_pipeline_test.go.

import (
	"bytes"
	"strings"
	"testing"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func TestArtifactAssemblyDeterministic(t *testing.T) {
	procedure := strings.Replace(scriptProcedure("scripts/compute.py"), "      arguments: {}\n", "      arguments:\n        value: context.ueRef\n", 1)
	req := v4TestRequest(v4TestContract, procedure, v4TestPredicates, v4TestSchema)
	req.Files = append(req.Files, File{Path: "skills/pdu/scripts/compute.py", Bytes: []byte(procedureScriptABI)})

	files := map[string][]byte{}
	parsed := map[string]*yaml.Node{}
	add := func(string, *yaml.Node, string, string) {}
	for _, file := range req.Files {
		files[file.Path] = file.Bytes
	}
	for _, filename := range sortedStringMapKeys(files) {
		if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
			parsed[filename] = v4ParseYAML(filename, files[filename], add)
		}
	}
	ast := parseV1Alpha4SkillAST("pdu", parsed)
	if ast == nil {
		t.Fatal("fixture AST did not parse")
	}
	identity := validateV1Alpha4DependencyIdentity(req, ast)
	ownerLinking := validateV1Alpha4OwnerLinking(req, ast, identity)
	scripts := validateV1Alpha4ScriptABI(req, identity, ast, files, "skills/pdu/assurance/procedure.yaml", "skills/pdu/assurance/predicates.yaml", add)
	if len(scripts.Outputs) != 1 {
		t.Fatalf("expected one derived Script Output Schema, got %d", len(scripts.Outputs))
	}

	first, err := generateV1Alpha4Artifact(req, "pdu", ast, identity, ownerLinking, scripts)
	if err != nil {
		t.Fatalf("artifact assembly: %v", err)
	}
	second, err := generateV1Alpha4Artifact(req, "pdu", ast, identity, ownerLinking, scripts)
	if err != nil {
		t.Fatalf("artifact assembly (second): %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("non-deterministic file count: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].Path != second[i].Path || !bytes.Equal(first[i].Bytes, second[i].Bytes) {
			t.Fatalf("non-deterministic artifact output at %s", first[i].Path)
		}
	}
	hasSchema, hasManifest := false, false
	for _, file := range first {
		switch {
		case strings.HasSuffix(file.Path, "/schemas/generated/compute.output.schema.json"):
			hasSchema = true
			if !bytes.Contains(file.Bytes, []byte(`"$id":"urn:connectionagent:skill:pdu:script-output:compute.py"`)) {
				t.Fatalf("generated Schema has the wrong $id: %s", file.Bytes)
			}
		case file.Path == "skills/pdu/generated/artifact-manifest.json":
			hasManifest = true
		}
	}
	if !hasSchema || !hasManifest {
		names := make([]string, 0, len(first))
		for _, file := range first {
			names = append(names, file.Path)
		}
		t.Fatalf("artifact tree is missing generated Schema or manifest: %v", names)
	}
}
