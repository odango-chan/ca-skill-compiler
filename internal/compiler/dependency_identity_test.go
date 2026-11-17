package compiler

// WP3A conformance fixtures: DependencySnapshot Identity and Trust Binding.
//
// Every fixture in tests/conformance/v1alpha4/wp3a-fixture-plan.json is exercised
// here. The fixture tree mirrors Appendix E.1-E.4 of the frozen CA-SL v1alpha4
// specification (ca-skills@7f84393). Negative cases use the normative Appendix
// E.6 minimal-change method: start from the successful baseline and change only
// the documented input.

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

const (
	wp3aFixtureRoot    = "../../tests/conformance/v1alpha4/fixtures/dependency-identity"
	wp3aSkillName      = "verify-ue-registration"
	wp3aContractPath   = "skills/verify-ue-registration/assurance/contract.yaml"
	wp3aProcedurePath  = "skills/verify-ue-registration/assurance/procedure.yaml"
	wp3aPredicatesPath = "skills/verify-ue-registration/assurance/predicates.yaml"

	wp3aHostContextPackageID = "urn:example:ca:host-context"
	wp3aToolsPackageID       = "urn:example:ca:tools"
)

var wp3aDependencyDiagnosticCodes = map[string]bool{
	"DEPENDENCY_REQUIREMENT_INVALID": true,
	"DEPENDENCY_ALIAS_MISSING":       true,
	"DEPENDENCY_SNAPSHOT_MISSING":    true,
	"DEPENDENCY_IDENTITY_UNVERIFIED": true,
	"DEPENDENCY_ID_MISMATCH":         true,
	"DEPENDENCY_DUPLICATE":           true,
	"UNUSED_DEPENDENCY":              true,
}

type wp3aFixture struct {
	files        map[string][]byte
	dependencies []ResolvedDependency
	bindings     []TrustedDependencyBinding
}

func readFixtureTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = data
		return nil
	})
	if err != nil {
		t.Fatalf("read fixture tree %s: %v", root, err)
	}
	if len(files) == 0 {
		t.Fatalf("fixture tree %s is empty", root)
	}
	return files
}

func fixtureFiles(files map[string][]byte) []File {
	paths := sortedStringMapKeys(files)
	out := make([]File, 0, len(paths))
	for _, path := range paths {
		out = append(out, File{Path: path, Bytes: files[path]})
	}
	return out
}

func snapshotFromTree(t *testing.T, dependencyID, root string) ResolvedDependency {
	t.Helper()
	files := fixtureFiles(readFixtureTree(t, filepath.Join(wp3aFixtureRoot, "base", "snapshots", root)))
	digest, err := ComputeDependencySnapshotDigest(files)
	if err != nil {
		t.Fatalf("compute digest for %s: %v", dependencyID, err)
	}
	return ResolvedDependency{DependencyID: dependencyID, Digest: digest, Files: files}
}

func wp3aBaseFixture(t *testing.T) wp3aFixture {
	t.Helper()
	fixture := wp3aFixture{
		files: readFixtureTree(t, filepath.Join(wp3aFixtureRoot, "base", "files")),
		dependencies: []ResolvedDependency{
			snapshotFromTree(t, "ca-context", "ca-context"),
			snapshotFromTree(t, "ca", "ca"),
		},
		bindings: []TrustedDependencyBinding{
			{
				DependencyID:     "ca-context",
				StablePackageID:  wp3aHostContextPackageID,
				AuthorizedOwners: []string{"example-host-owner"},
			},
			{
				DependencyID:     "ca",
				StablePackageID:  wp3aToolsPackageID,
				AuthorizedOwners: []string{"example-ue-owner"},
			},
		},
	}
	for index := range fixture.bindings {
		for _, dependency := range fixture.dependencies {
			if dependency.DependencyID == fixture.bindings[index].DependencyID {
				fixture.bindings[index].Digest = dependency.Digest
			}
		}
	}
	return fixture
}

func (fixture wp3aFixture) request(t *testing.T) BuildRequest {
	t.Helper()
	dependencies := append([]ResolvedDependency(nil), fixture.dependencies...)
	bindings := append([]TrustedDependencyBinding(nil), fixture.bindings...)
	return BuildRequest{
		APIVersion:           BuildRequestAPIVersion,
		LanguageTarget:       V1Alpha4Target,
		Files:                fixtureFiles(fixture.files),
		ResolvedDependencies: dependencies,
		TrustedBuildContext:  &TrustedBuildContext{DependencyBindings: bindings},
	}
}

func (fixture wp3aFixture) replaceFile(path, old, replacement string) {
	fixture.files[path] = []byte(strings.Replace(string(fixture.files[path]), old, replacement, 1))
}

func (fixture *wp3aFixture) withoutBinding(dependencyID string) {
	kept := make([]TrustedDependencyBinding, 0, len(fixture.bindings))
	for _, binding := range fixture.bindings {
		if binding.DependencyID != dependencyID {
			kept = append(kept, binding)
		}
	}
	fixture.bindings = kept
}

func (fixture *wp3aFixture) withoutDependency(dependencyID string) {
	kept := make([]ResolvedDependency, 0, len(fixture.dependencies))
	for _, dependency := range fixture.dependencies {
		if dependency.DependencyID != dependencyID {
			kept = append(kept, dependency)
		}
	}
	fixture.dependencies = kept
}

func wp3aParseSkillAST(t *testing.T, req BuildRequest, name string) *V1Alpha4SkillAST {
	t.Helper()
	base := "skills/" + name + "/"
	parsed := map[string]*yaml.Node{}
	for _, file := range req.Files {
		if !strings.HasPrefix(file.Path, base) {
			continue
		}
		if !strings.HasSuffix(file.Path, ".yaml") && !strings.HasSuffix(file.Path, ".yml") {
			continue
		}
		node, err := yaml.Parse(file.Bytes)
		if err != nil {
			t.Fatalf("parse fixture %s: %v", file.Path, err)
		}
		parsed[file.Path] = node
	}
	return parseV1Alpha4SkillAST(name, parsed)
}

func wp3aIdentity(t *testing.T, req BuildRequest) dependencyIdentityResult {
	t.Helper()
	return validateV1Alpha4DependencyIdentity(req, wp3aParseSkillAST(t, req, wp3aSkillName))
}

func wp3aDiagnostics(diagnostics []Diagnostic) []Diagnostic {
	filtered := make([]Diagnostic, 0)
	for _, diagnostic := range diagnostics {
		if wp3aDependencyDiagnosticCodes[diagnostic.Code] {
			filtered = append(filtered, diagnostic)
		}
	}
	sortDiagnostics(filtered)
	return filtered
}

func wp3aCheckBuildAgreement(t *testing.T, req BuildRequest) []Diagnostic {
	t.Helper()
	check := wp3aDiagnostics(Check(req))
	build := wp3aDiagnostics(Build(req).Diagnostics)
	if !reflect.DeepEqual(check, build) {
		t.Fatalf("Check and Build disagree on WP3A diagnostics:\ncheck=%+v\nbuild=%+v", check, build)
	}
	return check
}

func assertFailureAtomic(t *testing.T, result BuildResult) {
	t.Helper()
	if result.Success {
		t.Fatalf("WP3A slice must not claim Build success: %+v", result)
	}
	if len(result.OutputFiles) != 0 {
		t.Fatalf("failed Build must not emit output files: %+v", result.OutputFiles)
	}
	if len(result.DependencyClosure) != 0 {
		t.Fatalf("failed Build must not emit a partial dependency closure: %+v", result.DependencyClosure)
	}
}

func assertNoRequestEnvelopeDiagnostics(t *testing.T, req BuildRequest) {
	t.Helper()
	if diagnostics := validateBuildRequest(req); len(diagnostics) != 0 {
		t.Fatalf("fixture unexpectedly rejected at the request envelope: %+v", diagnostics)
	}
}

func expectOnlyWP3ADiagnostics(t *testing.T, diagnostics []Diagnostic, expected []Diagnostic) {
	t.Helper()
	if len(diagnostics) != len(expected) {
		t.Fatalf("WP3A diagnostics = %+v, want exactly %+v", diagnostics, expected)
	}
	for index := range expected {
		if !diagnostics[index].matchesExpected(expected[index]) {
			t.Fatalf("WP3A diagnostic[%d] = %+v, want %+v", index, diagnostics[index], expected[index])
		}
	}
}

// matchesExpected compares the frozen identity of a diagnostic plus the source
// location when the expectation carries one.
func (diagnostic Diagnostic) matchesExpected(expected Diagnostic) bool {
	if diagnostic.Code != expected.Code || diagnostic.Phase != expected.Phase || diagnostic.Severity != expected.Severity {
		return false
	}
	if expected.Path != diagnostic.Path {
		return false
	}
	if expected.Line != 0 && diagnostic.Line != expected.Line {
		return false
	}
	if expected.Column != 0 && diagnostic.Column != expected.Column {
		return false
	}
	return true
}

// WP3A-001: exact positive Alias -> snapshot -> trusted binding -> Package ID.
func TestWP3A001PositiveIdentityAndPermutationStability(t *testing.T) {
	req := wp3aBaseFixture(t).request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)

	identity := wp3aIdentity(t, req)
	if len(identity.Diagnostics) != 0 {
		t.Fatalf("positive identity emitted WP3A diagnostics: %+v", identity.Diagnostics)
	}
	wantIdentities := []dependencyIdentity{
		{Alias: "ca", StablePackageID: wp3aToolsPackageID},
		{Alias: "ca-context", StablePackageID: wp3aHostContextPackageID},
	}
	if len(identity.Identities) != len(wantIdentities) {
		t.Fatalf("resolved identities = %+v, want %+v", identity.Identities, wantIdentities)
	}
	for index, want := range wantIdentities {
		got := identity.Identities[index]
		if got.Alias != want.Alias || got.StablePackageID != want.StablePackageID {
			t.Fatalf("identity[%d] = %+v, want %+v", index, got, want)
		}
		if !validDigest(got.MeasuredDigest) {
			t.Fatalf("identity[%d] measured digest is not a frozen sha256 digest: %q", index, got.MeasuredDigest)
		}
		for _, dependency := range req.ResolvedDependencies {
			if dependency.DependencyID == got.Alias && dependency.Digest != got.MeasuredDigest {
				t.Fatalf("identity[%d] digest is not the measured original-byte digest", index)
			}
		}
	}
	if want := []string{"ca", "ca-context"}; !reflect.DeepEqual(identity.Closure, want) {
		t.Fatalf("dependency closure = %v, want %v", identity.Closure, want)
	}
	if len(wp3aCheckBuildAgreement(t, req)) != 0 {
		t.Fatal("positive identity must not emit WP3A diagnostics through Check/Build")
	}

	result := Build(req)
	assertFailureAtomic(t, result)
	if !hasCode(result.Diagnostics, "CAPABILITY_INPUT_UNTRUSTED") {
		t.Fatalf("later unimplemented stages must still fail closed: %+v", result.Diagnostics)
	}

	// Reorder source files, snapshot files, snapshots and bindings. The measured
	// identity tuple and the diagnostics must not change.
	permuted := req
	permuted.Files = reverseFiles(req.Files)
	permuted.ResolvedDependencies = make([]ResolvedDependency, len(req.ResolvedDependencies))
	for index, dependency := range req.ResolvedDependencies {
		permuted.ResolvedDependencies[len(req.ResolvedDependencies)-1-index] = ResolvedDependency{
			DependencyID: dependency.DependencyID,
			Digest:       dependency.Digest,
			Files:        reverseFiles(dependency.Files),
		}
	}
	permutedBindings := append([]TrustedDependencyBinding(nil), req.TrustedBuildContext.DependencyBindings...)
	sort.Slice(permutedBindings, func(i, j int) bool { return permutedBindings[i].DependencyID > permutedBindings[j].DependencyID })
	permuted.TrustedBuildContext = &TrustedBuildContext{DependencyBindings: permutedBindings}

	permutedIdentity := wp3aIdentity(t, permuted)
	if !reflect.DeepEqual(identity.Identities, permutedIdentity.Identities) ||
		!reflect.DeepEqual(identity.Closure, permutedIdentity.Closure) ||
		len(permutedIdentity.Diagnostics) != 0 {
		t.Fatalf("identity is not permutation stable: %+v vs %+v", identity, permutedIdentity)
	}
	if !reflect.DeepEqual(Check(req), Check(permuted)) {
		t.Fatalf("Check diagnostics are not permutation stable:\n%+v\n%+v", Check(req), Check(permuted))
	}
}

func reverseFiles(files []File) []File {
	reversed := make([]File, len(files))
	for index, file := range files {
		reversed[len(files)-1-index] = file
	}
	return reversed
}

// WP3A-002 / E03: Contract Package ID differs from the trusted Package ID.
func TestWP3A002DependencyPackageIDMismatch(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aContractPath, "ca: urn:example:ca:tools", "ca: urn:example:ca:other-tools")
	req := fixture.request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)

	diagnostics := wp3aCheckBuildAgreement(t, req)
	expectOnlyWP3ADiagnostics(t, diagnostics, []Diagnostic{{
		Severity: "error",
		Phase:    "C_STATIC",
		Code:     "DEPENDENCY_ID_MISMATCH",
		Path:     wp3aContractPath,
		Line:     5,
		Column:   7,
	}})
	result := Build(req)
	assertFailureAtomic(t, result)
	for _, diagnostic := range result.Diagnostics {
		if strings.Contains(strings.ToLower(diagnostic.Message), "filename") {
			t.Fatalf("identity must not be inferred from a filename: %+v", diagnostic)
		}
	}
}

// WP3A-003 / E04: trusted dependency binding is missing.
func TestWP3A003MissingTrustedBinding(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.withoutBinding("ca")
	req := fixture.request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)

	diagnostics := wp3aCheckBuildAgreement(t, req)
	expectOnlyWP3ADiagnostics(t, diagnostics, []Diagnostic{{
		Severity: "error",
		Phase:    "C_STATIC",
		Code:     "DEPENDENCY_IDENTITY_UNVERIFIED",
		Path:     "",
	}})
	if diagnostics[0].Line != 0 || diagnostics[0].Column != 0 {
		t.Fatalf("trusted metadata diagnostic must use zero location: %+v", diagnostics[0])
	}
	assertFailureAtomic(t, Build(req))
}

// WP3A-004 / E04: trusted binding refers to a different measured digest.
func TestWP3A004BindingDigestMismatchIsUnverified(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	req := fixture.request(t)
	invalidDigest := "sha256:" + strings.Repeat("0", 64)
	for index := range req.TrustedBuildContext.DependencyBindings {
		if req.TrustedBuildContext.DependencyBindings[index].DependencyID == "ca" {
			req.TrustedBuildContext.DependencyBindings[index].Digest = invalidDigest
		}
	}
	diagnostics := wp3aCheckBuildAgreement(t, req)
	expectOnlyWP3ADiagnostics(t, diagnostics, []Diagnostic{{
		Severity: "error",
		Phase:    "C_STATIC",
		Code:     "DEPENDENCY_IDENTITY_UNVERIFIED",
		Path:     "",
	}})
	assertFailureAtomic(t, Build(req))

	// Unit-level: matching dependencyId alone is insufficient; the exact measured
	// digest is part of the binding lookup key.
	unitReq := wp3aBaseFixture(t).request(t)
	unitReq.TrustedBuildContext.DependencyBindings = []TrustedDependencyBinding{{
		DependencyID:     "ca",
		Digest:           invalidDigest,
		StablePackageID:  wp3aToolsPackageID,
		AuthorizedOwners: []string{"example-ue-owner"},
	}}
	unit := wp3aIdentity(t, unitReq)
	found := false
	for _, diagnostic := range unit.Diagnostics {
		if diagnostic.Code == "DEPENDENCY_IDENTITY_UNVERIFIED" {
			found = true
		}
	}
	if !found {
		t.Fatalf("digest-only binding mismatch was not rejected: %+v", unit.Diagnostics)
	}
}

// WP3A-005: a used Contract Alias has no supplied snapshot.
func TestWP3A005UsedAliasSnapshotMissing(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.withoutDependency("ca")
	req := fixture.request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)

	diagnostics := wp3aCheckBuildAgreement(t, req)
	expectOnlyWP3ADiagnostics(t, diagnostics, []Diagnostic{{
		Severity: "error",
		Phase:    "C_STATIC",
		Code:     "DEPENDENCY_SNAPSHOT_MISSING",
		Path:     "",
	}})
	result := Build(req)
	assertFailureAtomic(t, result)
	for _, masked := range []string{"EXPORT_ID_MISSING", "SCHEMA_MISSING"} {
		if hasCode(result.Diagnostics, masked) {
			t.Fatalf("missing snapshot must not be replaced with %s: %+v", masked, result.Diagnostics)
		}
	}
}

// WP3A-006: multiple snapshots or trusted bindings ambiguously match one Alias.
func TestWP3A006DuplicateSnapshotAndBinding(t *testing.T) {
	t.Run("duplicate snapshot", func(t *testing.T) {
		fixture := wp3aBaseFixture(t)
		duplicate := fixture.dependencies[1]
		fixture.dependencies = append(fixture.dependencies, duplicate)
		req := fixture.request(t)
		diagnostics := wp3aCheckBuildAgreement(t, req)
		expectOnlyWP3ADiagnostics(t, diagnostics, []Diagnostic{{
			Severity: "error",
			Phase:    "C_STATIC",
			Code:     "DEPENDENCY_DUPLICATE",
			Path:     "",
		}})
		assertFailureAtomic(t, Build(req))
	})

	t.Run("duplicate trusted binding", func(t *testing.T) {
		fixture := wp3aBaseFixture(t)
		req := fixture.request(t)
		duplicate := req.TrustedBuildContext.DependencyBindings[1]
		forward := append([]TrustedDependencyBinding(nil), req.TrustedBuildContext.DependencyBindings...)
		forward = append(forward, duplicate)
		backward := append([]TrustedDependencyBinding{duplicate}, req.TrustedBuildContext.DependencyBindings...)
		for name, bindings := range map[string][]TrustedDependencyBinding{"forward": forward, "backward": backward} {
			req.TrustedBuildContext = &TrustedBuildContext{DependencyBindings: bindings}
			diagnostics := wp3aCheckBuildAgreement(t, req)
			expectOnlyWP3ADiagnostics(t, diagnostics, []Diagnostic{{
				Severity: "error",
				Phase:    "C_STATIC",
				Code:     "DEPENDENCY_DUPLICATE",
				Path:     "",
			}})
			if name == "forward" {
				assertFailureAtomic(t, Build(req))
			}
		}
	})
}

// WP3A-007 / E35: Contract dependency value is not a Stable Package ID string.
func TestWP3A007DependencyRequirementInvalid(t *testing.T) {
	cases := map[string]string{
		"mapping value":  "ca:\n    id: urn:example:ca:tools",
		"sequence value": "ca: [urn:example:ca:tools]",
		"relative value": "ca: relative/package",
		"empty value":    "ca: \"\"",
	}
	for name, replacement := range cases {
		t.Run(name, func(t *testing.T) {
			fixture := wp3aBaseFixture(t)
			fixture.replaceFile(wp3aContractPath, "ca: urn:example:ca:tools", replacement)
			req := fixture.request(t)
			assertNoRequestEnvelopeDiagnostics(t, req)
			diagnostics := wp3aCheckBuildAgreement(t, req)
			if len(diagnostics) != 1 || diagnostics[0].Code != "DEPENDENCY_REQUIREMENT_INVALID" ||
				diagnostics[0].Phase != "C_STATIC" || diagnostics[0].Severity != "error" {
				t.Fatalf("WP3A diagnostics = %+v, want one DEPENDENCY_REQUIREMENT_INVALID/C_STATIC", diagnostics)
			}
			if diagnostics[0].Path != wp3aContractPath || diagnostics[0].Line == 0 || diagnostics[0].Column == 0 {
				t.Fatalf("Contract source location was not retained: %+v", diagnostics[0])
			}
			assertFailureAtomic(t, Build(req))
		})
	}
}

// WP3A-008: a supported external reference uses an undeclared Alias.
func TestWP3A008UndeclaredAliasReference(t *testing.T) {
	t.Run("callable reference", func(t *testing.T) {
		fixture := wp3aBaseFixture(t)
		fixture.replaceFile(wp3aProcedurePath, "tool: ca::ue.registration.current", "tool: undeclared::ue.registration.current")
		// A string containing "::" in an argument is not a supported reference
		// position and must not be scanned.
		fixture.replaceFile(wp3aProcedurePath, "ueRef: context.ueRef", "ueRef: context.ueRef\n        note: 'arbitrary::text'")
		req := fixture.request(t)
		assertNoRequestEnvelopeDiagnostics(t, req)
		diagnostics := wp3aCheckBuildAgreement(t, req)
		if len(diagnostics) != 1 || diagnostics[0].Code != "DEPENDENCY_ALIAS_MISSING" || diagnostics[0].Phase != "C_STATIC" {
			t.Fatalf("WP3A diagnostics = %+v, want one DEPENDENCY_ALIAS_MISSING/C_STATIC", diagnostics)
		}
		if diagnostics[0].Path != wp3aProcedurePath || diagnostics[0].Line == 0 || diagnostics[0].Column == 0 {
			t.Fatalf("reference source location was not retained: %+v", diagnostics[0])
		}
		assertFailureAtomic(t, Build(req))
	})

	t.Run("external schema reference in Contract", func(t *testing.T) {
		fixture := wp3aBaseFixture(t)
		fixture.replaceFile(wp3aContractPath, "ca-context::urn:example:ca:invocation-context:v1", "missing-context::urn:example:ca:invocation-context:v1")
		fixture.replaceFile(wp3aContractPath, "  ca-context: urn:example:ca:host-context\n", "")
		req := fixture.request(t)
		diagnostics := wp3aCheckBuildAgreement(t, req)
		if len(diagnostics) != 1 || diagnostics[0].Code != "DEPENDENCY_ALIAS_MISSING" {
			t.Fatalf("WP3A diagnostics = %+v, want one DEPENDENCY_ALIAS_MISSING", diagnostics)
		}
	})

	t.Run("external schema reference in predicate parameter", func(t *testing.T) {
		fixture := wp3aBaseFixture(t)
		fixture.replaceFile(wp3aPredicatesPath, "registration: ca::urn:example:ca:registration:v1", "registration: missing-schema::urn:example:ca:registration:v1")
		req := fixture.request(t)
		diagnostics := wp3aCheckBuildAgreement(t, req)
		if len(diagnostics) != 1 || diagnostics[0].Code != "DEPENDENCY_ALIAS_MISSING" {
			t.Fatalf("WP3A diagnostics = %+v, want one DEPENDENCY_ALIAS_MISSING", diagnostics)
		}
	})

	t.Run("external schema $ref in Skill-owned schema", func(t *testing.T) {
		fixture := wp3aBaseFixture(t)
		fixture.replaceFile(wp3aContractPath, "contextSchema:", "requestSchema: schemas/request.schema.yaml\ncontextSchema:")
		fixture.files["skills/verify-ue-registration/schemas/request.schema.yaml"] = []byte(`$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:request:v1
$ref: missing-schema::urn:example:external:request:v1
`)
		req := fixture.request(t)
		diagnostics := wp3aCheckBuildAgreement(t, req)
		if len(diagnostics) != 1 || diagnostics[0].Code != "DEPENDENCY_ALIAS_MISSING" {
			t.Fatalf("WP3A diagnostics = %+v, want one DEPENDENCY_ALIAS_MISSING", diagnostics)
		}
		if diagnostics[0].Path != "skills/verify-ue-registration/schemas/request.schema.yaml" {
			t.Fatalf("schema reference location was not retained: %+v", diagnostics[0])
		}
	})
}

// WP3A-009 / E16: an unreferenced extra snapshot cannot become a global export.
func TestWP3A009UnreferencedExtraSnapshot(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	extra := snapshotFromTree(t, "unreferenced-extra", "unreferenced-extra")
	fixture.dependencies = append(fixture.dependencies, extra)
	req := fixture.request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)

	if diagnostics := wp3aCheckBuildAgreement(t, req); len(diagnostics) != 0 {
		t.Fatalf("extra unreferenced snapshot created WP3A diagnostics: %+v", diagnostics)
	}
	identity := wp3aIdentity(t, req)
	if want := []string{"ca", "ca-context"}; !reflect.DeepEqual(identity.Closure, want) {
		t.Fatalf("extra snapshot entered the selected closure: %v", identity.Closure)
	}
	for _, entry := range identity.Identities {
		if entry.Alias == "unreferenced-extra" {
			t.Fatal("extra snapshot was exposed as a dependency identity")
		}
	}
	result := Build(req)
	assertFailureAtomic(t, result)

	// Request-level raw-byte integrity still applies to every supplied snapshot,
	// including ones outside the selected closure.
	corrupted := wp3aBaseFixture(t)
	corruptedExtra := snapshotFromTree(t, "unreferenced-extra", "unreferenced-extra")
	corruptedExtra.Digest = "sha256:" + strings.Repeat("0", 64)
	corrupted.dependencies = append(corrupted.dependencies, corruptedExtra)
	corruptedReq := corrupted.request(t)
	if !hasCode(wp3aCheckBuildAgreement(t, corruptedReq), "DEPENDENCY_IDENTITY_UNVERIFIED") {
		t.Fatal("extra snapshot bytes escaped request-level digest verification")
	}
}

// WP3A-010: a declared but unused Contract dependency is a C_WARNING.
func TestWP3A010UnusedDependencyWarning(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aContractPath, "  ca: urn:example:ca:tools", "  ca: urn:example:ca:tools\n  ca-unused: urn:example:ca:unused")
	req := fixture.request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)

	diagnostics := wp3aCheckBuildAgreement(t, req)
	if len(diagnostics) != 1 || diagnostics[0].Code != "UNUSED_DEPENDENCY" {
		t.Fatalf("WP3A diagnostics = %+v, want only UNUSED_DEPENDENCY", diagnostics)
	}
	warning := diagnostics[0]
	if warning.Severity != "warning" || warning.Phase != "C_WARNING" {
		t.Fatalf("UNUSED_DEPENDENCY must be warning/C_WARNING: %+v", warning)
	}
	if warning.Path != wp3aContractPath || warning.Line == 0 || warning.Column == 0 {
		t.Fatalf("unused dependency location was not retained: %+v", warning)
	}
	identity := wp3aIdentity(t, req)
	if want := []string{"ca", "ca-context"}; !reflect.DeepEqual(identity.Closure, want) {
		t.Fatalf("unused Alias entered the dependency closure: %v", identity.Closure)
	}
	result := Build(req)
	assertFailureAtomic(t, result)
	if hasCode(result.Diagnostics, "UNUSED_DEPENDENCY") && !hasCode(result.Diagnostics, "CAPABILITY_INPUT_UNTRUSTED") {
		t.Fatalf("the warning itself must not be the only failure cause: %+v", result.Diagnostics)
	}
}

// Multiple selected Skills must keep their Alias sets and closures isolated.
func TestWP3AMultipleSkillsDoNotLeakDependencies(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	otherSkill := "other-skill"
	base := "skills/" + otherSkill + "/"
	fixture.files[base+"SKILL.md"] = []byte("---\nname: other-skill\n---\n")
	fixture.files[base+"assurance/contract.yaml"] = []byte(`goal: other.goal
dependencies:
  shared: urn:example:ca:shared
`)
	fixture.files[base+"assurance/procedure.yaml"] = []byte(`start: act
nodes:
  act:
    operation:
      tool: ca::ue.registration.current
      arguments: {}
      routes:
        SUCCEEDED: SUCCEEDED
        FAILED: FAILED
`)
	fixture.files[base+"assurance/predicates.yaml"] = []byte("predicates: {}\n")
	req := fixture.request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)

	diagnostics := wp3aCheckBuildAgreement(t, req)
	expected := map[string]bool{
		"DEPENDENCY_ALIAS_MISSING": true,
		"UNUSED_DEPENDENCY":        true,
	}
	if len(diagnostics) != 2 {
		t.Fatalf("cross-Skill isolation diagnostics = %+v, want alias-missing + unused warning", diagnostics)
	}
	for _, diagnostic := range diagnostics {
		if !expected[diagnostic.Code] {
			t.Fatalf("unexpected isolation diagnostic: %+v", diagnostic)
		}
		if !strings.HasPrefix(diagnostic.Path, base) {
			t.Fatalf("diagnostic leaked from the other Skill: %+v", diagnostic)
		}
	}
	for _, diagnostic := range diagnostics {
		if strings.HasPrefix(diagnostic.Path, "skills/"+wp3aSkillName+"/") {
			t.Fatalf("first Skill was affected by the second Skill: %+v", diagnostic)
		}
	}
}

// validStablePackageID is a frozen identity lexer: no normalization or coercion.
func TestWP3AStablePackageIDLexer(t *testing.T) {
	valid := []string{
		"urn:example:ca:tools",
		"urn:example:connectionagent:5gc-tool-contracts",
		"https://example.test/packages/ca-tools",
		"pkg+json:example/tools",
	}
	for _, value := range valid {
		if !validStablePackageID(value) {
			t.Fatalf("valid Stable Package ID rejected: %q", value)
		}
	}
	invalid := []string{
		"",
		"relative/package",
		"urn:",
		"Urn example:x",
		"urn:example/../escape",
		"urn:example: /space",
		"urn:example:tools\\path",
		"1urn:example",
	}
	for _, value := range invalid {
		if validStablePackageID(value) {
			t.Fatalf("invalid Stable Package ID accepted: %q", value)
		}
	}
}
