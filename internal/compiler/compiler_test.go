package compiler

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func source(name, entry, procedure string) []File {
	prefix := "skills/" + name + "/"
	return []File{
		{Path: prefix + "SKILL.md", Bytes: []byte("# " + name + "\n")},
		{Path: prefix + "assurance/contract.yaml", Bytes: []byte(
			"apiVersion: connectionagent.io/v1alpha1\nkind: SkillContract\nmetadata:\n  name: " + name +
				"\nspec:\n  skillType: executable\n  entryProcedure: " + entry + "\n  allowedResources:\n    capabilities: [demo.read]\n")},
		{Path: prefix + "assurance/procedure.yaml", Bytes: []byte(procedure)},
	}
}

func toolCatalog() File {
	return File{Path: "lib/tools/demo.yaml", Bytes: []byte(
		"apiVersion: ca-agent-demo.tool/v1\nkind: ToolCatalog\nx-common:\n  ref: &ref {type: string, minLength: 1}\nspec:\n  contracts:\n  - operation: demo.read\n    effect: READ\n    inputSchema:\n      type: object\n      required: [ref]\n      properties:\n        ref: *ref\n    outputSchema: {type: object}\n")}
}

func testProject() []File {
	parent := source("root", "start",
		"procedures:\n  start:\n    start: lookup\n    nodes:\n      lookup:\n        call: demo.read\n        with: {ref: {value: sample}}\n      nested:\n        procedure: {skill: child, procedure: run}\n")
	child := source("child", "run",
		"procedures:\n  run:\n    start: done\n    nodes:\n      done:\n        terminal: SUCCEEDED\n")
	return append(append(parent, child...), toolCatalog())
}

func findDiagnostic(got []Diagnostic, code string) bool {
	for _, d := range got {
		if d.Code == code {
			return true
		}
	}
	return false
}

func TestBuildMultiSkillAndImportedToolCatalog(t *testing.T) {
	req := Request{Files: testProject(), Targets: []string{"root"}}
	result := Build(req)
	if !result.Success {
		t.Fatalf("unexpected diagnostics: %#v", result.Diagnostics)
	}
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
		if f.Path != wantPaths[i] {
			t.Fatalf("output[%d] = %q; want %q", i, f.Path, wantPaths[i])
		}
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
	if !result.Success {
		t.Fatalf("unexpected diagnostics: %#v", result.Diagnostics)
	}
	for _, f := range result.OutputFiles {
		if strings.HasPrefix(f.Path, "skills/root/") || strings.HasPrefix(f.Path, "lib/") {
			t.Fatalf("unrelated material linked: %s", f.Path)
		}
	}
}

func TestTargetBuildIgnoresBrokenUnrelatedSkill(t *testing.T) {
	files := testProject()
	files = append(files, source("unrelated", "broken", "procedures: [invalid")...)
	got := Build(Request{Files: files, Targets: []string{"root"}})
	if !got.Success {
		t.Fatalf("unrelated invalid Skill blocked target closure: %#v", got.Diagnostics)
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

func TestMissingBuildTargetIsDiagnosed(t *testing.T) {
	got := Build(Request{Files: testProject(), Targets: []string{"missing"}})
	if got.Success || !findDiagnostic(got.Diagnostics, "TARGET_NOT_FOUND") {
		t.Fatalf("missing build target did not fail: %#v", got)
	}
}

func TestVerifyResolvesPredicateDeclaredInSameSkill(t *testing.T) {
	files := source("pred", "check", "procedures:\n  check:\n    start: v\n    nodes:\n      v:\n        verify: allowed\n")
	files = append(files, File{Path: "skills/pred/assurance/predicates.yaml", Bytes: []byte("predicates:\n  allowed: {evaluator: {type: cel, expression: 'true'}}\n")})
	got := Build(Request{Files: files})
	if !got.Success {
		t.Fatalf("valid Predicate reference failed: %#v", got.Diagnostics)
	}
	for i := range files {
		if files[i].Path == "skills/pred/assurance/procedure.yaml" {
			files[i].Bytes = []byte(strings.Replace(string(files[i].Bytes), "verify: allowed", "verify: absent", 1))
		}
	}
	if d := Check(Request{Files: files}); !findDiagnostic(d, "PREDICATE_NOT_FOUND") {
		t.Fatalf("missing Predicate was not diagnosed: %#v", d)
	}
}

func TestDuplicateOperationFails(t *testing.T) {
	files := testProject()
	two := toolCatalog()
	two.Path = "lib/tools/duplicate.yaml"
	files = append(files, two)
	d := Check(Request{Files: files})
	if !findDiagnostic(d, "LIB_DUPLICATE_OPERATION") {
		t.Fatalf("missing duplicate diagnostic: %#v", d)
	}
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
	if !findDiagnostic(d, "ENTRY_NOT_FOUND") {
		t.Fatalf("missing entry diagnostic: %#v", d)
	}
}

func TestInvalidYAMLFails(t *testing.T) {
	files := testProject()
	for i := range files {
		if files[i].Path == "lib/tools/demo.yaml" {
			files[i].Bytes = []byte("kind: [")
		}
	}
	d := Check(Request{Files: files})
	if !findDiagnostic(d, "YAML_PARSE") {
		t.Fatalf("missing YAML diagnostic: %#v", d)
	}
}

func TestDuplicateYAMLKeyFails(t *testing.T) {
	files := []File{{Path: "lib/tools/bad.yaml", Bytes: []byte("kind: ToolCatalog\nkind: ToolCatalog\nspec: {contracts: [{operation: demo.read, inputSchema: {type: object}, effect: READ}]}\n")}}
	if got := InspectLibraries(Request{Files: files}); !findDiagnostic(got.Diagnostics, "YAML_DUPLICATE_KEY") {
		t.Fatalf("duplicate YAML key accepted: %#v", got.Diagnostics)
	}
}

func TestRecursiveYAMLAliasFailsWithoutPanic(t *testing.T) {
	files := []File{{Path: "lib/tools/cycle.yaml", Bytes: []byte("kind: ToolCatalog\nspec: {contracts: []}\nx: &x [*x]\n")}}
	got := InspectLibraries(Request{Files: files})
	if !findDiagnostic(got.Diagnostics, "YAML_LIMIT") {
		t.Fatalf("recursive YAML alias was not bounded: %#v", got.Diagnostics)
	}
}

func TestDeterminismAndJSONWire(t *testing.T) {
	req := Request{Files: testProject()}
	one, two := Build(req), Build(req)
	if !reflect.DeepEqual(one, two) {
		t.Fatalf("nondeterministic result")
	}
	b, err := json.Marshal(one)
	if err != nil {
		t.Fatal(err)
	}
	var parsed Result
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(one, parsed) {
		t.Fatalf("wire roundtrip differs: %#v vs %#v", one, parsed)
	}
}

func TestManifestAndReadonlyLibraryInspection(t *testing.T) {
	files := []File{toolCatalog(), {Path: "lib/manifest.yaml", Bytes: []byte("libraries:\n  - path: lib/tools/demo.yaml\n    status: proposed\n")}}
	got := InspectLibraries(Request{Files: files})
	if len(got.Diagnostics) != 0 || len(got.Tools) != 1 || got.Tools[0].SourceStatus != "proposed" || got.Tools[0].DeclaredEffect != "READ" {
		t.Fatalf("unexpected library inspection: %#v", got)
	}
	if got.Tools[0].InputSchema == nil || got.Tools[0].OutputSchema == nil {
		t.Fatalf("schema projections missing: %#v", got.Tools[0])
	}
}

func TestGeneratedStrictScriptOutput(t *testing.T) {
	files := []File{
		{Path: "skills/s/assurance/contract.yaml", Bytes: []byte("metadata: {name: s}\nspec: {entryProcedure: p}\n")},
		{Path: "skills/s/SKILL.md", Bytes: []byte("# s")},
		{Path: "skills/s/assurance/procedure.yaml", Bytes: []byte("procedures:\n  p:\n    start: n\n    nodes:\n      n:\n        script: scripts/out.py\n        with: {}\n")},
		{Path: "skills/s/scripts/out.py", Bytes: []byte("from typing import TypedDict, Literal, NotRequired, Annotated\nclass Child(TypedDict):\n    ok: Literal[True]\nclass Output(TypedDict):\n    label: Annotated[str, {\"minLength\": 1}]\n    child: Child\n    extra: NotRequired[int]\ndef main(input: dict) -> Output:\n    return {}\n")},
	}
	got := Build(Request{Files: files})
	if !got.Success {
		t.Fatalf("build failed: %#v", got.Diagnostics)
	}
	var generated []byte
	for _, f := range got.OutputFiles {
		if strings.HasSuffix(f.Path, "out.output.schema.json") {
			generated = f.Bytes
		}
	}
	if len(generated) == 0 || !strings.Contains(string(generated), "additionalProperties") || !strings.Contains(string(generated), "minLength") || !strings.Contains(string(generated), "required") {
		t.Fatalf("generated output schema incomplete: %s", generated)
	}
}

func TestBroadScriptOutputIsRejected(t *testing.T) {
	if _, err := parseScriptOutput("def main(input: dict) -> dict:\n    return {}\n"); err == nil {
		t.Fatal("broad dict output accepted")
	}
}

func TestAnnotatedOutputConstraintsAreStrict(t *testing.T) {
	for _, constraint := range []string{`{"minLength": -1}`, `{"mystery": 1}`, `{"pattern": "["}`} {
		src := "from typing import TypedDict, Annotated\nclass Output(TypedDict):\n    value: Annotated[str, " + constraint + "]\ndef main(input: dict) -> Output:\n    return {}\n"
		if _, err := parseScriptOutput(src); err == nil {
			t.Errorf("invalid constraint %s was accepted", constraint)
		}
	}
}

func TestTypedDictMustHaveTypingImport(t *testing.T) {
	src := "class Output(TypedDict):\n    value: str\ndef main(input: dict) -> Output:\n    return {}\n"
	if _, err := parseScriptOutput(src); err == nil {
		t.Fatal("undefined TypedDict annotation accepted")
	}
}

func TestPinnedToolCatalogsInspectWithProvenance(t *testing.T) {
	paths := []string{"../../examples/ca-agent-demo-project/lib/tools/createdsession.tool.yaml", "../../examples/ca-agent-demo-project/lib/tools/mas-proposed.tool.yaml", "../../examples/ca-agent-demo-project/lib/manifest.yaml"}
	var files []File
	for _, p := range paths {
		b, err := os.ReadFile(filepath.Clean(p))
		if err != nil {
			t.Fatal(err)
		}
		rel := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(p)), "../../examples/ca-agent-demo-project/")
		files = append(files, File{Path: rel, Bytes: b})
	}
	got := InspectLibraries(Request{Files: files})
	if len(got.Diagnostics) != 0 {
		t.Fatalf("catalog inspection diagnostics: %#v", got.Diagnostics)
	}
	if len(got.Tools) != 28 {
		t.Fatalf("wanted 28 imported Operations, got %d", len(got.Tools))
	}
	proposed := 0
	for _, tool := range got.Tools {
		if tool.SourceStatus == "proposed" {
			proposed++
		}
		if tool.InputSchema == nil {
			t.Errorf("missing input schema for %s", tool.OperationID)
		}
	}
	if proposed != 16 {
		t.Fatalf("wanted 16 proposed Operations, got %d", proposed)
	}
}

func TestBindingRejectsUnknownProducerField(t *testing.T) {
	files := testProject()
	for i := range files {
		if files[i].Path == "skills/root/assurance/procedure.yaml" {
			files[i].Bytes = []byte(strings.Replace(string(files[i].Bytes), "with: {ref: {value: sample}}", "with: {ref: {from: lookup.output.missing}}", 1))
		}
	}
	if got := Check(Request{Files: files}); !findDiagnostic(got, "SOURCE_FIELD_UNKNOWN") {
		t.Fatalf("unknown producer field was accepted: %#v", got)
	}
}

func TestBindingRequiresImportedHostSchema(t *testing.T) {
	files := testProject()
	for i := range files {
		if files[i].Path == "skills/root/assurance/procedure.yaml" {
			files[i].Bytes = []byte(strings.Replace(string(files[i].Bytes), "with: {ref: {value: sample}}", "with: {ref: {from: context.facts.unowned}}", 1))
		}
	}
	if got := Check(Request{Files: files}); !findDiagnostic(got, "EXTERNAL_SCHEMA_MISSING") {
		t.Fatalf("missing Host contract was accepted: %#v", got)
	}
}

func TestMinimalFixtureNativeOutputOrder(t *testing.T) {
	paths := []string{"skills/reader/SKILL.md", "skills/reader/assurance/contract.yaml", "skills/reader/assurance/procedure.yaml", "skills/writer/SKILL.md", "skills/writer/assurance/contract.yaml", "skills/writer/assurance/procedure.yaml", "lib/tools/demo.tool.yaml"}
	var files []File
	for _, p := range paths {
		b, err := os.ReadFile(filepath.Join("../../examples/minimal-project", p))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, File{Path: p, Bytes: b})
	}
	got := Build(Request{Files: files})
	if !got.Success {
		t.Fatalf("minimal project build failed: %#v", got.Diagnostics)
	}
	want := []string{"lib/tools/demo.tool.yaml", "skills/reader/SKILL.md", "skills/reader/assurance/contract.yaml", "skills/reader/assurance/procedure.yaml", "skills/writer/SKILL.md", "skills/writer/assurance/contract.yaml", "skills/writer/assurance/procedure.yaml"}
	if len(got.OutputFiles) != len(want) {
		t.Fatalf("got %d output files", len(got.OutputFiles))
	}
	for i, f := range got.OutputFiles {
		if f.Path != want[i] {
			t.Fatalf("output[%d]=%s, want %s", i, f.Path, want[i])
		}
	}
}
