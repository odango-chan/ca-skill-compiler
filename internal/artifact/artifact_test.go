package artifact

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

func sampleInput() Input {
	schema, err := ScriptOutputSchemaFile("establish-pdu-session", "scripts/common/build_policy.py", map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           map[string]any{"result": map[string]any{"type": "string"}},
		"required":             []string{"result"},
	})
	if err != nil {
		panic(err)
	}
	return Input{
		SkillName:        "establish-pdu-session",
		Goal:             "pdu_session.established",
		GeneratorVersion: "v1alpha4-dev",
		GeneratorCommit:  "30f71ca",
		SourceFiles: []File{
			{Path: "skills/establish-pdu-session/SKILL.md", Bytes: []byte("# skill\n")},
			{Path: "skills/establish-pdu-session/assurance/contract.yaml", Bytes: []byte("goal: pdu_session.established\n")},
		},
		GeneratedSchemas: []File{schema},
		Dependency: []DependencyClosureEntry{
			{DependencyID: "ca", StablePackageID: "urn:example:5gc-tool-contracts", Exports: []ExportRef{
				{ExportKind: "operation", ExportID: "example.session.create", Owner: "example-ue-owner"},
			}},
		},
	}
}

func TestBuildDeterministicAndManifestValid(t *testing.T) {
	first, err := Build(sampleInput())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	second, err := Build(sampleInput())
	if err != nil {
		t.Fatalf("Build twice: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("non-deterministic file count: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].Path != second[i].Path || !bytes.Equal(first[i].Bytes, second[i].Bytes) {
			t.Fatalf("non-deterministic output at %d: %s", i, first[i].Path)
		}
	}
	manifestIndex := -1
	for i, file := range first {
		if file.Path == ManifestPath("establish-pdu-session") {
			manifestIndex = i
		}
	}
	if manifestIndex < 0 {
		t.Fatalf("manifest missing from output tree")
	}
	manifestBytes := first[manifestIndex].Bytes
	if bytes.HasSuffix(manifestBytes, []byte("\n")) {
		t.Fatalf("manifest must not have a trailing newline")
	}
	validateAgainstFrozenSchema(t, manifestBytes)
}

func validateAgainstFrozenSchema(t *testing.T, manifestBytes []byte) {
	t.Helper()
	raw, err := os.ReadFile("../../contracts/v1alpha4/artifact-manifest.schema.json")
	if err != nil {
		t.Fatalf("read manifest schema: %v", err)
	}
	var schemaValue any
	if err := json.Unmarshal(raw, &schemaValue); err != nil {
		t.Fatalf("decode manifest schema: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err := compiler.AddResource("urn:test:artifact-manifest", schemaValue); err != nil {
		t.Fatalf("register manifest schema: %v", err)
	}
	schema, err := compiler.Compile("urn:test:artifact-manifest")
	if err != nil {
		t.Fatalf("compile manifest schema: %v", err)
	}
	var instance any
	decoder := json.NewDecoder(bytes.NewReader(manifestBytes))
	decoder.UseNumber()
	if err := decoder.Decode(&instance); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if err := schema.Validate(instance); err != nil {
		t.Fatalf("manifest violates the frozen schema: %v", err)
	}
}

func TestBuildRejectsInvalidTrees(t *testing.T) {
	cases := map[string]func(*Input){
		"missing goal":       func(in *Input) { in.Goal = "" },
		"missing source":     func(in *Input) { in.SourceFiles = nil },
		"duplicate path":     func(in *Input) { in.SourceFiles = append(in.SourceFiles, in.SourceFiles[0]) },
		"unsorted generated": func(in *Input) { in.GeneratedSchemas = append(in.GeneratedSchemas, in.GeneratedSchemas[0]) },
		"manifest collision": func(in *Input) {
			in.SourceFiles = append(in.SourceFiles, File{Path: ManifestPath(in.SkillName), Bytes: []byte("{}")})
		},
	}
	for name, mutate := range cases {
		input := sampleInput()
		mutate(&input)
		if _, err := Build(input); err == nil {
			t.Fatalf("%s: invalid Artifact tree accepted", name)
		}
	}
}

func TestScriptOutputSchemaIdentity(t *testing.T) {
	path, err := ScriptOutputSchemaPath("establish-pdu-session", "scripts/common/build_policy.py")
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	if path != "skills/establish-pdu-session/schemas/generated/common/build_policy.output.schema.json" {
		t.Fatalf("unexpected Schema path: %s", path)
	}
	id, err := ScriptOutputSchemaID("establish-pdu-session", "scripts/common/build_policy.py")
	if err != nil {
		t.Fatalf("id: %v", err)
	}
	if id != "urn:connectionagent:skill:establish-pdu-session:script-output:common%2Fbuild_policy.py" {
		t.Fatalf("unexpected Schema $id: %s", id)
	}

	file, err := ScriptOutputSchemaFile("establish-pdu-session", "scripts/common/build_policy.py", map[string]any{"type": "object", "properties": map[string]any{}})
	if err != nil {
		t.Fatalf("schema file: %v", err)
	}
	if bytes.HasSuffix(file.Bytes, []byte("\n")) {
		t.Fatalf("generated Schema must not have a trailing newline")
	}
	if !bytes.Contains(file.Bytes, []byte(`"$id":"urn:connectionagent:skill:establish-pdu-session:script-output:common%2Fbuild_policy.py"`)) {
		t.Fatalf("generated Schema lacks the exact $id: %s", file.Bytes)
	}
	if !bytes.HasPrefix(file.Bytes, []byte(`{"$id":`)) {
		t.Fatalf("JCS must sort keys with $id first: %s", file.Bytes)
	}
}

func TestCanonicalizeValueSortsKeys(t *testing.T) {
	canonical, err := CanonicalizeValue(map[string]any{"b": 1, "a": "x"})
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}
	if string(canonical) != `{"a":"x","b":1}` {
		t.Fatalf("unexpected canonical bytes: %s", canonical)
	}
	if _, err := CanonicalizeJSON([]byte(`{"a":1,}`)); err == nil {
		t.Fatalf("invalid JSON must be rejected")
	}
}
