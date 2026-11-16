package compiler

// WP3D conformance fixtures: snapshot-scoped DependencyInspection, selected-Skill
// closure isolation and same-snapshot Reconcile dedicated-READ anti-bypass.

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const wp3dFixtureRoot = "../../tests/conformance/v1alpha4/fixtures/integrated-linking"

var wp3dDiagnosticCodes = map[string]bool{
	"RECONCILE_CONTRACT_INVALID":         true,
	"EXPORT_ID_AMBIGUOUS":                true,
	"DEPENDENCY_IDENTITY_UNVERIFIED":     true,
	"DEPENDENCY_OWNER_MISMATCH":          true,
	"DEPENDENCY_EXPORT_OWNER_UNVERIFIED": true,
}

func wp3dInspectRequest(t *testing.T, fixture wp3aFixture) BuildRequest {
	t.Helper()
	req := validEnvelopeRequest()
	req.ResolvedDependencies = append([]ResolvedDependency(nil), fixture.dependencies...)
	bindings := append([]TrustedDependencyBinding(nil), fixture.bindings...)
	req.TrustedBuildContext = &TrustedBuildContext{DependencyBindings: bindings}
	return req
}

func wp3dSessionCatalog(t *testing.T) []byte {
	t.Helper()
	files := readFixtureTree(t, filepath.Join(wp3dFixtureRoot, "base"))
	content, ok := files["session-catalog.yaml"]
	if !ok {
		t.Fatal("integrated-linking fixture session-catalog.yaml is missing")
	}
	return content
}

func wp3dCheckBuildDiagnostics(t *testing.T, req BuildRequest) []Diagnostic {
	t.Helper()
	check := wp3dDiagnostics(Check(req))
	build := wp3dDiagnostics(Build(req).Diagnostics)
	if !reflect.DeepEqual(check, build) {
		t.Fatalf("Check and Build disagree on WP3D diagnostics:\ncheck=%+v\nbuild=%+v", check, build)
	}
	return check
}

func wp3dDiagnostics(diagnostics []Diagnostic) []Diagnostic {
	filtered := make([]Diagnostic, 0)
	for _, diagnostic := range diagnostics {
		if diagnostic.Message == v1alpha4ResidualFailClosedMessage {
			continue
		}
		if wp3dDiagnosticCodes[diagnostic.Code] {
			filtered = append(filtered, diagnostic)
		}
	}
	sortDiagnostics(filtered)
	return filtered
}

func expectOnlyWP3DDiagnostic(t *testing.T, diagnostics []Diagnostic, code string, count int) {
	t.Helper()
	if len(diagnostics) != count {
		t.Fatalf("WP3D diagnostics = %+v, want %d x %s", diagnostics, count, code)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code != code || diagnostic.Phase != "C_STATIC" || diagnostic.Severity != "error" {
			t.Fatalf("WP3D diagnostic = %+v, want %s/C_STATIC", diagnostic, code)
		}
	}
}

// WP3D-001: positive snapshot-scoped inventory with frozen ordering.
func TestWP3D001PositiveDependencyInspection(t *testing.T) {
	req := wp3dInspectRequest(t, wp3aBaseFixture(t))
	inspection := InspectDependencies(req)
	if inspection.APIVersion != DependencyInspectionAPIVersion {
		t.Fatalf("DependencyInspection apiVersion = %q", inspection.APIVersion)
	}
	if len(inspection.Diagnostics) != 0 {
		t.Fatalf("positive inspection emitted diagnostics: %+v", inspection.Diagnostics)
	}
	wantCallables := []InspectedCallable{{
		DependencyID:         "ca",
		Owner:                "example-ue-owner",
		Kind:                 "observation",
		ID:                   "ue.registration.current",
		Effect:               "READ",
		SourcePath:           "tool-catalog.yaml",
		QueryContextSchemaID: "urn:example:ca:registration-query:v1",
		OutputSchemaID:       "urn:example:ca:registration:v1",
	}}
	if !reflect.DeepEqual(inspection.Callables, wantCallables) {
		t.Fatalf("inspection callables = %+v, want %+v", inspection.Callables, wantCallables)
	}
	wantSchemas := []InspectedSchema{
		{DependencyID: "ca", Owner: "example-ue-owner", ID: "urn:example:ca:registration-query:v1", SourcePath: "registration-query.schema.yaml"},
		{DependencyID: "ca", Owner: "example-ue-owner", ID: "urn:example:ca:registration:v1", SourcePath: "registration.schema.yaml"},
		{DependencyID: "ca-context", Owner: "example-host-owner", ID: "urn:example:ca:invocation-context:v1", SourcePath: "invocation-context.schema.yaml"},
	}
	if !reflect.DeepEqual(inspection.Schemas, wantSchemas) {
		t.Fatalf("inspection schemas = %+v, want %+v", inspection.Schemas, wantSchemas)
	}

	// Inspection is snapshot-scoped: input order must not change the projection.
	permuted := req
	permuted.ResolvedDependencies = append([]ResolvedDependency(nil), req.ResolvedDependencies...)
	for index := range permuted.ResolvedDependencies {
		permuted.ResolvedDependencies[index].Files = reverseFiles(permuted.ResolvedDependencies[index].Files)
	}
	permuted.ResolvedDependencies[0], permuted.ResolvedDependencies[1] = permuted.ResolvedDependencies[1], permuted.ResolvedDependencies[0]
	if !reflect.DeepEqual(inspection, InspectDependencies(permuted)) {
		t.Fatalf("DependencyInspection is not input-order stable")
	}
}

// WP3D-002: a supplied snapshot without a trusted binding contributes nothing.
func TestWP3D002InspectionMissingBinding(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.withoutBinding("ca")
	req := wp3dInspectRequest(t, fixture)
	inspection := InspectDependencies(req)
	if !hasCode(inspection.Diagnostics, "DEPENDENCY_IDENTITY_UNVERIFIED") {
		t.Fatalf("unbound snapshot was not rejected: %+v", inspection.Diagnostics)
	}
	for _, callable := range inspection.Callables {
		if callable.DependencyID == "ca" {
			t.Fatalf("unverified snapshot leaked a callable: %+v", callable)
		}
	}
	for _, schema := range inspection.Schemas {
		if schema.DependencyID == "ca" {
			t.Fatalf("unverified snapshot leaked a Schema: %+v", schema)
		}
	}
}

// WP3D-003: multi-Owner snapshot omits exports without trusted attribution.
func TestWP3D003InspectionOmitsUnattributedExports(t *testing.T) {
	fixture := multiOwnerFixture(t)
	for index := range fixture.bindings {
		if fixture.bindings[index].DependencyID != "ca" {
			continue
		}
		fixture.bindings[index].ExportOwners = []ExportOwner{
			{ExportKind: "observation", ExportID: "ue.registration.current", Owner: "example-ue-owner"},
		}
	}
	req := wp3dInspectRequest(t, fixture)
	inspection := InspectDependencies(req)
	if len(inspection.Callables) != 1 || inspection.Callables[0].ID != "ue.registration.current" {
		t.Fatalf("inspection callables = %+v, want the mapped observation only", inspection.Callables)
	}
	for _, schema := range inspection.Schemas {
		if schema.DependencyID == "ca" {
			t.Fatalf("unattributed Schema export was exposed: %+v", schema)
		}
	}
	unverified := 0
	for _, diagnostic := range inspection.Diagnostics {
		if diagnostic.Code == "DEPENDENCY_EXPORT_OWNER_UNVERIFIED" {
			unverified++
		}
	}
	if unverified != 2 {
		t.Fatalf("DEPENDENCY_EXPORT_OWNER_UNVERIFIED count = %d, want 2: %+v", unverified, inspection.Diagnostics)
	}
}

// WP3D-004: unauthorized catalog Owner removes that catalog's callables.
func TestWP3D004InspectionUntrustedCatalogOwner(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceSnapshotFile("ca", "tool-catalog.yaml", "owner: example-ue-owner", "owner: example-untrusted-owner")
	fixture.refreshSnapshotDigests(t)
	req := wp3dInspectRequest(t, fixture)
	inspection := InspectDependencies(req)
	if len(inspection.Callables) != 0 {
		t.Fatalf("untrusted catalog callables were exposed: %+v", inspection.Callables)
	}
	if len(inspection.Schemas) != 3 {
		t.Fatalf("independent Schema exports should still be listed: %+v", inspection.Schemas)
	}
	found := false
	for _, diagnostic := range inspection.Diagnostics {
		if diagnostic.Code == "DEPENDENCY_OWNER_MISMATCH" {
			found = true
		}
	}
	if !found {
		t.Fatalf("untrusted catalog Owner was not reported: %+v", inspection.Diagnostics)
	}
}

// WP3D-005: ambiguous export identity is reported and omitted from inventory.
func TestWP3D005InspectionOmitsAmbiguousExports(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.addSnapshotFile("ca", "second-catalog.yaml", []byte(`apiVersion: connectionagent.io/toolcatalog/v1alpha4
kind: ToolCatalog
metadata:
  owner: example-ue-owner
spec:
  contracts:
    - kind: observation
      id: ue.registration.current
      effect: READ
      ownerOutcomes: [SUCCEEDED, FAILED]
      queryContextSchema:
        $ref: urn:example:ca:registration-query:v1
      outputSchema:
        $ref: urn:example:ca:registration:v1
`))
	fixture.refreshSnapshotDigests(t)
	inspection := InspectDependencies(wp3dInspectRequest(t, fixture))
	if len(inspection.Callables) != 0 {
		t.Fatalf("ambiguous callable was exposed: %+v", inspection.Callables)
	}
	if !hasCode(inspection.Diagnostics, "EXPORT_ID_AMBIGUOUS") {
		t.Fatalf("ambiguous export identity was not reported: %+v", inspection.Diagnostics)
	}
}

// WP3D-006: a dedicated same-snapshot Reconcile READ cannot be an ordinary tool.
func TestWP3D006ReconcileAntiBypass(t *testing.T) {
	control := wp3aBaseFixture(t)
	control.addSnapshotFile("ca", "session-catalog.yaml", wp3dSessionCatalog(t))
	control.refreshSnapshotDigests(t)
	controlReq := control.request(t)
	if diagnostics := wp3dCheckBuildDiagnostics(t, controlReq); len(diagnostics) != 0 {
		t.Fatalf("unused Mutation/Reconcile contract created diagnostics: %+v", diagnostics)
	}

	fixture := wp3aBaseFixture(t)
	fixture.addSnapshotFile("ca", "session-catalog.yaml", wp3dSessionCatalog(t))
	fixture.refreshSnapshotDigests(t)
	fixture.replaceFile(wp3aProcedurePath,
		"    observation:\n      tool: ca::ue.registration.current",
		"    operation:\n      tool: ca::example.session.reconcile")
	req := fixture.request(t)
	diagnostics := wp3dCheckBuildDiagnostics(t, req)
	expectOnlyWP3DDiagnostic(t, diagnostics, "RECONCILE_CONTRACT_INVALID", 1)
	if diagnostics[0].Path != wp3aProcedurePath || diagnostics[0].Line == 0 || diagnostics[0].Column == 0 {
		t.Fatalf("anti-bypass diagnostic must retain the Skill source reference location: %+v", diagnostics[0])
	}
	assertFailureAtomic(t, Build(req))
}

// WP3D-007: selected-Skill closures stay isolated and deterministic.
func TestWP3D007SelectedSkillClosureIsolation(t *testing.T) {
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

	first := wp3aIdentityForSkill(t, req, wp3aSkillName)
	second := wp3aIdentityForSkill(t, req, otherSkill)
	if want := []string{"ca", "ca-context"}; !reflect.DeepEqual(first.Closure, want) {
		t.Fatalf("first Skill closure = %v, want %v", first.Closure, want)
	}
	if len(second.Closure) != 0 {
		t.Fatalf("second Skill leaked a dependency closure: %v", second.Closure)
	}
	for _, diagnostic := range second.Diagnostics {
		if diagnostic.Code == "DEPENDENCY_ALIAS_MISSING" && !strings.HasPrefix(diagnostic.Path, base) {
			t.Fatalf("second Skill diagnostic leaked outside its package: %+v", diagnostic)
		}
	}
}

func wp3aIdentityForSkill(t *testing.T, req BuildRequest, name string) dependencyIdentityResult {
	t.Helper()
	return validateV1Alpha4DependencyIdentity(req, wp3aParseSkillAST(t, req, name))
}
