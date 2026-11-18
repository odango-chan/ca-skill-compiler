package compiler

// WP3C conformance fixtures: trusted CapabilityContracts deployment input, exact
// callable membership, Effect consistency and Scope-policy presence. Derived from
// Appendix E.4 and the frozen minimal-change cases E06/E07.

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	wp3cFixtureRoot      = "../../tests/conformance/v1alpha4/fixtures/capability-linking"
	wp3cDeploymentID     = "urn:example:deployment:smf"
	wp3cPublisherID      = "urn:example:publisher:deployment"
	wp3cAuthorizationRef = "urn:example:authorization:example-deployment"
)

var wp3cDiagnosticCodes = map[string]bool{
	"CAPABILITY_INPUT_UNTRUSTED":          true,
	"CAPABILITY_DEFINITION_CONFLICT":      true,
	"CAPABILITY_MEMBERSHIP_DUPLICATE":     true,
	"CAPABILITY_SCOPE_UNRESOLVED":         true,
	"CAPABILITY_SCOPE_EMPTY_INTERSECTION": true,
	"CAPABILITY_NOT_GRANTED":              true,
	"CAPABILITY_MAPPING_MISSING":          true,
	"EFFECT_MISMATCH":                     true,
	"IDENTIFIER_INVALID":                  true,
}

func wp3cBaseBundle(t *testing.T) map[string][]byte {
	t.Helper()
	return readFixtureTree(t, filepath.Join(wp3cFixtureRoot, "base"))
}

func wp3cContext(t *testing.T, files []File) *TrustedBuildContext {
	t.Helper()
	digest, err := ComputeDependencySnapshotDigest(files)
	if err != nil {
		t.Fatalf("compute capability bundle digest: %v", err)
	}
	authorizations := []DeploymentAuthorization{{
		DeploymentID:     wp3cDeploymentID,
		PublisherID:      wp3cPublisherID,
		AuthorizationRef: wp3cAuthorizationRef,
	}}
	return &TrustedBuildContext{
		CapabilityContracts: &CapabilityContractsBundle{
			DeploymentID: wp3cDeploymentID,
			PublisherID:  wp3cPublisherID,
			Digest:       digest,
			Files:        files,
		},
		DeploymentAuthorizations: &authorizations,
	}
}

func wp3cRequest(t *testing.T, fixture wp3aFixture, bundle map[string][]byte) BuildRequest {
	t.Helper()
	req := fixture.request(t)
	context := wp3cContext(t, fixtureFiles(bundle))
	context.DependencyBindings = req.TrustedBuildContext.DependencyBindings
	req.TrustedBuildContext = context
	return req
}

func wp3cLinking(t *testing.T, req BuildRequest) capabilityLinkingResult {
	t.Helper()
	identity := wp3aIdentity(t, req)
	ownerLinking := validateOwnerLinking(req, wp3aParseSkillAST(t, req, wp3aSkillName), identity)
	return validateCapabilityLinking(req, wp3aParseSkillAST(t, req, wp3aSkillName), ownerLinking)
}

func wp3cDiagnostics(diagnostics []Diagnostic) []Diagnostic {
	filtered := make([]Diagnostic, 0)
	for _, diagnostic := range diagnostics {
		if strings.HasPrefix(diagnostic.Message, "fail-closed:") {
			continue
		}
		if wp3cDiagnosticCodes[diagnostic.Code] {
			filtered = append(filtered, diagnostic)
		}
	}
	sortDiagnostics(filtered)
	return filtered
}

func wp3cCheckBuildAgreement(t *testing.T, req BuildRequest) []Diagnostic {
	t.Helper()
	check := wp3cDiagnostics(Check(req))
	build := wp3cDiagnostics(Build(req).Diagnostics)
	if !reflect.DeepEqual(check, build) {
		t.Fatalf("Check and Build disagree on WP3C diagnostics:\ncheck=%+v\nbuild=%+v", check, build)
	}
	return check
}

func expectOnlyWP3CDiagnostics(t *testing.T, diagnostics []Diagnostic, code string, count int) {
	t.Helper()
	if len(diagnostics) != count {
		t.Fatalf("WP3C diagnostics = %+v, want %d x %s", diagnostics, count, code)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code != code || diagnostic.Phase != "C_STATIC" || diagnostic.Severity != "error" {
			t.Fatalf("WP3C diagnostic = %+v, want %s/C_STATIC", diagnostic, code)
		}
		if diagnostic.Path != "" || diagnostic.Line != 0 || diagnostic.Column != 0 {
			t.Fatalf("trusted deployment diagnostic must use empty path and zero location: %+v", diagnostic)
		}
	}
}

// WP3C-001: positive trusted CapabilityContracts link.
func TestWP3C001PositiveCapabilityLink(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	req := wp3cRequest(t, fixture, wp3cBaseBundle(t))
	assertNoRequestEnvelopeDiagnostics(t, req)

	linking := wp3cLinking(t, req)
	if len(linking.Diagnostics) != 0 {
		t.Fatalf("positive capability link emitted diagnostics: %+v", linking.Diagnostics)
	}
	if want := []string{"example.registration-read"}; !reflect.DeepEqual(linking.Grants, want) {
		t.Fatalf("parsed Skill grants = %v, want %v", linking.Grants, want)
	}
	if len(linking.Capabilities) != 1 || linking.Capabilities[0].id != "example.registration-read" {
		t.Fatalf("parsed capabilities = %+v", linking.Capabilities)
	}
	if diagnostics := wp3cCheckBuildAgreement(t, req); len(diagnostics) != 0 {
		t.Fatalf("positive capability link emitted diagnostics through Check/Build: %+v", diagnostics)
	}
	result := Build(req)
	if !result.Success {
		t.Fatalf("fully grounded positive capability link must Build after gate removal: %+v", result.Diagnostics)
	}
	if len(result.OutputFiles) == 0 {
		t.Fatal("successful Build emitted no artifact files")
	}

	// Bundle file order must not change the result.
	twoFiles := wp3cBaseBundle(t)
	twoFiles["second-capability.yaml"] = []byte(`apiVersion: connectionagent.io/capability-contracts/v1alpha4
kind: CapabilityContracts
spec:
  capabilities:
    - id: example.other-read
      owner: deployment
      callables:
        - owner: example-other-owner
          id: other.observation.read
          kind: observation
          effect: READ
          scopePolicyRef: deployment.other-read
`)
	ordered := fixtureFiles(twoFiles)
	reversed := make([]File, len(ordered))
	for index := range ordered {
		reversed[len(ordered)-1-index] = ordered[index]
	}
	permutedReq := wp3aBaseFixture(t).request(t)
	firstContext := wp3cContext(t, ordered)
	firstContext.DependencyBindings = permutedReq.TrustedBuildContext.DependencyBindings
	permutedReq.TrustedBuildContext = firstContext
	firstLinking := wp3cLinking(t, permutedReq)
	secondReq := wp3aBaseFixture(t).request(t)
	secondContext := wp3cContext(t, reversed)
	secondContext.DependencyBindings = secondReq.TrustedBuildContext.DependencyBindings
	secondReq.TrustedBuildContext = secondContext
	secondLinking := wp3cLinking(t, secondReq)
	if len(firstLinking.Diagnostics) != 0 || len(secondLinking.Diagnostics) != 0 {
		t.Fatalf("valid two-file bundle emitted diagnostics: %+v / %+v", firstLinking.Diagnostics, secondLinking.Diagnostics)
	}
	if !reflect.DeepEqual(firstLinking.Capabilities, secondLinking.Capabilities) {
		t.Fatalf("capability indexing is not file-order stable: %+v vs %+v", firstLinking.Capabilities, secondLinking.Capabilities)
	}
}

// WP3C-002 / E06: the selected Skill grants no Capability for the callable.
func TestWP3C002CapabilityNotGrantedE06(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aContractPath, "capabilities:\n  - example.registration-read", "capabilities: []")
	req := wp3cRequest(t, fixture, wp3cBaseBundle(t))
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3cCheckBuildAgreement(t, req)
	expectOnlyWP3CDiagnostics(t, diagnostics, "CAPABILITY_NOT_GRANTED", 1)
	assertFailureAtomic(t, Build(req))
}

// WP3C-003 / E07: deployment-side member Effect conflicts with ToolCatalog.
func TestWP3C003EffectMismatchE07(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	bundle := wp3cBaseBundle(t)
	bundle["capability-contracts.yaml"] = []byte(strings.Replace(string(bundle["capability-contracts.yaml"]), "effect: READ", "effect: MUTATION", 1))
	req := wp3cRequest(t, fixture, bundle)
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3cCheckBuildAgreement(t, req)
	expectOnlyWP3CDiagnostics(t, diagnostics, "EFFECT_MISMATCH", 1)
	assertFailureAtomic(t, Build(req))
}

// WP3C-004: the callable has no Capability member record at all.
func TestWP3C004CapabilityMappingMissing(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	bundle := wp3cBaseBundle(t)
	bundle["capability-contracts.yaml"] = []byte(strings.Replace(string(bundle["capability-contracts.yaml"]), "owner: example-ue-owner", "owner: example-other-owner", 1))
	req := wp3cRequest(t, fixture, bundle)
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3cCheckBuildAgreement(t, req)
	expectOnlyWP3CDiagnostics(t, diagnostics, "CAPABILITY_MAPPING_MISSING", 1)
	assertFailureAtomic(t, Build(req))
}

// WP3C-005: a callable reference exists but no trusted deployment input is supplied.
func TestWP3C005MissingTrustedDeploymentInput(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	req := fixture.request(t)
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3cCheckBuildAgreement(t, req)
	expectOnlyWP3CDiagnostics(t, diagnostics, "CAPABILITY_INPUT_UNTRUSTED", 1)
	assertFailureAtomic(t, Build(req))
}

// WP3C-006: deployment authorization does not bind the supplied deployment.
func TestWP3C006DeploymentAuthorizationMismatch(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	req := wp3cRequest(t, fixture, wp3cBaseBundle(t))
	authorizations := []DeploymentAuthorization{{
		DeploymentID:     "urn:example:deployment:other",
		PublisherID:      wp3cPublisherID,
		AuthorizationRef: wp3cAuthorizationRef,
	}}
	req.TrustedBuildContext.DeploymentAuthorizations = &authorizations
	diagnostics := wp3cCheckBuildAgreement(t, req)
	expectOnlyWP3CDiagnostics(t, diagnostics, "CAPABILITY_INPUT_UNTRUSTED", 1)
	assertFailureAtomic(t, Build(req))
}

// WP3C-007: the same Capability ID is defined more than once in the artifact set.
func TestWP3C007CapabilityDefinitionConflict(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	bundle := wp3cBaseBundle(t)
	bundle["second-capability.yaml"] = []byte(`apiVersion: connectionagent.io/capability-contracts/v1alpha4
kind: CapabilityContracts
spec:
  capabilities:
    - id: example.registration-read
      owner: deployment
      callables:
        - owner: example-ue-owner
          id: ue.registration.current
          kind: observation
          effect: READ
          scopePolicyRef: deployment.registration-read
`)
	req := wp3cRequest(t, fixture, bundle)
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3cCheckBuildAgreement(t, req)
	expectOnlyWP3CDiagnostics(t, diagnostics, "CAPABILITY_DEFINITION_CONFLICT", 1)
	assertFailureAtomic(t, Build(req))
}

// WP3C-008: the same member is declared twice inside one Capability.
func TestWP3C008CapabilityMembershipDuplicate(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	bundle := wp3cBaseBundle(t)
	bundle["capability-contracts.yaml"] = []byte(strings.Replace(string(bundle["capability-contracts.yaml"]), `          scopePolicyRef: deployment.registration-read`, `          scopePolicyRef: deployment.registration-read
        - owner: example-ue-owner
          id: ue.registration.current
          kind: observation
          effect: READ
          scopePolicyRef: deployment.registration-read`, 1))
	req := wp3cRequest(t, fixture, bundle)
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3cCheckBuildAgreement(t, req)
	expectOnlyWP3CDiagnostics(t, diagnostics, "CAPABILITY_MEMBERSHIP_DUPLICATE", 1)
	assertFailureAtomic(t, Build(req))
}

// WP3C-009: a granted member has no trusted resolvable Scope policy reference.
func TestWP3C009CapabilityScopeUnresolved(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	bundle := wp3cBaseBundle(t)
	bundle["capability-contracts.yaml"] = []byte(strings.Replace(string(bundle["capability-contracts.yaml"]), "          scopePolicyRef: deployment.registration-read\n", "", 1))
	req := wp3cRequest(t, fixture, bundle)
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3cCheckBuildAgreement(t, req)
	expectOnlyWP3CDiagnostics(t, diagnostics, "CAPABILITY_SCOPE_UNRESOLVED", 1)
	assertFailureAtomic(t, Build(req))
}

// WP3C-010: an ungranted member still participates in Effect consistency checks.
func TestWP3C010UngrantedMemberEffectConflict(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	bundle := map[string][]byte{"capability-contracts.yaml": []byte(`apiVersion: connectionagent.io/capability-contracts/v1alpha4
kind: CapabilityContracts
spec:
  capabilities:
    - id: example.registration-read
      owner: deployment
      callables:
        - owner: example-ue-owner
          id: ue.registration.current
          kind: observation
          effect: READ
          scopePolicyRef: deployment.registration-read
    - id: example.other-authorization
      owner: deployment
      callables:
        - owner: example-ue-owner
          id: ue.registration.current
          kind: observation
          effect: MUTATION
          scopePolicyRef: deployment.other-authorization
`)}
	req := wp3cRequest(t, fixture, bundle)
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3cCheckBuildAgreement(t, req)
	expectOnlyWP3CDiagnostics(t, diagnostics, "EFFECT_MISMATCH", 1)
	assertFailureAtomic(t, Build(req))
}

// WP3C-011: an invalid CapabilityId in the Skill Grant list is an identifier error.
func TestWP3C011InvalidGrantIdentifier(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aContractPath, "capabilities:\n  - example.registration-read", "capabilities:\n  - example.registration-read\n  - \"bad capability id\"")
	req := wp3cRequest(t, fixture, wp3cBaseBundle(t))
	assertNoRequestEnvelopeDiagnostics(t, req)
	diagnostics := wp3cCheckBuildAgreement(t, req)
	if len(diagnostics) != 1 || diagnostics[0].Code != "IDENTIFIER_INVALID" || diagnostics[0].Phase != "C_STATIC" {
		t.Fatalf("WP3C diagnostics = %+v, want one IDENTIFIER_INVALID/C_STATIC", diagnostics)
	}
	if diagnostics[0].Path != wp3aContractPath || diagnostics[0].Line == 0 || diagnostics[0].Column == 0 {
		t.Fatalf("Contract source location was not retained: %+v", diagnostics[0])
	}
}
