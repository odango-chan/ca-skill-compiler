package compiler

// WP10 artifact integrity: VerifyBuildResult accepts a real successful Build and
// rejects tampering, missing listings and non-canonical manifests.

import (
	"strings"
	"testing"
)

func verifiedBuildRequest(t *testing.T) BuildResult {
	t.Helper()
	result := Build(evidenceFreeBaseRequest(t))
	if !result.Success {
		t.Fatalf("expected successful Build, got %+v", result.Diagnostics)
	}
	if err := VerifyBuildResult(result); err != nil {
		t.Fatalf("successful Build failed artifact verification: %v", err)
	}
	return result
}

func TestVerifyBuildResultRejectsDuplicatePath(t *testing.T) {
	result := verifiedBuildRequest(t)
	result.OutputFiles = append(result.OutputFiles, result.OutputFiles[0])
	if err := VerifyBuildResult(result); err == nil {
		t.Fatal("duplicate artifact path accepted")
	}
}

func TestVerifyBuildResultRejectsNonCanonicalManifest(t *testing.T) {
	result := verifiedBuildRequest(t)
	for index := range result.OutputFiles {
		if strings.HasSuffix(result.OutputFiles[index].Path, "/generated/artifact-manifest.json") {
			result.OutputFiles[index].Bytes = append(append([]byte{}, result.OutputFiles[index].Bytes...), ' ')
		}
	}
	if err := VerifyBuildResult(result); err == nil || !strings.Contains(err.Error(), "canonical") {
		t.Fatalf("non-canonical manifest accepted: %v", err)
	}
}

func TestVerifyBuildResultRejectsUnlistedFile(t *testing.T) {
	result := verifiedBuildRequest(t)
	result.OutputFiles = append(result.OutputFiles, File{Path: "skills/verify-ue-registration/generated/extra.txt", Bytes: []byte("x")})
	if err := VerifyBuildResult(result); err == nil || !strings.Contains(err.Error(), "not listed") {
		t.Fatalf("unlisted artifact file accepted: %v", err)
	}
}

func TestVerifyBuildResultRejectsTamperedSchemaIdentity(t *testing.T) {
	result := verifiedBuildRequest(t)
	tampered := false
	for index := range result.OutputFiles {
		if strings.Contains(result.OutputFiles[index].Path, "/generated/script-output-schemas/") {
			result.OutputFiles[index].Bytes = []byte(strings.Replace(string(result.OutputFiles[index].Bytes), "\"urn:connectionagent:skill:", "\"urn:connectionagent:evil:", 1))
			tampered = true
		}
	}
	if !tampered {
		t.Skip("fixture produced no generated Script Output Schema")
	}
	if err := VerifyBuildResult(result); err == nil {
		t.Fatal("tampered generated Schema identity accepted")
	}
}
