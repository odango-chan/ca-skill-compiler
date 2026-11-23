package compiler

// WP8: frozen v1alpha5 deterministic Build artifact assembly tests.
//
// These fixtures are SYNTHETIC trusted static-compiler fixtures built from the
// WP3/WP6 test helpers; they are not real Owner artifacts, deployment profiles or
// 5GC/3GPP publication claims.

import (
	"bytes"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const v1alpha5WaitFreeProcedure = `start: verify-registration
nodes:
  verify-registration:
    operation:
      tool: ca::demo.observe_registration
      arguments: {}
      routes:
        SUCCEEDED: classify-registration
        FAILED: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
  classify-registration:
    postcondition:
      predicates:
        - predicate: demo.registration_satisfied
          arguments: []
      routes:
        PASS: SUCCEEDED
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
`

func v1alpha5ArtifactPaths(result BuildResult) []string {
	paths := make([]string, 0, len(result.OutputFiles))
	for _, file := range result.OutputFiles {
		paths = append(paths, file.Path)
	}
	sort.Strings(paths)
	return paths
}

func v1alpha5ResultFile(result BuildResult, path string) ([]byte, bool) {
	for _, file := range result.OutputFiles {
		if file.Path == path {
			return file.Bytes, true
		}
	}
	return nil, false
}

func v1alpha5DecodeManifest(t *testing.T, result BuildResult) v1alpha5ArtifactManifest {
	t.Helper()
	raw, ok := v1alpha5ResultFile(result, V1Alpha5ArtifactManifestPath)
	if !ok {
		t.Fatalf("BuildResult has no %s", V1Alpha5ArtifactManifestPath)
	}
	var manifest v1alpha5ArtifactManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode v1alpha5 manifest: %v", err)
	}
	return manifest
}

// TestV1Alpha5BuildAssemblesTrustedArtifact is the full trusted WP3 positive
// path through the public Build entry: zero diagnostics, a complete deterministic
// tree, exact manifest/link fields and a passing v1alpha5 artifact verification.
func TestV1Alpha5BuildAssemblesTrustedArtifact(t *testing.T) {
	request := wp3BuildRequest(t, wp3FixtureOptions{})
	if diagnostics := Check(request); len(diagnostics) != 0 {
		t.Fatalf("trusted fixture must pass Check, got %#v", diagnostics)
	}
	result := Build(request)
	if !result.Success {
		t.Fatalf("trusted v1alpha5 Build must succeed, diagnostics = %#v", result.Diagnostics)
	}
	if result.APIVersion != BuildResultAPIVersion {
		t.Fatalf("BuildResult.apiVersion = %q", result.APIVersion)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("successful Build must not report diagnostics: %#v", result.Diagnostics)
	}
	if len(result.OutputFiles) != 3 {
		t.Fatalf("expected 3 artifact files, got %v", v1alpha5ArtifactPaths(result))
	}
	wantPaths := []string{
		V1Alpha5ArtifactManifestPath,
		"skills/example/compiled/ingress-links.json",
		"skills/example/compiled/procedure.json",
	}
	if got := v1alpha5ArtifactPaths(result); strings.Join(got, ",") != strings.Join(wantPaths, ",") {
		t.Fatalf("artifact paths = %v, want %v", got, wantPaths)
	}
	if strings.Join(result.DependencyClosure, ",") != "integration,tools" {
		t.Fatalf("dependencyClosure = %v, want [integration tools]", result.DependencyClosure)
	}
	if err := VerifyBuildResult(result); err != nil {
		t.Fatalf("VerifyBuildResult: %v", err)
	}

	manifest := v1alpha5DecodeManifest(t, result)
	if manifest.APIVersion != V1Alpha5ArtifactAPIVersion || manifest.LanguageTarget != V1Alpha5Target {
		t.Fatalf("unexpected manifest identity: %#v", manifest)
	}
	if manifest.Generator.Name != v1alpha5ArtifactGeneratorName || manifest.Generator.Version != V1Alpha5FrozenGeneratorVersion {
		t.Fatalf("unexpected manifest generator: %#v", manifest.Generator)
	}
	if len(manifest.DependencyClosure) != 2 {
		t.Fatalf("manifest dependencyClosure = %#v", manifest.DependencyClosure)
	}
	if manifest.DependencyClosure[0].DependencyID != "integration" || manifest.DependencyClosure[0].StablePackageID != wp3IntegrationPackageID {
		t.Fatalf("manifest integration closure = %#v", manifest.DependencyClosure[0])
	}
	if got := []string{
		manifest.DependencyClosure[0].Exports[0].ExportKind,
		manifest.DependencyClosure[0].Exports[0].ExportID,
		manifest.DependencyClosure[0].Exports[0].Owner,
	}; strings.Join(got, "|") != "ingressOperation|"+wp3IngressExportID+"|"+wp3IntegrationOwner {
		t.Fatalf("manifest ingress export = %#v", manifest.DependencyClosure[0].Exports)
	}
	if manifest.DependencyClosure[1].DependencyID != "tools" || manifest.DependencyClosure[1].StablePackageID != wp3ToolsPackageID {
		t.Fatalf("manifest tools closure = %#v", manifest.DependencyClosure[1])
	}
	if got := []string{
		manifest.DependencyClosure[1].Exports[0].ExportKind,
		manifest.DependencyClosure[1].Exports[0].ExportID,
		manifest.DependencyClosure[1].Exports[0].Owner,
	}; strings.Join(got, "|") != "operation|"+wp3TriggerExportID+"|"+wp3ToolsOwner {
		t.Fatalf("manifest trigger export = %#v", manifest.DependencyClosure[1].Exports)
	}

	if len(manifest.Skills) != 1 {
		t.Fatalf("manifest skills = %#v", manifest.Skills)
	}
	skill := manifest.Skills[0]
	if skill.Name != "example" || !strings.HasPrefix(skill.SourceDigest, "sha256:") {
		t.Fatalf("unexpected manifest Skill: %#v", skill)
	}
	if strings.Join(skill.SourceFiles, ",") != wp3SkillMarkdownPath+","+wp3ContractPath+","+wp3PredicatesPath+","+wp3ProcedurePath {
		t.Fatalf("manifest sourceFiles = %v", skill.SourceFiles)
	}
	if strings.Join(skill.CompiledFiles, ",") != "skills/example/compiled/ingress-links.json,skills/example/compiled/procedure.json" {
		t.Fatalf("manifest compiledFiles = %v", skill.CompiledFiles)
	}
	if len(skill.WaitLinks) != 1 {
		t.Fatalf("manifest waitLinks = %#v", skill.WaitLinks)
	}
	link := skill.WaitLinks[0]
	if link.NodeID != "wait-sm-context" || link.Event.Source != "AMF" ||
		link.Event.Service != "Nsmf_PDUSession" || link.Event.Operation != "UpdateSMContext" {
		t.Fatalf("unexpected Wait link: %#v", link)
	}
	if link.Timeout != "PT30S" || link.TimeoutRoute != "handle-update-timeout" {
		t.Fatalf("unexpected Wait timeout/route: %#v", link)
	}
	if link.Ingress.DependencyAlias != wp3IntegrationDependency ||
		link.Ingress.CatalogOwner != wp3IntegrationOwner ||
		link.Ingress.SnapshotPath != wp3CatalogPath ||
		link.Ingress.OperationID != wp3IngressExportID {
		t.Fatalf("unexpected Ingress identity: %#v", link.Ingress)
	}
	_, digests := wp3Artifacts(t)
	if link.Ingress.NormalizedSchemaRef != wp3NormalizedArtifactRef ||
		link.Ingress.NormalizedSchemaDigest != digests[wp3NormalizedArtifactRef] ||
		link.Ingress.NormalizedSchemaSnapshotPath != wp3NormalizedPath {
		t.Fatalf("unexpected normalized Schema link: %#v", link.Ingress)
	}
	if link.Ingress.Protocol.Specification != "3GPP TS 29.502" || link.Ingress.Protocol.Release != "19.5.0" ||
		link.Ingress.Protocol.APIVersion != "v1" || link.Ingress.Protocol.OperationID != "UpdateSmContext" ||
		link.Ingress.Protocol.Method != "POST" || link.Ingress.Protocol.Path != "/sm-contexts/{smContextRef}/modify" {
		t.Fatalf("unexpected protocol metadata: %#v", link.Ingress.Protocol)
	}
	if link.Response.Mode != "none" || link.Response.Declared || link.Response.BoundaryNodeID != "wait-sm-context" {
		t.Fatalf("unexpected response metadata: %#v", link.Response)
	}

	// The manifest must list every emitted file except itself with the exact
	// digest of its bytes.
	if len(manifest.OutputFiles) != len(result.OutputFiles)-1 {
		t.Fatalf("manifest outputFiles = %d, want %d", len(manifest.OutputFiles), len(result.OutputFiles)-1)
	}
	for _, entry := range manifest.OutputFiles {
		raw, ok := v1alpha5ResultFile(result, entry.Path)
		if !ok {
			t.Fatalf("manifest lists missing %q", entry.Path)
		}
		if entry.SHA256 != sha256Hex(raw) {
			t.Fatalf("manifest digest for %q = %q", entry.Path, entry.SHA256)
		}
	}

	// The compiled Procedure document must be canonical and preserve the
	// validated source shape.
	procedureRaw, _ := v1alpha5ResultFile(result, "skills/example/compiled/procedure.json")
	canonical, err := canonicalizeJSON(procedureRaw)
	if err != nil || !bytes.Equal(canonical, procedureRaw) {
		t.Fatalf("procedure.json is not canonical JCS: %v", err)
	}
	var procedure map[string]json.RawMessage
	if err := json.Unmarshal(procedureRaw, &procedure); err != nil {
		t.Fatal(err)
	}
	if string(procedure["start"]) != `"trigger-transfer"` {
		t.Fatalf("compiled Procedure start = %s", procedure["start"])
	}
}

// TestV1Alpha5BuildIsDeterministic proves the same BuildRequest yields
// byte-identical files and identical result identity across repeated builds,
// and that an equivalent permuted request does too.
func TestV1Alpha5BuildIsDeterministic(t *testing.T) {
	first := Build(wp3BuildRequest(t, wp3FixtureOptions{}))
	if !first.Success {
		t.Fatalf("first Build failed: %#v", first.Diagnostics)
	}
	second := Build(wp3BuildRequest(t, wp3FixtureOptions{}))
	if !second.Success {
		t.Fatalf("second Build failed: %#v", second.Diagnostics)
	}
	if len(first.OutputFiles) != len(second.OutputFiles) {
		t.Fatalf("file counts differ: %d vs %d", len(first.OutputFiles), len(second.OutputFiles))
	}
	for index := range first.OutputFiles {
		left, right := first.OutputFiles[index], second.OutputFiles[index]
		if left.Path != right.Path || !bytes.Equal(left.Bytes, right.Bytes) {
			t.Fatalf("repeated Build differs at %q", left.Path)
		}
	}
	if strings.Join(first.DependencyClosure, ",") != strings.Join(second.DependencyClosure, ",") {
		t.Fatalf("dependencyClosure differs: %v vs %v", first.DependencyClosure, second.DependencyClosure)
	}
}

// TestV1Alpha5BuildWaitFreeSucceeds proves a fully admitted Wait-free v1alpha5
// Skill also Builds successfully; the artifact has an empty wait-link list and
// only the dependencies the Skill actually uses.
func TestV1Alpha5BuildWaitFreeSucceeds(t *testing.T) {
	request := wp3BuildRequest(t, wp3FixtureOptions{
		procedure: wp3WaitFreeProcedure,
		contract:  wp3WaitFreeContract,
	})
	if diagnostics := Check(request); len(diagnostics) != 0 {
		t.Fatalf("Wait-free procedure must pass Check, got %#v", diagnostics)
	}
	result := Build(request)
	if !result.Success {
		t.Fatalf("Wait-free v1alpha5 Build must succeed, diagnostics = %#v", result.Diagnostics)
	}
	if err := VerifyBuildResult(result); err != nil {
		t.Fatalf("VerifyBuildResult: %v", err)
	}
	if strings.Join(result.DependencyClosure, ",") != "integration,"+wp3ToolsDependency {
		t.Fatalf("Wait-free dependencyClosure = %v", result.DependencyClosure)
	}
	manifest := v1alpha5DecodeManifest(t, result)
	if len(manifest.DependencyClosure) != 2 || len(manifest.Skills) != 1 {
		t.Fatalf("unexpected Wait-free manifest: %#v", manifest)
	}
	if manifest.Skills[0].WaitLinks == nil || len(manifest.Skills[0].WaitLinks) != 0 {
		t.Fatalf("Wait-free waitLinks must be an empty array: %#v", manifest.Skills[0].WaitLinks)
	}
	linksRaw, ok := v1alpha5ResultFile(result, "skills/example/compiled/ingress-links.json")
	if !ok {
		t.Fatal("Wait-free artifact is missing ingress-links.json")
	}
	if string(linksRaw) != `{"links":[]}` {
		t.Fatalf("empty ingress-links document = %s", linksRaw)
	}
}

// TestV1Alpha5BuildPairedResponseMetadata proves WP6 paired-response producer,
// boundary and result-Schema metadata is carried into the artifact manifest.
func TestV1Alpha5BuildPairedResponseMetadata(t *testing.T) {
	request := wp6BuildRequest(t, wp6FixtureOptions{})
	if diagnostics := Check(request); len(diagnostics) != 0 {
		t.Fatalf("positive WP6 fixture must pass Check, got %#v", diagnostics)
	}
	result := Build(request)
	if !result.Success {
		t.Fatalf("paired-response v1alpha5 Build must succeed, diagnostics = %#v", result.Diagnostics)
	}
	if err := VerifyBuildResult(result); err != nil {
		t.Fatalf("VerifyBuildResult: %v", err)
	}
	manifest := v1alpha5DecodeManifest(t, result)
	if len(manifest.Skills) != 1 || len(manifest.Skills[0].WaitLinks) != 1 {
		t.Fatalf("unexpected manifest: %#v", manifest.Skills)
	}
	response := manifest.Skills[0].WaitLinks[0].Response
	if response.Mode != "paired" || response.Declared || response.BoundaryNodeID != "update-sm-context" {
		t.Fatalf("unexpected paired response metadata: %#v", response)
	}
	if response.ProducerStage != "operation" || response.ResultSchemaRef != wp6ResponseResultRef {
		t.Fatalf("unexpected response producer metadata: %#v", response)
	}
	if strings.Join(response.EarlyFailureTerminals, ",") != "SAFE_STOPPED,UNRESOLVED" {
		t.Fatalf("unexpected early failure terminals: %v", response.EarlyFailureTerminals)
	}
	if !isSHA256Value(response.ResultSchemaDigest) {
		t.Fatalf("response result Schema digest = %q", response.ResultSchemaDigest)
	}
}

// TestV1Alpha5BuildInvalidEmitsNoArtifact keeps the fail-closed source path
// atomic: real WAIT_* diagnostics block Build and no artifact file is emitted.
func TestV1Alpha5BuildInvalidEmitsNoArtifact(t *testing.T) {
	procedure := `start: wait-only-node
nodes:
  wait-only-node:
    wait:
      event:
        source: AMF
        service: Nsmf_PDUSession
        operation: ""
      timeout: PT0S
      routes:
        TIMEOUT: SUCCEEDED
`
	result := Build(v1alpha5TestProcedureRequest(procedure))
	if result.Success {
		t.Fatal("invalid v1alpha5 procedure must not Build")
	}
	if result.APIVersion != BuildResultAPIVersion {
		t.Fatalf("failed BuildResult.apiVersion = %q", result.APIVersion)
	}
	if len(result.OutputFiles) != 0 || len(result.DependencyClosure) != 0 {
		t.Fatalf("failed Build emitted artifact data: %#v", result)
	}
	for _, code := range []string{"WAIT_NODE_STAGE_REQUIRED", "WAIT_DECLARATION_INVALID", "WAIT_TIMEOUT_INVALID", "WAIT_TIMEOUT_ROUTE_INVALID"} {
		if !hasV1Alpha5Diagnostic(result.Diagnostics, code) {
			t.Fatalf("failed Build is missing %s: %#v", code, result.Diagnostics)
		}
	}
	for _, diagnostic := range result.Diagnostics {
		if strings.HasPrefix(diagnostic.Message, "fail-closed: ") {
			t.Fatalf("real source diagnostics must not be masked: %#v", result.Diagnostics)
		}
	}
}

// TestV1Alpha5ArtifactVerificationDetectsTampering proves VerifyBuildResult
// rejects mutated bytes, mutated digests, a missing file and duplicate paths.
func TestV1Alpha5ArtifactVerificationDetectsTampering(t *testing.T) {
	result := Build(wp3BuildRequest(t, wp3FixtureOptions{}))
	if !result.Success {
		t.Fatalf("fixture Build failed: %#v", result.Diagnostics)
	}
	if err := VerifyBuildResult(result); err != nil {
		t.Fatalf("baseline verification failed: %v", err)
	}

	mutated := result
	mutated.OutputFiles = append([]File(nil), result.OutputFiles...)
	for index := range mutated.OutputFiles {
		if mutated.OutputFiles[index].Path == "skills/example/compiled/procedure.json" {
			bytesCopy := append([]byte(nil), mutated.OutputFiles[index].Bytes...)
			bytesCopy[0] ^= 0xff
			mutated.OutputFiles[index].Bytes = bytesCopy
		}
	}
	if err := VerifyBuildResult(mutated); err == nil {
		t.Fatal("mutated artifact bytes were accepted")
	}

	duplicated := result
	duplicated.OutputFiles = append(append([]File(nil), result.OutputFiles...), result.OutputFiles[0])
	if err := VerifyBuildResult(duplicated); err == nil {
		t.Fatal("duplicate artifact path was accepted")
	}

	missing := result
	missing.OutputFiles = result.OutputFiles[1:]
	if err := VerifyBuildResult(missing); err == nil {
		t.Fatal("artifact with a missing file was accepted")
	}

	unlisted := result
	unlisted.OutputFiles = append(append([]File(nil), result.OutputFiles...), File{
		Path:  "skills/example/compiled/extra.json",
		Bytes: []byte(`{}`),
	})
	if err := VerifyBuildResult(unlisted); err == nil {
		t.Fatal("artifact with an unlisted file was accepted")
	}

	// A tampered manifest digest must be rejected even when the tree files are
	// unchanged: rebuild the manifest with a wrong sha256 and re-verify.
	tamperedManifest := result
	tamperedManifest.OutputFiles = append([]File(nil), result.OutputFiles...)
	for index := range tamperedManifest.OutputFiles {
		if tamperedManifest.OutputFiles[index].Path != V1Alpha5ArtifactManifestPath {
			continue
		}
		var manifest v1alpha5ArtifactManifest
		if err := json.Unmarshal(tamperedManifest.OutputFiles[index].Bytes, &manifest); err != nil {
			t.Fatal(err)
		}
		manifest.OutputFiles[0].SHA256 = "sha256:" + strings.Repeat("0", 64)
		rebuilt, err := canonicalizeValue(manifest)
		if err != nil {
			t.Fatal(err)
		}
		tamperedManifest.OutputFiles[index].Bytes = rebuilt
	}
	if err := VerifyBuildResult(tamperedManifest); err == nil {
		t.Fatal("tampered manifest digest was accepted")
	}
}

// TestV1Alpha5ArtifactManifestMatchesFrozenSchema keeps the emitted manifest and
// the shipped frozen contract from drifting. The schema is loaded from the
// repository contract file with no network access, and its metadata must declare
// the frozen v2.0.0 contract status rather than a candidate/draft/partial state.
func TestV1Alpha5ArtifactManifestMatchesFrozenSchema(t *testing.T) {
	schemaBytes, err := os.ReadFile("../../contracts/v1alpha5/artifact-manifest.schema.json")
	if err != nil {
		t.Fatalf("read frozen v1alpha5 manifest schema: %v", err)
	}
	var schemaDocument any
	if err := json.Unmarshal(schemaBytes, &schemaDocument); err != nil {
		t.Fatalf("decode frozen schema: %v", err)
	}
	schemaObject, ok := schemaDocument.(map[string]any)
	if !ok {
		t.Fatal("frozen artifact manifest schema must be a JSON object")
	}
	title, _ := schemaObject["title"].(string)
	description, _ := schemaObject["description"].(string)
	declaration := strings.ToLower(title + "\n" + description)
	if !strings.Contains(declaration, "frozen") {
		t.Fatalf("artifact manifest schema must declare its frozen status: title=%q description=%q", title, description)
	}
	for _, retired := range []string{"candidate", "draft", "partial"} {
		if strings.Contains(declaration, retired) {
			t.Fatalf("artifact manifest schema still describes the contract as %q: title=%q description=%q", retired, title, description)
		}
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err := compiler.AddResource("urn:connectionagent:compiler:artifact-manifest:v1alpha5", schemaDocument); err != nil {
		t.Fatalf("register frozen schema: %v", err)
	}
	compiled, err := compiler.Compile("urn:connectionagent:compiler:artifact-manifest:v1alpha5")
	if err != nil {
		t.Fatalf("compile frozen schema: %v", err)
	}

	result := Build(wp3BuildRequest(t, wp3FixtureOptions{}))
	if !result.Success {
		t.Fatalf("trusted v1alpha5 Build failed: %+v", result.Diagnostics)
	}
	manifest := v1alpha5DecodeManifest(t, result)
	if manifest.ValidationScope.Status != v1alpha5ArtifactValidationStatus {
		t.Fatalf("validationScope.status = %q", manifest.ValidationScope.Status)
	}
	if len(manifest.ValidationScope.ValidatedPasses) == 0 || len(manifest.ValidationScope.NotValidated) == 0 {
		t.Fatalf("validationScope must declare validated and not-validated passes: %#v", manifest.ValidationScope)
	}
	manifestBytes, ok := v1alpha5ResultFile(result, V1Alpha5ArtifactManifestPath)
	if !ok {
		t.Fatalf("BuildResult has no %s", V1Alpha5ArtifactManifestPath)
	}
	var manifestValue any
	if err := json.Unmarshal(manifestBytes, &manifestValue); err != nil {
		t.Fatalf("decode emitted manifest: %v", err)
	}
	if err := compiled.Validate(manifestValue); err != nil {
		t.Fatalf("emitted manifest does not satisfy the frozen schema: %v", err)
	}
}
