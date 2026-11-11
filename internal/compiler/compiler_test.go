package compiler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func source(name, entry, procedure string) []File {
	prefix := "skills/" + name + "/"
	return []File{
		{Path: prefix + "SKILL.md", Bytes: []byte("---\nname: " + name + "\ndescription: Test fixture for compiler validation.\n---\n\n# " + name + "\n")},
		{Path: prefix + "assurance/contract.yaml", Bytes: []byte(
			"apiVersion: connectionagent.io/v1alpha1\nkind: SkillContract\nmetadata:\n  name: " + name +
				"\nspec:\n  skillType: executable\n  entryProcedure: " + entry + "\n  allowedResources:\n    capabilities: [demo.read]\n")},
		{Path: prefix + "assurance/procedure.yaml", Bytes: []byte(procedure)},
	}
}

func loadProjectFilesForTest(t *testing.T, root string) []File {
	t.Helper()
	var files []File
	for _, dir := range []string{"skills", "lib"} {
		base := filepath.Join(root, dir)
		err := filepath.WalkDir(base, func(filename string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			content, err := os.ReadFile(filename)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, filename)
			if err != nil {
				return err
			}
			files = append(files, File{Path: filepath.ToSlash(rel), Bytes: content})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return files
}

func TestNativeDiagnosticResultsMatchBrowserGoldens(t *testing.T) {
	files := loadProjectFilesForTest(t, filepath.Join("..", "..", "examples", "script-contract-project"))
	var golden struct {
		TargetNotFound               []Diagnostic `json:"targetNotFound"`
		InvalidBinding               []Diagnostic `json:"invalidBinding"`
		BindingTypeMismatch          []Diagnostic `json:"bindingTypeMismatch"`
		UnknownOperation             []Diagnostic `json:"unknownOperation"`
		ProcedureOutputMissingSchema []Diagnostic `json:"procedureOutputMissingSchema"`
		InvalidScriptOutput          []Diagnostic `json:"invalidScriptOutput"`
		DuplicateLibrary             []Diagnostic `json:"duplicateLibrary"`
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "web", "diagnostics-golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatal(err)
	}
	if got := Check(Request{Files: files, Targets: []string{"missing"}}); !reflect.DeepEqual(got, golden.TargetNotFound) {
		t.Fatalf("Native missing-target diagnostics differ from browser golden:\ngot  %#v\nwant %#v", got, golden.TargetNotFound)
	}
	invalidBinding := make([]File, len(files))
	for i, file := range files {
		invalidBinding[i] = File{Path: file.Path, Bytes: bytes.Clone(file.Bytes)}
		if file.Path == "skills/normalize/assurance/procedure.yaml" {
			invalidBinding[i].Bytes = []byte(strings.Replace(string(file.Bytes), "          key: {from: context.request.dnn}", "          unexpected: {value: sample}", 1))
			if bytes.Equal(invalidBinding[i].Bytes, file.Bytes) {
				t.Fatal("negative binding fixture mutation did not apply")
			}
		}
	}
	if got := Check(Request{Files: invalidBinding}); !reflect.DeepEqual(got, golden.InvalidBinding) {
		t.Fatalf("Native invalid-binding diagnostics differ from browser golden:\ngot  %#v\nwant %#v", got, golden.InvalidBinding)
	}
	typeMismatch := mutateProjectFile(t, files, "skills/normalize/schemas/normalize-request.schema.yaml", "dnn: {type: string, minLength: 1}", "dnn: {type: integer}")
	if got := Check(Request{Files: typeMismatch}); !reflect.DeepEqual(got, golden.BindingTypeMismatch) {
		t.Fatalf("Native binding type diagnostics differ from browser golden:\ngot  %#v\nwant %#v", got, golden.BindingTypeMismatch)
	}
	unknownOperation := mutateProjectFile(t, files, "skills/normalize/assurance/procedure.yaml", "call: demo.read", "call: demo.missing")
	if got := Check(Request{Files: unknownOperation}); !reflect.DeepEqual(got, golden.UnknownOperation) {
		t.Fatalf("Native unknown-operation diagnostics differ from browser golden:\ngot  %#v\nwant %#v", got, golden.UnknownOperation)
	}
	procedureOutput := make([]File, len(files))
	for i, file := range files {
		procedureOutput[i] = File{Path: file.Path, Bytes: bytes.Clone(file.Bytes)}
		if file.Path == "skills/normalize/assurance/procedure.yaml" {
			procedureOutput[i].Bytes = []byte("procedures:\n  normalize:\n    start: invoke\n    nodes:\n      invoke:\n        procedure: helper\n        'on':\n          SUCCEEDED: normalize-step\n          FAILED: failed\n      normalize-step:\n        preconditions:\n          steps:\n            - name: normalize-value\n              script: scripts/normalize.py\n\n\n\n\n              with:\n                value: {from: invoke.output.value}\n        terminal: SUCCEEDED\n      failed: {terminal: FAILED}\n  helper:\n    start: done\n    nodes:\n      done: {terminal: SUCCEEDED}\n")
		}
	}
	if got := Check(Request{Files: procedureOutput}); !reflect.DeepEqual(got, golden.ProcedureOutputMissingSchema) {
		t.Fatalf("Native missing-Procedure-output diagnostics differ from browser golden:\\ngot  %#v\\nwant %#v", got, golden.ProcedureOutputMissingSchema)
	}
	invalidOutput := mutateProjectFile(t, files, "skills/normalize/scripts/normalize.py", "normalized: Annotated[str, {\"minLength\": 1, \"pattern\": r\"^[a-z0-9.-]+$\"}]", "normalized: Any")
	if got := Check(Request{Files: invalidOutput}); !reflect.DeepEqual(got, golden.InvalidScriptOutput) {
		t.Fatalf("Native invalid-Script diagnostics differ from browser golden:\ngot  %#v\nwant %#v", got, golden.InvalidScriptOutput)
	}
	duplicate := toolCatalog()
	duplicate.Path = "lib/tools/duplicate.tool.yaml"
	duplicateResult := InspectLibraries(Request{Files: append(append([]File(nil), files...), duplicate)})
	if !reflect.DeepEqual(duplicateResult.Diagnostics, golden.DuplicateLibrary) {
		t.Fatalf("Native InspectLibraries duplicate-operation diagnostics differ from browser golden:\ngot  %#v\nwant %#v", duplicateResult.Diagnostics, golden.DuplicateLibrary)
	}
}

func mutateProjectFile(t *testing.T, files []File, path, before, after string) []File {
	t.Helper()
	mutated := make([]File, len(files))
	changed := false
	for i, file := range files {
		mutated[i] = File{Path: file.Path, Bytes: bytes.Clone(file.Bytes)}
		if file.Path == path {
			text := strings.Replace(string(file.Bytes), before, after, 1)
			if text == string(file.Bytes) {
				t.Fatalf("test mutation did not apply to %s", path)
			}
			mutated[i].Bytes = []byte(text)
			changed = true
		}
	}
	if !changed {
		t.Fatalf("test file missing: %s", path)
	}
	return mutated
}

func toolCatalog() File {
	return File{Path: "lib/tools/demo.yaml", Bytes: []byte(
		"apiVersion: ca-agent-demo.tool/v1\nkind: ToolCatalog\nmetadata: {name: test, owner: tests}\nx-common:\n  ref: &ref {type: string, minLength: 1}\nspec:\n  contracts:\n  - operation: demo.read\n    effect: READ\n    inputSchema:\n      type: object\n      required: [ref]\n      properties:\n        ref: *ref\n    outputSchema: {type: object}\n")}
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

func TestRequestFileOrderDoesNotChangeBuildOrDiagnostics(t *testing.T) {
	files := testProject()
	baseline := Build(Request{Files: files})
	for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
		files[i], files[j] = files[j], files[i]
	}
	shuffled := Build(Request{Files: files})
	if !reflect.DeepEqual(baseline, shuffled) {
		t.Fatalf("input file order changed Build result:\nbaseline=%#v\nshuffled=%#v", baseline, shuffled)
	}
	broken := append(testProject(), File{Path: "skills/broken/assurance/procedure.yaml", Bytes: []byte("procedures: [broken")})
	check := Check(Request{Files: broken})
	for i, j := 0, len(broken)-1; i < j; i, j = i+1, j-1 {
		broken[i], broken[j] = broken[j], broken[i]
	}
	if got := Check(Request{Files: broken}); !reflect.DeepEqual(check, got) {
		t.Fatalf("input file order changed diagnostics:\nfirst=%#v\nsecond=%#v", check, got)
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

func TestOutcomeBranchesRejectEmptyMapsAndNames(t *testing.T) {
	for _, branch := range []string{"'on': {}", "'on': {\"\": done}", "postconditions: {on: {}}"} {
		files := source("branch", "run", "procedures:\n  run:\n    start: read\n    nodes:\n      read:\n        call: demo.read\n        with: {ref: {value: sample}}\n        "+branch+"\n      done: {terminal: SUCCEEDED}\n")
		files = append(files, toolCatalog())
		got := Build(Request{Files: files})
		if got.Success || !findDiagnostic(got.Diagnostics, "PROCEDURE_UNSUPPORTED") {
			t.Fatalf("malformed outcome branch %q was accepted: %#v", branch, got.Diagnostics)
		}
		for _, d := range got.Diagnostics {
			if d.Code == "PROCEDURE_UNSUPPORTED" && d.Path != "skills/branch/assurance/procedure.yaml" {
				t.Fatalf("branch diagnostic has wrong source path: %#v", d)
			}
		}
	}
}

func TestMissingLibraryOperationFails(t *testing.T) {
	files := testProject()
	files = files[:len(files)-1]
	result := Build(Request{Files: files})
	if result.Success || !findDiagnostic(result.Diagnostics, "LIB_OPERATION_UNKNOWN") || len(result.OutputFiles) != 0 {
		t.Fatalf("must fail without the imported tool: %#v", result)
	}
}

func TestProjectLocalSchemaRefsAreResolvedWithoutFilesystemOrNetwork(t *testing.T) {
	files := source("refs", "run", `procedures:
  run:
    start: read
    nodes:
      read:
        call: demo.read
        with: {definitions: {value: sample}}
`)
	files = append(files, File{Path: "lib/tools/ref.yaml", Bytes: []byte(`apiVersion: ca-agent-demo.tool/v1
kind: ToolCatalog
x-common:
  Ref: {$ref: '../schemas/fields.yaml#/definitions/Ref'}
metadata: {name: test, owner: tests}
spec:
  contracts:
  - operation: demo.read
    effect: READ
    inputSchema:
      type: object
      required: [definitions]
      properties:
        definitions: {$ref: '#/x-common/Ref'}
    outputSchema: {type: object}
`)}, File{Path: "lib/schemas/fields.yaml", Bytes: []byte("definitions:\n  Ref: {type: string, minLength: 1}\n")})
	got := Build(Request{Files: files})
	if !got.Success {
		t.Fatalf("local JSON Pointer ref should resolve: %#v", got.Diagnostics)
	}
	foundReferencedLibrary := false
	for _, file := range got.OutputFiles {
		if file.Path == "lib/schemas/fields.yaml" {
			foundReferencedLibrary = true
		}
	}
	if !foundReferencedLibrary {
		t.Fatal("transitively referenced read-only Library Schema was not packaged")
	}

	for i := range files {
		if files[i].Path == "lib/tools/ref.yaml" {
			files[i].Bytes = []byte(strings.Replace(string(files[i].Bytes), "#/x-common/Ref", "https://example.invalid/schema.json", 1))
		}
	}
	got = Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "LIB_SCHEMA_INVALID") {
		t.Fatalf("remote schema reference should fail closed: %#v", got.Diagnostics)
	}
}

func TestSchemaReferenceCyclesAreDiagnosed(t *testing.T) {
	files := source("refs", "run", `procedures:
  run:
    start: read
    nodes:
      read: {call: demo.read, with: {ref: {value: sample}}}
`)
	files = append(files, File{Path: "lib/tools/ref.yaml", Bytes: []byte(`apiVersion: ca-agent-demo.tool/v1
kind: ToolCatalog
metadata: {name: test, owner: tests}
x-common:
  Ref: {$ref: '#/x-common/Ref'}
spec:
  contracts:
  - operation: demo.read
    effect: READ
    inputSchema:
      type: object
      required: [ref]
      properties:
        ref: {$ref: '#/x-common/Ref'}
`)})
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "LIB_SCHEMA_INVALID") {
		t.Fatalf("cyclic Schema reference should be diagnosed: %#v", got.Diagnostics)
	}
}

func TestMalformedUnusedLibrarySchemaIsDiagnosed(t *testing.T) {
	files := source("unused", "run", "procedures:\n  run:\n    start: done\n    nodes:\n      done: {terminal: SUCCEEDED}\n")
	files = append(files, File{Path: "lib/tools/unused.yaml", Bytes: []byte(`apiVersion: ca-agent-demo.tool/v1
kind: ToolCatalog
metadata: {name: test, owner: tests}
spec:
  contracts:
  - operation: demo.unused
    effect: READ
    inputSchema: {type: object}
    outputSchema:
      type: object
      properties:
        value: invalid
`)})
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "LIB_SCHEMA_INVALID") {
		t.Fatalf("malformed unused Tool Schema was ignored: %#v", got.Diagnostics)
	}
}

func TestSchemaShapeValidatesDraft2020ConstraintTypes(t *testing.T) {
	valid := schemaNode(map[string]any{
		"type": "array", "items": map[string]any{"type": "object"},
		"minItems": 1, "maxItems": 4, "contains": map[string]any{"type": "string"},
		"minContains": 1, "maxContains": 2, "unevaluatedItems": false,
	})
	if err := validateSchemaShape(valid); err != nil {
		t.Fatalf("valid Draft 2020 constraints rejected: %v", err)
	}
	invalid := []map[string]any{
		{"type": "object", "minProperties": "1"},
		{"type": "string", "minLength": 1.5},
		{"type": "object", "required": []any{"key", "key"}},
		{"type": "object", "patternProperties": map[string]any{"[": map[string]any{"type": "string"}}},
		{"type": "object", "additionalProperties": "true"},
		{"type": "array", "contains": map[string]any{}, "minContains": 3, "maxContains": 2},
		{"type": "string", "enum": []any{"same", "same"}},
		{"type": "number", "enum": []any{1, 1.0}},
		{"type": "array", "unevaluatedItems": "false"},
	}
	for _, source := range []string{"type: object\ntrue: {type: string}\n", "type: object\nproperties:\n  true: {type: string}\n", "type: object\ndependentRequired:\n  true: [value]\n"} {
		var document yaml.Node
		if err := yaml.Unmarshal([]byte(source), &document); err != nil {
			t.Fatal(err)
		}
		if err := validateSchemaShape(document.Content[0]); err == nil {
			t.Errorf("non-string JSON object key was accepted: %q", source)
		}
	}
	for _, candidate := range invalid {
		if err := validateSchemaShape(schemaNode(candidate)); err == nil {
			t.Errorf("invalid Schema accepted: %#v", candidate)
		}
	}
	var document yaml.Node
	if err := yaml.Unmarshal([]byte("type: number\nminimum: 9007199254740993\nmaximum: 9007199254740994\nmultipleOf: 1e-400\n"), &document); err != nil {
		t.Fatal(err)
	}
	if err := validateSchemaShape(document.Content[0]); err != nil {
		t.Fatalf("exact large bounds and small positive multipleOf were rejected: %v", err)
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
	files = append(files, File{Path: "skills/pred/assurance/predicates.yaml", Bytes: []byte(predicateSetSource("pred", "predicates:\n  allowed: {evaluator: {type: cel, expression: 'true'}}\n"))})
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

func TestPredicateDefinitionsRequireStaticSourceShape(t *testing.T) {
	files := source("pred", "check", "procedures:\n  check:\n    start: v\n    nodes:\n      v:\n        verify: allowed\n")
	files = append(files, File{Path: "skills/pred/assurance/predicates.yaml", Bytes: []byte(predicateSetSource("pred", "predicates:\n  allowed: {observations: []}\n"))})
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "PREDICATE_SOURCE_INCOMPLETE") {
		t.Fatalf("incomplete Predicate definition accepted: %#v", got.Diagnostics)
	}
}

func TestPredicateEvaluatorAndObservationContractsAreStrict(t *testing.T) {
	cases := []struct {
		name       string
		predicates string
	}{
		{"unknown PredicateSet field", "runtimeOptions: legacy\npredicates:\n  ready: {evaluator: {type: cel, expression: 'true'}}\n"},
		{"CEL expression must be string", "predicates:\n  ready:\n    evaluator: {type: cel, expression: true}\n"},
		{"unsupported CEL field", "predicates:\n  ready:\n    evaluator: {type: cel, expression: 'true', ref: skill.check}\n"},
		{"unsupported script field", "predicates:\n  ready:\n    evaluator: {type: script, ref: skill.check, expression: 'true'}\n"},
		{"observation ref must be string", "predicates:\n  ready:\n    observations: {state: {ref: true}}\n    evaluator: {type: cel, expression: 'true'}\n"},
		{"observation scope must be string", "predicates:\n  ready:\n    observations: {state: {ref: state.read, scope: true}}\n    evaluator: {type: cel, expression: 'true'}\n"},
		{"observation freshness must be string", "predicates:\n  ready:\n    observations: {state: {ref: state.read, freshness: 5}}\n    evaluator: {type: cel, expression: 'true'}\n"},
		{"unknown observation field", "predicates:\n  ready:\n    observations: {state: {ref: state.read, runtime: active}}\n    evaluator: {type: cel, expression: 'true'}\n"},
		{"unknown Predicate field", "predicates:\n  ready:\n    customRuntime: enabled\n    evaluator: {type: cel, expression: 'true'}\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := source("strict-predicate", "run", "procedures:\n  run:\n    start: done\n    nodes: {done: {terminal: SUCCEEDED}}\n")
			files = append(files, File{Path: "skills/strict-predicate/assurance/predicates.yaml", Bytes: []byte(predicateSetSource("strict-predicate", tc.predicates))})
			got := Build(Request{Files: files})
			if got.Success || !findDiagnostic(got.Diagnostics, "PREDICATE_SOURCE_INCOMPLETE") {
				t.Fatalf("unsupported Predicate declaration was accepted: %#v", got.Diagnostics)
			}
		})
	}
}

func TestPredicateSetAllowsStandardEnvelopeFields(t *testing.T) {
	files := source("predicate-envelope", "run", "procedures:\n  run:\n    start: check\n    nodes:\n      check: {verify: ready}\n")
	files = append(files, File{Path: "skills/predicate-envelope/assurance/predicates.yaml", Bytes: []byte(predicateSetSource("predicate-envelope", "predicates:\n  ready: {evaluator: {type: cel, expression: 'true'}}\n"))})
	if got := Build(Request{Files: files}); !got.Success {
		t.Fatalf("standard PredicateSet envelope was rejected: %#v", got.Diagnostics)
	}
}

func TestScriptPredicateEvaluatorUsesPackageRelativeReference(t *testing.T) {
	files := source("pred", "check", "procedures:\n  check:\n    start: v\n    nodes:\n      v:\n        verify: allowed\n")
	for i := range files {
		if files[i].Path == "skills/pred/assurance/contract.yaml" {
			src := string(files[i].Bytes)
			src = strings.Replace(src, "  entryProcedure: check\n", "  entryProcedure: check\n  inputs: {requestSchema: schemas/request.yaml}\n  context:\n    schemas:\n      subject: lib/host/subject.yaml\n      facts: lib/host/facts.yaml\n      observations: lib/observations/context.yaml\n      evidence: lib/host/evidence.yaml\n", 1)
			files[i].Bytes = []byte(src)
		}
	}
	files = append(files,
		File{Path: "skills/pred/assurance/predicates.yaml", Bytes: []byte(predicateSetSource("pred", "predicates:\n  allowed:\n    evaluator: {type: script, ref: scripts/evaluate.py}\n"))},
		File{Path: "skills/pred/scripts/evaluate.py", Bytes: []byte("from typing import TypedDict\nclass Output(TypedDict):\n    result: str\ndef main(input: dict) -> Output:\n    return {}\n")},
		File{Path: "skills/pred/schemas/request.yaml", Bytes: []byte("type: object\n")},
		File{Path: "lib/host/subject.yaml", Bytes: []byte("type: object\n")},
		File{Path: "lib/host/facts.yaml", Bytes: []byte("type: object\n")},
		File{Path: "lib/host/evidence.yaml", Bytes: []byte("type: object\n")},
		File{Path: "lib/observations/context.yaml", Bytes: []byte("type: object\n")},
	)
	got := Build(Request{Files: files})
	if !got.Success {
		t.Fatalf("package-relative Predicate Script should compile: %#v", got.Diagnostics)
	}
	found := false
	for _, file := range got.OutputFiles {
		if file.Path == "skills/pred/schemas/generated/evaluate.output.schema.json" {
			found = true
		}
	}
	if !found {
		t.Fatal("Predicate Script Output Schema was not generated")
	}
	for i := range files {
		if files[i].Path == "skills/pred/assurance/predicates.yaml" {
			files[i].Bytes = []byte(predicateSetSource("pred", "predicates:\n  allowed:\n    evaluator: {type: script, ref: skill.evaluate-allowed}\n"))
		}
	}
	if got := Build(Request{Files: files}); got.Success || !findDiagnostic(got.Diagnostics, "PREDICATE_SOURCE_INCOMPLETE") {
		t.Fatalf("non-path Predicate Script identity was accepted: %#v", got.Diagnostics)
	}
}

func predicateSetSource(skill, body string) string {
	return "apiVersion: connectionagent.io/v1alpha1\nkind: PredicateSet\nmetadata: {skill: " + skill + "}\n" + body
}

func TestContractPredicateGroupsAndEntriesAreValidated(t *testing.T) {
	cases := []struct {
		name  string
		field string
	}{
		{"group must be a sequence", "preconditions: true"},
		{"entry must be a mapping", "postconditions: [ready]"},
		{"predicate ID must be a string", "preconditions: [{predicate: 7}]"},
		{"expectation must be a string", "postconditions: [{predicate: ready, expect: 1}]"},
		{"unknown condition field", "postconditions: [{predicate: ready, dynamic: true}]"},
		{"unknown contract applicability field", "applicability: {predicate: ready}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := source("contract-predicates", "run", "procedures:\n  run:\n    start: done\n    nodes: {done: {terminal: SUCCEEDED}}\n")
			for i := range files {
				if files[i].Path == "skills/contract-predicates/assurance/contract.yaml" {
					files[i].Bytes = []byte(strings.Replace(string(files[i].Bytes), "  entryProcedure: run\n", "  entryProcedure: run\n  "+tc.field+"\n", 1))
				}
			}
			got := Build(Request{Files: files})
			if got.Success || (!findDiagnostic(got.Diagnostics, "PREDICATE_SOURCE_INCOMPLETE") && !findDiagnostic(got.Diagnostics, "CONTRACT_SPEC")) {
				t.Fatalf("malformed contract Predicate declaration was accepted: %#v", got.Diagnostics)
			}
		})
	}
}

func TestUnknownContractPredicateReferenceIsDiagnosed(t *testing.T) {
	files := source("contract-predicate-ref", "run", "procedures:\n  run:\n    start: done\n    nodes: {done: {terminal: SUCCEEDED}}\n")
	for i := range files {
		if files[i].Path == "skills/contract-predicate-ref/assurance/contract.yaml" {
			files[i].Bytes = []byte(strings.Replace(string(files[i].Bytes), "  entryProcedure: run\n", "  entryProcedure: run\n  preconditions: [{predicate: absent}]\n", 1))
		}
	}
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "PREDICATE_NOT_FOUND") {
		t.Fatalf("unknown contract Predicate reference was accepted: %#v", got.Diagnostics)
	}
}

func TestLocalProcedureInvocationCycleFails(t *testing.T) {
	files := source("cycle", "a", "procedures:\n  a:\n    start: call-b\n    nodes:\n      call-b: {procedure: b}\n  b:\n    start: call-a\n    nodes:\n      call-a: {procedure: a}\n")
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "PROCEDURE_CYCLE") {
		t.Fatalf("recursive Procedure graph accepted: %#v", got.Diagnostics)
	}
}

func TestSkillDependencyCycleFails(t *testing.T) {
	files := append(source("root", "start", "procedures:\n  start:\n    start: invoke-child\n    nodes:\n      invoke-child: {procedure: {skill: child, procedure: run}}\n"), source("child", "run", "procedures:\n  run:\n    start: invoke-root\n    nodes:\n      invoke-root: {procedure: {skill: root, procedure: start}}\n")...)
	got := Build(Request{Files: files, Targets: []string{"root"}})
	if got.Success || !findDiagnostic(got.Diagnostics, "DEPENDENCY_CYCLE") {
		t.Fatalf("Skill dependency cycle accepted: %#v", got.Diagnostics)
	}
}

func TestDuplicateOperationFails(t *testing.T) {
	files := testProject()
	two := toolCatalog()
	two.Path = "lib/tools/duplicate.yaml"
	files = append(files, two)
	d := Check(Request{Files: files})
	if !findDiagnostic(d, "LIB_OPERATION_DUPLICATE") {
		t.Fatalf("missing duplicate diagnostic: %#v", d)
	}
	inspected := InspectLibraries(Request{Files: files})
	for _, tool := range inspected.Tools {
		if tool.OperationID == "demo.read" {
			t.Fatal("ambiguous Operation was selected for inspection")
		}
	}
}

func TestNonNormalizedAndDuplicatePathsFail(t *testing.T) {
	input := testProject()
	input = append(input, File{Path: "skills/../escape.txt", Bytes: []byte("bad")}, input[0])
	result := Build(Request{Files: input})
	if !findDiagnostic(result.Diagnostics, "PROJECT_PATH_INVALID") || !findDiagnostic(result.Diagnostics, "PROJECT_DUPLICATE_FILE") || result.Success {
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

func TestSkillMarkdownFrontmatterIsValidated(t *testing.T) {
	cases := []struct {
		name string
		md   string
		code string
	}{
		{"missing frontmatter", "# run\n", "SKILL_METADATA_INVALID"},
		{"missing description", "---\nname: run\n---\n", "SKILL_METADATA_INVALID"},
		{"directory mismatch", "---\nname: other\ndescription: A test Skill.\n---\n", "SKILL_NAME_MISMATCH"},
		{"unknown field", "---\nname: run\ndescription: A test Skill.\nunsafeSemantic: true\n---\n", "SKILL_METADATA_UNSUPPORTED"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := source("run", "p", "procedures:\n  p:\n    start: done\n    nodes: {done: {terminal: SUCCEEDED}}\n")
			files[0].Bytes = []byte(tc.md)
			if got := Check(Request{Files: files}); !findDiagnostic(got, tc.code) {
				t.Fatalf("frontmatter error %s was not diagnosed: %#v", tc.code, got)
			}
		})
	}
}

func TestBuildRequestTargetsAreUniqueAndBounded(t *testing.T) {
	files := source("request", "run", "procedures:\n  run:\n    start: done\n    nodes: {done: {terminal: SUCCEEDED}}\n")
	for _, targets := range [][]string{{"request", "request"}, {strings.Repeat("x", 81)}, {strings.Repeat("界", 81)}} {
		got := Build(Request{Files: files, Targets: targets})
		if got.Success || !findDiagnostic(got.Diagnostics, "REQUEST_INVALID") || len(got.OutputFiles) != 0 {
			t.Fatalf("invalid target vector accepted or yielded partial output: %#v", got)
		}
	}
	if got := Build(Request{}); got.Success || !findDiagnostic(got.Diagnostics, "REQUEST_INVALID") {
		t.Fatalf("empty file vector accepted: %#v", got)
	}
}

func TestDiagnosticStringsStayWithinBuildResultLimits(t *testing.T) {
	a := &analysis{}
	longPath := "skills/" + strings.Repeat("界", 234)
	a.errorAt(longPath, 0, "PROJECT_PATH_INVALID", strings.Repeat("é", 9000))
	got := a.sortedDiagnostics()
	if len(got) != 1 || len([]rune(got[0].Path)) > 240 || len([]rune(got[0].Message)) > 8192 {
		t.Fatalf("diagnostic exceeded BuildResult string limits: pathRunes=%d messageRunes=%d", len([]rune(got[0].Path)), len([]rune(got[0].Message)))
	}
}

func TestBuildResultFileLimitRejectsPartialPackage(t *testing.T) {
	files := loadProjectFilesForTest(t, filepath.Join("..", "..", "examples", "script-contract-project"))
	extra := maxFiles - len(files)
	for i := 0; i < extra; i++ {
		files = append(files, File{Path: filepath.ToSlash(filepath.Join("skills", "normalize", "references", "limit", strconv.Itoa(i)+".txt")), Bytes: []byte("fixture")})
	}
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "OUTPUT_LIMIT") || len(got.OutputFiles) != 0 {
		t.Fatalf("generated file beyond the v0.1 output limit was accepted or returned partially: success=%v outputs=%d diags=%#v", got.Success, len(got.OutputFiles), got.Diagnostics)
	}
}

func TestToolCatalogRejectsUnknownFieldsAndRequiresMetadata(t *testing.T) {
	catalog := toolCatalog()
	catalog.Bytes = []byte(strings.Replace(string(catalog.Bytes), "metadata: {name: test, owner: tests}", "metadata: {name: test, owner: tests, version: latest}", 1))
	got := InspectLibraries(Request{Files: []File{catalog}})
	if !findDiagnostic(got.Diagnostics, "LIB_SCHEMA_INVALID") {
		t.Fatalf("unknown ToolCatalog metadata field was silently accepted: %#v", got)
	}
	catalog.Bytes = []byte(strings.Replace(string(catalog.Bytes), "metadata: {name: test, owner: tests, version: latest}\n", "", 1))
	got = InspectLibraries(Request{Files: []File{catalog}})
	if !findDiagnostic(got.Diagnostics, "LIB_SCHEMA_INVALID") {
		t.Fatalf("ToolCatalog without owner metadata was accepted: %#v", got)
	}
}

func TestToolCatalogEnforcesWireStringLimits(t *testing.T) {
	catalog := toolCatalog()
	catalog.Bytes = []byte(strings.Replace(string(catalog.Bytes), "operation: demo.read", "operation: "+strings.Repeat("a", 161), 1))
	if got := InspectLibraries(Request{Files: []File{catalog}}); !findDiagnostic(got.Diagnostics, "LIB_SCHEMA_INVALID") || len(got.Tools) != 0 {
		t.Fatal("Operation ID beyond the contract limit was accepted")
	}
	catalog = toolCatalog()
	catalog.Bytes = []byte(strings.Replace(string(catalog.Bytes), "operation: demo.read", "operation: demo.read\n    description: "+strings.Repeat("d", 8193), 1))
	if got := InspectLibraries(Request{Files: []File{catalog}}); !findDiagnostic(got.Diagnostics, "LIB_SCHEMA_INVALID") || len(got.Tools) != 0 {
		t.Fatal("Tool description beyond the contract limit was accepted")
	}
}

func TestReconcileRequiresEarlierToolNodeWithDeclaredSchema(t *testing.T) {
	files := source("reconcile", "run", `procedures:
  run:
    start: create
    nodes:
      create:
        call: demo.read
        with: {ref: {value: sample}}
        'on': {SUCCEEDED: recover}
      recover:
        reconcile: create
        'on': {SUCCEEDED: consume}
      consume:
        call: demo.read
        with: {ref: {from: recover.output.state}}
        'on': {SUCCEEDED: done}
      done: {terminal: done}
`)
	files = append(files, toolCatalog())
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "PROCEDURE_UNSUPPORTED") {
		t.Fatalf("reconcile without Tool reconcileOutputSchema should fail: %#v", got.Diagnostics)
	}

	for i := range files {
		if files[i].Path == "lib/tools/demo.yaml" {
			files[i].Bytes = []byte(strings.Replace(string(files[i].Bytes), "outputSchema: {type: object}", "outputSchema: {type: object}\n    reconcileOutputSchema: {type: object, required: [state], properties: {state: {type: string, minLength: 1}}}", 1))
		}
	}
	got = Build(Request{Files: files})
	if !got.Success {
		t.Fatalf("valid earlier reconcile target rejected: %#v", got.Diagnostics)
	}
}

func TestReconcileCannotReferenceLaterNode(t *testing.T) {
	files := source("reconcile", "run", `procedures:
  run:
    start: recover
    nodes:
      recover:
        reconcile: create
      create:
        call: demo.read
`)
	files = append(files, toolCatalog())
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "NODE_TARGET_UNKNOWN") {
		t.Fatalf("reconcile target declared later should fail: %#v", got.Diagnostics)
	}
}

func TestCrossSkillProcedureInputsAreLinkedFromProducerSchemas(t *testing.T) {
	root := source("root", "run", `procedures:
  run:
    start: lookup
    nodes:
      lookup:
        call: demo.read
        with: {ref: {value: sample}}
      child:
        procedure: {skill: child, procedure: write}
        with:
          value: {from: lookup.output.result}
`)
	child := source("child", "write", `procedures:
  write:
    inputs:
      value: {type: string, from: context.request.name}
    start: done
    nodes:
      done: {terminal: SUCCEEDED}
`)
	for i := range child {
		if child[i].Path == "skills/child/assurance/contract.yaml" {
			child[i].Bytes = []byte(strings.Replace(string(child[i].Bytes), "entryProcedure: write\n", "entryProcedure: write\n  inputs:\n    requestSchema: schemas/request.yaml\n", 1))
		}
	}
	catalog := toolCatalog()
	catalog.Bytes = []byte(`apiVersion: ca-agent-demo.tool/v1
kind: ToolCatalog
metadata: {name: test, owner: tests}
spec:
  contracts:
  - operation: demo.read
    effect: READ
    inputSchema: {type: object, required: [ref], properties: {ref: {type: string}}}
    outputSchema: {type: object, required: [result], properties: {result: {type: string}}}
`)
	requestSchema := File{Path: "skills/child/schemas/request.yaml", Bytes: []byte("type: object\nproperties:\n  name: {type: string}\n")}
	files := append(append(append(root, child...), catalog), requestSchema)
	if got := Build(Request{Files: files}); !got.Success {
		t.Fatalf("valid cross-Skill Procedure input rejected: %#v", got.Diagnostics)
	}

	missing := strings.Replace(string(root[2].Bytes), "        with:\n          value: {from: lookup.output.result}\n", "", 1)
	broken := append([]File(nil), files...)
	broken[2].Bytes = []byte(missing)
	if got := Build(Request{Files: broken}); got.Success || !findDiagnostic(got.Diagnostics, "BINDING_UNRESOLVED") {
		t.Fatalf("missing required Procedure input was not diagnosed: %#v", got.Diagnostics)
	}

	extra := append([]File(nil), files...)
	extra[2].Bytes = []byte(strings.Replace(string(extra[2].Bytes), "          value: {from: lookup.output.result}\n", "          value: {from: lookup.output.result}\n          unexpected: {value: extra}\n", 1))
	if got := Build(Request{Files: extra}); got.Success || !findDiagnostic(got.Diagnostics, "SOURCE_FIELD_UNKNOWN") {
		t.Fatalf("unknown Procedure input was not diagnosed: %#v", got.Diagnostics)
	}

	wrong := append([]File(nil), files...)
	wrong[2].Bytes = []byte(strings.Replace(string(wrong[2].Bytes), "lookup.output.result", "lookup.output.missing", 1))
	if got := Build(Request{Files: wrong}); got.Success || !findDiagnostic(got.Diagnostics, "SOURCE_FIELD_UNKNOWN") {
		t.Fatalf("unknown Procedure input producer field was not diagnosed: %#v", got.Diagnostics)
	}

	typeMismatch := append([]File(nil), files...)
	typeMismatch[2].Bytes = []byte(strings.Replace(string(typeMismatch[2].Bytes), "lookup.output.result", "lookup.output", 1))
	if got := Build(Request{Files: typeMismatch}); got.Success || !findDiagnostic(got.Diagnostics, "BINDING_TYPE_MISMATCH") {
		t.Fatalf("Procedure input type mismatch was not diagnosed: %#v", got.Diagnostics)
	}
}

func TestProcedureNodeOutputWithoutDeclaredSchemaIsUnresolved(t *testing.T) {
	files := source("procedure-output", "run", `procedures:
  run:
    start: invoke
    nodes:
      invoke: {procedure: helper}
      consume:
        call: demo.read
        with: {ref: {from: invoke.output.result}}
  helper:
    start: done
    nodes:
      done: {terminal: SUCCEEDED}
`)
	files = append(files, toolCatalog())
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "BINDING_UNRESOLVED") {
		t.Fatalf("Procedure output without an authoritative Schema was accepted: %#v", got.Diagnostics)
	}
	foundExplanation := false
	for _, diagnostic := range got.Diagnostics {
		if diagnostic.Code == "BINDING_UNRESOLVED" && strings.Contains(diagnostic.Message, "Procedure node has no declared output Schema: invoke") {
			foundExplanation = true
		}
	}
	if !foundExplanation {
		t.Fatalf("missing Procedure output Schema was reported as an unknown node: %#v", got.Diagnostics)
	}
}

func TestIntegerProducerSatisfiesNumberInput(t *testing.T) {
	integer := schemaNode(map[string]any{"type": "integer"})
	number := schemaNode(map[string]any{"type": "number"})
	if !schemasCompatible(integer, number) {
		t.Fatal("integer Producer should satisfy a JSON Schema number consumer")
	}
	if schemasCompatible(number, integer) {
		t.Fatal("number Producer must not satisfy an integer consumer")
	}
}

func TestIntegerLiteralSatisfiesNumberInput(t *testing.T) {
	number := schemaNode(map[string]any{"type": "number"})
	integer := schemaNode(map[string]any{"type": "integer"})
	if !literalMatchesSchema(schemaNode(7), number) {
		t.Fatal("integer literal should satisfy a JSON Schema number consumer")
	}
	if literalMatchesSchema(schemaNode(7.5), integer) {
		t.Fatal("fractional literal must not satisfy an integer consumer")
	}
}

func TestNumericLiteralConstraintsAreEnforced(t *testing.T) {
	for _, tc := range []struct {
		name   string
		value  any
		schema map[string]any
		want   bool
	}{
		{"exclusive minimum accepts larger value", 2, map[string]any{"type": "number", "exclusiveMinimum": 1}, true},
		{"exclusive minimum rejects boundary", 1, map[string]any{"type": "number", "exclusiveMinimum": 1}, false},
		{"exclusive maximum accepts smaller value", 1, map[string]any{"type": "number", "exclusiveMaximum": 2}, true},
		{"exclusive maximum rejects boundary", 2, map[string]any{"type": "number", "exclusiveMaximum": 2}, false},
		{"multipleOf accepts multiple", 6, map[string]any{"type": "number", "multipleOf": 2}, true},
		{"multipleOf rejects nonmultiple", 7, map[string]any{"type": "number", "multipleOf": 2}, false},
		{"decimal multiple is exact", 0.3, map[string]any{"type": "number", "multipleOf": 0.1}, true},
		{"near decimal multiple is rejected", 0.30000000001, map[string]any{"type": "number", "multipleOf": 0.1}, false},
		{"large integer below minimum is rejected exactly", int64(9007199254740992), map[string]any{"type": "number", "minimum": int64(9007199254740993)}, false},
		{"large integer at minimum is accepted exactly", int64(9007199254740993), map[string]any{"type": "number", "minimum": int64(9007199254740993)}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := literalMatchesSchema(schemaNode(tc.value), schemaNode(tc.schema)); got != tc.want {
				t.Fatalf("literalMatchesSchema() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestProducerNumericConstraintsMustSatisfyConsumer(t *testing.T) {
	source := func(values map[string]any) *yaml.Node {
		values["type"] = "number"
		return schemaNode(values)
	}
	cases := []struct {
		name string
		from map[string]any
		to   map[string]any
		want bool
	}{
		{"stronger exclusive lower bound", map[string]any{"minimum": 5}, map[string]any{"exclusiveMinimum": 4}, true},
		{"equal minimum does not satisfy exclusive lower bound", map[string]any{"minimum": 5}, map[string]any{"exclusiveMinimum": 5}, false},
		{"strong minimum dominates weaker exclusive minimum", map[string]any{"minimum": 5, "exclusiveMinimum": 3}, map[string]any{"exclusiveMinimum": 4}, true},
		{"strong exclusive minimum satisfies regular minimum", map[string]any{"minimum": 0, "exclusiveMinimum": 10}, map[string]any{"minimum": 5}, true},
		{"strong maximum dominates weaker exclusive maximum", map[string]any{"maximum": 5, "exclusiveMaximum": 7}, map[string]any{"exclusiveMaximum": 6}, true},
		{"strong exclusive maximum satisfies regular maximum", map[string]any{"maximum": 10, "exclusiveMaximum": 4}, map[string]any{"maximum": 5}, true},
		{"producer multiple is also consumer multiple", map[string]any{"multipleOf": 6}, map[string]any{"multipleOf": 2}, true},
		{"producer multiple does not imply consumer multiple", map[string]any{"multipleOf": 2}, map[string]any{"multipleOf": 3}, false},
		{"large integer bound below consumer bound", map[string]any{"minimum": int64(9007199254740992)}, map[string]any{"minimum": int64(9007199254740993)}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := schemasCompatible(source(tc.from), source(tc.to)); got != tc.want {
				t.Fatalf("schemasCompatible() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSchemaCompatibilityIgnoresConstraintsForOtherTypes(t *testing.T) {
	cases := []struct {
		name   string
		typ    string
		source map[string]any
		target map[string]any
	}{
		{"string ignores array item count", "string", map[string]any{}, map[string]any{"minItems": 2}},
		{"string length ignores numeric bounds", "string", map[string]any{"minLength": 2, "maxLength": 5, "exclusiveMinimum": 0, "exclusiveMaximum": 100}, map[string]any{"minLength": 2, "maxLength": 5}},
		{"integer ignores string pattern", "integer", map[string]any{}, map[string]any{"pattern": "^[0-9]+$"}},
		{"array ignores numeric bounds", "array", map[string]any{"items": map[string]any{"type": "string"}}, map[string]any{"items": map[string]any{"type": "string"}, "minimum": 2}},
		{"string ignores array uniqueness", "string", map[string]any{}, map[string]any{"uniqueItems": true}},
		{"number ignores string length", "number", map[string]any{}, map[string]any{"minLength": 5}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := map[string]any{"type": tc.typ}
			target := map[string]any{"type": tc.typ}
			for key, value := range tc.source {
				source[key] = value
			}
			for key, value := range tc.target {
				target[key] = value
			}
			if !schemasCompatible(schemaNode(source), schemaNode(target)) {
				t.Fatal("inapplicable JSON Schema keywords changed compatibility")
			}
		})
	}
}

func TestJSONNumericValuesCompareMathematically(t *testing.T) {
	parse := func(source string) *yaml.Node {
		t.Helper()
		var document yaml.Node
		if err := yaml.Unmarshal([]byte(source), &document); err != nil {
			t.Fatal(err)
		}
		return document.Content[0]
	}
	producer := parse("type: number\nenum: [1]\n")
	consumer := parse("type: number\nenum: [1.0]\n")
	if !schemasCompatible(producer, consumer) {
		t.Fatal("JSON Schema numeric enum values 1 and 1.0 should be equal")
	}
	if !literalMatchesSchema(parse("1.0\n"), parse("type: number\nconst: 1\n")) {
		t.Fatal("JSON number literal 1.0 did not satisfy const 1")
	}
	if literalMatchesSchema(parse("[1, 1.0]\n"), parse("type: array\nitems: {type: number}\nuniqueItems: true\n")) {
		t.Fatal("uniqueItems accepted mathematically equal numeric values")
	}
	if err := validateSchemaShape(parse("type: number\nenum: [1, 1.0]\n")); err == nil {
		t.Fatal("duplicate mathematical enum values were accepted")
	}
	if !jsonValuesEqual(parse("{nested: [1, {value: 2.00}]}\n"), parse("{nested: [1.0, {value: 2}]}\n")) {
		t.Fatal("nested JSON values did not compare numeric members mathematically")
	}
}

func TestObjectSchemaCompatibilityHonorsAdditionalProperties(t *testing.T) {
	object := func(properties map[string]any, additional any) *yaml.Node {
		value := map[string]any{"type": "object", "properties": properties}
		if additional != nil {
			value["additionalProperties"] = additional
		}
		return schemaNode(value)
	}
	stringSchema := map[string]any{"type": "string"}
	numberSchema := map[string]any{"type": "number"}
	cases := []struct {
		name   string
		source *yaml.Node
		target *yaml.Node
		want   bool
	}{
		{"unrestricted producer cannot meet typed additional properties", object(map[string]any{}, true), object(map[string]any{}, stringSchema), false},
		{"matching additional property schemas", object(map[string]any{}, stringSchema), object(map[string]any{}, stringSchema), true},
		{"closed producer meets typed additional properties", object(map[string]any{}, false), object(map[string]any{}, stringSchema), true},
		{"declared extra field checked against target additional schema", object(map[string]any{"extra": stringSchema}, false), object(map[string]any{}, stringSchema), true},
		{"declared incompatible extra field rejected", object(map[string]any{"extra": numberSchema}, false), object(map[string]any{}, stringSchema), false},
		{"unrestricted producer cannot meet closed consumer", object(map[string]any{}, nil), object(map[string]any{}, false), false},
		{"closed producer meets closed consumer", object(map[string]any{"key": stringSchema}, false), object(map[string]any{"key": stringSchema}, false), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := schemasCompatible(tc.source, tc.target); got != tc.want {
				t.Fatalf("schemasCompatible() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestObjectLiteralBindingsHonorAdditionalProperties(t *testing.T) {
	value := func(extra any) *yaml.Node { return schemaNode(map[string]any{"key": "value", "extra": extra}) }
	base := map[string]any{"type": "object", "properties": map[string]any{"key": map[string]any{"type": "string"}}, "required": []string{"key"}}
	cases := []struct {
		name  string
		value *yaml.Node
		extra any
		want  bool
	}{
		{"default additional properties accept JSON values", value(1), nil, true},
		{"false rejects unlisted fields", value(1), false, false},
		{"schema checks unlisted fields", value("ok"), map[string]any{"type": "string"}, true},
		{"schema rejects incompatible unlisted fields", value(1), map[string]any{"type": "string"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			schema := make(map[string]any, len(base)+1)
			for key, item := range base {
				schema[key] = item
			}
			if tc.extra != nil {
				schema["additionalProperties"] = tc.extra
			}
			if got := literalMatchesSchema(tc.value, schemaNode(schema)); got != tc.want {
				t.Fatalf("literalMatchesSchema() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestScriptInputProjectionRejectsUnsupportedNestedSchemas(t *testing.T) {
	unsupported := schemaNode(map[string]any{
		"type":       "object",
		"properties": map[string]any{"nested": map[string]any{"anyOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "integer"}}}},
	})
	if projectionSchemaSupported(unsupported) {
		t.Fatal("unsupported nested union was accepted in a Script input projection")
	}
	incompleteArray := schemaNode(map[string]any{"type": "array"})
	if projectionSchemaSupported(incompleteArray) {
		t.Fatal("array with no item Schema was accepted in a Script input projection")
	}
}

func TestUnusedProcedureInputsAreLinkedAgainstTheirProducer(t *testing.T) {
	root := source("root", "run", `procedures:
  run:
    start: call
    nodes:
      call:
        procedure: {skill: child, procedure: write}
`)
	child := source("child", "write", `procedures:
  write:
    inputs:
      unused: {type: string, from: context.request.missing}
    start: done
    nodes:
      done: {terminal: SUCCEEDED}
`)
	for i := range child {
		if child[i].Path == "skills/child/assurance/contract.yaml" {
			child[i].Bytes = []byte(strings.Replace(string(child[i].Bytes), "entryProcedure: write\n", "entryProcedure: write\n  inputs:\n    requestSchema: schemas/request.yaml\n", 1))
		}
	}
	requestSchema := File{Path: "skills/child/schemas/request.yaml", Bytes: []byte(`type: object
properties:
  name: {type: string}
`)}
	files := append(append(root, child...), requestSchema)
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "SOURCE_FIELD_UNKNOWN") {
		t.Fatalf("unused Procedure input with an unknown producer field was accepted: %#v", got.Diagnostics)
	}

	wrongType := append([]File(nil), files...)
	for i := range wrongType {
		if wrongType[i].Path == "skills/child/assurance/procedure.yaml" {
			wrongType[i].Bytes = []byte(strings.Replace(string(wrongType[i].Bytes), "type: string, from: context.request.missing", "type: integer, from: context.request.name", 1))
		}
	}
	got = Build(Request{Files: wrongType})
	if got.Success || !findDiagnostic(got.Diagnostics, "BINDING_TYPE_MISMATCH") {
		t.Fatalf("unused Procedure input with a mismatched declared type was accepted: %#v", got.Diagnostics)
	}
}

func TestProcedureInputSourceCyclesFailWithoutRecursing(t *testing.T) {
	files := source("cycle", "run", `procedures:
  run:
    inputs:
      first: {type: string, from: input.second}
      second: {type: string, from: input.first}
    start: read
    nodes:
      read:
        call: demo.read
        with: {ref: {from: input.first}}
`)
	files = append(files, toolCatalog())
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "BINDING_UNRESOLVED") {
		t.Fatalf("cyclic Procedure input source should fail with a link diagnostic: %#v", got.Diagnostics)
	}
}

func TestExplicitSelfSkillProcedureReferenceIsLinked(t *testing.T) {
	files := source("self", "run", `procedures:
  run:
    start: child
    nodes:
      child:
        procedure: {skill: self, procedure: helper}
  helper:
    start: done
    nodes:
      done: {terminal: SUCCEEDED}
`)
	if got := Build(Request{Files: files}); !got.Success {
		t.Fatalf("valid explicit self-Skill Procedure reference rejected: %#v", got.Diagnostics)
	}

	for i := range files {
		if files[i].Path == "skills/self/assurance/procedure.yaml" {
			files[i].Bytes = []byte(strings.Replace(string(files[i].Bytes), "procedure: helper", "procedure: missing", 1))
		}
	}
	if got := Build(Request{Files: files}); got.Success || !findDiagnostic(got.Diagnostics, "PROCEDURE_NOT_FOUND") {
		t.Fatalf("unknown explicit self-Skill Procedure was not diagnosed: %#v", got.Diagnostics)
	}
}

func TestMalformedActionShapeFailsClosed(t *testing.T) {
	files := source("shape", "run", `procedures:
  run:
    start: call
    nodes:
      call:
        call: {operation: demo.read}
`)
	files = append(files, toolCatalog())
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "PROCEDURE_UNSUPPORTED") {
		t.Fatalf("mapping-valued call action should be rejected: %#v", got.Diagnostics)
	}
}

func TestMalformedProcedureContainerShapesAreDiagnosed(t *testing.T) {
	cases := []struct {
		name      string
		procedure string
		code      string
	}{
		{"procedures must be a mapping", "procedures: []\n", "PROCEDURES_MISSING"},
		{"procedure definition must be a mapping", "procedures:\n  run: invalid\n", "PROCEDURE_UNSUPPORTED"},
		{"nodes must be a mapping", "procedures:\n  run:\n    start: done\n    nodes: []\n", "PROCEDURE_UNSUPPORTED"},
		{"node names must not be empty", "procedures:\n  run:\n    start: \"\"\n    nodes:\n      \"\": {terminal: SUCCEEDED}\n", "PROCEDURE_UNSUPPORTED"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := source("shape", "run", tc.procedure)
			got := Build(Request{Files: files})
			if got.Success || !findDiagnostic(got.Diagnostics, tc.code) {
				t.Fatalf("malformed Procedure shape was accepted or misdiagnosed: %#v", got.Diagnostics)
			}
		})
	}
}

func TestNestedVerifyStepReportsMissingRequestSchemaForBindings(t *testing.T) {
	files := source("nested-verify", "run", `procedures:
  run:
    start: done
    nodes:
      done:
        preconditions:
          steps:
            - name: verify-ready
              verify: ready
              with: {value: {from: context.request.value}}
        terminal: SUCCEEDED
`)
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "EXTERNAL_SCHEMA_MISSING") {
		t.Fatalf("verify binding without a request Schema was not reported: %#v", got.Diagnostics)
	}
}

func TestNestedStepRejectsEmptyExtraActionField(t *testing.T) {
	files := source("nested-action", "run", `procedures:
  run:
    start: done
    nodes:
      done:
        preconditions:
          steps:
            - name: read
              call: demo.read
              script: ""
        terminal: SUCCEEDED
`)
	files = append(files, toolCatalog())
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "PROCEDURE_UNSUPPORTED") {
		t.Fatalf("nested step with an empty extra action field was accepted: %#v", got.Diagnostics)
	}
}

func TestProcedureIdentifiersAndActionsRequireStrings(t *testing.T) {
	cases := []struct {
		name      string
		procedure string
	}{
		{"procedure key", "procedures:\n  7:\n    start: done\n    nodes: {done: {terminal: SUCCEEDED}}\n"},
		{"input key", "procedures:\n  run:\n    inputs:\n      7: {type: string, from: context.request.name}\n    start: done\n    nodes: {done: {terminal: SUCCEEDED}}\n"},
		{"procedure input source must be a string", "procedures:\n  run:\n    inputs:\n      value: {type: string, from: true}\n    start: done\n    nodes: {done: {terminal: SUCCEEDED}}\n"},
		{"node key", "procedures:\n  run:\n    start: 7\n    nodes:\n      7: {terminal: SUCCEEDED}\n"},
		{"non-string node action", "procedures:\n  run:\n    start: done\n    nodes: {done: {terminal: true}}\n"},
		{"non-string nested step name", "procedures:\n  run:\n    start: done\n    nodes:\n      done:\n        preconditions:\n          steps: [{name: 7, script: scripts/run.py}]\n        terminal: SUCCEEDED\n"},
		{"incomplete explicit Procedure reference", "procedures:\n  run:\n    start: invoke\n    nodes:\n      invoke: {procedure: {procedure: helper}}\n  helper:\n    start: done\n    nodes: {done: {terminal: SUCCEEDED}}\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := source("string-identifiers", "run", tc.procedure)
			got := Build(Request{Files: files})
			if got.Success || !findDiagnostic(got.Diagnostics, "PROCEDURE_UNSUPPORTED") {
				t.Fatalf("non-string or incomplete Procedure syntax was accepted: %#v", got.Diagnostics)
			}
		})
	}
}

func TestBindingFieldNamesMustBeStrings(t *testing.T) {
	files := source("numeric-binding-key", "run", `procedures:
  run:
    start: read
    nodes:
      read:
        call: demo.read
        with:
          7: {value: sample}
`)
	files = append(files, toolCatalog())
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "BINDING_UNSUPPORTED") {
		t.Fatalf("non-string binding field name was accepted: %#v", got.Diagnostics)
	}
}

func TestBindingSourcesMustBeStrings(t *testing.T) {
	for _, value := range []string{"true", "7", "[]", "{}"} {
		t.Run(value, func(t *testing.T) {
			files := source("nonstring-binding-source", "run", `procedures:
  run:
    start: read
    nodes:
      read:
        call: demo.read
        with:
          ref: {from: `+value+`}
`)
			files = append(files, toolCatalog())
			got := Build(Request{Files: files})
			if got.Success || !findDiagnostic(got.Diagnostics, "BINDING_UNSUPPORTED") {
				t.Fatalf("non-string binding source %s was accepted: %#v", value, got.Diagnostics)
			}
		})
	}
}

func TestUnsupportedBindingSchemasAreNotReportedAsTypeMismatches(t *testing.T) {
	cases := []struct {
		name, path, before, after string
	}{
		{
			name:   "consumer format constraint on producer binding",
			path:   "lib/tools/demo.tool.yaml",
			before: "          key: *text",
			after:  "          key: {type: string, format: email}",
		},
		{
			name:   "producer format constraint",
			path:   "skills/normalize/schemas/normalize-request.schema.yaml",
			before: "  dnn: {type: string, minLength: 1}",
			after:  "  dnn: {type: string, format: email}",
		},
		{
			name:   "consumer format constraint on literal binding",
			path:   "lib/tools/demo.tool.yaml",
			before: "          key: *text",
			after:  "          key: {type: string, format: email}",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := loadProjectFilesForTest(t, filepath.Join("..", "..", "examples", "script-contract-project"))
			if tc.name == "consumer format constraint on literal binding" {
				for i := range files {
					if files[i].Path == "skills/normalize/assurance/procedure.yaml" {
						files[i].Bytes = []byte(strings.Replace(string(files[i].Bytes), "key: {from: context.request.dnn}", "key: {value: sample}", 1))
					}
				}
			}
			changed := false
			for i := range files {
				if files[i].Path == tc.path {
					text := strings.Replace(string(files[i].Bytes), tc.before, tc.after, 1)
					changed = text != string(files[i].Bytes)
					files[i].Bytes = []byte(text)
				}
			}
			if !changed {
				t.Fatal("test mutation did not apply")
			}
			got := Build(Request{Files: files})
			if got.Success || !findDiagnostic(got.Diagnostics, "BINDING_UNSUPPORTED") {
				t.Fatalf("unsupported binding Schema was not reported explicitly: %#v", got.Diagnostics)
			}
			if findDiagnostic(got.Diagnostics, "BINDING_TYPE_MISMATCH") {
				t.Fatalf("unsupported Schema was mislabeled as a type mismatch: %#v", got.Diagnostics)
			}
		})
	}
}

func TestPreAndPostconditionStepsHaveDistinctOutputScopes(t *testing.T) {
	files := source("scopes", "run", `procedures:
  run:
    start: producer
    nodes:
      producer:
        call: demo.read
        with: {ref: {value: sample}}
        preconditions:
          steps:
            - name: same
              call: demo.read
              with: {ref: {value: sample}}
        postconditions:
          steps:
            - name: same
              call: demo.integer
              with: {ref: {value: sample}}
      remote-consumer:
        call: demo.read
        with: {ref: {from: producer.preconditions.same.output.result}}
      local-consumer:
        call: demo.read
        with: {ref: {from: preconditions.same.output.result}}
        preconditions:
          steps:
            - name: same
              call: demo.read
              with: {ref: {value: sample}}
        postconditions:
          steps:
            - name: same
              call: demo.integer
              with: {ref: {value: sample}}
`)
	files[1].Bytes = []byte(strings.Replace(string(files[1].Bytes), "capabilities: [demo.read]", "capabilities: [demo.read, demo.integer]", 1))
	files = append(files, File{Path: "lib/tools/scopes.yaml", Bytes: []byte(`apiVersion: ca-agent-demo.tool/v1
kind: ToolCatalog
metadata: {name: scopes, owner: tests}
spec:
  contracts:
  - operation: demo.read
    effect: READ
    inputSchema: {type: object, required: [ref], properties: {ref: {type: string}}}
    outputSchema: {type: object, properties: {result: {type: string}}}
  - operation: demo.integer
    effect: READ
    inputSchema: {type: object, required: [ref], properties: {ref: {type: string}}}
    outputSchema: {type: object, properties: {result: {type: integer}}}
`)})
	if got := Build(Request{Files: files}); !got.Success {
		t.Fatalf("pre/postcondition outputs leaked across scopes: %#v", got.Diagnostics)
	}
}

func TestScriptBindingsRejectNonJSONValuesAndUnknownBindingFields(t *testing.T) {
	files := []File{
		{Path: "skills/s/assurance/contract.yaml", Bytes: []byte("apiVersion: connectionagent.io/v1alpha1\nkind: SkillContract\nmetadata: {name: s}\nspec: {entryProcedure: p}\n")},
		{Path: "skills/s/SKILL.md", Bytes: []byte("---\nname: s\ndescription: Test fixture.\n---\n")},
		{Path: "skills/s/assurance/procedure.yaml", Bytes: []byte(`procedures:
  p:
    start: n
    nodes:
      n:
        script: scripts/out.py
        with:
          date: {value: 2026-11-11}
`)},
		{Path: "skills/s/scripts/out.py", Bytes: []byte("from typing import TypedDict\nclass Output(TypedDict):\n    value: str\ndef main(input: dict) -> Output:\n    return {}\n")},
	}
	if got := Build(Request{Files: files}); got.Success || !findDiagnostic(got.Diagnostics, "BINDING_UNSUPPORTED") {
		t.Fatalf("YAML timestamp accepted as JSON binding literal: %#v", got.Diagnostics)
	}
	for i := range files {
		if files[i].Path == "skills/s/assurance/procedure.yaml" {
			files[i].Bytes = []byte(strings.Replace(string(files[i].Bytes), "date: {value: 2026-11-11}", "date: {value: sample, mode: dynamic}", 1))
		}
	}
	if got := Build(Request{Files: files}); got.Success || !findDiagnostic(got.Diagnostics, "BINDING_UNSUPPORTED") {
		t.Fatalf("unknown binding descriptor field was ignored: %#v", got.Diagnostics)
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

func TestFileInputOrderDoesNotChangeDiagnosticsOrOutputs(t *testing.T) {
	files := testProject()
	reversed := append([]File(nil), files...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	one, two := Build(Request{Files: files}), Build(Request{Files: reversed})
	if !reflect.DeepEqual(one, two) {
		t.Fatalf("file ordering changed compiler result:\n%#v\n%#v", one, two)
	}
	bad := append([]File(nil), files...)
	for i := range bad {
		if bad[i].Path == "skills/root/assurance/procedure.yaml" {
			bad[i].Bytes = []byte("procedures: [invalid")
		}
	}
	badReverse := append([]File(nil), bad...)
	for left, right := 0, len(badReverse)-1; left < right; left, right = left+1, right-1 {
		badReverse[left], badReverse[right] = badReverse[right], badReverse[left]
	}
	if !reflect.DeepEqual(Check(Request{Files: bad}), Check(Request{Files: badReverse})) {
		t.Fatal("file ordering changed diagnostics")
	}
}

func TestReadonlyLibraryInspectionHasNoSourceLifecycleState(t *testing.T) {
	got := InspectLibraries(Request{Files: []File{toolCatalog()}})
	if len(got.Diagnostics) != 0 || len(got.Tools) != 1 || got.Tools[0].DeclaredEffect != "READ" {
		t.Fatalf("unexpected library inspection: %#v", got)
	}
	if got.Tools[0].InputSchema == nil || got.Tools[0].OutputSchema == nil {
		t.Fatalf("schema projections missing: %#v", got.Tools[0])
	}
	wire, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(wire), "sourceStatus") || strings.Contains(string(wire), "proposed") {
		t.Fatalf("Compiler inspection leaked source lifecycle metadata: %s", wire)
	}
}

func TestV01RejectsLegacyLocalToolGrantField(t *testing.T) {
	files := source("no-local-tools", "run", "procedures:\n  run:\n    start: done\n    nodes: {done: {terminal: SUCCEEDED}}\n")
	for i := range files {
		if files[i].Path == "skills/no-local-tools/assurance/contract.yaml" {
			files[i].Bytes = []byte(strings.Replace(string(files[i].Bytes), "capabilities: [demo.read]", "capabilities: [demo.read]\n    localTools: [legacy.script]", 1))
		}
	}
	if got := Build(Request{Files: files}); got.Success || !findDiagnostic(got.Diagnostics, "CONTRACT_SPEC") {
		t.Fatalf("v0 localTools compatibility field was accepted: %#v", got.Diagnostics)
	}
}

func TestGeneratedStrictScriptOutput(t *testing.T) {
	files := []File{
		{Path: "skills/s/assurance/contract.yaml", Bytes: []byte("apiVersion: connectionagent.io/v1alpha1\nkind: SkillContract\nmetadata: {name: s}\nspec: {entryProcedure: p}\n")},
		{Path: "skills/s/SKILL.md", Bytes: []byte("---\nname: s\ndescription: Test fixture.\n---\n")},
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

func TestGeneratedOutputPathRejectsFileDirectoryCollisions(t *testing.T) {
	files := []File{
		{Path: "skills/s/assurance/contract.yaml", Bytes: []byte("apiVersion: connectionagent.io/v1alpha1\nkind: SkillContract\nmetadata: {name: s}\nspec: {entryProcedure: p}\n")},
		{Path: "skills/s/SKILL.md", Bytes: []byte("---\nname: s\ndescription: Test fixture.\n---\n")},
		{Path: "skills/s/assurance/procedure.yaml", Bytes: []byte("procedures:\n  p:\n    start: n\n    nodes:\n      n:\n        script: scripts/out.py\n        with: {}\n")},
		{Path: "skills/s/scripts/out.py", Bytes: []byte("from typing import TypedDict\nclass Output(TypedDict):\n    value: str\ndef main(input: dict) -> Output:\n    return {}\n")},
		{Path: "skills/s/schemas/generated/out.output.schema.json/child.txt", Bytes: []byte("collision")},
	}
	got := Build(Request{Files: files})
	if got.Success || !findDiagnostic(got.Diagnostics, "PROJECT_DUPLICATE_FILE") {
		t.Fatalf("generated file/directory collision was not rejected: %#v", got.Diagnostics)
	}
}

func TestBroadScriptOutputIsRejected(t *testing.T) {
	if _, err := parseScriptOutput("def main(input: dict) -> dict:\n    return {}\n"); err == nil {
		t.Fatal("broad dict output accepted")
	}
}

func TestAnnotatedOutputConstraintsAreStrict(t *testing.T) {
	for _, constraint := range []string{`{"minLength": -1}`, `{"mystery": 1}`, `{"pattern": "["}`, `{"minLength": 1, "minLength": 2}`, `{minLength: 1}`, `{"minLength: 1}`} {
		src := "from typing import TypedDict, Annotated\nclass Output(TypedDict):\n    value: Annotated[str, " + constraint + "]\ndef main(input: dict) -> Output:\n    return {}\n"
		if _, err := parseScriptOutput(src); err == nil {
			t.Errorf("invalid constraint %s was accepted", constraint)
		}
	}
}

func TestScriptLiteralsKeepQuotedPunctuation(t *testing.T) {
	src := `from typing import Annotated, Literal, TypedDict
class Output(TypedDict):
    value: Annotated[str, {"minLength": 1, "pattern": r"^(foo,bar)[0-9]+$"}]
    marker: Literal["a,b"]
    missing: Literal[None]
    repeated: Literal["same", "same"]
    precise: Literal[9007199254740993]
    threshold: Annotated[int, {"minimum": 9007199254740993, "maximum": 9007199254740994}]
def main(input: dict) -> Output:
    return {}`
	schema, err := parseScriptOutput(src)
	if err != nil {
		t.Fatalf("documented raw pattern string rejected: %v", err)
	}
	properties := schema["properties"].(map[string]any)
	value := properties["value"].(map[string]any)
	if got := value["pattern"]; got != "^(foo,bar)[0-9]+$" {
		t.Fatalf("raw pattern changed during parsing: %#v", got)
	}
	marker := properties["marker"].(map[string]any)
	if marker["const"] != "a,b" {
		t.Fatalf("quoted Literal changed during parsing: %#v", marker)
	}
	missing := properties["missing"].(map[string]any)
	if missing["type"] != "null" || missing["const"] != nil {
		t.Fatalf("Literal[None] was not represented as JSON null: %#v", missing)
	}
	repeated := properties["repeated"].(map[string]any)
	if repeated["const"] != "same" {
		t.Fatalf("duplicate Literal values were not normalized: %#v", repeated)
	}
	precise := properties["precise"].(map[string]any)
	if precise["const"] != json.Number("9007199254740993") {
		t.Fatalf("large integer Literal lost precision: %#v", precise)
	}
	threshold := properties["threshold"].(map[string]any)
	if threshold["minimum"] != json.Number("9007199254740993") || threshold["maximum"] != json.Number("9007199254740994") {
		t.Fatalf("large Annotated numeric constraints lost precision: %#v", threshold)
	}
	encoded, err := json.Marshal(schema)
	if err != nil || !strings.Contains(string(encoded), "9007199254740993") {
		t.Fatalf("serialized Script Schema lost exact integer values: %s, %v", encoded, err)
	}
}

func TestTypedDictBodyRejectsNonFieldStatements(t *testing.T) {
	src := "from typing import TypedDict\nclass Output(TypedDict):\n    value: str\n    computed = dynamic_value()\ndef main(input: dict) -> Output:\n    return {}\n"
	if _, err := parseScriptOutput(src); err == nil {
		t.Fatal("non-field TypedDict body statement was silently ignored")
	}
}

func TestTypedDictMustHaveTypingImport(t *testing.T) {
	src := "class Output(TypedDict):\n    value: str\ndef main(input: dict) -> Output:\n    return {}\n"
	if _, err := parseScriptOutput(src); err == nil {
		t.Fatal("undefined TypedDict annotation accepted")
	}
}

func TestScriptDeclarationsInsideMultilineStringsAreIgnored(t *testing.T) {
	src := `"""
from typing import TypedDict
class Output(TypedDict):
    value: str
def main(input: dict) -> Output:
    return {}
"""
`
	if _, err := parseScriptOutput(src); err == nil {
		t.Fatal("declarations inside a multiline string were treated as Python code")
	}
}

func TestScriptDeclarationsRejectAmbiguityAndInvalidLiterals(t *testing.T) {
	cases := []string{
		"from typing import TypedDict\nclass Output(TypedDict):\n    value: str\ndef main(input: dict) -> Output:\n    return {}\ndef main(input: dict) -> Output:\n    return {}\n",
		"from typing import TypedDict\nclass Output(TypedDict):\n    value: str\n    value: int\ndef main(input: dict) -> Output:\n    return {}\n",
		"from typing import TypedDict, Literal\nclass Output(TypedDict):\n    value: Literal[None, 'text']\ndef main(input: dict) -> Output:\n    return {}\n",
	}
	for _, src := range cases {
		if _, err := parseScriptOutput(src); err == nil {
			t.Fatal("ambiguous or invalid Script declaration accepted")
		}
	}
}

func TestNodeOutputMustDominateConsumer(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("start: first\nnodes:\n  first: {on: {PASS: second, FAIL: failed}}\n  second: {on: {PASS: done}}\n  failed: {terminal: failed}\n  done: {terminal: done}\n"), &doc); err != nil {
		t.Fatal(err)
	}
	proc := doc.Content[0]
	if !nodeOutputDominates(proc, "first", "second") {
		t.Fatal("first output should be available on every path to second")
	}
	if nodeOutputDominates(proc, "second", "first") {
		t.Fatal("future node output must not be available to first")
	}
	if nodeOutputDominates(proc, "second", "failed") {
		t.Fatal("branch-exclusive output must not be available to failed")
	}
}

func TestPhaseStepOutputsRespectExecutionOrder(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("start: first\nnodes:\n  first:\n    call: demo.read\n    preconditions:\n      steps:\n        - {name: prep1, call: demo.read}\n        - {name: prep2, call: demo.read}\n    postconditions:\n      steps:\n        - {name: post1, call: demo.read}\n        - {name: post2, call: demo.read}\n    'on': {PASS: second}\n  second: {terminal: done}\n"), &doc); err != nil {
		t.Fatal(err)
	}
	proc := doc.Content[0]
	for _, tc := range []struct {
		ref, phase, consumer string
		index                int
		want                 bool
	}{
		{"preconditions.prep1.output.value", "preconditions", "first", 1, true},
		{"preconditions.prep2.output.value", "preconditions", "first", 1, false},
		{"preconditions.prep1.output.value", "main", "first", -1, true},
		{"postconditions.post1.output.value", "main", "first", -1, false},
		{"postconditions.post1.output.value", "postconditions", "first", 1, true},
		{"postconditions.post2.output.value", "postconditions", "first", 0, false},
		{"first.postconditions.post2.output.value", "main", "second", -1, true},
	} {
		got, _ := bindingSourceAvailable(proc, tc.ref, tc.consumer, tc.phase, tc.index)
		if got != tc.want {
			t.Errorf("bindingSourceAvailable(%q, %q, %q, %d) = %v, want %v", tc.ref, tc.consumer, tc.phase, tc.index, got, tc.want)
		}
	}
}

func TestNestedProcedureBindingsAreCheckedAgainstSchemas(t *testing.T) {
	target := schemaNode(map[string]any{"type": "object", "required": []any{"count"}, "additionalProperties": false, "properties": map[string]any{"count": map[string]any{"type": "integer", "minimum": 1}}})
	valid := schemaNode(map[string]any{"type": "object", "required": []any{"count"}, "additionalProperties": false, "properties": map[string]any{"count": map[string]any{"type": "integer", "minimum": 1}}})
	wrong := schemaNode(map[string]any{"type": "object", "required": []any{"count"}, "additionalProperties": false, "properties": map[string]any{"count": map[string]any{"type": "string"}}})
	if !schemasCompatible(valid, target) || schemasCompatible(wrong, target) {
		t.Fatal("nested producer schema compatibility was not checked")
	}
	if !literalMatchesSchema(schemaNode(map[string]any{"count": 2}), target) || literalMatchesSchema(schemaNode(map[string]any{"count": "two"}), target) || literalMatchesSchema(schemaNode(map[string]any{}), target) {
		t.Fatal("nested literal validation was not strict")
	}
}

func TestCurrentProcedureVerifyStepsMayBeUnnamedAndAcceptBindings(t *testing.T) {
	files := loadProjectFilesForTest(t, filepath.Join("..", "..", "examples", "script-contract-project"))
	files = append(files, File{Path: "skills/normalize/assurance/predicates.yaml", Bytes: []byte(predicateSetSource("normalize", "predicates:\n  demo.result:\n    evaluator: {type: cel, expression: 'true'}\n"))})
	for i := range files {
		if files[i].Path == "skills/normalize/assurance/procedure.yaml" {
			src := string(files[i].Bytes)
			src = strings.Replace(src, "        'on':\n          SUCCEEDED: normalize-step", "        postconditions:\n          steps:\n            - verify: demo.result\n              with:\n                result: {from: read.output.value}\n          'on':\n            PASS: normalize-step\n            FAIL: failed\n            UNKNOWN: failed\n        'on':\n          SUCCEEDED: normalize-step", 1)
			files[i].Bytes = []byte(src)
		}
	}
	if got := Check(Request{Files: files}); !findDiagnostic(got, "EXTERNAL_SCHEMA_MISSING") || findDiagnostic(got, "PROCEDURE_UNSUPPORTED") {
		t.Fatalf("current unnamed verify step should parse, while its missing consumer Schema stays explicit: %#v", got)
	}
	for i := range files {
		if files[i].Path == "skills/normalize/assurance/procedure.yaml" {
			files[i].Bytes = []byte(strings.Replace(string(files[i].Bytes), "read.output.value", "read.output.unknown", 1))
		}
	}
	if got := Check(Request{Files: files}); !findDiagnostic(got, "SOURCE_FIELD_UNKNOWN") {
		t.Fatalf("verify binding must resolve against its producer schema: %#v", got)
	}
}

func TestPinnedOfficialToolCatalogInspection(t *testing.T) {
	files := loadProjectFilesForTest(t, filepath.Join("..", "..", "examples", "ca-skills-v0.2"))
	got := InspectLibraries(Request{Files: files})
	if len(got.Diagnostics) != 0 {
		t.Fatalf("catalog inspection diagnostics: %#v", got.Diagnostics)
	}
	if len(got.Tools) != 12 {
		t.Fatalf("wanted 12 approved imported Operations, got %d", len(got.Tools))
	}
	for _, tool := range got.Tools {
		if tool.InputSchema == nil {
			t.Errorf("missing input schema for %s", tool.OperationID)
		}
	}
}

func TestCurrentOfficialCaSkillsSnapshotParsesAndKeepsOwnerGaps(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "ca-skills-v0.2")
	checksums, err := os.ReadFile(filepath.Join(root, "UPSTREAM_SOURCE.sha256"))
	if err != nil {
		t.Fatal(err)
	}
	verified := 0
	for _, line := range strings.Split(strings.TrimSpace(string(checksums)), "\n") {
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 {
			t.Fatalf("invalid upstream checksum entry: %q", line)
		}
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(parts[1])))
		if err != nil {
			t.Fatalf("read pinned source %s: %v", parts[1], err)
		}
		digest := sha256.Sum256(content)
		if hex.EncodeToString(digest[:]) != parts[0] {
			t.Fatalf("pinned upstream source changed: %s", parts[1])
		}
		verified++
	}
	if verified != 62 {
		t.Fatalf("expected checksums for 61 official Skill files and one ToolCatalog, got %d", verified)
	}
	files := loadProjectFilesForTest(t, root)
	pdu := Build(Request{Files: files, Targets: []string{"establish-pdu-session"}})
	if pdu.Success || !findDiagnostic(pdu.Diagnostics, "EXTERNAL_SCHEMA_MISSING") {
		t.Fatalf("official PDU source must retain explicit Owner/Tool contract gaps: %#v", pdu.Diagnostics)
	}
	for _, unsupported := range []string{"SKILL_METADATA_INVALID", "CONTRACT_SPEC", "CASE_INVALID", "PREDICATE_SOURCE_INCOMPLETE", "PROCEDURE_UNSUPPORTED", "SCRIPT_TYPE_UNSUPPORTED", "LOCAL_TOOL_RESOLUTION_UNSPECIFIED"} {
		if findDiagnostic(pdu.Diagnostics, unsupported) {
			t.Errorf("official PDU source form was not parsed according to Skill Language v0.2 (%s): %#v", unsupported, pdu.Diagnostics)
		}
	}
	mas := Build(Request{Files: files, Targets: []string{"establish-pdu-session-mas"}})
	if mas.Success || !findDiagnostic(mas.Diagnostics, "LIB_OPERATION_UNKNOWN") || !findDiagnostic(mas.Diagnostics, "EXTERNAL_SCHEMA_MISSING") {
		t.Fatalf("MAS source must retain missing approved Tool/Owner contract diagnostics: %#v", mas.Diagnostics)
	}
	for _, unsupported := range []string{"SKILL_METADATA_INVALID", "CONTRACT_SPEC", "CASE_INVALID", "PREDICATE_SOURCE_INCOMPLETE"} {
		if findDiagnostic(mas.Diagnostics, unsupported) {
			t.Errorf("current MAS source form was not parsed according to the source Language (%s): %#v", unsupported, mas.Diagnostics)
		}
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
