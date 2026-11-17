package compiler

// WP2: v1alpha5 API admission and target dispatch tests.

import (
	"strings"
	"testing"
)

func TestV1Alpha5APIVersionsAreIndependent(t *testing.T) {
	versions := map[string]string{
		"request":  BuildRequestAPIVersionV1Alpha5,
		"result":   BuildResultAPIVersionV1Alpha5,
		"inspect":  DependencyInspectionAPIVersionV1Alpha5,
		"v4req":    BuildRequestAPIVersion,
		"v4result": BuildResultAPIVersion,
	}
	for name, version := range versions {
		if !strings.Contains(version, "/v1alpha5") && !strings.Contains(version, "/v1alpha4") {
			t.Fatalf("%s version is not versioned: %q", name, version)
		}
	}
	if BuildRequestAPIVersionV1Alpha5 == BuildRequestAPIVersion {
		t.Fatal("v1alpha5 request API version must not reuse the v1alpha4 identity")
	}
	if BuildResultAPIVersionV1Alpha5 == BuildResultAPIVersion {
		t.Fatal("v1alpha5 result API version must not reuse the v1alpha4 identity")
	}
	if DependencyInspectionAPIVersionV1Alpha5 == DependencyInspectionAPIVersion {
		t.Fatal("v1alpha5 inspection API version must not reuse the v1alpha4 identity")
	}
	if V1Alpha5Target != "connectionagent.io/v1alpha5" {
		t.Fatalf("unexpected v1alpha5 language target: %q", V1Alpha5Target)
	}
}

func TestCheckV1Alpha5AcceptsValidProcedure(t *testing.T) {
	// End-to-end public Check positive path: a complete synthetic v1alpha5
	// request with its trusted Ingress/ToolCatalog/Schema dependency context.
	diagnostics := Check(wp3BuildRequest(t, wp3FixtureOptions{}))
	if len(diagnostics) != 0 {
		t.Fatalf("expected zero v1alpha5 diagnostics, got %#v", diagnostics)
	}
}

func TestValidateBuildRequestV1Alpha5RejectsWrongAPIVersion(t *testing.T) {
	req := v1alpha5TestProcedureRequest(validV1Alpha5WaitProcedure())
	req.APIVersion = BuildRequestAPIVersion // v1alpha4 envelope
	diagnostics := CheckV1Alpha5(req)
	assertV1Alpha5DiagnosticCodeOnly(t, diagnostics, "REQUEST_INVALID")
}

func TestValidateBuildRequestV1Alpha5RejectsWrongLanguageTarget(t *testing.T) {
	req := v1alpha5TestProcedureRequest(validV1Alpha5WaitProcedure())
	req.LanguageTarget = V1Alpha4Target
	diagnostics := CheckV1Alpha5(req)
	assertV1Alpha5DiagnosticCodeOnly(t, diagnostics, "LANGUAGE_TARGET_UNSUPPORTED")
}

func TestValidateBuildRequestV1Alpha5RejectsEmptyFiles(t *testing.T) {
	req := BuildRequest{APIVersion: BuildRequestAPIVersionV1Alpha5, LanguageTarget: V1Alpha5Target}
	diagnostics := CheckV1Alpha5(req)
	assertV1Alpha5DiagnosticCodeOnly(t, diagnostics, "SOURCE_ROOT_INVALID")
}

func TestValidateBuildRequestV1Alpha5RejectsUnnormalizedPath(t *testing.T) {
	req := BuildRequest{
		APIVersion:     BuildRequestAPIVersionV1Alpha5,
		LanguageTarget: V1Alpha5Target,
		Files: []File{
			{Path: "skills/../example/assurance/procedure.yaml", Bytes: []byte("start: a\nnodes: {}\n")},
		},
	}
	diagnostics := CheckV1Alpha5(req)
	assertV1Alpha5DiagnosticCodeOnly(t, diagnostics, "PACKAGE_PATH_INVALID")
}

func TestValidateBuildRequestV1Alpha5RejectsDuplicatePath(t *testing.T) {
	req := BuildRequest{
		APIVersion:     BuildRequestAPIVersionV1Alpha5,
		LanguageTarget: V1Alpha5Target,
		Files: []File{
			{Path: v1alpha5TestProcedurePath, Bytes: []byte("start: a\nnodes: {}\n")},
			{Path: v1alpha5TestProcedurePath, Bytes: []byte("start: b\nnodes: {}\n")},
		},
	}
	diagnostics := CheckV1Alpha5(req)
	assertV1Alpha5DiagnosticCodeOnly(t, diagnostics, "PACKAGE_PATH_INVALID")
}

func TestValidateBuildRequestV1Alpha5RejectsNilBytes(t *testing.T) {
	req := BuildRequest{
		APIVersion:     BuildRequestAPIVersionV1Alpha5,
		LanguageTarget: V1Alpha5Target,
		Files:          []File{{Path: v1alpha5TestProcedurePath}},
	}
	diagnostics := CheckV1Alpha5(req)
	assertV1Alpha5DiagnosticCodeOnly(t, diagnostics, "REQUEST_INVALID")
}

func TestCheckDispatchesV1Alpha5TargetAndFailsClosedOnUnknownTarget(t *testing.T) {
	// Exact v1alpha5 dispatch: the full trusted synthetic request is admitted by
	// the v1alpha5 entry, not the v1alpha4 envelope that hard-rejects the target.
	if diagnostics := Check(wp3BuildRequest(t, wp3FixtureOptions{})); len(diagnostics) != 0 {
		t.Fatalf("public Check must dispatch v1alpha5 exactly, got %#v", diagnostics)
	}

	unknown := v1alpha5TestProcedureRequest(validV1Alpha5WaitProcedure())
	unknown.LanguageTarget = "connectionagent.io/v1alpha6"
	diagnostics := Check(unknown)
	assertV1Alpha5DiagnosticCodeOnly(t, diagnostics, "LANGUAGE_TARGET_UNSUPPORTED")
}

func TestBuildV1Alpha5SucceedsAfterFullValidation(t *testing.T) {
	// The complete trusted fixture validates without source diagnostics, so the
	// candidate Build assembles the deterministic artifact tree.
	result := Build(wp3BuildRequest(t, wp3FixtureOptions{}))
	if !result.Success {
		t.Fatalf("v1alpha5 Build must succeed after zero-error validation, diagnostics = %#v", result.Diagnostics)
	}
	if result.APIVersion != BuildResultAPIVersionV1Alpha5 {
		t.Fatalf("BuildResult.apiVersion = %q, want %q", result.APIVersion, BuildResultAPIVersionV1Alpha5)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("successful Build must not report diagnostics, got %#v", result.Diagnostics)
	}
	if len(result.OutputFiles) < 3 {
		t.Fatalf("successful Build must emit the artifact tree, got %v", v1alpha5ArtifactPaths(result))
	}
	if strings.Join(result.DependencyClosure, ",") != "integration,tools" {
		t.Fatalf("dependencyClosure = %v, want [integration tools]", result.DependencyClosure)
	}
	if err := VerifyBuildResult(result); err != nil {
		t.Fatalf("VerifyBuildResult: %v", err)
	}

	// A Wait whose trusted Ingress export cannot exist is a real source
	// diagnostic and still blocks Build.
	dependencyFree := Build(v1alpha5TestProcedureRequest(validV1Alpha5WaitProcedure()))
	if dependencyFree.Success {
		t.Fatal("dependency-free v1alpha5 Build must not succeed")
	}
	if !hasV1Alpha5Diagnostic(dependencyFree.Diagnostics, "WAIT_INGRESS_EXPORT_NOT_FOUND") {
		t.Fatalf("dependency-free Wait must report WAIT_INGRESS_EXPORT_NOT_FOUND, got %#v", dependencyFree.Diagnostics)
	}
	if len(dependencyFree.OutputFiles) != 0 {
		t.Fatalf("failed Build emitted artifact files: %v", v1alpha5ArtifactPaths(dependencyFree))
	}
}

func TestBuildV1Alpha5ReportsWaitDiagnosticsAndEmitsNoArtifact(t *testing.T) {
	procedure := `start: wait-node
nodes:
  wait-node:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT0S
      routes:
        TIMEOUT: SUCCEEDED
    operation:
      tool: ca::smf.x
      arguments: {}
`
	result := Build(v1alpha5TestProcedureRequest(procedure))
	if result.Success {
		t.Fatal("invalid v1alpha5 Wait must not build")
	}
	if result.APIVersion != BuildResultAPIVersionV1Alpha5 {
		t.Fatalf("BuildResult.apiVersion = %q, want %q", result.APIVersion, BuildResultAPIVersionV1Alpha5)
	}
	if !hasErrorDiagnostic(result.Diagnostics) {
		t.Fatalf("expected wait diagnostics, got %#v", result.Diagnostics)
	}
	if !hasV1Alpha5Diagnostic(result.Diagnostics, "WAIT_TIMEOUT_INVALID") ||
		!hasV1Alpha5Diagnostic(result.Diagnostics, "WAIT_TIMEOUT_ROUTE_INVALID") {
		t.Fatalf("expected exact Wait diagnostics, got %#v", result.Diagnostics)
	}
	if len(result.OutputFiles) != 0 || len(result.DependencyClosure) != 0 {
		t.Fatalf("failed Build emitted artifact data: %#v", result)
	}
	for _, diagnostic := range result.Diagnostics {
		if strings.HasPrefix(diagnostic.Message, "fail-closed: ") {
			t.Fatalf("artifact assembly must not mask real source diagnostics: %#v", result.Diagnostics)
		}
	}
}

func TestV1Alpha4TargetStillRejectsAuthorWait(t *testing.T) {
	req := BuildRequest{
		APIVersion:           BuildRequestAPIVersion,
		LanguageTarget:       V1Alpha4Target,
		ResolvedDependencies: []ResolvedDependency{},
		Files: []File{
			{Path: "skills/example/SKILL.md", Bytes: []byte("# example\n")},
			{Path: v1alpha5TestProcedurePath, Bytes: []byte(validV1Alpha5WaitProcedure())},
		},
	}
	diagnostics := Check(req)
	if !hasV1Alpha5Diagnostic(diagnostics, "UNSUPPORTED_WAIT") {
		t.Fatalf("v1alpha4 must keep rejecting author wait with its frozen diagnostic, got %#v", diagnostics)
	}
	if hasV1Alpha5Diagnostic(diagnostics, "WAIT_DECLARATION_INVALID") {
		t.Fatalf("v1alpha4 must not emit v1alpha5 Wait diagnostics: %#v", diagnostics)
	}
}

func assertV1Alpha5DiagnosticCodeOnly(t *testing.T, diagnostics []Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("expected diagnostic %s, got %#v", code, diagnostics)
}
