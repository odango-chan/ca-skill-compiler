package parity

// Static guard for the checked-in synthetic v1alpha5 trusted parity fixture. It
// keeps web/parity-v1alpha5/trusted-build-request.json decodable and Buildable
// through the public compiler API so the real-Chromium harness cannot silently
// use a stale or invalid request. This test does not need a browser.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/odango-chan/ca-skill-compiler/internal/compiler"
)

func TestV1Alpha5TrustedParityFixtureBuilds(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "web", "parity-v1alpha5", "trusted-build-request.json"))
	if err != nil {
		t.Fatalf("read trusted v1alpha5 parity fixture: %v", err)
	}
	request, err := compiler.DecodeBuildRequestJSON(raw)
	if err != nil {
		t.Fatalf("trusted v1alpha5 parity fixture does not decode strictly: %v", err)
	}
	if diagnostics := compiler.Check(request); len(diagnostics) != 0 {
		t.Fatalf("trusted v1alpha5 parity fixture must Check clean, got %#v", diagnostics)
	}
	result := compiler.Build(request)
	if !result.Success {
		t.Fatalf("trusted v1alpha5 parity fixture must Build, diagnostics = %#v", result.Diagnostics)
	}
	if result.APIVersion != compiler.BuildResultAPIVersion {
		t.Fatalf("BuildResult.apiVersion = %q", result.APIVersion)
	}
	if len(result.DependencyClosure) != 2 {
		t.Fatalf("dependencyClosure = %v, want [integration tools]", result.DependencyClosure)
	}
	if err := compiler.VerifyBuildResult(result); err != nil {
		t.Fatalf("VerifyBuildResult: %v", err)
	}

	second := compiler.Build(request)
	if !second.Success || len(second.OutputFiles) != len(result.OutputFiles) {
		t.Fatalf("repeated Build of the trusted fixture differs in shape")
	}
	for index := range result.OutputFiles {
		if result.OutputFiles[index].Path != second.OutputFiles[index].Path ||
			!bytes.Equal(result.OutputFiles[index].Bytes, second.OutputFiles[index].Bytes) {
			t.Fatalf("repeated Build of the trusted fixture differs at %q", result.OutputFiles[index].Path)
		}
	}

	var manifest map[string]json.RawMessage
	var manifestRaw []byte
	for _, file := range result.OutputFiles {
		if file.Path == compiler.V1Alpha5ArtifactManifestPath {
			manifestRaw = file.Bytes
		}
	}
	if manifestRaw == nil {
		t.Fatal("trusted fixture artifact has no manifest.json")
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Skills []struct {
			WaitLinks []struct {
				NodeID  string `json:"nodeId"`
				Ingress struct {
					DependencyAlias string `json:"dependencyAlias"`
					OperationID     string `json:"operationId"`
					SnapshotPath    string `json:"snapshotPath"`
				} `json:"ingress"`
				Response struct {
					Mode string `json:"mode"`
				} `json:"response"`
			} `json:"waitLinks"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(manifestRaw, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Skills) != 1 || len(decoded.Skills[0].WaitLinks) != 1 {
		t.Fatalf("trusted fixture manifest waitLinks shape is unexpected")
	}
	waitLink := decoded.Skills[0].WaitLinks[0]
	if waitLink.NodeID != "wait-sm-context" ||
		waitLink.Ingress.DependencyAlias != "integration" ||
		waitLink.Ingress.SnapshotPath != "ingress/catalog.yaml" ||
		waitLink.Ingress.OperationID != "nsmf-pdusession.update-sm-context" {
		t.Fatalf("trusted fixture linked Ingress identity is unexpected: %#v", waitLink)
	}
	if waitLink.Response.Mode != "none" {
		t.Fatalf("trusted fixture response mode = %q", waitLink.Response.Mode)
	}
}
