package compiler

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func validEnvelopeRequest() BuildRequest {
	return BuildRequest{
		APIVersion:          BuildRequestAPIVersion,
		LanguageTarget:      V1Alpha4Target,
		Files:               []File{{Path: "skills/demo/SKILL.md", Bytes: []byte("# demo\n")}},
		ResolvedDependencies: []ResolvedDependency{},
	}
}

func hasCode(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func TestDependencySnapshotDigestGoldenVectors(t *testing.T) {
	raw, err := os.ReadFile("../../contracts/v1alpha4/dependency-snapshot-digest-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors struct {
		Vectors []struct {
			Name           string `json:"name"`
			Files          []struct {
				Path  string `json:"path"`
				Bytes string `json:"bytes"`
			} `json:"files"`
			ExpectedDigest string `json:"expectedDigest"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors.Vectors) != 3 {
		t.Fatalf("expected 3 frozen vectors, got %d", len(vectors.Vectors))
	}
	for _, vector := range vectors.Vectors {
		t.Run(vector.Name, func(t *testing.T) {
			files := make([]File, 0, len(vector.Files))
			for _, item := range vector.Files {
				decoded, err := base64.StdEncoding.DecodeString(item.Bytes)
				if err != nil {
					t.Fatal(err)
				}
				if decoded == nil {
					decoded = []byte{}
				}
				files = append(files, File{Path: item.Path, Bytes: decoded})
			}
			got, err := ComputeDependencySnapshotDigest(files)
			if err != nil {
				t.Fatal(err)
			}
			if got != vector.ExpectedDigest {
				t.Fatalf("digest = %s, want %s", got, vector.ExpectedDigest)
			}
		})
	}
}

func TestDependencySnapshotDigestRejectsUnsafeAndDuplicatePaths(t *testing.T) {
	for _, path := range []string{"", "/absolute", "../escape", "a/../b", "a//b", "a\\b", "a:b", "a/./b"} {
		t.Run(path, func(t *testing.T) {
			if _, err := ComputeDependencySnapshotDigest([]File{{Path: path, Bytes: []byte("x")}}); err == nil {
				t.Fatalf("unsafe path accepted: %q", path)
			}
		})
	}
	if _, err := ComputeDependencySnapshotDigest([]File{
		{Path: "a.yaml", Bytes: []byte("one")},
		{Path: "a.yaml", Bytes: []byte("two")},
	}); err == nil {
		t.Fatal("duplicate path accepted")
	}
}

func TestDependencySnapshotDigestIsOrderIndependentAndContentSensitive(t *testing.T) {
	files := []File{
		{Path: "z/file", Bytes: []byte("z")},
		{Path: "a/file", Bytes: []byte("a")},
	}
	first, err := ComputeDependencySnapshotDigest(files)
	if err != nil {
		t.Fatal(err)
	}
	reversed, err := ComputeDependencySnapshotDigest([]File{files[1], files[0]})
	if err != nil {
		t.Fatal(err)
	}
	if first != reversed {
		t.Fatalf("file order changed digest: %s != %s", first, reversed)
	}
	changed, err := ComputeDependencySnapshotDigest([]File{
		{Path: "z/file", Bytes: []byte("z")},
		{Path: "a/file", Bytes: []byte("A")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first == changed {
		t.Fatal("content change did not change digest")
	}
}

func TestBuildRequestRequiresExactIndependentVersions(t *testing.T) {
	req := validEnvelopeRequest()
	req.APIVersion = "connectionagent.io/compiler-build-request/v1alpha3"
	if got := Check(req); !hasCode(got, "REQUEST_INVALID") {
		t.Fatalf("unsupported API version was not rejected: %+v", got)
	}

	req = validEnvelopeRequest()
	req.LanguageTarget = "connectionagent.io/v1alpha3"
	if got := Check(req); !hasCode(got, "LANGUAGE_TARGET_UNSUPPORTED") {
		t.Fatalf("unsupported language target was not rejected: %+v", got)
	}
}

func TestBuildRequestRequiresExplicitEmptyDependencyArray(t *testing.T) {
	req := validEnvelopeRequest()
	req.ResolvedDependencies = nil
	if got := Check(req); !hasCode(got, "REQUEST_INVALID") {
		t.Fatalf("missing resolvedDependencies array was accepted: %+v", got)
	}
}

func TestBuildRequestValidatesSourcePathsAndDuplicates(t *testing.T) {
	req := validEnvelopeRequest()
	req.Files[0].Path = "skills/demo/../SKILL.md"
	if got := Check(req); !hasCode(got, "PACKAGE_PATH_INVALID") {
		t.Fatalf("traversal source path was accepted: %+v", got)
	}

	req = validEnvelopeRequest()
	req.Files = append(req.Files, req.Files[0])
	if got := Check(req); !hasCode(got, "PACKAGE_PATH_INVALID") {
		t.Fatalf("duplicate source path was accepted: %+v", got)
	}

	req = validEnvelopeRequest()
	req.Files[0].Path = "lib/tools/catalog.yaml"
	if got := Check(req); !hasCode(got, "PACKAGE_PATH_INVALID") {
		t.Fatalf("legacy lib path was accepted as author source: %+v", got)
	}
}

func TestBuildRequestVerifiesDependencySnapshotDigest(t *testing.T) {
	req := validEnvelopeRequest()
	req.ResolvedDependencies = []ResolvedDependency{{
		DependencyID: "host-contracts",
		Digest:       "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		Files:        []File{{Path: "catalog.yaml", Bytes: []byte("apiVersion: v1\n")}},
	}}
	if got := Check(req); !hasCode(got, "DEPENDENCY_IDENTITY_UNVERIFIED") {
		t.Fatalf("digest mismatch was not rejected: %+v", got)
	}

	actual, err := ComputeDependencySnapshotDigest(req.ResolvedDependencies[0].Files)
	if err != nil {
		t.Fatal(err)
	}
	req.ResolvedDependencies[0].Digest = actual
	got := Check(req)
	if hasCode(got, "DEPENDENCY_IDENTITY_UNVERIFIED") {
		t.Fatalf("correct snapshot digest was rejected at the request boundary: %+v", got)
	}
}

func TestBuildResultUsesFrozenOutputEnvelopeAndFailureAtomicity(t *testing.T) {
	result := Build(validEnvelopeRequest())
	if result.APIVersion != BuildResultAPIVersion {
		t.Fatalf("BuildResult apiVersion = %q", result.APIVersion)
	}
	if result.Success {
		t.Fatal("source-only partial compiler must not claim a successful build")
	}
	if result.DependencyClosure == nil || len(result.DependencyClosure) != 0 {
		t.Fatalf("failed build must have empty dependencyClosure: %#v", result.DependencyClosure)
	}
	if result.OutputFiles == nil || len(result.OutputFiles) != 0 {
		t.Fatalf("failed build must have empty outputFiles: %#v", result.OutputFiles)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Phase != "C_STATIC" || diagnostic.Severity != "error" || diagnostic.Message == "" {
			t.Fatalf("invalid v1alpha4 diagnostic shape: %+v", diagnostic)
		}
	}
}

func TestInspectDependenciesFailsClosedUntilLinkerExists(t *testing.T) {
	req := validEnvelopeRequest()
	result := InspectDependencies(req)
	if result.APIVersion != DependencyInspectionAPIVersion {
		t.Fatalf("DependencyInspection apiVersion = %q", result.APIVersion)
	}
	if result.Callables == nil || result.Schemas == nil {
		t.Fatal("empty inspection arrays must be non-null")
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("empty dependency inventory should be inspectable: %+v", result.Diagnostics)
	}

	req.ResolvedDependencies = []ResolvedDependency{{
		DependencyID: "host-contracts",
		Files:        []File{{Path: "catalog.yaml", Bytes: []byte("apiVersion: v1\n")}},
	}}
	req.ResolvedDependencies[0].Digest, _ = ComputeDependencySnapshotDigest(req.ResolvedDependencies[0].Files)
	result = InspectDependencies(req)
	if !hasCode(result.Diagnostics, "DEPENDENCY_IDENTITY_UNVERIFIED") {
		t.Fatalf("nonempty dependency inventory must fail closed before WP3: %+v", result)
	}
}

func TestDecodeBuildRequestJSONIsStrictAndDecodesBase64(t *testing.T) {
	wire := `{"apiVersion":"connectionagent.io/compiler-build-request/v1alpha4","languageTarget":"connectionagent.io/v1alpha4","files":[{"path":"skills/demo/SKILL.md","bytes":"IyBkZW1vCg=="}],"resolvedDependencies":[]}`
	req, err := DecodeBuildRequestJSON([]byte(wire))
	if err != nil {
		t.Fatal(err)
	}
	if string(req.Files[0].Bytes) != "# demo\n" {
		t.Fatalf("base64 bytes decoded incorrectly: %q", req.Files[0].Bytes)
	}
	if _, err := DecodeBuildRequestJSON([]byte(strings.Replace(wire, `"files":`, `"unknown":true,"files":`, 1))); err == nil {
		t.Fatal("unknown JSON property accepted")
	}
	if _, err := DecodeBuildRequestJSON([]byte(strings.Replace(wire, `"bytes":"IyBkZW1vCg=="`, `"bytes":"not-base64"`, 1))); err == nil {
		t.Fatal("invalid base64 byte field accepted")
	}
	if _, err := DecodeBuildRequestJSON([]byte(wire + ` {}`)); err == nil {
		t.Fatal("trailing JSON value accepted")
	}
}
