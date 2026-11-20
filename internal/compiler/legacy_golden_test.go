package compiler_test

// Frozen-entry golden baseline.
//
// These cases exercise the released v1alpha5 public entries (Check, Build,
// InspectDependencies, VerifyBuildResult) through the same explicit requests the
// release acceptance used. The expected document was generated at the v2.0.2
// baseline commit before the Development Build change; the test proves the new
// code path did not alter the frozen entry behavior. Regenerate only on a
// deliberate public-contract change with UPDATE_LEGACY_GOLDEN=1.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/odango-chan/ca-skill-compiler/internal/compiler"
	"github.com/odango-chan/ca-skill-compiler/internal/projectinput"
)

type legacyGoldenDiagnostic struct {
	Severity string `json:"severity"`
	Phase    string `json:"phase"`
	Code     string `json:"code"`
	Path     string `json:"path"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Message  string `json:"message"`
}

type legacyGoldenFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type legacyGoldenCallable struct {
	DependencyID         string `json:"dependencyId"`
	Owner                string `json:"owner"`
	Kind                 string `json:"kind"`
	ID                   string `json:"id"`
	Effect               string `json:"effect"`
	SourcePath           string `json:"sourcePath"`
	InputSchemaID        string `json:"inputSchemaId,omitempty"`
	QueryContextSchemaID string `json:"queryContextSchemaId,omitempty"`
	OutputSchemaID       string `json:"outputSchemaId"`
	ReceiptSchemaID      string `json:"receiptSchemaId,omitempty"`
}

type legacyGoldenSchema struct {
	DependencyID string `json:"dependencyId"`
	Owner        string `json:"owner"`
	ID           string `json:"id"`
	SourcePath   string `json:"sourcePath"`
}

type legacyGoldenIngress struct {
	DependencyID   string `json:"dependencyId"`
	Owner          string `json:"owner"`
	Kind           string `json:"kind"`
	ID             string `json:"id"`
	EventSource    string `json:"eventSource"`
	EventService   string `json:"eventService"`
	EventOperation string `json:"eventOperation"`
	Method         string `json:"method"`
	Path           string `json:"path"`
}

type legacyGoldenCase struct {
	Name              string                   `json:"name"`
	Check             []legacyGoldenDiagnostic `json:"check"`
	BuildSuccess      bool                     `json:"buildSuccess"`
	BuildDiagnostics  []legacyGoldenDiagnostic `json:"buildDiagnostics"`
	DependencyClosure []string                 `json:"dependencyClosure"`
	OutputFiles       []legacyGoldenFile       `json:"outputFiles"`
	Inspect           legacyGoldenInspection   `json:"inspect"`
	VerifyBuildResult string                   `json:"verifyBuildResult"`
}

type legacyGoldenInspection struct {
	Diagnostics       []legacyGoldenDiagnostic `json:"diagnostics"`
	Callables         []legacyGoldenCallable   `json:"callables"`
	Schemas           []legacyGoldenSchema     `json:"schemas"`
	IngressOperations []legacyGoldenIngress    `json:"ingressOperations"`
}

type legacyGoldenDocument struct {
	BaselineCommit string             `json:"baselineCommit"`
	Cases          []legacyGoldenCase `json:"cases"`
}

func TestLegacyEntryGoldenBaseline(t *testing.T) {
	goldenPath := filepath.Join("..", "..", "tests", "golden", "legacy-entry-v2.0.2.json")
	actual := legacyGoldenActual(t)
	if os.Getenv("UPDATE_LEGACY_GOLDEN") == "1" {
		raw, err := json.MarshalIndent(actual, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, append(raw, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote legacy golden baseline to %s", goldenPath)
		return
	}
	raw, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read legacy golden baseline: %v", err)
	}
	var expected legacyGoldenDocument
	if err := json.Unmarshal(raw, &expected); err != nil {
		t.Fatalf("decode legacy golden baseline: %v", err)
	}
	if !reflect.DeepEqual(normalizeGoldenDocument(actual), normalizeGoldenDocument(expected)) {
		actualRaw, _ := json.MarshalIndent(normalizeGoldenDocument(actual), "", "  ")
		expectedRaw, _ := json.MarshalIndent(normalizeGoldenDocument(expected), "", "  ")
		t.Fatalf("frozen v1alpha5 entry behavior changed.\nactual:\n%s\nexpected:\n%s", actualRaw, expectedRaw)
	}
}

func normalizeGoldenDocument(document legacyGoldenDocument) legacyGoldenDocument {
	// The baseline commit is provenance, not behavior: compare the cases only.
	document.BaselineCommit = ""
	return document
}

func legacyGoldenActual(t *testing.T) legacyGoldenDocument {
	t.Helper()
	root := filepath.Join("..", "..")
	document := legacyGoldenDocument{BaselineCommit: "e98a7b26acabfa2a5e1587974ab9be5742e70bf7"}
	for _, testCase := range []struct {
		name    string
		request compiler.BuildRequest
	}{
		{"v1alpha5-trusted", mustDecodeLegacyRequest(t, filepath.Join(root, "web", "parity-v1alpha5", "trusted-build-request.json"))},
		{"v1alpha5-invalid-procedure", legacyInvalidProcedureRequest(t, root)},
		{"parity-success-project", mustLoadLegacyProject(t, filepath.Join(root, "web", "parity-success"))},
		{"parity-success-no-evidence", legacyNoEvidenceRequest(t, root)},
		{"parity-project", mustLoadLegacyProject(t, filepath.Join(root, "web", "parity-project"))},
	} {
		document.Cases = append(document.Cases, legacyGoldenRun(testCase.name, testCase.request))
	}
	return document
}

func legacyGoldenRun(name string, request compiler.BuildRequest) legacyGoldenCase {
	check := compiler.Check(request)
	build := compiler.Build(request)
	inspection := compiler.InspectDependencies(request)
	verify := compiler.VerifyBuildResult(build)
	caseResult := legacyGoldenCase{
		Name:              name,
		Check:             goldenDiagnostics(check),
		BuildSuccess:      build.Success,
		BuildDiagnostics:  goldenDiagnostics(build.Diagnostics),
		DependencyClosure: append([]string{}, build.DependencyClosure...),
		OutputFiles:       []legacyGoldenFile{},
		VerifyBuildResult: "",
	}
	for _, file := range build.OutputFiles {
		sum := sha256.Sum256(file.Bytes)
		caseResult.OutputFiles = append(caseResult.OutputFiles, legacyGoldenFile{Path: file.Path, SHA256: "sha256:" + hex.EncodeToString(sum[:])})
	}
	sort.Slice(caseResult.OutputFiles, func(i, j int) bool { return caseResult.OutputFiles[i].Path < caseResult.OutputFiles[j].Path })
	if caseResult.DependencyClosure == nil {
		caseResult.DependencyClosure = []string{}
	}
	if verify != nil {
		caseResult.VerifyBuildResult = verify.Error()
	}
	caseResult.Inspect = legacyGoldenInspection{
		Diagnostics:       goldenDiagnostics(inspection.Diagnostics),
		Callables:         []legacyGoldenCallable{},
		Schemas:           []legacyGoldenSchema{},
		IngressOperations: []legacyGoldenIngress{},
	}
	for _, callable := range inspection.Callables {
		caseResult.Inspect.Callables = append(caseResult.Inspect.Callables, legacyGoldenCallable{
			DependencyID: callable.DependencyID, Owner: callable.Owner, Kind: callable.Kind, ID: callable.ID,
			Effect: callable.Effect, SourcePath: callable.SourcePath, InputSchemaID: callable.InputSchemaID,
			QueryContextSchemaID: callable.QueryContextSchemaID, OutputSchemaID: callable.OutputSchemaID,
			ReceiptSchemaID: callable.ReceiptSchemaID,
		})
	}
	for _, schema := range inspection.Schemas {
		caseResult.Inspect.Schemas = append(caseResult.Inspect.Schemas, legacyGoldenSchema{
			DependencyID: schema.DependencyID, Owner: schema.Owner, ID: schema.ID, SourcePath: schema.SourcePath,
		})
	}
	for _, operation := range inspection.IngressOperations {
		caseResult.Inspect.IngressOperations = append(caseResult.Inspect.IngressOperations, legacyGoldenIngress{
			DependencyID: operation.DependencyID, Owner: operation.Owner, Kind: operation.Kind, ID: operation.ID,
			EventSource: operation.EventSource, EventService: operation.EventService, EventOperation: operation.EventOperation,
			Method: operation.Method, Path: operation.Path,
		})
	}
	return caseResult
}

func goldenDiagnostics(diagnostics []compiler.Diagnostic) []legacyGoldenDiagnostic {
	result := make([]legacyGoldenDiagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		result = append(result, legacyGoldenDiagnostic{
			Severity: diagnostic.Severity, Phase: diagnostic.Phase, Code: diagnostic.Code, Path: diagnostic.Path,
			Line: diagnostic.Line, Column: diagnostic.Column, Message: diagnostic.Message,
		})
	}
	return result
}

func mustDecodeLegacyRequest(t *testing.T, path string) compiler.BuildRequest {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	request, err := compiler.DecodeBuildRequestJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func mustLoadLegacyProject(t *testing.T, path string) compiler.BuildRequest {
	t.Helper()
	absolute, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	request, err := projectinput.Load(absolute)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func legacyInvalidProcedureRequest(t *testing.T, root string) compiler.BuildRequest {
	t.Helper()
	request := mustDecodeLegacyRequest(t, filepath.Join(root, "web", "parity-v1alpha5", "trusted-build-request.json"))
	invalid, err := os.ReadFile(filepath.Join(root, "web", "parity-v1alpha5", "invalid-procedure.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for index := range request.Files {
		if filepath.Base(request.Files[index].Path) == "procedure.yaml" {
			request.Files[index].Bytes = invalid
		}
	}
	return request
}

func legacyNoEvidenceRequest(t *testing.T, root string) compiler.BuildRequest {
	t.Helper()
	request := mustLoadLegacyProject(t, filepath.Join(root, "web", "parity-success"))
	for index := range request.ResolvedDependencies {
		files := request.ResolvedDependencies[index].Files[:0]
		for _, file := range request.ResolvedDependencies[index].Files {
			if file.Path == "evidence-guarantees.yaml" {
				continue
			}
			files = append(files, file)
		}
		request.ResolvedDependencies[index].Files = files
		if digest, err := compiler.ComputeDependencySnapshotDigest(files); err == nil {
			request.ResolvedDependencies[index].Digest = digest
			for bindingIndex := range request.TrustedBuildContext.DependencyBindings {
				if request.TrustedBuildContext.DependencyBindings[bindingIndex].DependencyID == request.ResolvedDependencies[index].DependencyID {
					request.TrustedBuildContext.DependencyBindings[bindingIndex].Digest = digest
				}
			}
		}
	}
	return request
}
