package compiler

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func source(name, entry, procedure string) []File {
	prefix := "skills/" + name + "/"
	return []File{
		{Path: prefix+"SKILL.md", Bytes: []byte("# "+name+"\n")},
		{Path: prefix+"assurance/contract.yaml", Bytes: []byte(
			"apiVersion: connectionagent.io/v1alpha1\nkind: SkillContract\nmetadata:\n  name: "+name+
			"\nspec:\n  skillType: executable\n  entryProcedure: "+entry+"\n  allowedResources:\n    capabilities: [demo.read]\n")},
		{Path: prefix+"assurance/procedure.yaml", Bytes: []byte(procedure)},
	}
}

func toolCatalog() File {
	return File{Path: "lib/tools/demo.yaml", Bytes: []byte(
		"apiVersion: ca-agent-demo.tool/v1\nkind: ToolCatalog\nx-common:\n  ref: &ref {type: string, minLength: 1}\nspec:\n  contracts:\n  - operation: demo.read\n    effect: READ\n    inputSchema:\n      type: object\n      required: [ref]\n      properties:\n        ref: *ref\n    outputSchema: {type: object}\n")}
}

func testProject() []File {
	parent := source("root", "start",
		"procedures:\n  start:\n    start: lookup\n    nodes:\n      lookup:\n        call: demo.read\n        with: {}\n      nested:\n        procedure: {skill: child, procedure: run}\n")
	child := source("child", "run",
		"procedures:\n  run:\n    start: done\n    nodes:\n      done:\n        terminal: SUCCEEDED\n")
	return append(append(parent, child...), toolCatalog())
}

func findDiagnostic(got []Diagnostic, code string) bool {
	for _, d := range got { if d.Code == code { return true } }
	return false
}

func TestBuildMultiSkillAndImportedToolCatalog(t *testing.T) {
	req := Request{Files: testProject(), Targets: []string{"root"}}
	result := Build(req)
	if !result.Success { t.Fatalf("unexpected diagnostics: %#v", result.Diagnostics) }
	if len(result.OutputFiles) != 7 {
		t.Fatalf("wanted 7 output files, got %d: %#v", len(result.OutputFiles), result.OutputFiles)
	}
	wantPaths := []string{
		"lib/tools/demo.yaml",
		"skills/child/SKILL.md", "skills/child/assurance/contract.yaml",
		"skills/child/assurance/procedure.yaml",
		"skills/root/SKILL.md", "skills/root/assurance/contract.yaml",
		"skills/root/assurance/procedure.yaml",
	}
	for i, f := range result.OutputFiles {
		if f.Path != wantPaths[i] { t.Fatalf("output[%d] = %q; want %q", i, f.Path, wantPaths[i]) }
	}
	// Snapshot must not share mutable byte arrays with the inputs.
	req.Files[0].Bytes[0] = 'x'
	if bytes.Equal(result.OutputFiles[4].Bytes, req.Files[0].Bytes) {
		t.Fatal("output leaked a mutable source byte slice")
	}
}

func TestBuildCanSelectOneSkill(t *testing.T) {
	req := Request{Files: testProject(), Targets: []string{"child"}}
	result := Build(req)
	if !result.Success { t.Fatalf("unexpected diagnostics: %#v", result.Diagnostics) }
	for _, f := range result.OutputFiles {
		if strings.HasPrefix(f.Path, "skills/root/") || strings.HasPrefix(f.Path, "lib/") {
			t.Fatalf("unrelated material linked: %s", f.Path)
		}
	}
}

func TestMissingLibraryOperationFails(t *testing.T) {
	files := testProject()
	files = files[:len(files)-1]
	result := Build(Request{Files: files})
	if result.Success || !findDiagnostic(result.Diagnostics, "TOOL_NOT_FOUND") || len(result.OutputFiles) != 0 {
		t.Fatalf("must fail without the imported tool: %#v", result)
	}
}

func TestDuplicateOperationFails(t *testing.T) {
	files := testProject()
	two := toolCatalog()
	two.Path = "lib/tools/duplicate.yaml"
	files = append(files, two)
	d := Check(Request{Files: files})
	if !findDiagnostic(d, "LIB_DUPLICATE_OPERATION") { t.Fatalf("missing duplicate diagnostic: %#v", d) }
}

func TestNonNormalizedAndDuplicatePathsFail(t *testing.T) {
	input := testProject()
	input = append(input, File{Path: "skills/../escape.txt", Bytes: []byte("bad")}, input[0])
	result := Build(Request{Files: input})
	if !findDiagnostic(result.Diagnostics, "PATH_INVALID") || !findDiagnostic(result.Diagnostics, "DUPLICATE_PATH") || result.Success {
		t.Fatalf("unsafe input accepted: %#v", result.Diagnostics)
	}
}

func TestMissingEntryProcedureFails(t *testing.T) {
	files := testProject()
	for i := range files {
		if files[i].Path == "skills/root/assurance/contract.yaml" {
			files[i].Bytes = []byte(strings.ReplaceAll(string(files[i].Bytes), "entryProcedure: start", "entryProcedure: missing"))
		}
	}
	d := Check(Request{Files: files})
	if !findDiagnostic(d, "ENTRY_NOT_FOUND") { t.Fatalf("missing entry diagnostic: %#v", d) }
}

func TestInvalidYAMLFails(t *testing.T) {
	files := testProject()
	for i := range files {
		if files[i].Path == "lib/tools/demo.yaml" { files[i].Bytes = []byte("kind: [") }
	}
	d := Check(Request{Files: files})
	if !findDiagnostic(d, "YAML_PARSE") { t.Fatalf("missing YAML diagnostic: %#v", d) }
}

func TestDeterminismAndJSONWire(t *testing.T) {
	req := Request{Files: testProject()}
	one, two := Build(req), Build(req)
	if !reflect.DeepEqual(one, two) { t.Fatalf("nondeterministic result") }
	b, err := json.Marshal(one)
	if err != nil { t.Fatal(err) }
	var parsed Result
	if err := json.Unmarshal(b, &parsed); err != nil { t.Fatal(err) }
	if !reflect.DeepEqual(one, parsed) { t.Fatalf("wire roundtrip differs: %#v vs %#v", one, parsed) }
}
