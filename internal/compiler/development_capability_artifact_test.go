package compiler

// Development Artifact capability-link tests.
//
// These tests prove that the Capability a Skill statically selects is delivered
// to the Host as an exact Artifact fact: the CapabilityCatalog Package identity,
// Owner, export, SourcePath, Callable members and the Skill declaration position
// all travel with the Artifact, and Verify rejects every attempted rewrite even
// after the tamperer re-seals the outer digests.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func developmentManifestOf(t *testing.T, result DevelopmentBuildResult) developmentArtifactManifest {
	t.Helper()
	var manifest developmentArtifactManifest
	if err := json.Unmarshal(developmentOutputFile(t, result, developmentManifestFileName), &manifest); err != nil {
		t.Fatalf("decode Development manifest: %v", err)
	}
	return manifest
}

func developmentSnapshotKey(t *testing.T, request DevelopmentBuildRequest, packageID string) DevelopmentPackageKey {
	t.Helper()
	for _, snapshot := range request.Snapshots {
		if snapshot.Key.PackageID == packageID {
			return snapshot.Key
		}
	}
	t.Fatalf("request has no package %s", packageID)
	return DevelopmentPackageKey{}
}

// TestDevelopmentCapabilityGoldenBaseline pins the exact canonical capability
// requirement of the released Development fixture. Any field, identity or order
// change is a baseline diff, not a silent artifact change.
func TestDevelopmentCapabilityGoldenBaseline(t *testing.T) {
	result := BuildDevelopment(loadDevelopmentFixture(t))
	if !result.Success {
		t.Fatalf("fixture build failed: %#v", result.Diagnostics)
	}
	manifest := developmentManifestOf(t, result)
	raw, err := canonicalizeValue(map[string]any{
		"apiVersion":             DevelopmentArtifactAPIVersion,
		"capabilityRequirements": manifest.CapabilityRequirements,
	})
	if err != nil {
		t.Fatalf("canonicalize capability baseline: %v", err)
	}
	golden, err := os.ReadFile(filepath.Join("..", "..", "tests", "golden", "development-capability-requirements-v1.json"))
	if err != nil {
		t.Fatalf("read capability golden baseline: %v", err)
	}
	if string(raw)+"\n" != string(golden) {
		t.Fatalf("capability requirement differs from the golden baseline:\n got: %s\nwant: %s", raw, golden)
	}
}

// TestDevelopmentCapabilityRequirementInArtifact proves the positive case: the
// Skill declares exactly one Capability and the final manifest carries its exact
// static requirement, so the Host never reads the SKILL source to recover it.
func TestDevelopmentCapabilityRequirementInArtifact(t *testing.T) {
	request := loadDevelopmentFixture(t)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("fixture build failed: %#v", result.Diagnostics)
	}
	manifest := developmentManifestOf(t, result)
	if len(manifest.CapabilityRequirements) != 1 {
		t.Fatalf("manifest capability requirements = %#v, want exactly one", manifest.CapabilityRequirements)
	}
	requirement := manifest.CapabilityRequirements[0]
	toolsKey := developmentSnapshotKey(t, request, "urn:example:ca:tools")
	if requirement.Skill != "verify-ue-registration" {
		t.Fatalf("capability requirement skill = %q", requirement.Skill)
	}
	if requirement.CapabilityID != "example.registration-read" {
		t.Fatalf("capability requirement id = %q, want the declared Capability", requirement.CapabilityID)
	}
	if requirement.Catalog.Key != toolsKey {
		t.Fatalf("capability Catalog key = %#v, want %#v", requirement.Catalog.Key, toolsKey)
	}
	if requirement.Catalog.Owner != "example-ue-owner" {
		t.Fatalf("capability Catalog owner = %q, want the locked Package publisher", requirement.Catalog.Owner)
	}
	if requirement.Catalog.ExportID != "example.registration-read" || requirement.Catalog.SourcePath != "capabilities.json" {
		t.Fatalf("capability Catalog export = %#v", requirement.Catalog)
	}
	if requirement.Catalog.Pointer != "/capabilities/0" {
		t.Fatalf("capability Catalog pointer = %q", requirement.Catalog.Pointer)
	}
	if requirement.Declaration.Path != "skills/verify-ue-registration/assurance/contract.yaml" ||
		requirement.Declaration.Pointer != "/capabilities/0" {
		t.Fatalf("capability declaration = %#v", requirement.Declaration)
	}
	if len(requirement.Callables) != 1 {
		t.Fatalf("capability Callable members = %#v", requirement.Callables)
	}
	callable := requirement.Callables[0]
	if callable.Key != toolsKey || callable.Owner != "example-ue-owner" ||
		callable.ExportID != "ue.registration.current" || callable.Kind != "observation" ||
		callable.Effect != "READ" || callable.SourcePath != "tool-catalog.yaml" {
		t.Fatalf("capability Callable member = %#v", callable)
	}
	// The Catalog Owner is a formal external fact: Build records it and never
	// claims to have verified it. The Artifact remains NOT_ADMITTED.
	owner, ok := developmentRequirementByKind(t, result, DevelopmentRequirementOwnerTrust, "example.registration-read")
	if !ok {
		t.Fatal("missing OWNER_TRUST Host requirement for the selected CapabilityCatalog")
	}
	if owner.Status != DevelopmentHostRequirementStatus {
		t.Fatalf("CapabilityCatalog OWNER_TRUST status = %q", owner.Status)
	}
	if owner.Location.Stage != "capabilities" || owner.Location.Node != "example.registration-read" ||
		owner.Location.Selector != "/capabilities/0" {
		t.Fatalf("CapabilityCatalog OWNER_TRUST location = %#v", owner.Location)
	}
	if len(owner.Subjects) != 1 || owner.Subjects[0].Key != toolsKey || owner.Subjects[0].ID != "example.registration-read" ||
		owner.Subjects[0].SourcePath != "capabilities.json" {
		t.Fatalf("CapabilityCatalog OWNER_TRUST subjects = %#v", owner.Subjects)
	}
	if manifest.ExecutionStatus != DevelopmentExecutionStatus {
		t.Fatalf("manifest executionStatus = %q", manifest.ExecutionStatus)
	}
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("VerifyDevelopment: %v", err)
	}
}

// TestDevelopmentCapabilitySelectionIsExact proves the central semantic: two
// Capabilities may contain the same Callable, but only the one the Skill
// declares becomes the static requirement. The Artifact never infers the
// Capability from the Callable set and never promotes every matching Capability.
func TestDevelopmentCapabilitySelectionIsExact(t *testing.T) {
	request := loadDevelopmentFixture(t)
	adminCatalog := []byte(`{
  "capabilities": [
    {
      "id": "example.registration-admin",
      "members": [
        {
          "callableId": "ue.registration.current"
        }
      ]
    }
  ]
}
`)
	request = addDevelopmentExportAndFile(t, request, "urn:example:ca:tools", "capabilities-admin.json", adminCatalog,
		map[string]any{"kind": "CapabilityCatalog", "id": "example.registration-admin", "path": "capabilities-admin.json"})
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("two-Capability build failed: %#v", result.Diagnostics)
	}
	manifest := developmentManifestOf(t, result)
	if len(manifest.CapabilityRequirements) != 1 {
		t.Fatalf("manifest capability requirements = %#v, want exactly the declared one", manifest.CapabilityRequirements)
	}
	if manifest.CapabilityRequirements[0].CapabilityID != "example.registration-read" {
		t.Fatalf("manifest selected Capability = %q, want example.registration-read", manifest.CapabilityRequirements[0].CapabilityID)
	}
	// The undeclared Capability must not appear as an Owner Trust requirement
	// either: it is a static selection, not a discovered Callable property.
	if _, ok := developmentRequirementByKind(t, result, DevelopmentRequirementOwnerTrust, "example.registration-admin"); ok {
		t.Fatal("an undeclared Capability became a Host Admission requirement")
	}
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("VerifyDevelopment: %v", err)
	}
}

// TestDevelopmentCapabilityDeclarationIsRequired proves that a declared
// Capability without a locked CapabilityCatalog is a static failure rather than
// a silently dropped requirement.
func TestDevelopmentCapabilityDeclarationIsRequired(t *testing.T) {
	request := loadDevelopmentFixture(t)
	contract := developmentSkillFile(t, request, "skills/verify-ue-registration/assurance/contract.yaml")
	updated := strings.Replace(string(contract), "- example.registration-read", "- example.registration-missing", 1)
	request = replaceDevelopmentSkillFileBytes(t, request, "skills/verify-ue-registration/assurance/contract.yaml", []byte(updated))
	result := BuildDevelopment(request)
	if result.Success {
		t.Fatal("a declared Capability with no locked CapabilityCatalog must fail the Build")
	}
	if !hasDevelopmentCode(result.Diagnostics, "CAPABILITY_MAPPING_MISSING") {
		t.Fatalf("diagnostics = %#v, want CAPABILITY_MAPPING_MISSING", result.Diagnostics)
	}
}

func developmentSkillFile(t *testing.T, request DevelopmentBuildRequest, path string) []byte {
	t.Helper()
	for _, file := range request.Files {
		if file.Path == path {
			return append([]byte(nil), file.Bytes...)
		}
	}
	t.Fatalf("request has no Skill file %s", path)
	return nil
}

func replaceDevelopmentSkillFileBytes(t *testing.T, request DevelopmentBuildRequest, path string, content []byte) DevelopmentBuildRequest {
	t.Helper()
	request = cloneDevelopmentRequest(t, request)
	for index := range request.Files {
		if request.Files[index].Path == path {
			request.Files[index].Bytes = content
			return request
		}
	}
	t.Fatalf("request has no Skill file %s", path)
	return request
}

// TestDevelopmentVerifyRejectsCapabilityTampering proves every capability field
// is part of Artifact integrity. Each case mutates the manifest and re-seals the
// file digests and artifactDigest, so only semantic verification can reject it.
func TestDevelopmentVerifyRejectsCapabilityTampering(t *testing.T) {
	request := loadDevelopmentFixture(t)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("fixture build failed: %#v", result.Diagnostics)
	}
	cases := []struct {
		name   string
		mutate func(manifest *developmentArtifactManifest)
	}{
		{
			name: "delete capability requirement",
			mutate: func(manifest *developmentArtifactManifest) {
				manifest.CapabilityRequirements = nil
			},
		},
		{
			name: "replace capability id",
			mutate: func(manifest *developmentArtifactManifest) {
				manifest.CapabilityRequirements[0].CapabilityID = "example.registration-admin"
				manifest.CapabilityRequirements[0].Catalog.ExportID = "example.registration-admin"
			},
		},
		{
			name: "replace catalog package key",
			mutate: func(manifest *developmentArtifactManifest) {
				manifest.CapabilityRequirements[0].Catalog.Key.SnapshotDigest = "sha256:" + strings.Repeat("0", 64)
			},
		},
		{
			name: "replace catalog owner",
			mutate: func(manifest *developmentArtifactManifest) {
				manifest.CapabilityRequirements[0].Catalog.Owner = "attacker-owner"
			},
		},
		{
			name: "replace catalog export id",
			mutate: func(manifest *developmentArtifactManifest) {
				manifest.CapabilityRequirements[0].Catalog.ExportID = "example.registration-other"
			},
		},
		{
			name: "replace catalog source path",
			mutate: func(manifest *developmentArtifactManifest) {
				manifest.CapabilityRequirements[0].Catalog.SourcePath = "other.json"
			},
		},
		{
			name: "replace catalog pointer",
			mutate: func(manifest *developmentArtifactManifest) {
				manifest.CapabilityRequirements[0].Catalog.Pointer = "/capabilities/7"
			},
		},
		{
			name: "delete callable member",
			mutate: func(manifest *developmentArtifactManifest) {
				manifest.CapabilityRequirements[0].Callables = nil
			},
		},
		{
			name: "replace callable export id",
			mutate: func(manifest *developmentArtifactManifest) {
				manifest.CapabilityRequirements[0].Callables[0].ExportID = "ue.registration.other"
			},
		},
		{
			name: "replace callable key",
			mutate: func(manifest *developmentArtifactManifest) {
				manifest.CapabilityRequirements[0].Callables[0].Key.PackageID = "urn:example:attacker"
			},
		},
		{
			name: "replace declaration pointer",
			mutate: func(manifest *developmentArtifactManifest) {
				manifest.CapabilityRequirements[0].Declaration.Pointer = "/capabilities/1"
			},
		},
		{
			name: "replace declaration path",
			mutate: func(manifest *developmentArtifactManifest) {
				manifest.CapabilityRequirements[0].Declaration.Path = "skills/verify-ue-registration/SKILL.md"
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			tampered := tamperDevelopmentArtifact(t, result, func(files map[string][]byte) {
				var manifest developmentArtifactManifest
				if err := json.Unmarshal(files[developmentManifestFileName], &manifest); err != nil {
					t.Fatal(err)
				}
				testCase.mutate(&manifest)
				files[developmentManifestFileName] = canonicalTestJSON(t, manifest)
			})
			if err := VerifyDevelopment(request, tampered); err == nil {
				t.Fatal("VerifyDevelopment accepted a re-sealed Artifact with a tampered capability requirement")
			}
		})
	}
}
