package compiler

// WP3B conformance fixtures: ToolCatalog export identity and trusted Owner
// attribution, derived from the Appendix E baseline and the frozen minimal-change
// method (E05, E39, E43, E91). See docs/workstreams/wp3b-owner-export-linking-v1alpha4.md.

import (
	"reflect"
	"strings"
	"testing"
)

var wp3bDiagnosticCodes = map[string]bool{
	"EXPORT_ID_AMBIGUOUS":                true,
	"EXPORT_ID_MISSING":                  true,
	"SCHEMA_MISSING":                     true,
	"DEPENDENCY_OWNER_MISMATCH":          true,
	"DEPENDENCY_EXPORT_OWNER_UNVERIFIED": true,
}

func (fixture *wp3aFixture) addSnapshotFile(dependencyID, path string, content []byte) {
	for index := range fixture.dependencies {
		if fixture.dependencies[index].DependencyID != dependencyID {
			continue
		}
		fixture.dependencies[index].Files = append(fixture.dependencies[index].Files, File{Path: path, Bytes: content})
	}
}

func (fixture *wp3aFixture) replaceSnapshotFile(dependencyID, path, old, replacement string) {
	for index := range fixture.dependencies {
		if fixture.dependencies[index].DependencyID != dependencyID {
			continue
		}
		for fileIndex := range fixture.dependencies[index].Files {
			file := &fixture.dependencies[index].Files[fileIndex]
			if file.Path == path {
				file.Bytes = []byte(strings.Replace(string(file.Bytes), old, replacement, 1))
			}
		}
	}
}

func (fixture *wp3aFixture) refreshSnapshotDigests(t *testing.T) {
	t.Helper()
	for index := range fixture.dependencies {
		digest, err := ComputeDependencySnapshotDigest(fixture.dependencies[index].Files)
		if err != nil {
			t.Fatalf("recompute digest for %s: %v", fixture.dependencies[index].DependencyID, err)
		}
		fixture.dependencies[index].Digest = digest
	}
	for index := range fixture.bindings {
		for _, dependency := range fixture.dependencies {
			if dependency.DependencyID == fixture.bindings[index].DependencyID {
				fixture.bindings[index].Digest = dependency.Digest
			}
		}
	}
}

func wp3bOwnerLinking(t *testing.T, req BuildRequest) (dependencyIdentityResult, ownerLinkingResult) {
	t.Helper()
	identity := wp3aIdentity(t, req)
	return identity, validateV1Alpha4OwnerLinking(req, wp3aParseSkillAST(t, req, wp3aSkillName), identity)
}

func wp3bDiagnostics(diagnostics []Diagnostic) []Diagnostic {
	filtered := make([]Diagnostic, 0)
	for _, diagnostic := range diagnostics {
		if wp3bDiagnosticCodes[diagnostic.Code] {
			filtered = append(filtered, diagnostic)
		}
	}
	sortDiagnostics(filtered)
	return filtered
}

func wp3bCheckBuildAgreement(t *testing.T, req BuildRequest) []Diagnostic {
	t.Helper()
	check := wp3bDiagnostics(Check(req))
	build := wp3bDiagnostics(Build(req).Diagnostics)
	if !reflect.DeepEqual(check, build) {
		t.Fatalf("Check and Build disagree on WP3B diagnostics:\ncheck=%+v\nbuild=%+v", check, build)
	}
	return check
}

func expectOnlyWP3BDiagnostics(t *testing.T, diagnostics []Diagnostic, code string, count int) {
	t.Helper()
	if len(diagnostics) != count {
		t.Fatalf("WP3B diagnostics = %+v, want %d x %s", diagnostics, count, code)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code != code || diagnostic.Phase != "C_STATIC" || diagnostic.Severity != "error" {
			t.Fatalf("WP3B diagnostic = %+v, want %s/C_STATIC", diagnostic, code)
		}
	}
}

// multiOwnerFixture is the Appendix E multi-Owner baseline: the ca snapshot has
// two authorized Owners and explicit attribution for every referenced export.
func multiOwnerFixture(t *testing.T) wp3aFixture {
	t.Helper()
	fixture := wp3aBaseFixture(t)
	for index := range fixture.bindings {
		if fixture.bindings[index].DependencyID != "ca" {
			continue
		}
		fixture.bindings[index].AuthorizedOwners = []string{"example-ue-owner", "example-other-owner"}
		fixture.bindings[index].ExportOwners = []ExportOwner{
			{ExportKind: "observation", ExportID: "ue.registration.current", Owner: "example-ue-owner"},
			{ExportKind: "schema", ExportID: "urn:example:ca:registration:v1", Owner: "example-ue-owner"},
			{ExportKind: "evidenceGuarantee", ExportID: "ue.registration-state", Owner: "example-ue-owner"},
		}
	}
	return fixture
}

// WP3B-001: positive export identity and trusted Owner attribution.
func TestWP3B001PositiveExportIdentity(t *testing.T) {
	req := wp3aBaseFixture(t).request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)
	_, linking := wp3bOwnerLinking(t, req)
	if len(linking.Diagnostics) != 0 {
		t.Fatalf("positive baseline emitted WP3B diagnostics: %+v", linking.Diagnostics)
	}

	want := []linkedExport{
		{DependencyAlias: "ca", ExportKind: "evidenceGuarantee", ExportID: "ue.registration-state", TrustedOwner: "example-ue-owner"},
		{
			DependencyAlias: "ca", ExportKind: "observation", ExportID: "ue.registration.current",
			TrustedOwner: "example-ue-owner", SelfDeclaredOwner: "example-ue-owner",
			SourcePath: "tool-catalog.yaml", Effect: "READ",
			QueryContextSchemaID: "urn:example:ca:registration-query:v1",
			OutputSchemaID:       "urn:example:ca:registration:v1",
			OwnerOutcomes:        []string{"SUCCEEDED", "FAILED"},
			EvidenceProvides:     []evidenceProvide{{ID: "ue.registration-state", Path: "output"}},
		},
		{
			DependencyAlias: "ca", ExportKind: "schema", ExportID: "urn:example:ca:registration:v1",
			TrustedOwner: "example-ue-owner", SourcePath: "registration.schema.yaml",
		},
		{
			DependencyAlias: "ca-context", ExportKind: "schema", ExportID: "urn:example:ca:invocation-context:v1",
			TrustedOwner: "example-host-owner", SourcePath: "invocation-context.schema.yaml",
		},
	}
	if !reflect.DeepEqual(linking.Exports, want) {
		t.Fatalf("linked exports = %+v, want %+v", linking.Exports, want)
	}
	if diagnostics := wp3bCheckBuildAgreement(t, req); len(diagnostics) != 0 {
		t.Fatalf("positive baseline emitted WP3B diagnostics through Check/Build: %+v", diagnostics)
	}
	result := Build(req)
	assertFailureAtomic(t, result)
	if !hasCode(result.Diagnostics, "CAPABILITY_INPUT_UNTRUSTED") {
		t.Fatalf("later unimplemented stages must still fail closed: %+v", result.Diagnostics)
	}

	// Snapshot file order must not change attribution or diagnostics.
	permuted := req
	permuted.ResolvedDependencies = append([]ResolvedDependency(nil), req.ResolvedDependencies...)
	for index := range permuted.ResolvedDependencies {
		files := permuted.ResolvedDependencies[index].Files
		reversed := make([]File, len(files))
		for fileIndex := range files {
			reversed[len(files)-1-fileIndex] = files[fileIndex]
		}
		permuted.ResolvedDependencies[index].Files = reversed
	}
	_, permutedLinking := wp3bOwnerLinking(t, permuted)
	if !reflect.DeepEqual(linking.Exports, permutedLinking.Exports) || len(permutedLinking.Diagnostics) != 0 {
		t.Fatalf("WP3B attribution is not snapshot-order stable: %+v", permutedLinking)
	}
}

// WP3B-002 / E05: ToolCatalog metadata.owner is not an authorized Owner.
func TestWP3B002OwnerNotAuthorizedE05(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceSnapshotFile("ca", "tool-catalog.yaml", "owner: example-ue-owner", "owner: example-untrusted-owner")
	fixture.refreshSnapshotDigests(t)
	req := fixture.request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)

	diagnostics := wp3bCheckBuildAgreement(t, req)
	expectOnlyWP3BDiagnostics(t, diagnostics, "DEPENDENCY_OWNER_MISMATCH", 1)
	if diagnostics[0].Path != "" || diagnostics[0].Line != 0 || diagnostics[0].Column != 0 {
		t.Fatalf("imported-contract diagnostic must use empty path and zero location: %+v", diagnostics[0])
	}
	assertFailureAtomic(t, Build(req))
}

// WP3B-003 / E39: multi-Owner snapshot lacks attribution for a linked export.
func TestWP3B003MultiOwnerMissingAttributionE39(t *testing.T) {
	baseline := multiOwnerFixture(t)
	baselineReq := baseline.request(t)
	assertNoRequestEnvelopeDiagnostics(t, baselineReq)
	if _, linking := wp3bOwnerLinking(t, baselineReq); len(linking.Diagnostics) != 0 {
		t.Fatalf("valid multi-Owner baseline must link: %+v", linking.Diagnostics)
	}

	fixture := multiOwnerFixture(t)
	for index := range fixture.bindings {
		if fixture.bindings[index].DependencyID != "ca" {
			continue
		}
		kept := make([]ExportOwner, 0, len(fixture.bindings[index].ExportOwners))
		for _, exportOwner := range fixture.bindings[index].ExportOwners {
			if exportOwner.ExportID == "urn:example:ca:registration:v1" {
				continue
			}
			kept = append(kept, exportOwner)
		}
		fixture.bindings[index].ExportOwners = kept
	}
	req := fixture.request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3bCheckBuildAgreement(t, req)
	expectOnlyWP3BDiagnostics(t, diagnostics, "DEPENDENCY_EXPORT_OWNER_UNVERIFIED", 1)
	if diagnostics[0].Path != "" || diagnostics[0].Line != 0 || diagnostics[0].Column != 0 {
		t.Fatalf("trusted-attribution diagnostic must use empty path and zero location: %+v", diagnostics[0])
	}
	assertFailureAtomic(t, Build(req))
}

// WP3B-004 / E43: unique attribution names an Owner outside the authorized set.
func TestWP3B004MultiOwnerUnauthorizedOwnerE43(t *testing.T) {
	fixture := multiOwnerFixture(t)
	for index := range fixture.bindings {
		if fixture.bindings[index].DependencyID != "ca" {
			continue
		}
		for exportIndex := range fixture.bindings[index].ExportOwners {
			if fixture.bindings[index].ExportOwners[exportIndex].ExportID == "urn:example:ca:registration:v1" {
				fixture.bindings[index].ExportOwners[exportIndex].Owner = "example-untrusted-owner"
			}
		}
	}
	req := fixture.request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3bCheckBuildAgreement(t, req)
	expectOnlyWP3BDiagnostics(t, diagnostics, "DEPENDENCY_OWNER_MISMATCH", 1)
	if diagnostics[0].Path != "" || diagnostics[0].Line != 0 || diagnostics[0].Column != 0 {
		t.Fatalf("imported-contract diagnostic must use empty path and zero location: %+v", diagnostics[0])
	}
	assertFailureAtomic(t, Build(req))
}

// WP3B-005 / E91: duplicate callable ID across ToolCatalog files in one snapshot.
func TestWP3B005DuplicateCallableIDE91(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.addSnapshotFile("ca", "second-catalog.yaml", []byte(`apiVersion: connectionagent.io/toolcatalog/v1alpha4
kind: ToolCatalog
metadata:
  name: second-catalog
  owner: example-ue-owner
spec:
  contracts:
    - kind: operation
      id: ue.registration.current
      effect: READ
      ownerOutcomes: [SUCCEEDED]
      inputSchema:
        $ref: urn:example:ca:registration-query:v1
      outputSchema:
        $ref: urn:example:ca:registration:v1
`))
	fixture.refreshSnapshotDigests(t)
	req := fixture.request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3bCheckBuildAgreement(t, req)
	expectOnlyWP3BDiagnostics(t, diagnostics, "EXPORT_ID_AMBIGUOUS", 1)
	assertFailureAtomic(t, Build(req))
}

// WP3B-006: referenced callable export does not exist.
func TestWP3B006MissingCallableExport(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "tool: ca::ue.registration.current", "tool: ca::ue.registration.missing")
	req := fixture.request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3bCheckBuildAgreement(t, req)
	expectOnlyWP3BDiagnostics(t, diagnostics, "EXPORT_ID_MISSING", 1)
	if diagnostics[0].Path != wp3aProcedurePath || diagnostics[0].Line == 0 || diagnostics[0].Column == 0 {
		t.Fatalf("missing export diagnostic must retain the Skill source reference location: %+v", diagnostics[0])
	}
	assertFailureAtomic(t, Build(req))
}

// WP3B-007: referenced Schema $id does not exist in the selected snapshot.
func TestWP3B007MissingSchemaExport(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aPredicatesPath, "ca::urn:example:ca:registration:v1", "ca::urn:example:ca:missing:v1")
	req := fixture.request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3bCheckBuildAgreement(t, req)
	expectOnlyWP3BDiagnostics(t, diagnostics, "SCHEMA_MISSING", 1)
	if diagnostics[0].Path != wp3aPredicatesPath || diagnostics[0].Line == 0 || diagnostics[0].Column == 0 {
		t.Fatalf("missing Schema diagnostic must retain the Skill source reference location: %+v", diagnostics[0])
	}
	assertFailureAtomic(t, Build(req))
}

// WP3B-008: callable exists but not with the required kind.
func TestWP3B008CallableKindMismatch(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceSnapshotFile("ca", "tool-catalog.yaml", "kind: observation", "kind: operation")
	fixture.replaceSnapshotFile("ca", "tool-catalog.yaml", "queryContextSchema:", "inputSchema:")
	fixture.refreshSnapshotDigests(t)
	req := fixture.request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3bCheckBuildAgreement(t, req)
	expectOnlyWP3BDiagnostics(t, diagnostics, "EXPORT_ID_MISSING", 1)
	assertFailureAtomic(t, Build(req))
}

// WP3B-009: duplicate Schema $id in one selected snapshot.
func TestWP3B009DuplicateSchemaIDE91(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.addSnapshotFile("ca", "registration-copy.schema.yaml", []byte(`$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:ca:registration:v1
type: object
`))
	fixture.refreshSnapshotDigests(t)
	req := fixture.request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3bCheckBuildAgreement(t, req)
	expectOnlyWP3BDiagnostics(t, diagnostics, "EXPORT_ID_AMBIGUOUS", 1)
	assertFailureAtomic(t, Build(req))
}

// validExportID is the frozen DottedId lexer for callable / Guarantee exports.
func TestWP3BExportIDLexer(t *testing.T) {
	for _, value := range []string{"ue.registration.current", "x", "a-b.c_d"} {
		if !validExportID(value) {
			t.Fatalf("valid ExportId rejected: %q", value)
		}
	}
	for _, value := range []string{"", ".a", "a.", "a..b", "1a", "a:b", "a b"} {
		if validExportID(value) {
			t.Fatalf("invalid ExportId accepted: %q", value)
		}
	}
}
