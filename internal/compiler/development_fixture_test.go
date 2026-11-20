package compiler

// Shared test helpers for the Development Build validation matrix.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func loadDevelopmentFixture(t *testing.T) DevelopmentBuildRequest {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "design", "fixtures", "registration-development-request.json"))
	if err != nil {
		t.Fatalf("read Development fixture: %v", err)
	}
	request, err := DecodeDevelopmentBuildRequestJSON(raw)
	if err != nil {
		t.Fatalf("decode Development fixture: %v", err)
	}
	return request
}

func cloneDevelopmentRequest(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
	t.Helper()
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("clone Development request: %v", err)
	}
	var clone DevelopmentBuildRequest
	if err := json.Unmarshal(raw, &clone); err != nil {
		t.Fatalf("clone Development request: %v", err)
	}
	return clone
}

func canonicalTestJSON(t *testing.T, value any) []byte {
	t.Helper()
	out, err := canonicalizeValue(value)
	if err != nil {
		t.Fatalf("canonicalize test value: %v", err)
	}
	return out
}

// modifyDevelopmentPackageFile replaces one snapshot file, updates the package
// manifest file digest and recomputes every lock/snapshot reference to that
// package's new snapshot digest.
func modifyDevelopmentPackageFile(t *testing.T, request DevelopmentBuildRequest, packageID, path string, content []byte) DevelopmentBuildRequest {
	t.Helper()
	request = cloneDevelopmentRequest(t, request)
	snapshotIndex := -1
	for index, snapshot := range request.Snapshots {
		if snapshot.Key.PackageID == packageID {
			snapshotIndex = index
			break
		}
	}
	if snapshotIndex < 0 {
		t.Fatalf("package %s not found in fixture", packageID)
	}
	files := request.Snapshots[snapshotIndex].Files
	replaced := false
	for index := range files {
		if files[index].Path == path {
			files[index].Bytes = content
			replaced = true
			break
		}
	}
	if !replaced {
		files = append(files, File{Path: path, Bytes: content})
	}
	// Keep package.json files[] in sync with the actual file set.
	var manifest map[string]any
	for index := range files {
		if files[index].Path == "package.json" {
			if err := json.Unmarshal(files[index].Bytes, &manifest); err != nil {
				t.Fatalf("decode package.json: %v", err)
			}
		}
	}
	if manifest != nil {
		sum := sha256.Sum256(content)
		entries, _ := manifest["files"].([]any)
		found := false
		for _, entry := range entries {
			object, _ := entry.(map[string]any)
			if object["path"] == path {
				object["sha256"] = "sha256:" + hex.EncodeToString(sum[:])
				found = true
			}
		}
		if !found {
			entries = append(entries, map[string]any{"path": path, "sha256": "sha256:" + hex.EncodeToString(sum[:])})
		}
		manifest["files"] = entries
		raw, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			t.Fatalf("encode package.json: %v", err)
		}
		for index := range files {
			if files[index].Path == "package.json" {
				files[index].Bytes = append(raw, '\n')
			}
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	request.Snapshots[snapshotIndex].Files = files
	digest, err := ComputeDependencySnapshotDigest(files)
	if err != nil {
		t.Fatalf("recompute snapshot digest: %v", err)
	}
	oldDigest := request.Snapshots[snapshotIndex].Key.SnapshotDigest
	request.Snapshots[snapshotIndex].Key.SnapshotDigest = digest
	for index := range request.Lock.Roots {
		if request.Lock.Roots[index].Key.PackageID == packageID {
			request.Lock.Roots[index].Key.SnapshotDigest = digest
		}
	}
	for index := range request.Lock.Packages {
		if request.Lock.Packages[index].Key.PackageID == packageID {
			request.Lock.Packages[index].Key.SnapshotDigest = digest
		}
		for edgeIndex := range request.Lock.Packages[index].Dependencies {
			if request.Lock.Packages[index].Dependencies[edgeIndex].Key.PackageID == packageID {
				request.Lock.Packages[index].Dependencies[edgeIndex].Key.SnapshotDigest = digest
			}
		}
	}
	for index := range request.Assurance {
		if request.Assurance[index].Key.PackageID == packageID && request.Assurance[index].Key.SnapshotDigest == oldDigest {
			request.Assurance[index].Key.SnapshotDigest = digest
		}
	}
	return request
}

func addDevelopmentSnapshot(t *testing.T, request DevelopmentBuildRequest, snapshot DevelopmentSnapshot) DevelopmentBuildRequest {
	t.Helper()
	request = cloneDevelopmentRequest(t, request)
	request.Snapshots = append(request.Snapshots, snapshot)
	return request
}

// tamperDevelopmentArtifact mutates artifact files and then recomputes the
// manifest file digests and artifactDigest, simulating an attacker who removes an
// obligation or edits a Procedure and re-seals the outer digest.
func tamperDevelopmentArtifact(t *testing.T, result DevelopmentBuildResult, mutate func(files map[string][]byte)) DevelopmentBuildResult {
	t.Helper()
	files := map[string][]byte{}
	for _, file := range result.OutputFiles {
		files[file.Path] = append([]byte(nil), file.Bytes...)
	}
	mutate(files)
	var manifest developmentArtifactManifest
	if err := json.Unmarshal(files[developmentManifestFileName], &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	manifest.Files = nil
	for path, raw := range files {
		if path == developmentManifestFileName {
			continue
		}
		sum := sha256.Sum256(raw)
		manifest.Files = append(manifest.Files, developmentManifestFile{Path: path, SHA256: "sha256:" + hex.EncodeToString(sum[:])})
	}
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	manifestBytes := canonicalTestJSON(t, manifest)
	files[developmentManifestFileName] = manifestBytes
	sum := sha256.Sum256(manifestBytes)
	result.ArtifactDigest = "sha256:" + hex.EncodeToString(sum[:])
	result.OutputFiles = nil
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		result.OutputFiles = append(result.OutputFiles, File{Path: path, Bytes: files[path]})
	}
	return result
}

func hasDevelopmentCode(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func developmentObligationByRule(t *testing.T, result DevelopmentBuildResult, rule, subjectID string) (DevelopmentObligation, bool) {
	t.Helper()
	raw := developmentOutputFile(t, result, developmentObligationsFileName)
	var obligations []DevelopmentObligation
	if err := json.Unmarshal(raw, &obligations); err != nil {
		t.Fatalf("decode obligations: %v", err)
	}
	for _, obligation := range obligations {
		if obligation.Rule != rule {
			continue
		}
		for _, subject := range obligation.Subjects {
			if subject.ID == subjectID {
				return obligation, true
			}
		}
	}
	return DevelopmentObligation{}, false
}

func developmentOutputFile(t *testing.T, result DevelopmentBuildResult, path string) []byte {
	t.Helper()
	for _, file := range result.OutputFiles {
		if file.Path == path {
			return file.Bytes
		}
	}
	t.Fatalf("Development artifact has no %s", path)
	return nil
}
