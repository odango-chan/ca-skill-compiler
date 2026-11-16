package compiler

// WP10 success-path conformance: a fully validated v1alpha4 Skill (no semantics
// that remain feature-fail-closed) produces a real, deterministic Artifact tree.

import (
	"bytes"
	"strings"
	"testing"
)

func evidenceFreeBaseRequest(t *testing.T) BuildRequest {
	t.Helper()
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aPredicatesPath,
		"    evidence:\n      requires:\n        registration: [ca::ue.registration-state]\n",
		"")
	req := wp3cRequest(t, fixture, wp3cBaseBundle(t))
	if diagnostics := validateBuildRequest(req); len(diagnostics) != 0 {
		t.Fatalf("fixture rejected at the request envelope: %+v", diagnostics)
	}
	return req
}

func TestV1Alpha4ValidSkillBuildsDeterministicArtifact(t *testing.T) {
	req := evidenceFreeBaseRequest(t)
	check := Check(req)
	for _, diagnostic := range check {
		if diagnostic.Severity == "error" {
			t.Fatalf("valid Skill produced an error diagnostic: %+v", diagnostic)
		}
	}
	result := Build(req)
	if !result.Success {
		t.Fatalf("valid Skill Build did not succeed: %+v", result.Diagnostics)
	}
	if result.APIVersion != BuildResultAPIVersion {
		t.Fatalf("BuildResult apiVersion = %q", result.APIVersion)
	}
	if len(result.OutputFiles) == 0 {
		t.Fatal("successful Build emitted no files")
	}
	var manifest []byte
	hasProvenance := false
	for _, file := range result.OutputFiles {
		switch {
		case file.Path == "skills/verify-ue-registration/generated/artifact-manifest.json":
			manifest = file.Bytes
		case strings.HasSuffix(file.Path, "assurance/contract.yaml"):
			hasProvenance = true
		}
	}
	if manifest == nil {
		t.Fatalf("artifact manifest missing: %v", outputPaths(result))
	}
	if !hasProvenance {
		t.Fatalf("author source files were not preserved: %v", outputPaths(result))
	}
	if len(result.DependencyClosure) == 0 {
		t.Fatal("successful Build must record the used dependency closure")
	}
	if !sortedStringsAreSorted(result.DependencyClosure) {
		t.Fatalf("dependency closure is not sorted: %v", result.DependencyClosure)
	}

	second := Build(req)
	if !second.Success || len(second.OutputFiles) != len(result.OutputFiles) {
		t.Fatalf("second Build diverged: %+v", second)
	}
	for index := range result.OutputFiles {
		if result.OutputFiles[index].Path != second.OutputFiles[index].Path ||
			!bytes.Equal(result.OutputFiles[index].Bytes, second.OutputFiles[index].Bytes) {
			t.Fatalf("Build output is not deterministic at %s", result.OutputFiles[index].Path)
		}
	}
}

func outputPaths(result BuildResult) []string {
	paths := make([]string, 0, len(result.OutputFiles))
	for _, file := range result.OutputFiles {
		paths = append(paths, file.Path)
	}
	return paths
}

func sortedStringsAreSorted(values []string) bool {
	for index := 1; index < len(values); index++ {
		if values[index-1] > values[index] {
			return false
		}
	}
	return true
}
