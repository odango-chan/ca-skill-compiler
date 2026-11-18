package compiler

// Single-version public API tests: exact admission, one dispatch path and a
// fail-closed artifact boundary.

import (
	"strings"
	"testing"
)

func TestPublicAPIVersionsAreSingleVersion(t *testing.T) {
	for name, version := range map[string]string{
		"request": BuildRequestAPIVersion,
		"result":  BuildResultAPIVersion,
		"inspect": DependencyInspectionAPIVersion,
	} {
		if !strings.Contains(version, "/v1alpha5") {
			t.Fatalf("%s API version is not the v1alpha5 identity: %q", name, version)
		}
		if strings.Contains(version, "/v1alpha4") {
			t.Fatalf("%s API version still exposes the retired v1alpha4 identity: %q", name, version)
		}
	}
	if V1Alpha5Target != "connectionagent.io/v1alpha5" {
		t.Fatalf("unexpected language target: %q", V1Alpha5Target)
	}
}

func TestCheckAcceptsValidProcedure(t *testing.T) {
	// End-to-end public Check positive path: a complete synthetic request with
	// its trusted Ingress/ToolCatalog/Schema dependency context.
	diagnostics := Check(wp3BuildRequest(t, wp3FixtureOptions{}))
	if len(diagnostics) != 0 {
		t.Fatalf("expected zero diagnostics, got %#v", diagnostics)
	}
}

func TestAdmissionRejectsRetiredRequestAPIVersion(t *testing.T) {
	req := v1alpha5TestProcedureRequest(validV1Alpha5WaitProcedure())
	req.APIVersion = "connectionagent.io/compiler-build-request/v1alpha4"
	diagnostics := Check(req)
	assertDiagnosticCodeOnly(t, diagnostics, "REQUEST_INVALID")
}

func TestAdmissionRejectsRetiredLanguageTarget(t *testing.T) {
	req := v1alpha5TestProcedureRequest(validV1Alpha5WaitProcedure())
	req.LanguageTarget = "connectionagent.io/v1alpha4"
	diagnostics := Check(req)
	assertDiagnosticCodeOnly(t, diagnostics, "LANGUAGE_TARGET_UNSUPPORTED")
}

func TestAdmissionRejectsEmptyFiles(t *testing.T) {
	req := BuildRequest{APIVersion: BuildRequestAPIVersion, LanguageTarget: V1Alpha5Target}
	diagnostics := Check(req)
	assertDiagnosticCodeOnly(t, diagnostics, "SOURCE_ROOT_INVALID")
}

func TestAdmissionRejectsUnnormalizedPath(t *testing.T) {
	req := BuildRequest{
		APIVersion:     BuildRequestAPIVersion,
		LanguageTarget: V1Alpha5Target,
		Files: []File{
			{Path: "skills/../example/assurance/procedure.yaml", Bytes: []byte("start: a\nnodes: {}\n")},
		},
	}
	diagnostics := Check(req)
	assertDiagnosticCodeOnly(t, diagnostics, "PACKAGE_PATH_INVALID")
}

func TestAdmissionRejectsDuplicatePath(t *testing.T) {
	req := BuildRequest{
		APIVersion:     BuildRequestAPIVersion,
		LanguageTarget: V1Alpha5Target,
		Files: []File{
			{Path: v1alpha5TestProcedurePath, Bytes: []byte("start: a\nnodes: {}\n")},
			{Path: v1alpha5TestProcedurePath, Bytes: []byte("start: b\nnodes: {}\n")},
		},
	}
	diagnostics := Check(req)
	assertDiagnosticCodeOnly(t, diagnostics, "PACKAGE_PATH_INVALID")
}

func TestAdmissionRejectsNilBytes(t *testing.T) {
	req := BuildRequest{
		APIVersion:     BuildRequestAPIVersion,
		LanguageTarget: V1Alpha5Target,
		Files:          []File{{Path: v1alpha5TestProcedurePath}},
	}
	diagnostics := Check(req)
	assertDiagnosticCodeOnly(t, diagnostics, "REQUEST_INVALID")
}

func TestCheckFailsClosedOnUnknownTarget(t *testing.T) {
	unknown := v1alpha5TestProcedureRequest(validV1Alpha5WaitProcedure())
	unknown.LanguageTarget = "connectionagent.io/v1alpha6"
	diagnostics := Check(unknown)
	assertDiagnosticCodeOnly(t, diagnostics, "LANGUAGE_TARGET_UNSUPPORTED")
}

func TestBuildSucceedsAfterFullValidation(t *testing.T) {
	// The complete trusted fixture validates without source diagnostics, so
	// Build assembles the deterministic artifact tree.
	result := Build(wp3BuildRequest(t, wp3FixtureOptions{}))
	if !result.Success {
		t.Fatalf("Build must succeed after zero-error validation, diagnostics = %#v", result.Diagnostics)
	}
	if result.APIVersion != BuildResultAPIVersion {
		t.Fatalf("BuildResult.apiVersion = %q, want %q", result.APIVersion, BuildResultAPIVersion)
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
		t.Fatal("dependency-free Build must not succeed")
	}
	if !hasV1Alpha5Diagnostic(dependencyFree.Diagnostics, "WAIT_INGRESS_EXPORT_NOT_FOUND") {
		t.Fatalf("dependency-free Wait must report WAIT_INGRESS_EXPORT_NOT_FOUND, got %#v", dependencyFree.Diagnostics)
	}
	if len(dependencyFree.OutputFiles) != 0 {
		t.Fatalf("failed Build emitted artifact files: %v", v1alpha5ArtifactPaths(dependencyFree))
	}
}

func TestBuildReportsWaitDiagnosticsAndEmitsNoArtifact(t *testing.T) {
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
		t.Fatal("invalid Wait must not build")
	}
	if result.APIVersion != BuildResultAPIVersion {
		t.Fatalf("BuildResult.apiVersion = %q, want %q", result.APIVersion, BuildResultAPIVersion)
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

func TestVerifyBuildResultRejectsRetiredResultVersion(t *testing.T) {
	result := Build(wp3BuildRequest(t, wp3FixtureOptions{}))
	if !result.Success {
		t.Fatalf("fixture must build: %#v", result.Diagnostics)
	}
	result.APIVersion = "connectionagent.io/compiler-build-result/v1alpha4"
	if err := VerifyBuildResult(result); err == nil {
		t.Fatal("retired v1alpha4 BuildResult passed current artifact verification")
	}
}

func assertDiagnosticCodeOnly(t *testing.T, diagnostics []Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			continue
		}
		t.Fatalf("expected only %s, got %#v", code, diagnostics)
	}
	if !hasCode(diagnostics, code) {
		t.Fatalf("expected diagnostic %s, got %#v", code, diagnostics)
	}
}
