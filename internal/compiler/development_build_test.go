package compiler

// Development Build acceptance matrix: positive fixture, deterministic rebuild,
// negative interface closures, Assurance Descriptor three-state coverage and
// Verify tamper resistance.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestDevelopmentBuildPositiveFixture(t *testing.T) {
	request := loadDevelopmentFixture(t)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("Development fixture must build, diagnostics = %#v", result.Diagnostics)
	}
	if result.APIVersion != DevelopmentBuildResultAPIVersion {
		t.Fatalf("result.apiVersion = %q", result.APIVersion)
	}
	if result.SourceStatus != DevelopmentSourceValidated || result.InterfaceLinkStatus != DevelopmentInterfaceLinked ||
		result.ArtifactStatus != DevelopmentArtifactBuilt || result.ExecutionStatus != DevelopmentExecutionStatus {
		t.Fatalf("unexpected statuses: %+v", result)
	}
	if result.ArtifactDigest == "" {
		t.Fatal("successful Development result has no artifactDigest")
	}
	paths := make([]string, 0, len(result.OutputFiles))
	for _, file := range result.OutputFiles {
		paths = append(paths, file.Path)
	}
	if !developmentContainsString(paths, developmentManifestFileName) || !developmentContainsString(paths, developmentRequirementsFileName) {
		t.Fatalf("Development artifact files = %v", paths)
	}
	if !developmentContainsString(paths, "skills/verify-ue-registration/compiled/procedure.json") {
		t.Fatalf("Development artifact has no compiled Procedure: %v", paths)
	}
	if len(paths) != 4 {
		t.Fatalf("Development artifact file count = %d, want 4", len(paths))
	}
	// The fixture declares a valid Evidence Descriptor export. The static
	// Evidence checks therefore pass and no Evidence requirement is recorded.
	if _, ok := developmentRequirementByKind(t, result, DevelopmentRequirementOwnerTrust, "ue.registration.current"); !ok {
		t.Fatal("missing OWNER_TRUST Host requirement for the used Observation export")
	}
	if _, ok := developmentRequirementByKind(t, result, DevelopmentRequirementOwnerTrust, "urn:example:ca:invocation-context:v1"); !ok {
		t.Fatal("missing OWNER_TRUST Host requirement for the used Context Schema export")
	}
	if _, ok := developmentRequirementByKind(t, result, DevelopmentRequirementEvidenceDomain, "ue.registration-state"); ok {
		t.Fatal("a statically complete Evidence Descriptor must not create an external Evidence requirement")
	}
	// Determinism: a repeated build is byte identical.
	second := BuildDevelopment(request)
	if !second.Success || second.ArtifactDigest != result.ArtifactDigest {
		t.Fatal("repeated Development build differs")
	}
	assertDevelopmentFilesEqual(t, result, second)
	// Full Verify: actual artifact bytes plus deterministic rebuild.
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("VerifyDevelopment: %v", err)
	}
}

func TestDevelopmentBuildIsReorderingInvariant(t *testing.T) {
	request := loadDevelopmentFixture(t)
	base := BuildDevelopment(request)
	if !base.Success {
		t.Fatalf("base build failed: %#v", base.Diagnostics)
	}
	permuted := cloneDevelopmentRequest(t, request)
	developmentReverseFiles(permuted.Files)
	for index := range permuted.Snapshots {
		developmentReverseFiles(permuted.Snapshots[index].Files)
	}
	reverseSnapshots(permuted.Snapshots)
	reverseRoots(permuted.Lock.Roots)
	reversePackages(permuted.Lock.Packages)
	for index := range permuted.Lock.Packages {
		reverseEdges(permuted.Lock.Packages[index].Dependencies)
	}
	result := BuildDevelopment(permuted)
	if !result.Success {
		t.Fatalf("permuted build failed: %#v", result.Diagnostics)
	}
	if result.ArtifactDigest != base.ArtifactDigest {
		t.Fatalf("reordered inputs changed artifactDigest: %s != %s", result.ArtifactDigest, base.ArtifactDigest)
	}
	assertDevelopmentFilesEqual(t, base, result)
}

func TestDevelopmentBuildNegativeClosures(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest
		wantCode string
	}{
		{
			name: "missing transitive schema package",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				request = cloneDevelopmentRequest(t, request)
				var snapshots []DevelopmentSnapshot
				for _, snapshot := range request.Snapshots {
					if snapshot.Key.PackageID == "urn:example:ca:types" {
						continue
					}
					snapshots = append(snapshots, snapshot)
				}
				request.Snapshots = snapshots
				var packages []DevelopmentLockPackage
				for _, entry := range request.Lock.Packages {
					if entry.Key.PackageID == "urn:example:ca:types" {
						continue
					}
					if entry.Key.PackageID == "urn:example:ca:tools" {
						entry.Dependencies = nil
					}
					packages = append(packages, entry)
				}
				request.Lock.Packages = packages
				return request
			},
			wantCode: developmentLockInvalid,
		},
		{
			name: "wrong schema reference",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				return modifyDevelopmentPackageFile(t, request, "urn:example:ca:tools", "registration-query.schema.yaml",
					[]byte("$schema: https://json-schema.org/draft/2020-12/schema\n$id: urn:example:ca:registration-query:v1\ntype: object\nadditionalProperties: false\nrequired: [ueRef]\nproperties:\n  ueRef:\n    $ref: urn:example:ca:missing-types:v1\n"))
			},
			wantCode: "SCHEMA_MISSING",
		},
		{
			name: "missing capability member",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				return modifyDevelopmentPackageFile(t, request, "urn:example:ca:tools", "capabilities.json",
					[]byte("{\n  \"capabilities\": [\n    {\n      \"id\": \"example.registration-read\",\n      \"members\": [\n        {\n          \"callableId\": \"ue.registration.missing\"\n        }\n      ]\n    }\n  ]\n}\n"))
			},
			wantCode: developmentSnapshotInvalid,
		},
		{
			name: "same version different digest",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				request = cloneDevelopmentRequest(t, request)
				var source DevelopmentSnapshot
				for _, snapshot := range request.Snapshots {
					if snapshot.Key.PackageID == "urn:example:ca:tools" {
						source = snapshot
					}
				}
				altered := append([]File(nil), source.Files...)
				for index := range altered {
					if altered[index].Path == "registration.schema.yaml" {
						altered[index].Bytes = append(append([]byte(nil), altered[index].Bytes...), '\n')
					}
				}
				return addDevelopmentSnapshot(t, request, DevelopmentSnapshot{Key: source.Key, Files: altered})
			},
			wantCode: developmentSnapshotInvalid,
		},
		{
			name: "unlocked external dependency",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				return modifyDevelopmentManifest(t, request, "urn:example:ca:tools", func(manifest map[string]any) {
					dependencies, _ := manifest["dependencies"].([]any)
					manifest["dependencies"] = append(dependencies, map[string]any{
						"alias": "extra", "packageId": "urn:example:ca:extra", "version": "1.0.0",
					})
				})
			},
			wantCode: developmentLockInvalid,
		},
		{
			name: "duplicate exported schema symbol",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				request = cloneDevelopmentRequest(t, request)
				// Add the tools registration Schema (same $id) to the host-context
				// package and export it there.
				var duplicate []byte
				for _, snapshot := range request.Snapshots {
					if snapshot.Key.PackageID != "urn:example:ca:tools" {
						continue
					}
					for _, file := range snapshot.Files {
						if file.Path == "registration.schema.yaml" {
							duplicate = file.Bytes
						}
					}
				}
				request = addDevelopmentExportAndFile(t, request, "urn:example:ca:host-context",
					"duplicate.schema.yaml", duplicate,
					map[string]any{"kind": "Schema", "id": "urn:example:ca:registration:v1", "path": "duplicate.schema.yaml"})
				return request
			},
			wantCode: developmentLockInvalid,
		},
		{
			name: "wrong callable effect",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				var current []byte
				for _, snapshot := range request.Snapshots {
					if snapshot.Key.PackageID != "urn:example:ca:tools" {
						continue
					}
					for _, file := range snapshot.Files {
						if file.Path == "tool-catalog.yaml" {
							current = file.Bytes
						}
					}
				}
				modified := bytes.Replace(current, []byte("effect: READ"), []byte("effect: MUTATION"), 1)
				return modifyDevelopmentPackageFile(t, request, "urn:example:ca:tools", "tool-catalog.yaml", modified)
			},
			wantCode: developmentSnapshotInvalid,
		},
		{
			name: "wrong evidence descriptor producer",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				descriptor, err := os.ReadFile(filepath.Join("..", "..", "tests", "conformance", "v1alpha4", "fixtures", "dependency-identity", "base", "snapshots", "ca", "evidence-guarantees.yaml"))
				if err != nil {
					t.Fatal(err)
				}
				modified := bytes.Replace(descriptor, []byte("path: output"), []byte("path: other"), 1)
				return modifyDevelopmentPackageFile(t, request, "urn:example:ca:tools", "evidence-guarantees.yaml", modified)
			},
			wantCode: "EVIDENCE_GUARANTEE_LINK_INVALID",
		},
		{
			name: "lock root missing",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				request = cloneDevelopmentRequest(t, request)
				var roots []DevelopmentLockRoot
				for _, root := range request.Lock.Roots {
					if root.Alias == "ca-context" {
						continue
					}
					roots = append(roots, root)
				}
				request.Lock.Roots = roots
				return request
			},
			wantCode: developmentLockInvalid,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := testCase.mutate(t, loadDevelopmentFixture(t))
			result := BuildDevelopment(request)
			if result.Success {
				t.Fatalf("negative case %q unexpectedly succeeded", testCase.name)
			}
			if len(result.OutputFiles) != 0 || result.ArtifactDigest != "" {
				t.Fatalf("failed build produced artifact bytes")
			}
			if !hasDevelopmentCode(result.Diagnostics, testCase.wantCode) {
				t.Fatalf("expected %s, got %#v", testCase.wantCode, result.Diagnostics)
			}
			if result.ExecutionStatus != DevelopmentExecutionStatus {
				t.Fatalf("failed build executionStatus = %q", result.ExecutionStatus)
			}
		})
	}
}

func TestDevelopmentEvidenceDescriptorStates(t *testing.T) {
	descriptorRaw, err := os.ReadFile(filepath.Join("..", "..", "tests", "conformance", "v1alpha4", "fixtures", "dependency-identity", "base", "snapshots", "ca", "evidence-guarantees.yaml"))
	if err != nil {
		t.Fatalf("read evidence descriptor fixture: %v", err)
	}
	withDescriptor := func(t *testing.T, content []byte) DevelopmentBuildRequest {
		return modifyDevelopmentPackageFile(t, loadDevelopmentFixture(t), "urn:example:ca:tools", "evidence-guarantees.yaml", content)
	}

	// Present and statically matching: the frozen Evidence semantics pass and no
	// external Evidence requirement is recorded.
	result := BuildDevelopment(withDescriptor(t, descriptorRaw))
	if !result.Success {
		t.Fatalf("Evidence Descriptor build failed: %#v", result.Diagnostics)
	}
	if _, ok := developmentRequirementByKind(t, result, DevelopmentRequirementEvidenceDomain, "ue.registration-state"); ok {
		t.Fatal("a statically complete Evidence Descriptor must not create an external Evidence requirement")
	}
	if err := VerifyDevelopment(withDescriptor(t, descriptorRaw), result); err != nil {
		t.Fatalf("VerifyDevelopment with Evidence Descriptor: %v", err)
	}

	// Missing static Descriptor: the frozen Evidence semantics cannot complete, so
	// the Build must fail instead of becoming an UNRESOLVED exemption.
	missing := modifyDevelopmentManifest(t, loadDevelopmentFixture(t), "urn:example:ca:tools", func(manifest map[string]any) {
		exports, _ := manifest["exports"].([]any)
		kept := make([]any, 0, len(exports))
		for _, export := range exports {
			object, _ := export.(map[string]any)
			if object["kind"] == "EvidenceDescriptor" {
				continue
			}
			kept = append(kept, export)
		}
		manifest["exports"] = kept
	})
	missingResult := BuildDevelopment(missing)
	if missingResult.Success {
		t.Fatal("missing Evidence Descriptor must fail the Build")
	}
	if !hasDevelopmentCode(missingResult.Diagnostics, "EVIDENCE_GUARANTEE_LINK_INVALID") {
		t.Fatalf("expected EVIDENCE_GUARANTEE_LINK_INVALID, got %#v", missingResult.Diagnostics)
	}

	// Present but contradictory: producer path must match provides.path.
	wrongPath := bytes.Replace(descriptorRaw, []byte("path: output"), []byte("path: other"), 1)
	wrong := BuildDevelopment(withDescriptor(t, wrongPath))
	if wrong.Success {
		t.Fatal("contradictory Evidence Descriptor must fail the Build")
	}
	if !hasDevelopmentCode(wrong.Diagnostics, "EVIDENCE_GUARANTEE_LINK_INVALID") {
		t.Fatalf("expected EVIDENCE_GUARANTEE_LINK_INVALID, got %#v", wrong.Diagnostics)
	}

	// Present and statically valid, but it declares an external source identity:
	// the Compiler cannot prove domain authority, so the Build succeeds and the
	// fact becomes a Host Admission requirement.
	withSource := append([]byte(nil), descriptorRaw...)
	withSource = append(withSource, []byte("  sourceIdentity: {domain: urn:example:authority:one, path: record.source}\n")...)
	trustDeferred := BuildDevelopment(withDescriptor(t, withSource))
	if !trustDeferred.Success {
		t.Fatalf("statically valid descriptor with external trust must build: %#v", trustDeferred.Diagnostics)
	}
	if _, ok := developmentRequirementByKind(t, trustDeferred, DevelopmentRequirementEvidenceDomain, "ue.registration-state"); !ok {
		t.Fatal("missing EVIDENCE_DOMAIN_TRUST Host requirement")
	}

	// A declared fact identity likewise requires an external CommonFact binding.
	withFact := append([]byte(nil), descriptorRaw...)
	withFact = append(withFact, []byte("  factIdentity: {namespace: urn:example:authority:facts, fact: ue.registration, valuePath: record.factValue, scopePaths: {ue: record.scope}}\n")...)
	factDeferred := BuildDevelopment(withDescriptor(t, withFact))
	if !factDeferred.Success {
		t.Fatalf("statically valid descriptor with factIdentity must build: %#v", factDeferred.Diagnostics)
	}
	if _, ok := developmentRequirementByKind(t, factDeferred, DevelopmentRequirementEvidenceFact, "ue.registration-state"); !ok {
		t.Fatal("missing EVIDENCE_FACT_TRUST Host requirement")
	}
}
func TestDevelopmentVerifyRejectsResealedTampering(t *testing.T) {
	request := loadDevelopmentFixture(t)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("fixture build failed: %#v", result.Diagnostics)
	}

	// Delete one host requirement and re-seal every outer digest.
	tampered := tamperDevelopmentArtifact(t, result, func(files map[string][]byte) {
		var requirements []DevelopmentHostRequirement
		if err := json.Unmarshal(files[developmentRequirementsFileName], &requirements); err != nil {
			t.Fatal(err)
		}
		files[developmentRequirementsFileName] = canonicalTestJSON(t, requirements[1:])
	})
	if err := VerifyDevelopment(request, tampered); err == nil {
		t.Fatal("VerifyDevelopment accepted a re-sealed artifact with a deleted host requirement")
	}

	// Edit the compiled Procedure and re-seal every outer digest.
	tampered = tamperDevelopmentArtifact(t, result, func(files map[string][]byte) {
		path := "skills/verify-ue-registration/compiled/procedure.json"
		edited := append([]byte(nil), files[path]...)
		edited[0] = ' '
		files[path] = edited
	})
	if err := VerifyDevelopment(request, tampered); err == nil {
		t.Fatal("VerifyDevelopment accepted a re-sealed artifact with an edited Procedure")
	}

	// Change the fixed execution status.
	statusTampered := result
	statusTampered.OutputFiles = append([]File(nil), result.OutputFiles...)
	statusTampered.ExecutionStatus = "ADMITTED"
	if err := VerifyDevelopment(request, statusTampered); err == nil {
		t.Fatal("VerifyDevelopment accepted a non-NOT_ADMITTED artifact")
	}
}

func TestDevelopmentRequestEnvelopeIsExclusive(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "design", "fixtures", "registration-development-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	// The Development request cannot be decoded by the frozen v1alpha5 entry...
	if _, err := DecodeBuildRequestJSON(raw); err == nil {
		t.Fatal("frozen BuildRequest decoder accepted a Development request")
	}
	// ...and the frozen request cannot be decoded by the Development entry.
	frozenRaw, err := os.ReadFile(filepath.Join("..", "..", "web", "parity-v1alpha5", "trusted-build-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeDevelopmentBuildRequestJSON(frozenRaw); err == nil {
		t.Fatal("Development decoder accepted a frozen v1alpha5 BuildRequest")
	}
	// Unknown fields and duplicate keys are rejected.
	if _, err := DecodeDevelopmentBuildRequestJSON([]byte(`{"apiVersion":"x","apiVersion":"y"}`)); err == nil {
		t.Fatal("duplicate JSON keys were accepted")
	}
}

func assertDevelopmentFilesEqual(t *testing.T, left, right DevelopmentBuildResult) {
	t.Helper()
	if len(left.OutputFiles) != len(right.OutputFiles) {
		t.Fatalf("artifact file counts differ: %d != %d", len(left.OutputFiles), len(right.OutputFiles))
	}
	for index := range left.OutputFiles {
		if left.OutputFiles[index].Path != right.OutputFiles[index].Path {
			t.Fatalf("artifact file order differs at %d", index)
		}
		if !bytes.Equal(left.OutputFiles[index].Bytes, right.OutputFiles[index].Bytes) {
			t.Fatalf("artifact bytes differ: %s", left.OutputFiles[index].Path)
		}
	}
}

func developmentContainsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func developmentReverseFiles(files []File) {
	for left, right := 0, len(files)-1; left < right; left, right = left+1, right-1 {
		files[left], files[right] = files[right], files[left]
	}
}

func reverseSnapshots(snapshots []DevelopmentSnapshot) {
	for left, right := 0, len(snapshots)-1; left < right; left, right = left+1, right-1 {
		snapshots[left], snapshots[right] = snapshots[right], snapshots[left]
	}
}

func reverseRoots(roots []DevelopmentLockRoot) {
	for left, right := 0, len(roots)-1; left < right; left, right = left+1, right-1 {
		roots[left], roots[right] = roots[right], roots[left]
	}
}

func reversePackages(packages []DevelopmentLockPackage) {
	for left, right := 0, len(packages)-1; left < right; left, right = left+1, right-1 {
		packages[left], packages[right] = packages[right], packages[left]
	}
}

func reverseEdges(edges []DevelopmentLockEdge) {
	for left, right := 0, len(edges)-1; left < right; left, right = left+1, right-1 {
		edges[left], edges[right] = edges[right], edges[left]
	}
}

// modifyDevelopmentManifest edits package.json and recomputes the snapshot
// digest without touching the declared files[] list.
func modifyDevelopmentManifest(t *testing.T, request DevelopmentBuildRequest, packageID string, mutate func(manifest map[string]any)) DevelopmentBuildRequest {
	t.Helper()
	request = cloneDevelopmentRequest(t, request)
	snapshotIndex := -1
	for index, snapshot := range request.Snapshots {
		if snapshot.Key.PackageID == packageID {
			snapshotIndex = index
		}
	}
	if snapshotIndex < 0 {
		t.Fatalf("package %s not found", packageID)
	}
	files := request.Snapshots[snapshotIndex].Files
	var manifest map[string]any
	for index := range files {
		if files[index].Path == "package.json" {
			if err := json.Unmarshal(files[index].Bytes, &manifest); err != nil {
				t.Fatal(err)
			}
			mutate(manifest)
			raw, err := json.MarshalIndent(manifest, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			files[index].Bytes = append(raw, '\n')
		}
	}
	digest, err := ComputeDependencySnapshotDigest(files)
	if err != nil {
		t.Fatal(err)
	}
	updateDevelopmentKeyDigest(&request, packageID, digest)
	return request
}

// addDevelopmentExportAndFile adds one file plus an export declaration to a
// package and recomputes the snapshot digest.
func addDevelopmentExportAndFile(t *testing.T, request DevelopmentBuildRequest, packageID, path string, content []byte, export map[string]any) DevelopmentBuildRequest {
	t.Helper()
	request = cloneDevelopmentRequest(t, request)
	snapshotIndex := -1
	for index, snapshot := range request.Snapshots {
		if snapshot.Key.PackageID == packageID {
			snapshotIndex = index
		}
	}
	if snapshotIndex < 0 {
		t.Fatalf("package %s not found", packageID)
	}
	files := request.Snapshots[snapshotIndex].Files
	for index := range files {
		if files[index].Path != "package.json" {
			continue
		}
		var manifest map[string]any
		if err := json.Unmarshal(files[index].Bytes, &manifest); err != nil {
			t.Fatal(err)
		}
		exports, _ := manifest["exports"].([]any)
		manifest["exports"] = append(exports, export)
		sum := sha256.Sum256(content)
		entries, _ := manifest["files"].([]any)
		manifest["files"] = append(entries, map[string]any{"path": path, "sha256": "sha256:" + hex.EncodeToString(sum[:])})
		raw, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		files[index].Bytes = append(raw, '\n')
	}
	files = append(files, File{Path: path, Bytes: content})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	request.Snapshots[snapshotIndex].Files = files
	digest, err := ComputeDependencySnapshotDigest(files)
	if err != nil {
		t.Fatal(err)
	}
	updateDevelopmentKeyDigest(&request, packageID, digest)
	return request
}

func updateDevelopmentKeyDigest(request *DevelopmentBuildRequest, packageID, digest string) {
	for index := range request.Snapshots {
		if request.Snapshots[index].Key.PackageID == packageID {
			request.Snapshots[index].Key.SnapshotDigest = digest
		}
	}
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
}

func TestDevelopmentMutationAndReconcileRequirements(t *testing.T) {
	legacy := recoveryFixture(t, recoveryDescriptor(true))
	request := developmentRequestFromLegacy(t, legacy)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("Mutation Development build failed: %#v", result.Diagnostics)
	}
	commit, ok := developmentRequirementByKind(t, result, DevelopmentRequirementCommitProof, "example.session.mutate")
	if !ok {
		t.Fatal("missing COMMIT_PROOF Host requirement for the Mutation callable")
	}
	if commit.Status != DevelopmentHostRequirementStatus {
		t.Fatalf("COMMIT_PROOF status = %q", commit.Status)
	}
	reconcile, ok := developmentRequirementByKind(t, result, DevelopmentRequirementReconcile, "example.session.reconcile")
	if !ok {
		t.Fatal("missing RECONCILE Host requirement for the Reconcile READ callable")
	}
	if reconcile.Status != DevelopmentHostRequirementStatus {
		t.Fatalf("RECONCILE status = %q", reconcile.Status)
	}
	// The dual-producer Evidence Descriptor is static input; it must be fully
	// checked and must not become an external requirement.
	if _, ok := developmentRequirementByKind(t, result, DevelopmentRequirementEvidenceDomain, "example.session-state"); ok {
		t.Fatal("a static dual-producer Evidence Descriptor must not create an external Evidence requirement")
	}
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("VerifyDevelopment: %v", err)
	}
}

func TestDevelopmentWaitIngressFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "web", "parity-v1alpha5", "trusted-build-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := DecodeBuildRequestJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	request := developmentRequestFromLegacy(t, legacy)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("Wait/Ingress Development build failed: %#v", result.Diagnostics)
	}
	links := developmentOutputFile(t, result, "skills/"+requestSkillName(t, request)+"/compiled/ingress-links.json")
	var ingress struct {
		Links []struct {
			NodeID  string `json:"nodeId"`
			Ingress struct {
				DependencyAlias string `json:"dependencyAlias"`
				OperationID     string `json:"operationId"`
			} `json:"ingress"`
		} `json:"links"`
	}
	if err := json.Unmarshal(links, &ingress); err != nil {
		t.Fatal(err)
	}
	if len(ingress.Links) != 1 || ingress.Links[0].Ingress.OperationID != "nsmf-pdusession.update-sm-context" {
		t.Fatalf("Development ingress links = %#v", ingress.Links)
	}
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("VerifyDevelopment: %v", err)
	}
}

func requestSkillName(t *testing.T, request DevelopmentBuildRequest) string {
	t.Helper()
	for _, file := range request.Files {
		parts := strings.Split(file.Path, "/")
		if len(parts) >= 2 && parts[0] == "skills" {
			return parts[1]
		}
	}
	t.Fatal("request has no Skill package")
	return ""
}


