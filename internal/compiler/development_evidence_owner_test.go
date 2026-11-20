package compiler

// Development Build Evidence Descriptor authority tests.
//
// An EvidenceDescriptor is a formal Interface Package export. Its declared
// `authority` is the Owner identity, so every consumed Descriptor must produce
// exactly one OWNER_TRUST Host Admission requirement at the real Predicate
// Evidence position. EVIDENCE_DOMAIN_TRUST and EVIDENCE_FACT_TRUST authorize a
// different external fact and never replace the Owner identity requirement.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	developmentDescriptorID       = "ue.registration-state"
	developmentPredicateID        = "example.ue.registered"
	developmentPredicatePath      = "skills/verify-ue-registration/assurance/predicates.yaml"
	developmentProcedurePath      = "skills/verify-ue-registration/assurance/procedure.yaml"
	developmentToolsPackageID     = "urn:example:ca:tools"
	developmentDescriptorFilePath = "evidence-guarantees.yaml"
)

func developmentSourceBytes(t *testing.T, request DevelopmentBuildRequest, path string) []byte {
	t.Helper()
	for _, file := range request.Files {
		if file.Path == path {
			return append([]byte(nil), file.Bytes...)
		}
	}
	t.Fatalf("source file %s not found in fixture", path)
	return nil
}

func developmentSnapshotBytes(t *testing.T, request DevelopmentBuildRequest, packageID, path string) []byte {
	t.Helper()
	for _, snapshot := range request.Snapshots {
		if snapshot.Key.PackageID != packageID {
			continue
		}
		for _, file := range snapshot.Files {
			if file.Path == path {
				return append([]byte(nil), file.Bytes...)
			}
		}
	}
	t.Fatalf("snapshot file %s/%s not found in fixture", packageID, path)
	return nil
}

func modifyDevelopmentSourceFile(t *testing.T, request DevelopmentBuildRequest, path string, content []byte) DevelopmentBuildRequest {
	t.Helper()
	request = cloneDevelopmentRequest(t, request)
	for index := range request.Files {
		if request.Files[index].Path == path {
			request.Files[index].Bytes = content
			return request
		}
	}
	t.Fatalf("source file %s not found in fixture", path)
	return request
}

// developmentDescriptorOwnerRequirements returns every OWNER_TRUST requirement
// whose Subject is the released Evidence Descriptor.
func developmentDescriptorOwnerRequirements(t *testing.T, result DevelopmentBuildResult) []DevelopmentHostRequirement {
	t.Helper()
	matched := make([]DevelopmentHostRequirement, 0)
	for _, requirement := range developmentRequirements(t, result, developmentRequirementsFileName) {
		if requirement.Kind != DevelopmentRequirementOwnerTrust {
			continue
		}
		for _, subject := range requirement.Subjects {
			if subject.ID == developmentDescriptorID {
				matched = append(matched, requirement)
			}
		}
	}
	return matched
}

// TestDevelopmentEvidenceDescriptorOwnerTrustIsExact proves that a consumed
// EvidenceDescriptor produces its own exactly located OWNER_TRUST requirement
// with the declared package identity, physical snapshot path and real Predicate
// Evidence JSON Pointer.
func TestDevelopmentEvidenceDescriptorOwnerTrustIsExact(t *testing.T) {
	request := loadDevelopmentFixture(t)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("fixture build failed: %#v", result.Diagnostics)
	}
	requirements := developmentDescriptorOwnerRequirements(t, result)
	if len(requirements) != 1 {
		t.Fatalf("Evidence Descriptor OWNER_TRUST count = %d, want 1: %#v", len(requirements), requirements)
	}
	requirement := requirements[0]
	if requirement.Status != DevelopmentHostRequirementStatus {
		t.Fatalf("Descriptor OWNER_TRUST status = %q", requirement.Status)
	}
	if len(requirement.Subjects) != 1 {
		t.Fatalf("Descriptor OWNER_TRUST subjects = %#v", requirement.Subjects)
	}
	subject := requirement.Subjects[0]
	if subject.Key.PackageID != developmentToolsPackageID || subject.Key.Version == "" || subject.Key.SnapshotDigest == "" {
		t.Fatalf("Descriptor OWNER_TRUST subject key = %#v", subject.Key)
	}
	if subject.ID != developmentDescriptorID {
		t.Fatalf("Descriptor OWNER_TRUST subject id = %q", subject.ID)
	}
	if subject.SourcePath != developmentDescriptorFilePath {
		t.Fatalf("Descriptor OWNER_TRUST sourcePath = %q, want the locked Descriptor path", subject.SourcePath)
	}
	if len(subject.SchemaIDs) != 0 {
		t.Fatalf("Descriptor OWNER_TRUST must not re-declare Callable Schemas: %#v", subject.SchemaIDs)
	}
	location := requirement.Location
	if location.Skill != "verify-ue-registration" || location.Path != developmentPredicatePath ||
		location.Node != developmentPredicateID || location.Stage != "evidence" {
		t.Fatalf("Descriptor OWNER_TRUST location = %#v", location)
	}
	if want := "/predicates/" + developmentPredicateID + "/evidence/requires/registration/0"; location.Selector != want {
		t.Fatalf("Descriptor OWNER_TRUST selector = %q, want %q", location.Selector, want)
	}
	// Owner authority is a distinct external fact: without a declared source or
	// fact identity the Descriptor produces no Domain/Fact requirement, and the
	// Owner requirement never substitutes for them.
	for _, kind := range []string{DevelopmentRequirementEvidenceDomain, DevelopmentRequirementEvidenceFact} {
		if _, ok := developmentRequirementByKind(t, result, kind, developmentDescriptorID); ok {
			t.Fatalf("Descriptor without external identity produced %s", kind)
		}
	}
	if result.ExecutionStatus != DevelopmentExecutionStatus {
		t.Fatalf("executionStatus = %q", result.ExecutionStatus)
	}
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("VerifyDevelopment: %v", err)
	}
}

// TestDevelopmentEvidenceDescriptorOwnerTrustPerReferencePosition proves that a
// Descriptor referenced from several Predicates and several list positions
// yields one precisely located OWNER_TRUST per real position instead of one
// collapsed requirement.
func TestDevelopmentEvidenceDescriptorOwnerTrustPerReferencePosition(t *testing.T) {
	request := loadDevelopmentFixture(t)

	predicates := developmentSourceBytes(t, request, developmentPredicatePath)
	predicates = append(predicates, []byte(`  example.ue.registered.secondary:
    parameters:
      - registration: ca::urn:example:ca:registration:v1
    evidence:
      requires:
        registration: [ca::ue.registration-state, ca::ue.registration-state]
    evaluator:
      type: cel
      expression: registration.state == "REGISTERED"
`)...)
	request = modifyDevelopmentSourceFile(t, request, developmentPredicatePath, predicates)

	procedure := developmentSourceBytes(t, request, developmentProcedurePath)
	call := []byte("          arguments:\n            - inspect-registration.observation.output\n")
	extra := []byte("          arguments:\n            - inspect-registration.observation.output\n        - predicate: example.ue.registered.secondary\n          arguments:\n            - inspect-registration.observation.output\n")
	if !bytes.Contains(procedure, call) {
		t.Fatalf("procedure fixture does not contain the expected postcondition call")
	}
	procedure = bytes.Replace(procedure, call, extra, 1)
	request = modifyDevelopmentSourceFile(t, request, developmentProcedurePath, procedure)

	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("multi-reference build failed: %#v", result.Diagnostics)
	}
	requirements := developmentDescriptorOwnerRequirements(t, result)
	if len(requirements) != 3 {
		t.Fatalf("Descriptor OWNER_TRUST count = %d, want 3: %#v", len(requirements), requirements)
	}
	got := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		got = append(got, requirement.Location.Node+"|"+requirement.Location.Selector)
	}
	sort.Strings(got)
	want := []string{
		"example.ue.registered|/predicates/example.ue.registered/evidence/requires/registration/0",
		"example.ue.registered.secondary|/predicates/example.ue.registered.secondary/evidence/requires/registration/0",
		"example.ue.registered.secondary|/predicates/example.ue.registered.secondary/evidence/requires/registration/1",
	}
	sort.Strings(want)
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("Descriptor OWNER_TRUST positions = %#v, want %#v", got, want)
		}
	}
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("VerifyDevelopment: %v", err)
	}
}

// developmentMultiProducerRequest adds a second Observation that publishes the
// same guarantee through a different output Schema.
func developmentMultiProducerRequest(t *testing.T) DevelopmentBuildRequest {
	t.Helper()
	request := loadDevelopmentFixture(t)

	altSchema := []byte(`$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:ca:registration-alt:v1
type: object
additionalProperties: false
required: [state]
properties:
  state:
    type: string
`)
	request = addDevelopmentExportAndFile(t, request, developmentToolsPackageID, "registration-alt.schema.yaml", altSchema,
		map[string]any{"kind": "Schema", "id": "urn:example:ca:registration-alt:v1", "path": "registration-alt.schema.yaml"})

	catalog := developmentSnapshotBytes(t, request, developmentToolsPackageID, "tool-catalog.yaml")
	catalog = append(catalog, []byte(`    - kind: observation
      id: ue.registration.alt
      effect: READ
      ownerOutcomes: [SUCCEEDED, FAILED]
      adapter: example-ue-query-alt
      queryContextSchema:
        $ref: urn:example:ca:registration-query:v1
      outputSchema:
        $ref: urn:example:ca:registration-alt:v1
      guarantees:
        evidence:
          provides:
            - guarantee: ue.registration-state
              path: output
`)...)
	request = modifyDevelopmentPackageFile(t, request, developmentToolsPackageID, "tool-catalog.yaml", catalog)
	request = modifyDevelopmentManifest(t, request, developmentToolsPackageID, func(manifest map[string]any) {
		exports, _ := manifest["exports"].([]any)
		manifest["exports"] = append(exports, map[string]any{
			"kind": "CallableCatalog", "id": "ue.registration.alt", "path": "tool-catalog.yaml",
		})
	})

	descriptor := developmentSnapshotBytes(t, request, developmentToolsPackageID, developmentDescriptorFilePath)
	descriptor = bytes.Replace(descriptor, []byte("  scope:\n"),
		[]byte("    - owner: example-ue-owner\n      kind: observation\n      id: ue.registration.alt\n      outcome: SUCCEEDED\n      path: output\n  scope:\n"), 1)
	return modifyDevelopmentPackageFile(t, request, developmentToolsPackageID, developmentDescriptorFilePath, descriptor)
}

// TestDevelopmentEvidenceMultipleProducersAndSchemas proves that one guarantee
// may be published by several Producers, that every Producer is matched to its
// own Callable and output Schema, and that the Descriptor still yields exactly
// one authority requirement.
func TestDevelopmentEvidenceMultipleProducersAndSchemas(t *testing.T) {
	request := developmentMultiProducerRequest(t)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("multi-producer build failed: %#v", result.Diagnostics)
	}
	requirements := developmentDescriptorOwnerRequirements(t, result)
	if len(requirements) != 1 {
		t.Fatalf("Descriptor OWNER_TRUST count = %d, want 1", len(requirements))
	}
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("VerifyDevelopment: %v", err)
	}
}

// TestDevelopmentEvidenceProducerSchemaAssociation proves the Compiler resolves
// the Schema of the Producer that actually supplies the argument instead of a
// single descriptor-level Schema: switching the producing Callable to an
// incompatible output Schema fails the static argument typing.
func TestDevelopmentEvidenceProducerSchemaAssociation(t *testing.T) {
	request := developmentMultiProducerRequest(t)
	procedure := developmentSourceBytes(t, request, developmentProcedurePath)
	procedure = bytes.Replace(procedure,
		[]byte("tool: ca::ue.registration.current"), []byte("tool: ca::ue.registration.alt"), 1)
	request = modifyDevelopmentSourceFile(t, request, developmentProcedurePath, procedure)

	result := BuildDevelopment(request)
	if result.Success {
		t.Fatal("an incompatible Producer output Schema must fail the Build")
	}
	if !hasDevelopmentCode(result.Diagnostics, "PREDICATE_TYPE_MISMATCH") {
		t.Fatalf("expected PREDICATE_TYPE_MISMATCH, got %#v", result.Diagnostics)
	}
	if len(result.OutputFiles) != 0 || result.ArtifactDigest != "" {
		t.Fatal("failed build produced artifact bytes")
	}
}

// TestDevelopmentEvidenceDescriptorNegativeMatrix proves that every static
// Descriptor contract violation is a Build failure, never a deferred
// requirement: missing/duplicate/illegal Descriptor, unknown Producer, wrong
// Owner, Outcome or provides.path, undeclared dependency, incompatible Producer
// Schema.
func TestDevelopmentEvidenceDescriptorNegativeMatrix(t *testing.T) {
	descriptor := func(t *testing.T, request DevelopmentBuildRequest) []byte {
		return developmentSnapshotBytes(t, request, developmentToolsPackageID, developmentDescriptorFilePath)
	}
	catalog := func(t *testing.T, request DevelopmentBuildRequest) []byte {
		return developmentSnapshotBytes(t, request, developmentToolsPackageID, "tool-catalog.yaml")
	}
	cases := []struct {
		name     string
		mutate   func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest
		wantCode string
	}{
		{
			name: "duplicate descriptor",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				raw := descriptor(t, request)
				raw = append(raw, []byte("ue.registration-state:\n  authority: example-ue-owner\n")...)
				return modifyDevelopmentPackageFile(t, request, developmentToolsPackageID, developmentDescriptorFilePath, raw)
			},
			wantCode: developmentSnapshotInvalid,
		},
		{
			name: "malformed descriptor document",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				raw := []byte("ue.registration-state: 5\n")
				return modifyDevelopmentPackageFile(t, request, developmentToolsPackageID, developmentDescriptorFilePath, raw)
			},
			wantCode: developmentSnapshotInvalid,
		},
		{
			name: "missing producer",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				raw := bytes.Replace(descriptor(t, request), []byte("id: ue.registration.current"), []byte("id: ue.registration.missing"), 1)
				return modifyDevelopmentPackageFile(t, request, developmentToolsPackageID, developmentDescriptorFilePath, raw)
			},
			wantCode: "EVIDENCE_GUARANTEE_LINK_INVALID",
		},
		{
			name: "wrong producer owner",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				raw := bytes.Replace(descriptor(t, request), []byte("owner: example-ue-owner"), []byte("owner: example-other-owner"), 1)
				return modifyDevelopmentPackageFile(t, request, developmentToolsPackageID, developmentDescriptorFilePath, raw)
			},
			wantCode: "EVIDENCE_GUARANTEE_LINK_INVALID",
		},
		{
			name: "wrong descriptor authority",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				raw := bytes.Replace(descriptor(t, request), []byte("authority: example-ue-owner"), []byte("authority: example-other-owner"), 1)
				return modifyDevelopmentPackageFile(t, request, developmentToolsPackageID, developmentDescriptorFilePath, raw)
			},
			wantCode: "DEPENDENCY_OWNER_MISMATCH",
		},
		{
			name: "unsupported producer outcome",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				raw := bytes.Replace(descriptor(t, request), []byte("outcome: SUCCEEDED"), []byte("outcome: FAILED"), 1)
				return modifyDevelopmentPackageFile(t, request, developmentToolsPackageID, developmentDescriptorFilePath, raw)
			},
			wantCode: "EVIDENCE_GUARANTEE_LINK_INVALID",
		},
		{
			name: "provides path mismatch",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				raw := bytes.Replace(catalog(t, request), []byte("path: output"), []byte("path: other"), 1)
				return modifyDevelopmentPackageFile(t, request, developmentToolsPackageID, "tool-catalog.yaml", raw)
			},
			wantCode: "EVIDENCE_GUARANTEE_LINK_INVALID",
		},
		{
			name: "undeclared dependency alias",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				predicates := developmentSourceBytes(t, request, developmentPredicatePath)
				predicates = bytes.Replace(predicates, []byte("ca::ue.registration-state"), []byte("zz::ue.registration-state"), 1)
				return modifyDevelopmentSourceFile(t, request, developmentPredicatePath, predicates)
			},
			wantCode: "EVIDENCE_GUARANTEE_LINK_INVALID",
		},
		{
			name: "incompatible producer output schema",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				raw := bytes.Replace(catalog(t, request),
					[]byte("$ref: urn:example:ca:registration:v1"), []byte("$ref: urn:example:ca:registration-ue-reference:v1"), 1)
				return modifyDevelopmentPackageFile(t, request, developmentToolsPackageID, "tool-catalog.yaml", raw)
			},
			wantCode: "PREDICATE_TYPE_MISMATCH",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := testCase.mutate(t, loadDevelopmentFixture(t))
			result := BuildDevelopment(request)
			if result.Success {
				t.Fatalf("case %q unexpectedly succeeded", testCase.name)
			}
			if len(result.OutputFiles) != 0 || result.ArtifactDigest != "" {
				t.Fatal("failed build produced artifact bytes")
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

// TestDevelopmentVerifyRejectsDeletedDescriptorOwnerTrust proves that deleting
// the Descriptor authority requirement and re-sealing every outer digest is
// still rejected: Verify re-derives the requirement set from the original
// request instead of trusting the artifact's own digest.
func TestDevelopmentVerifyRejectsDeletedDescriptorOwnerTrust(t *testing.T) {
	request := loadDevelopmentFixture(t)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("fixture build failed: %#v", result.Diagnostics)
	}
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("untampered artifact must verify: %v", err)
	}
	tampered := tamperDevelopmentArtifact(t, result, func(files map[string][]byte) {
		var requirements []DevelopmentHostRequirement
		if err := json.Unmarshal(files[developmentRequirementsFileName], &requirements); err != nil {
			t.Fatal(err)
		}
		kept := make([]DevelopmentHostRequirement, 0, len(requirements))
		for _, requirement := range requirements {
			if requirement.Kind == DevelopmentRequirementOwnerTrust && len(requirement.Subjects) == 1 &&
				requirement.Subjects[0].ID == developmentDescriptorID {
				continue
			}
			kept = append(kept, requirement)
		}
		if len(kept) == len(requirements) {
			t.Fatal("tamper target OWNER_TRUST requirement not found")
		}
		files[developmentRequirementsFileName] = canonicalTestJSON(t, kept)
	})
	if err := VerifyDevelopment(request, tampered); err == nil {
		t.Fatal("VerifyDevelopment accepted a re-sealed artifact with a deleted Descriptor OWNER_TRUST")
	}
}

// TestDevelopmentVerifyRejectsCoarseDescriptorOwnerTrust proves that a
// Descriptor OWNER_TRUST must keep its real Predicate Evidence JSON Pointer: a
// coarse /evidence block selector is rejected even after the requirement ID and
// every outer digest are recomputed.
func TestDevelopmentVerifyRejectsCoarseDescriptorOwnerTrust(t *testing.T) {
	request := loadDevelopmentFixture(t)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("fixture build failed: %#v", result.Diagnostics)
	}
	tampered := tamperDevelopmentArtifact(t, result, func(files map[string][]byte) {
		var requirements []DevelopmentHostRequirement
		if err := json.Unmarshal(files[developmentRequirementsFileName], &requirements); err != nil {
			t.Fatal(err)
		}
		found := false
		for index := range requirements {
			requirement := &requirements[index]
			if requirement.Kind != DevelopmentRequirementOwnerTrust || len(requirement.Subjects) != 1 ||
				requirement.Subjects[0].ID != developmentDescriptorID {
				continue
			}
			requirement.Location.Selector = "/predicates/" + developmentPredicateID + "/evidence"
			found = true
		}
		if !found {
			t.Fatal("tamper target OWNER_TRUST requirement not found")
		}
		files[developmentRequirementsFileName] = canonicalTestJSON(t, resealRequirements(t, requirements))
	})
	if err := VerifyDevelopment(request, tampered); err == nil {
		t.Fatal("VerifyDevelopment accepted a coarse Descriptor OWNER_TRUST selector")
	}
}

// TestDevelopmentEvidenceDescriptorOwnerTrustIsNotReplacedByDomainOrFact proves
// that a Descriptor declaring sourceIdentity and factIdentity keeps three
// independent Host Admission facts: the declared Owner identity, the external
// source domain authority and the CommonFact authorization.
func TestDevelopmentEvidenceDescriptorOwnerTrustIsNotReplacedByDomainOrFact(t *testing.T) {
	request := loadDevelopmentFixtureWithDescriptor(t, true, true)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("descriptor with both identities failed: %#v", result.Diagnostics)
	}
	owners := developmentDescriptorOwnerRequirements(t, result)
	if len(owners) != 1 {
		t.Fatalf("Descriptor OWNER_TRUST count = %d, want 1", len(owners))
	}
	domain, hasDomain := developmentRequirementByKind(t, result, DevelopmentRequirementEvidenceDomain, developmentDescriptorID)
	fact, hasFact := developmentRequirementByKind(t, result, DevelopmentRequirementEvidenceFact, developmentDescriptorID)
	if !hasDomain || !hasFact {
		t.Fatalf("source/fact identities must produce both requirements: domain=%v fact=%v", hasDomain, hasFact)
	}
	ids := map[string]bool{owners[0].RequirementID: true, domain.RequirementID: true, fact.RequirementID: true}
	if len(ids) != 3 {
		t.Fatalf("Owner, Domain and Fact requirements must be distinct: %#v", ids)
	}
	for _, requirement := range []DevelopmentHostRequirement{owners[0], domain, fact} {
		if requirement.Location.Selector == "" || requirement.Location.Stage != "evidence" {
			t.Fatalf("requirement is not at an exact Evidence position: %#v", requirement.Location)
		}
	}
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("VerifyDevelopment: %v", err)
	}
}

// TestDevelopmentBuildNeedsNoRuntimeEvidenceOrAttestation proves that a static
// Development Build needs neither real Runtime Evidence nor an Assurance
// Attestation: the request envelope cannot carry them, the Build succeeds, and
// the artifact stays NOT_ADMITTED.
func TestDevelopmentBuildNeedsNoRuntimeEvidenceOrAttestation(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "design", "fixtures", "registration-development-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"assurance", "attestation", "attestations", "runtimeEvidence", "evidenceTrustBindings", "deploymentGrant"} {
		if _, exists := envelope[forbidden]; exists {
			t.Fatalf("Development BuildRequest must not carry %q", forbidden)
		}
	}
	request := loadDevelopmentFixture(t)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("static Development build failed: %#v", result.Diagnostics)
	}
	if result.ExecutionStatus != DevelopmentExecutionStatus {
		t.Fatalf("executionStatus = %q, want %q", result.ExecutionStatus, DevelopmentExecutionStatus)
	}
	for _, file := range result.OutputFiles {
		lower := strings.ToLower(file.Path)
		if strings.Contains(lower, "attest") && !strings.Contains(lower, "requirements") {
			t.Fatalf("artifact carries an attestation payload: %s", file.Path)
		}
	}
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("VerifyDevelopment: %v", err)
	}
}
