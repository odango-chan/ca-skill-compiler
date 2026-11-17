package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/odango-chan/ca-skill-compiler/internal/compiler"
	"github.com/odango-chan/ca-skill-compiler/internal/projectinput"
)

// testBinary is a real skillc executable built once for CLI-level end-to-end
// tests. Package-internal tests cover helpers; these tests exercise the actual
// command line, process exit codes and files written to disk.
var testBinary string

func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "skillc-e2e-")
	if err != nil {
		panic(err)
	}
	testBinary = filepath.Join(directory, "skillc")
	goTool := filepath.Join(runtime.GOROOT(), "bin", "go")
	if runtime.GOOS == "windows" {
		testBinary += ".exe"
		goTool += ".exe"
	}
	if _, err := os.Stat(goTool); err != nil {
		goTool = "go"
	}
	build := exec.Command(goTool, "build", "-o", testBinary, ".")
	output, err := build.CombinedOutput()
	if err != nil {
		os.RemoveAll(directory)
		panic(fmt.Sprintf("build skillc binary: %v\n%s", err, output))
	}
	code := m.Run()
	os.RemoveAll(directory)
	os.Exit(code)
}

type cliResult struct {
	code   int
	stdout string
	stderr string
}

func runCLI(t *testing.T, args ...string) cliResult {
	t.Helper()
	command := exec.Command(testBinary, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("run skillc %v: %v", args, err)
		}
		code = exit.ExitCode()
	}
	return cliResult{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func repoRootForCLI(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func copyTree(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.Walk(source, func(name string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, name)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		content, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0644)
	})
	if err != nil {
		t.Fatalf("copy %s: %v", source, err)
	}
}

func readTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	err := filepath.Walk(root, func(name string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = content
		return nil
	})
	if err != nil {
		t.Fatalf("read tree: %v", err)
	}
	return files
}

func TestCLIEndToEndVersion(t *testing.T) {
	versioned := runCLI(t, "version")
	if versioned.code != 0 {
		t.Fatalf("version exit = %d, stderr = %s", versioned.code, versioned.stderr)
	}
	for _, want := range []string{"skillc " + compiler.CompilerVersion, "language target: " + compiler.V1Alpha4Target} {
		if !strings.Contains(versioned.stdout, want) {
			t.Fatalf("version output missing %q: %q", want, versioned.stdout)
		}
	}
	if result := runCLI(t, "version", "extra"); result.code != 2 {
		t.Fatalf("version with argument exit = %d, want 2", result.code)
	}
}

func TestCLIEndToEndTrustedBuildAndDeterminism(t *testing.T) {
	fixture := filepath.Join(repoRootForCLI(t), "web", "parity-success")

	checked := runCLI(t, "check", fixture)
	if checked.code != 0 {
		t.Fatalf("check exit = %d, stderr = %s", checked.code, checked.stderr)
	}
	if !strings.Contains(checked.stdout, "static check OK") {
		t.Fatalf("check stdout = %q", checked.stdout)
	}

	first := filepath.Join(t.TempDir(), "first")
	built := runCLI(t, "build", fixture, "--out", first)
	if built.code != 0 {
		t.Fatalf("build exit = %d, stderr = %s", built.code, built.stderr)
	}
	if !strings.Contains(built.stdout, "Skill build complete") {
		t.Fatalf("build stdout = %q", built.stdout)
	}
	firstTree := readTree(t, first)
	if len(firstTree) == 0 {
		t.Fatal("build wrote no files")
	}
	if _, ok := firstTree[filepath.ToSlash("skills/verify-ue-registration/generated/artifact-manifest.json")]; !ok {
		t.Fatalf("artifact manifest missing, files = %v", keysOf(firstTree))
	}

	second := filepath.Join(t.TempDir(), "second")
	if built := runCLI(t, "build", fixture, "--out", second); built.code != 0 {
		t.Fatalf("second build exit = %d, stderr = %s", built.code, built.stderr)
	}
	secondTree := readTree(t, second)
	if len(firstTree) != len(secondTree) {
		t.Fatalf("repeated build file count = %d, want %d", len(secondTree), len(firstTree))
	}
	for name, content := range firstTree {
		if !bytes.Equal(secondTree[name], content) {
			t.Fatalf("repeated build differs at %s", name)
		}
	}
}

func TestCLIEndToEndJSONContracts(t *testing.T) {
	fixture := filepath.Join(repoRootForCLI(t), "web", "parity-success")

	checked := runCLI(t, "check", fixture, "--json")
	if checked.code != 0 {
		t.Fatalf("check --json exit = %d, stderr = %s", checked.code, checked.stderr)
	}
	var checkDiagnostics []compiler.Diagnostic
	if err := json.Unmarshal([]byte(checked.stdout), &checkDiagnostics); err != nil {
		t.Fatalf("check --json output is not a Diagnostic array: %v\n%s", err, checked.stdout)
	}
	if len(checkDiagnostics) != 0 {
		t.Fatalf("unexpected check diagnostics: %+v", checkDiagnostics)
	}

	out := filepath.Join(t.TempDir(), "out")
	built := runCLI(t, "build", fixture, "--out", out, "--json")
	if built.code != 0 {
		t.Fatalf("build --json exit = %d, stderr = %s", built.code, built.stderr)
	}
	var buildResult compiler.BuildResult
	if err := json.Unmarshal([]byte(built.stdout), &buildResult); err != nil {
		t.Fatalf("build --json output is not a BuildResult document: %v", err)
	}
	if !buildResult.Success || len(buildResult.OutputFiles) == 0 {
		t.Fatalf("unexpected build result: success=%v files=%d", buildResult.Success, len(buildResult.OutputFiles))
	}
	for _, file := range buildResult.OutputFiles {
		if file.Path == "" || len(file.Bytes) == 0 {
			t.Fatalf("incomplete output file in machine result: %+v", file)
		}
	}

	inspected := runCLI(t, "inspect", fixture, "--json")
	if inspected.code != 0 {
		t.Fatalf("inspect --json exit = %d, stderr = %s", inspected.code, inspected.stderr)
	}
	var inspection compiler.DependencyInspection
	if err := json.Unmarshal([]byte(inspected.stdout), &inspection); err != nil {
		t.Fatalf("inspect --json output is not a DependencyInspection document: %v", err)
	}
	if inspection.APIVersion != compiler.DependencyInspectionAPIVersion {
		t.Fatalf("inspection apiVersion = %q", inspection.APIVersion)
	}
	if len(inspection.Callables) == 0 || len(inspection.Schemas) == 0 {
		t.Fatalf("inspection found callables=%d schemas=%d", len(inspection.Callables), len(inspection.Schemas))
	}
}

func TestCLIEndToEndHumanInspect(t *testing.T) {
	fixture := filepath.Join(repoRootForCLI(t), "web", "parity-success")
	inspected := runCLI(t, "inspect", fixture)
	if inspected.code != 0 {
		t.Fatalf("inspect exit = %d, stderr = %s", inspected.code, inspected.stderr)
	}
	for _, want := range []string{"Dependency inspection OK", "example-ue-owner", "example-host-owner"} {
		if !strings.Contains(inspected.stdout, want) {
			t.Fatalf("inspect stdout missing %q:\n%s", want, inspected.stdout)
		}
	}
}

func TestCLIEndToEndDiagnosticsArePositioned(t *testing.T) {
	root := t.TempDir()
	copyTree(t, filepath.Join(repoRootForCLI(t), "web", "parity-project"), root)
	procedure := filepath.Join(root, "skills", "parity", "assurance", "procedure.yaml")
	content, err := os.ReadFile(procedure)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(procedure, append(content, []byte("\nbroken: [unclosed\n")...), 0644); err != nil {
		t.Fatal(err)
	}

	checked := runCLI(t, "check", root)
	if checked.code != 1 {
		t.Fatalf("check exit = %d, want 1; stderr = %s", checked.code, checked.stderr)
	}
	pattern := regexp.MustCompile(`skills/parity/assurance/procedure\.yaml:\d+:\d+: error C_STATIC YAML_PARSE_ERROR: `)
	if !pattern.MatchString(checked.stderr) {
		t.Fatalf("diagnostic lacks file:line:column/code: %q", checked.stderr)
	}

	machine := runCLI(t, "check", root, "--json")
	if machine.code != 1 {
		t.Fatalf("check --json exit = %d, want 1", machine.code)
	}
	var diagnostics []compiler.Diagnostic
	if err := json.Unmarshal([]byte(machine.stdout), &diagnostics); err != nil {
		t.Fatalf("check --json: %v", err)
	}
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "YAML_PARSE_ERROR" {
			found = true
			if diagnostic.Path != "skills/parity/assurance/procedure.yaml" || diagnostic.Line == 0 || diagnostic.Column == 0 {
				t.Fatalf("diagnostic lacks source position: %+v", diagnostic)
			}
		}
	}
	if !found {
		t.Fatalf("YAML_PARSE_ERROR diagnostic missing: %+v", diagnostics)
	}
}

func TestCLIEndToEndMissingDependencyFailsClosed(t *testing.T) {
	root := t.TempDir()
	copyTree(t, filepath.Join(repoRootForCLI(t), "web", "parity-success", "files"), root)
	checked := runCLI(t, "check", root)
	if checked.code != 1 {
		t.Fatalf("check exit = %d, want 1; stdout = %s", checked.code, checked.stdout)
	}
	if !strings.Contains(checked.stderr, "DEPENDENCY_SNAPSHOT_MISSING") {
		t.Fatalf("missing dependency was not reported: %q", checked.stderr)
	}
	built := runCLI(t, "build", root, "--out", filepath.Join(t.TempDir(), "out"))
	if built.code != 1 {
		t.Fatalf("build exit = %d, want 1", built.code)
	}
}

func TestCLIEndToEndRequestTransport(t *testing.T) {
	fixture := filepath.Join(repoRootForCLI(t), "web", "parity-success")
	request, err := projectinput.Load(fixture)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	requestPath := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(requestPath, encoded, 0644); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"check", "--request", requestPath},
		{"inspect", "--request", requestPath, "--json"},
		{"build", "--request", requestPath, "--out", filepath.Join(t.TempDir(), "out")},
	} {
		if result := runCLI(t, args...); result.code != 0 {
			t.Fatalf("request transport %v exit = %d, stderr = %s", args, result.code, result.stderr)
		}
	}

	malformed := filepath.Join(t.TempDir(), "malformed.json")
	if err := os.WriteFile(malformed, []byte(`{"apiVersion":123}`), 0644); err != nil {
		t.Fatal(err)
	}
	if result := runCLI(t, "check", "--request", malformed); result.code != 2 {
		t.Fatalf("malformed request exit = %d, want 2", result.code)
	}
	if result := runCLI(t, "check", "--request", requestPath, fixture); result.code != 2 {
		t.Fatalf("PROJECT with --request exit = %d, want 2", result.code)
	}
	if result := runCLI(t, "build", "--request", requestPath); result.code != 2 {
		t.Fatalf("build --request without --out exit = %d, want 2", result.code)
	}
}

func TestCLIEndToEndOutputFreshness(t *testing.T) {
	fixture := filepath.Join(repoRootForCLI(t), "web", "parity-success")
	out := filepath.Join(t.TempDir(), "out")
	if result := runCLI(t, "build", fixture, "--out", out); result.code != 0 {
		t.Fatalf("first build exit = %d, stderr = %s", result.code, result.stderr)
	}
	second := runCLI(t, "build", fixture, "--out", out)
	if second.code != 1 {
		t.Fatalf("nonempty output accepted, exit = %d", second.code)
	}
	if !strings.Contains(second.stderr, "nonempty") {
		t.Fatalf("nonempty output diagnostic missing: %q", second.stderr)
	}
}

func TestCLIEndToEndSkillTarget(t *testing.T) {
	fixture := filepath.Join(repoRootForCLI(t), "web", "parity-success")
	if result := runCLI(t, "check", fixture, "--skill", "verify-ue-registration"); result.code != 0 {
		t.Fatalf("targeted check exit = %d, stderr = %s", result.code, result.stderr)
	}
	if result := runCLI(t, "check", fixture, "--skill", "no-such-skill"); result.code != 1 {
		t.Fatalf("unknown target exit = %d, want 1; stderr = %s", result.code, result.stderr)
	}
}

// v1alpha5CLIRequestPath writes the synthetic v1alpha5 author fixture as an
// explicit BuildRequest JSON document. It uses author files only: the v1alpha5
// Check slice needs no trusted dependency snapshot.
func v1alpha5CLIRequestPath(t *testing.T, fixture string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRootForCLI(t), "web", "parity-v1alpha5", fixture+"-procedure.yaml"))
	if err != nil {
		t.Fatalf("read v1alpha5 fixture %s: %v", fixture, err)
	}
	request := compiler.BuildRequest{
		APIVersion:           compiler.BuildRequestAPIVersionV1Alpha5,
		LanguageTarget:       compiler.V1Alpha5Target,
		Files:                []compiler.File{{Path: "skills/example/assurance/procedure.yaml", Bytes: raw}},
		ResolvedDependencies: []compiler.ResolvedDependency{},
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	requestPath := filepath.Join(t.TempDir(), "v1alpha5-"+fixture+"-request.json")
	if err := os.WriteFile(requestPath, encoded, 0644); err != nil {
		t.Fatal(err)
	}
	return requestPath
}

func TestCLIEndToEndV1Alpha5VersionAndHelp(t *testing.T) {
	versioned := runCLI(t, "version")
	if versioned.code != 0 {
		t.Fatalf("version exit = %d, stderr = %s", versioned.code, versioned.stderr)
	}
	for _, want := range []string{
		"skillc " + compiler.CompilerVersion,
		"language target: " + compiler.V1Alpha4Target,
		"language target: " + compiler.V1Alpha5Target,
		compiler.BuildRequestAPIVersion,
		compiler.BuildRequestAPIVersionV1Alpha5,
		compiler.BuildResultAPIVersion,
		compiler.BuildResultAPIVersionV1Alpha5,
	} {
		if !strings.Contains(versioned.stdout, want) {
			t.Fatalf("version output missing %q: %q", want, versioned.stdout)
		}
	}
	helped := runCLI(t, "help")
	if helped.code != 0 {
		t.Fatalf("help exit = %d", helped.code)
	}
	for _, want := range []string{v1alpha5LanguageTargetForCLITest, "--request", compiler.BuildRequestAPIVersion} {
		if !strings.Contains(helped.stdout, want) {
			t.Fatalf("help output missing %q", want)
		}
	}
}

const v1alpha5LanguageTargetForCLITest = "connectionagent.io/v1alpha5"

// TestCLIEndToEndV1Alpha5CheckRequest proves `check --request` dispatches to the
// v1alpha5 static Check: a valid Wait Procedure returns zero diagnostics and an
// invalid one returns deterministic positioned WAIT_* diagnostics.
func TestCLIEndToEndV1Alpha5CheckRequest(t *testing.T) {
	validPath := v1alpha5CLIRequestPath(t, "valid")
	validJSON := runCLI(t, "check", "--request", validPath, "--json")
	if validJSON.code != 0 {
		t.Fatalf("valid v1alpha5 check --json exit = %d, stderr = %s", validJSON.code, validJSON.stderr)
	}
	var validDiagnostics []compiler.Diagnostic
	if err := json.Unmarshal([]byte(validJSON.stdout), &validDiagnostics); err != nil {
		t.Fatalf("valid v1alpha5 check --json output: %v\n%s", err, validJSON.stdout)
	}
	if len(validDiagnostics) != 0 {
		t.Fatalf("valid v1alpha5 Wait procedure diagnostics = %+v, want []", validDiagnostics)
	}
	validHuman := runCLI(t, "check", "--request", validPath)
	if validHuman.code != 0 || !strings.Contains(validHuman.stdout, "static check OK") {
		t.Fatalf("valid v1alpha5 human check exit = %d stdout = %q stderr = %q", validHuman.code, validHuman.stdout, validHuman.stderr)
	}

	invalidPath := v1alpha5CLIRequestPath(t, "invalid")
	invalidJSON := runCLI(t, "check", "--request", invalidPath, "--json")
	if invalidJSON.code != 1 {
		t.Fatalf("invalid v1alpha5 check --json exit = %d, want 1; stderr = %s", invalidJSON.code, invalidJSON.stderr)
	}
	var diagnostics []compiler.Diagnostic
	if err := json.Unmarshal([]byte(invalidJSON.stdout), &diagnostics); err != nil {
		t.Fatalf("invalid v1alpha5 check --json output: %v\n%s", err, invalidJSON.stdout)
	}
	expected := []struct {
		code   string
		line   int
		column int
	}{
		{"WAIT_NODE_STAGE_REQUIRED", 5, 12},
		{"WAIT_DECLARATION_INVALID", 8, 20},
		{"WAIT_TIMEOUT_INVALID", 9, 16},
		{"WAIT_RESPONSE_BOUNDARY_INVALID", 11, 16},
		{"WAIT_TIMEOUT_ROUTE_INVALID", 13, 18},
		{"WAIT_TIMEOUT_ROUTE_INVALID", 14, 9},
	}
	if len(diagnostics) != len(expected) {
		t.Fatalf("invalid v1alpha5 diagnostics = %d, want %d: %+v", len(diagnostics), len(expected), diagnostics)
	}
	for index, want := range expected {
		diagnostic := diagnostics[index]
		if diagnostic.Severity != "error" || diagnostic.Phase != "C_STATIC" ||
			diagnostic.Code != want.code || diagnostic.Path != "skills/example/assurance/procedure.yaml" ||
			diagnostic.Line != want.line || diagnostic.Column != want.column {
			t.Fatalf("diagnostic[%d] = %+v, want code=%s %s:%d:%d", index, diagnostic, want.code, "skills/example/assurance/procedure.yaml", want.line, want.column)
		}
	}

	invalidHuman := runCLI(t, "check", "--request", invalidPath)
	if invalidHuman.code != 1 {
		t.Fatalf("invalid v1alpha5 human check exit = %d, want 1", invalidHuman.code)
	}
	positioned := regexp.MustCompile(`skills/example/assurance/procedure\.yaml:5:12: error C_STATIC WAIT_NODE_STAGE_REQUIRED: `)
	if !positioned.MatchString(invalidHuman.stderr) {
		t.Fatalf("v1alpha5 human diagnostic lacks file:line:column/code: %q", invalidHuman.stderr)
	}
}

// TestCLIEndToEndV1Alpha5BuildFailClosed proves Build is fail-closed for the
// candidate target in both directions: a valid Wait Procedure fails closed on
// the documented residual gate, while an invalid one preserves its WAIT_*
// diagnostics. Every v1alpha5 BuildResult is stamped with the v1alpha5 API
// version, and no artifact files are emitted.
func TestCLIEndToEndV1Alpha5BuildFailClosed(t *testing.T) {
	validPath := v1alpha5CLIRequestPath(t, "valid")
	valid := runCLI(t, "build", "--request", validPath, "--out", filepath.Join(t.TempDir(), "valid-out"), "--json")
	if valid.code != 1 {
		t.Fatalf("valid v1alpha5 build exit = %d, want 1; stderr = %s", valid.code, valid.stderr)
	}
	var validResult compiler.BuildResult
	if err := json.Unmarshal([]byte(valid.stdout), &validResult); err != nil {
		t.Fatalf("valid v1alpha5 build --json output: %v\n%s", err, valid.stdout)
	}
	if validResult.APIVersion != compiler.BuildResultAPIVersionV1Alpha5 || validResult.Success {
		t.Fatalf("valid v1alpha5 build result = apiVersion %q success %v", validResult.APIVersion, validResult.Success)
	}
	if len(validResult.Diagnostics) != 1 || validResult.Diagnostics[0].Code != "CAPABILITY_INPUT_UNTRUSTED" ||
		!strings.HasPrefix(validResult.Diagnostics[0].Message, "fail-closed:") {
		t.Fatalf("v1alpha5 fail-closed residual = %+v", validResult.Diagnostics)
	}
	if len(validResult.OutputFiles) != 0 || len(validResult.DependencyClosure) != 0 {
		t.Fatalf("fail-closed v1alpha5 build emitted artifact data: %+v", validResult)
	}

	invalidPath := v1alpha5CLIRequestPath(t, "invalid")
	invalid := runCLI(t, "build", "--request", invalidPath, "--out", filepath.Join(t.TempDir(), "invalid-out"), "--json")
	if invalid.code != 1 {
		t.Fatalf("invalid v1alpha5 build exit = %d, want 1; stderr = %s", invalid.code, invalid.stderr)
	}
	var invalidResult compiler.BuildResult
	if err := json.Unmarshal([]byte(invalid.stdout), &invalidResult); err != nil {
		t.Fatalf("invalid v1alpha5 build --json output: %v\n%s", err, invalid.stdout)
	}
	if invalidResult.APIVersion != compiler.BuildResultAPIVersionV1Alpha5 || invalidResult.Success {
		t.Fatalf("invalid v1alpha5 build result = apiVersion %q success %v", invalidResult.APIVersion, invalidResult.Success)
	}
	if len(invalidResult.Diagnostics) != 6 {
		t.Fatalf("invalid v1alpha5 build diagnostics = %+v, want 6", invalidResult.Diagnostics)
	}
	for _, diagnostic := range invalidResult.Diagnostics {
		if !strings.HasPrefix(diagnostic.Code, "WAIT_") || diagnostic.Line == 0 || diagnostic.Column == 0 {
			t.Fatalf("invalid v1alpha5 build diagnostic = %+v", diagnostic)
		}
	}
}

// TestCLIEndToEndV1Alpha5InspectDoesNotClaimIngressSupport keeps the deferred
// DependencyInspection ingress wiring explicit: v1alpha5 inspection currently
// exposes the inherited v1alpha4 envelope rejection instead of guessing a new
// inventory projection.
func TestCLIEndToEndV1Alpha5InspectDoesNotClaimIngressSupport(t *testing.T) {
	inspection := runCLI(t, "inspect", "--request", v1alpha5CLIRequestPath(t, "valid"), "--json")
	if inspection.code != 1 {
		t.Fatalf("v1alpha5 inspect exit = %d, want 1", inspection.code)
	}
	var result compiler.DependencyInspection
	if err := json.Unmarshal([]byte(inspection.stdout), &result); err != nil {
		t.Fatalf("v1alpha5 inspect --json output: %v\n%s", err, inspection.stdout)
	}
	if result.APIVersion != compiler.DependencyInspectionAPIVersion {
		t.Fatalf("v1alpha5 inspect apiVersion = %q", result.APIVersion)
	}
	codes := make(map[string]bool)
	for _, diagnostic := range result.Diagnostics {
		codes[diagnostic.Code] = true
	}
	if !codes["LANGUAGE_TARGET_UNSUPPORTED"] || !codes["REQUEST_INVALID"] {
		t.Fatalf("unexpected v1alpha5 inspect diagnostics: %+v", result.Diagnostics)
	}
	if len(result.Callables) != 0 || len(result.Schemas) != 0 {
		t.Fatalf("deferred v1alpha5 inspection must not claim ingress inventory: %+v", result)
	}
}

func keysOf(files map[string][]byte) []string {
	keys := make([]string, 0, len(files))
	for name := range files {
		keys = append(keys, name)
	}
	return keys
}
