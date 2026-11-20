package compiler

// Development Build final-architecture tests for the requirement digest domain,
// the exact Evidence reference positions and the Verify completeness matrix.
//
// The three subjects are tested together because they share one property: the
// artifact must never lose a real source position or a real class boundary in
// exchange for a tidier (but false) digest. Determinism is therefore stated
// precisely:
//
//   - identical source bytes and requests produce byte-identical artifacts;
//   - semantically unordered inputs (file order, snapshot order, lock order,
//     mapping order) are normalized and may not change any byte;
//   - a real list index inside a source document is part of the location, so
//     reordering that list is allowed to change the Requirement ID and the
//     artifact digest. Order independence must never be bought by dropping the
//     true index.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

// legacyDevelopmentRequirementDigestDomain is the Requirement ID digest domain
// that the pre-final slice used for both classes. It is kept in the test tree
// only so the rejection of artifacts produced with it stays proven.
const legacyDevelopmentRequirementDigestDomain = "CA-SKILL-DEVELOPMENT-HOST-REQUIREMENT\x00v1\x00"

// foreignDevelopmentRequirementDigestDomain is any digest domain other than the
// single unified one. A requirement derived from it must never verify.
const foreignDevelopmentRequirementDigestDomain = "CA-SKILL-DEVELOPMENT-OTHER-REQUIREMENT\x00v1\x00"

func replaceDevelopmentSkillFile(t *testing.T, request DevelopmentBuildRequest, path string, content []byte) DevelopmentBuildRequest {
	t.Helper()
	request = cloneDevelopmentRequest(t, request)
	for index := range request.Files {
		if request.Files[index].Path == path {
			request.Files[index].Bytes = append([]byte(nil), content...)
			return request
		}
	}
	t.Fatalf("fixture has no Skill file %s", path)
	return request
}

func developmentPredicatesFile(t *testing.T, request DevelopmentBuildRequest) []byte {
	t.Helper()
	for _, file := range request.Files {
		if file.Path == "skills/verify-ue-registration/assurance/predicates.yaml" {
			return append([]byte(nil), file.Bytes...)
		}
	}
	t.Fatal("fixture has no PredicateSet source")
	return nil
}

// developmentRequirementsOfKind returns every requirement of one kind in one
// document together with its decoded record.
func developmentRequirementsOfKind(t *testing.T, result DevelopmentBuildResult, document, kind string) []DevelopmentHostRequirement {
	t.Helper()
	matched := make([]DevelopmentHostRequirement, 0)
	for _, requirement := range developmentRequirements(t, result, document) {
		if requirement.Kind == kind {
			matched = append(matched, requirement)
		}
	}
	return matched
}

func developmentSelectors(requirements []DevelopmentHostRequirement) []string {
	selectors := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		selectors = append(selectors, requirement.Location.Selector)
	}
	sort.Strings(selectors)
	return selectors
}

// TestDevelopmentEvidenceLocationSinglePredicate proves the exact pointer of a
// Descriptor referenced once by one Predicate: the Predicate ID, the requires
// parameter key and the real array index are all present.
func TestDevelopmentEvidenceLocationSinglePredicate(t *testing.T) {
	request := loadDevelopmentFixtureWithDescriptor(t, true, true)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("fixture build failed: %#v", result.Diagnostics)
	}
	domain := developmentRequirementsOfKind(t, result, developmentRequirementsFileName, DevelopmentRequirementEvidenceDomain)
	fact := developmentRequirementsOfKind(t, result, developmentRequirementsFileName, DevelopmentRequirementEvidenceFact)
	const wantSelector = "/predicates/example.ue.registered/evidence/requires/registration/0"
	if len(domain) != 1 || domain[0].Location.Selector != wantSelector {
		t.Fatalf("domain trust selectors = %#v, want [%s]", developmentSelectors(domain), wantSelector)
	}
	if len(fact) != 1 || fact[0].Location.Selector != wantSelector {
		t.Fatalf("fact trust selectors = %#v, want [%s]", developmentSelectors(fact), wantSelector)
	}
	for _, requirement := range append(append([]DevelopmentHostRequirement{}, domain...), fact...) {
		if requirement.Location.Node != "example.ue.registered" || requirement.Location.Stage != "evidence" {
			t.Fatalf("evidence requirement location = %+v", requirement.Location)
		}
		if requirement.Location.Path != "skills/verify-ue-registration/assurance/predicates.yaml" {
			t.Fatalf("evidence requirement path = %q", requirement.Location.Path)
		}
	}
}

// TestDevelopmentEvidenceLocationMultiplePredicates proves that the same
// Descriptor referenced from two Predicates yields two requirements with
// different Predicate identities and different IDs; the shared Descriptor
// content is never used to collapse the positions.
func TestDevelopmentEvidenceLocationMultiplePredicates(t *testing.T) {
	base := loadDevelopmentFixtureWithDescriptor(t, true, false)
	source := string(developmentPredicatesFile(t, base))
	second := strings.Replace(source,
		"predicates:\n",
		"predicates:\n  example.ue.also-registered:\n    parameters:\n      - registration: ca::urn:example:ca:registration:v1\n    evidence:\n      requires:\n        registration: [ca::ue.registration-state]\n    evaluator:\n      type: cel\n      expression: 'registration.state == \"REGISTERED\"'\n", 1)
	request := replaceDevelopmentSkillFile(t, base, "skills/verify-ue-registration/assurance/predicates.yaml", []byte(second))
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("two-predicate build failed: %#v", result.Diagnostics)
	}
	domain := developmentRequirementsOfKind(t, result, developmentRequirementsFileName, DevelopmentRequirementEvidenceDomain)
	if len(domain) != 2 {
		t.Fatalf("domain trust requirements = %d, want 2", len(domain))
	}
	nodes := map[string]bool{}
	ids := map[string]bool{}
	for _, requirement := range domain {
		nodes[requirement.Location.Node] = true
		ids[requirement.RequirementID] = true
	}
	if !nodes["example.ue.registered"] || !nodes["example.ue.also-registered"] {
		t.Fatalf("evidence requirement predicates = %#v", nodes)
	}
	if len(ids) != 2 {
		t.Fatalf("two Predicate positions collapsed onto one requirement ID: %#v", ids)
	}
}

// TestDevelopmentEvidenceLocationRepeatedInOnePredicate proves that the same
// Descriptor referenced twice from one requires list produces one requirement
// per list index, not one merged requirement.
func TestDevelopmentEvidenceLocationRepeatedInOnePredicate(t *testing.T) {
	base := loadDevelopmentFixtureWithDescriptor(t, true, false)
	source := bytes.Replace(developmentPredicatesFile(t, base),
		[]byte("registration: [ca::ue.registration-state]"),
		[]byte("registration: [ca::ue.registration-state, ca::ue.registration-state]"), 1)
	request := replaceDevelopmentSkillFile(t, base, "skills/verify-ue-registration/assurance/predicates.yaml", source)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("repeated-reference build failed: %#v", result.Diagnostics)
	}
	domain := developmentRequirementsOfKind(t, result, developmentRequirementsFileName, DevelopmentRequirementEvidenceDomain)
	want := []string{
		"/predicates/example.ue.registered/evidence/requires/registration/0",
		"/predicates/example.ue.registered/evidence/requires/registration/1",
	}
	if got := developmentSelectors(domain); len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("repeated reference selectors = %#v, want %#v", got, want)
	}
	if domain[0].RequirementID == domain[1].RequirementID {
		t.Fatalf("two real list positions collapsed onto one requirement ID")
	}
}

// TestDevelopmentEvidenceLocationMultipleDescriptorsInOneList proves that each
// Descriptor in one requires list keeps its own index.
func TestDevelopmentEvidenceLocationMultipleDescriptorsInOneList(t *testing.T) {
	request := independenceDevelopmentFixture(t)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("independence fixture build failed: %#v", result.Diagnostics)
	}
	bySelector := map[string]DevelopmentHostRequirement{}
	for _, requirement := range developmentRequirements(t, result, developmentRequirementsFileName) {
		if requirement.Kind != DevelopmentRequirementEvidenceDomain {
			continue
		}
		for _, subject := range requirement.Subjects {
			if subject.ID != "ue.registration-state" && subject.ID != "ue.registration-freshness" {
				continue
			}
			bySelector[requirement.Location.Selector] = requirement
		}
	}
	for _, want := range []string{
		"/predicates/example.ue.registered/evidence/requires/registration/0",
		"/predicates/example.ue.registered/evidence/requires/registration/1",
	} {
		if _, exists := bySelector[want]; !exists {
			t.Fatalf("missing exact requires position %q; got %#v", want, sortedStringMapKeys(bySelector))
		}
	}
	// Both Descriptors are also referenced by the two Independence Groups, so
	// those positions must be present as well.
	for _, want := range []string{
		"/predicates/example.ue.registered/evidence/independence/0/members/0/guarantee",
		"/predicates/example.ue.registered/evidence/independence/1/members/0/guarantee",
	} {
		if _, exists := bySelector[want]; !exists {
			t.Fatalf("missing exact independence position %q; got %#v", want, sortedStringMapKeys(bySelector))
		}
	}
}

// TestDevelopmentEvidenceLocationBothIdentitiesShareOnePosition proves that one
// Descriptor declaring both identities yields two kinds with the same exact
// position, and that neither kind overwrites the other.
func TestDevelopmentEvidenceLocationBothIdentitiesShareOnePosition(t *testing.T) {
	result := buildDevelopmentWithIdentities(t, true, true)
	domain := developmentRequirementsOfKind(t, result, developmentRequirementsFileName, DevelopmentRequirementEvidenceDomain)
	fact := developmentRequirementsOfKind(t, result, developmentRequirementsFileName, DevelopmentRequirementEvidenceFact)
	if len(domain) != 1 || len(fact) != 1 {
		t.Fatalf("domain/fact requirements = %d/%d, want 1/1", len(domain), len(fact))
	}
	if domain[0].Location != fact[0].Location {
		t.Fatalf("the same Descriptor produced two different positions: %+v != %+v", domain[0].Location, fact[0].Location)
	}
	if domain[0].RequirementID == fact[0].RequirementID {
		t.Fatal("the two trust kinds collapsed onto one requirement ID")
	}
}

// TestDevelopmentEvidenceLocationDrivesRequirementID proves that changing a real
// source index changes the Requirement ID, and that two builds over identical
// bytes are byte-identical. Order independence is deliberately not claimed for a
// real list index.
func TestDevelopmentEvidenceLocationDrivesRequirementID(t *testing.T) {
	base := loadDevelopmentFixtureWithDescriptor(t, true, false)
	source := bytes.Replace(developmentPredicatesFile(t, base),
		[]byte("registration: [ca::ue.registration-state]"),
		[]byte("registration: [ca::ue.registration-state, ca::ue.registration-state]"), 1)
	request := replaceDevelopmentSkillFile(t, base, "skills/verify-ue-registration/assurance/predicates.yaml", source)
	first := BuildDevelopment(request)
	if !first.Success {
		t.Fatalf("first build failed: %#v", first.Diagnostics)
	}
	domain := developmentRequirementsOfKind(t, first, developmentRequirementsFileName, DevelopmentRequirementEvidenceDomain)
	if len(domain) != 2 {
		t.Fatalf("domain requirements = %d, want 2", len(domain))
	}
	bySelector := map[string]string{}
	for _, requirement := range domain {
		bySelector[requirement.Location.Selector] = requirement.RequirementID
	}
	low := bySelector["/predicates/example.ue.registered/evidence/requires/registration/0"]
	high := bySelector["/predicates/example.ue.registered/evidence/requires/registration/1"]
	if low == "" || high == "" || low == high {
		t.Fatalf("list index did not drive the requirement ID: %q vs %q", low, high)
	}

	// Identical input bytes must rebuild byte for byte.
	second := BuildDevelopment(cloneDevelopmentRequest(t, request))
	if !second.Success || second.ArtifactDigest != first.ArtifactDigest {
		t.Fatalf("repeated build over identical bytes changed the artifact")
	}
	if len(second.OutputFiles) != len(first.OutputFiles) {
		t.Fatalf("repeated build changed the file count")
	}
	for index := range first.OutputFiles {
		if first.OutputFiles[index].Path != second.OutputFiles[index].Path ||
			!bytes.Equal(first.OutputFiles[index].Bytes, second.OutputFiles[index].Bytes) {
			t.Fatalf("repeated build changed file bytes: %s", first.OutputFiles[index].Path)
		}
	}
}

// TestDevelopmentEvidenceLocationReorderIsLocal proves that reversing the same
// requires list only relocates the real indices: the requirement IDs for the two
// descriptors swap their selectors and the artifact digest is allowed to change.
// Semantically unordered request inputs still normalize to the same bytes.
func TestDevelopmentEvidenceLocationReorderIsLocal(t *testing.T) {
	request := independenceDevelopmentFixture(t)
	first := BuildDevelopment(request)
	if !first.Success {
		t.Fatalf("initial build failed: %#v", first.Diagnostics)
	}
	stateBefore := ""
	for _, requirement := range developmentRequirements(t, first, developmentRequirementsFileName) {
		if requirement.Kind != DevelopmentRequirementEvidenceDomain {
			continue
		}
		if requirement.Location.Selector != "/predicates/example.ue.registered/evidence/requires/registration/0" {
			continue
		}
		for _, subject := range requirement.Subjects {
			if subject.ID == "ue.registration-state" {
				stateBefore = requirement.RequirementID
			}
		}
	}
	if stateBefore == "" {
		t.Fatal("fixture has no ue.registration-state at requires index 0")
	}

	// Swap the first two guarantees of the requires list only.
	predicatesPath := "skills/verify-ue-registration/assurance/predicates.yaml"
	predicates := string(mustDevelopmentSkillFile(t, request, predicatesPath))
	const before = "registration: [ca::ue.registration-state, ca::ue.registration-freshness, ca::ue.registration-alternate]"
	const after = "registration: [ca::ue.registration-freshness, ca::ue.registration-state, ca::ue.registration-alternate]"
	if !strings.Contains(predicates, before) {
		t.Fatalf("independence fixture no longer has the expected requires list")
	}
	reordered := replaceDevelopmentSkillFile(t, request, predicatesPath, []byte(strings.Replace(predicates, before, after, 1)))
	second := BuildDevelopment(reordered)
	if !second.Success {
		t.Fatalf("reordered build failed: %#v", second.Diagnostics)
	}
	stateAfter := ""
	for _, requirement := range developmentRequirements(t, second, developmentRequirementsFileName) {
		if requirement.Kind != DevelopmentRequirementEvidenceDomain {
			continue
		}
		if requirement.Location.Selector != "/predicates/example.ue.registered/evidence/requires/registration/1" {
			continue
		}
		for _, subject := range requirement.Subjects {
			if subject.ID == "ue.registration-state" {
				stateAfter = requirement.RequirementID
			}
		}
	}
	if stateAfter == "" {
		t.Fatal("reordered source has no ue.registration-state at requires index 1")
	}
	if stateAfter == stateBefore {
		t.Fatal("the requirement ID ignored a real source index change")
	}
	if second.ArtifactDigest == first.ArtifactDigest {
		t.Fatal("relocating a real source position must change the artifact digest")
	}

	// Reordering the semantically unordered request inputs is still normalized.
	permuted := cloneDevelopmentRequest(t, request)
	developmentReverseFiles(permuted.Files)
	reverseSnapshots(permuted.Snapshots)
	reverseRoots(permuted.Lock.Roots)
	reversePackages(permuted.Lock.Packages)
	for index := range permuted.Lock.Packages {
		reverseEdges(permuted.Lock.Packages[index].Dependencies)
	}
	normalized := BuildDevelopment(permuted)
	if !normalized.Success || normalized.ArtifactDigest != first.ArtifactDigest {
		t.Fatalf("reordering semantically unordered inputs changed the artifact")
	}
}

func mustDevelopmentSkillFile(t *testing.T, request DevelopmentBuildRequest, path string) []byte {
	t.Helper()
	for _, file := range request.Files {
		if file.Path == path {
			return file.Bytes
		}
	}
	t.Fatalf("fixture has no Skill file %s", path)
	return nil
}

// developmentRequirementIDInDomain re-derives one requirement ID from the given
// digest domain, so a test can seal an artifact with a historic or foreign domain.
func developmentRequirementIDInDomain(t *testing.T, requirement DevelopmentHostRequirement, domain string) string {
	t.Helper()
	key := normalizeDevelopmentRequirementKey(developmentRequirementKey{
		Kind: requirement.Kind, Location: requirement.Location, Subjects: requirement.Subjects,
		Independence: requirement.Independence,
	})
	canonical, err := canonicalizeValue(key)
	if err != nil {
		t.Fatalf("canonicalize requirement key: %v", err)
	}
	sum := sha256.Sum256(append([]byte(domain), canonical...))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func resealRequirements(t *testing.T, requirements []DevelopmentHostRequirement) []DevelopmentHostRequirement {
	t.Helper()
	resealed := make([]DevelopmentHostRequirement, 0, len(requirements))
	for _, requirement := range requirements {
		requirement.RequirementID = developmentRequirementIDInDomain(t, requirement, developmentRequirementDigestDomain)
		resealed = append(resealed, requirement)
	}
	sort.Slice(resealed, func(i, j int) bool { return resealed[i].RequirementID < resealed[j].RequirementID })
	return resealed
}

// TestDevelopmentRequirementDigestDomainIsUnified proves that both classes are
// derived from the single class-neutral domain, that the historical Host-specific
// domain no longer reproduces any committed ID, and that Verify rejects both the
// legacy and an arbitrary foreign domain even after every outer digest is sealed.
func TestDevelopmentRequirementDigestDomainIsUnified(t *testing.T) {
	legacy := recoveryFixture(t, recoveryDescriptor(true))
	request := developmentRequestFromLegacy(t, legacy)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("recovery Development build failed: %#v", result.Diagnostics)
	}
	host := developmentRequirements(t, result, developmentRequirementsFileName)
	runtime := developmentRequirements(t, result, developmentRuntimeRequirementsFileName)
	if len(host) == 0 || len(runtime) == 0 {
		t.Fatalf("fixture must carry both classes, got host=%d runtime=%d", len(host), len(runtime))
	}
	for _, requirement := range append(append([]DevelopmentHostRequirement{}, host...), runtime...) {
		want := developmentRequirementIDInDomain(t, requirement, developmentRequirementDigestDomain)
		if requirement.RequirementID != want {
			t.Fatalf("%s requirement ID is not derived from the unified domain: %s", requirement.Kind, requirement.RequirementID)
		}
		legacyID := developmentRequirementIDInDomain(t, requirement, legacyDevelopmentRequirementDigestDomain)
		if requirement.RequirementID == legacyID {
			t.Fatalf("%s requirement ID still uses the historical Host-requirement domain", requirement.Kind)
		}
	}

	for _, testCase := range []struct {
		name   string
		domain string
	}{
		{name: "legacy host domain", domain: legacyDevelopmentRequirementDigestDomain},
		{name: "foreign domain", domain: foreignDevelopmentRequirementDigestDomain},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			forged := make([]DevelopmentHostRequirement, 0, len(host))
			for _, requirement := range host {
				requirement.RequirementID = developmentRequirementIDInDomain(t, requirement, testCase.domain)
				forged = append(forged, requirement)
			}
			sort.Slice(forged, func(i, j int) bool { return forged[i].RequirementID < forged[j].RequirementID })
			raw := canonicalTestJSON(t, forged)
			if err := verifyDevelopmentRequirementDocument(raw, "host"); err == nil {
				t.Fatalf("isolated verification accepted a requirement derived from the %s", testCase.name)
			}
			tampered := tamperDevelopmentArtifact(t, result, func(files map[string][]byte) {
				files[developmentRequirementsFileName] = raw
			})
			if err := VerifyDevelopment(request, tampered); err == nil {
				t.Fatalf("VerifyDevelopment accepted requirements derived from the %s", testCase.name)
			}
		})
	}
}

// TestDevelopmentVerifyRejectsDocumentTampering proves the Verify completeness
// matrix for the two requirement documents. Every case re-seals the manifest file
// digests and the artifactDigest, so only real verification can catch it.
func TestDevelopmentVerifyRejectsDocumentTampering(t *testing.T) {
	legacy := recoveryFixture(t, recoveryDescriptor(true))
	request := developmentRequestFromLegacy(t, legacy)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("recovery Development build failed: %#v", result.Diagnostics)
	}
	host := developmentRequirements(t, result, developmentRequirementsFileName)
	runtime := developmentRequirements(t, result, developmentRuntimeRequirementsFileName)
	if len(host) < 2 || len(runtime) < 2 {
		t.Fatalf("fixture must carry at least two requirements per class, got %d/%d", len(host), len(runtime))
	}

	cases := []struct {
		name string
		// isolated reports whether the artifact-only check must already reject the
		// tampering; deletions are only visible to the rebuild.
		isolated bool
		mutate   func(files map[string][]byte)
	}{
		{
			name: "deleted host requirement",
			mutate: func(files map[string][]byte) {
				files[developmentRequirementsFileName] = canonicalTestJSON(t, host[1:])
			},
		},
		{
			name: "deleted runtime requirement",
			mutate: func(files map[string][]byte) {
				files[developmentRuntimeRequirementsFileName] = canonicalTestJSON(t, runtime[1:])
			},
		},
		{
			name:     "runtime requirement moved into host document",
			isolated: true,
			mutate: func(files map[string][]byte) {
				migrated := append(append([]DevelopmentHostRequirement{}, host...), runtime[0])
				sort.Slice(migrated, func(i, j int) bool { return migrated[i].RequirementID < migrated[j].RequirementID })
				files[developmentRequirementsFileName] = canonicalTestJSON(t, migrated)
				files[developmentRuntimeRequirementsFileName] = canonicalTestJSON(t, runtime[1:])
			},
		},
		{
			name:     "host requirement moved into runtime document",
			isolated: true,
			mutate: func(files map[string][]byte) {
				migrated := append(append([]DevelopmentHostRequirement{}, runtime...), host[0])
				sort.Slice(migrated, func(i, j int) bool { return migrated[i].RequirementID < migrated[j].RequirementID })
				files[developmentRuntimeRequirementsFileName] = canonicalTestJSON(t, migrated)
				files[developmentRequirementsFileName] = canonicalTestJSON(t, host[1:])
			},
		},
		{
			name:     "edited status",
			isolated: true,
			mutate: func(files map[string][]byte) {
				edited := append([]DevelopmentHostRequirement{}, host...)
				edited[0].Status = DevelopmentRuntimeRequirementStatus
				files[developmentRequirementsFileName] = canonicalTestJSON(t, edited)
			},
		},
		{
			name:     "edited requirementId",
			isolated: true,
			mutate: func(files map[string][]byte) {
				edited := append([]DevelopmentHostRequirement{}, host...)
				edited[0].RequirementID = "sha256:" + strings.Repeat("0", 64)
				files[developmentRequirementsFileName] = canonicalTestJSON(t, edited)
			},
		},
		{
			name:     "edited location",
			isolated: true,
			mutate: func(files map[string][]byte) {
				edited := append([]DevelopmentHostRequirement{}, host...)
				edited[0].Location.Selector = "/predicates/other/evidence/requires/registration/0"
				files[developmentRequirementsFileName] = canonicalTestJSON(t, edited)
			},
		},
		{
			name:     "added unknown field",
			isolated: true,
			mutate: func(files map[string][]byte) {
				files[developmentRequirementsFileName] = canonicalTestJSON(t, withUnknownRequirementField(t, host, "admitted", true))
			},
		},
		{
			name:     "duplicated requirement",
			isolated: true,
			mutate: func(files map[string][]byte) {
				duplicated := append(append([]DevelopmentHostRequirement{}, host...), host[0])
				sort.Slice(duplicated, func(i, j int) bool { return duplicated[i].RequirementID < duplicated[j].RequirementID })
				files[developmentRequirementsFileName] = canonicalTestJSON(t, duplicated)
			},
		},
		{
			name:     "runtime requirement in host document with recomputed identity",
			isolated: true,
			mutate: func(files map[string][]byte) {
				migrated := append(append([]DevelopmentHostRequirement{}, host...), runtime[0])
				files[developmentRequirementsFileName] = canonicalTestJSON(t, resealRequirements(t, migrated))
				files[developmentRuntimeRequirementsFileName] = canonicalTestJSON(t, resealRequirements(t, runtime[1:]))
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			tampered := tamperDevelopmentArtifact(t, result, testCase.mutate)
			if testCase.isolated {
				if err := verifyDevelopmentArtifactFiles(tampered); err == nil {
					t.Fatalf("artifact-only verification accepted %q", testCase.name)
				}
			}
			if err := VerifyDevelopment(request, tampered); err == nil {
				t.Fatalf("VerifyDevelopment accepted %q", testCase.name)
			}
		})
	}

	// The untouched artifact must still verify, so the matrix above cannot pass
	// vacuously.
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("untouched artifact failed verification: %v", err)
	}
}

func withUnknownRequirementField(t *testing.T, requirements []DevelopmentHostRequirement, name string, value any) []map[string]any {
	t.Helper()
	raw, err := json.Marshal(requirements)
	if err != nil {
		t.Fatalf("marshal requirements: %v", err)
	}
	var objects []map[string]any
	if err := json.Unmarshal(raw, &objects); err != nil {
		t.Fatalf("unmarshal requirements: %v", err)
	}
	objects[0][name] = value
	return objects
}

// TestDevelopmentVerifyRejectsIndependenceTampering proves that the Independence
// detail is covered to the last member, domain pair and group index: even an
// attacker who recomputes every requirement ID and re-seals the manifest is
// rejected by the deterministic rebuild.
func TestDevelopmentVerifyRejectsIndependenceTampering(t *testing.T) {
	request := independenceDevelopmentFixture(t)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("independence fixture build failed: %#v", result.Diagnostics)
	}
	groups := developmentRequirementsOfKind(t, result, developmentRequirementsFileName, DevelopmentRequirementIndependence)
	if len(groups) != 2 {
		t.Fatalf("independence requirements = %d, want 2", len(groups))
	}
	for _, group := range groups {
		if group.Independence == nil || len(group.Independence.Members) < 2 {
			t.Fatalf("independence requirement has no exact detail: %+v", group)
		}
	}

	cases := []struct {
		name   string
		mutate func(host []DevelopmentHostRequirement, groupIndex int) []DevelopmentHostRequirement
	}{
		{
			name: "member domain edited",
			mutate: func(host []DevelopmentHostRequirement, index int) []DevelopmentHostRequirement {
				host[index].Independence.Members[0].Domain = "urn:example:record-domain:forged"
				return host
			},
		},
		{
			name: "member export id edited",
			mutate: func(host []DevelopmentHostRequirement, index int) []DevelopmentHostRequirement {
				host[index].Independence.Members[0].ExportID = "ue.registration-forged"
				return host
			},
		},
		{
			name: "member removed",
			mutate: func(host []DevelopmentHostRequirement, index int) []DevelopmentHostRequirement {
				host[index].Independence.Members = host[index].Independence.Members[1:]
				return host
			},
		},
		{
			name: "domain pair removed",
			mutate: func(host []DevelopmentHostRequirement, index int) []DevelopmentHostRequirement {
				host[index].Independence.DomainPairs = host[index].Independence.DomainPairs[1:]
				return host
			},
		},
		{
			name: "domain pair edited",
			mutate: func(host []DevelopmentHostRequirement, index int) []DevelopmentHostRequirement {
				host[index].Independence.DomainPairs[0] = []string{"urn:example:record-domain:one", "urn:example:record-domain:forged"}
				return host
			},
		},
		{
			name: "group index and selector edited",
			mutate: func(host []DevelopmentHostRequirement, index int) []DevelopmentHostRequirement {
				host[index].Independence.GroupIndex = 4
				host[index].Location.Selector = "/predicates/example.ue.registered/evidence/independence/4"
				return host
			},
		},
		{
			name: "group replaced by a coarse evidence position",
			mutate: func(host []DevelopmentHostRequirement, index int) []DevelopmentHostRequirement {
				host[index].Location.Selector = "/predicates/example.ue.registered/evidence"
				return host
			},
		},
		{
			name: "all groups merged into one record",
			mutate: func(host []DevelopmentHostRequirement, index int) []DevelopmentHostRequirement {
				merged := make([]DevelopmentHostRequirement, 0, len(host))
				for _, requirement := range host {
					if requirement.Kind == DevelopmentRequirementIndependence && requirement.RequirementID != host[index].RequirementID {
						continue
					}
					merged = append(merged, requirement)
				}
				return merged
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			// Decode the artifact documents again so every case starts clean.
			host := developmentRequirements(t, result, developmentRequirementsFileName)
			groupIndex := -1
			for index, requirement := range host {
				if requirement.Kind == DevelopmentRequirementIndependence {
					groupIndex = index
					break
				}
			}
			if groupIndex < 0 {
				t.Fatal("artifact has no independence requirement")
			}
			edited := testCase.mutate(host, groupIndex)
			// The attacker recomputes every identity from the edited detail and
			// re-seals the manifest, so only the deterministic rebuild can catch it.
			tampered := tamperDevelopmentArtifact(t, result, func(files map[string][]byte) {
				files[developmentRequirementsFileName] = canonicalTestJSON(t, resealRequirements(t, edited))
			})
			if err := VerifyDevelopment(request, tampered); err == nil {
				t.Fatalf("VerifyDevelopment accepted independence tampering: %s", testCase.name)
			}
		})
	}
}
