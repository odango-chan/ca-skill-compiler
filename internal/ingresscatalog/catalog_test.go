package ingresscatalog

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const (
	syntheticOpenAPIRef    = "urn:example:synthetic:openapi"
	syntheticRequestRef    = "urn:example:synthetic:schema:request"
	syntheticNormalizedRef = "urn:example:synthetic:schema:normalized"
	syntheticBrokenRef     = "urn:example:synthetic:schema:broken"
	syntheticDependency    = "integration.synthetic"
	syntheticCatalogPath   = "ingress/catalog.yaml"
)

type syntheticSnapshot struct {
	context *ResolutionContext
	exports map[string]RootRef
	values  map[string]string
}

func newSyntheticSnapshot(t *testing.T) *syntheticSnapshot {
	t.Helper()
	openAPI := `{"openapi":"3.0.0","info":{"title":"Synthetic Ingress API","version":"1.0.0"},"paths":{"/synthetic/sm-contexts/{smContextRef}/modify":{"post":{"operationId":"SyntheticUpdateSmContext","parameters":[{"name":"smContextRef","in":"path","required":true,"schema":{"type":"string"}}],"requestBody":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Request"}}}},"responses":{"204":{"description":"no content"}}}}},"components":{"schemas":{"Request":{"type":"object","properties":{"updateData":{"type":"object"}},"required":["updateData"]}}}}`
	requestSchema := `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"https://schemas.invalid/synthetic/request","type":"object","properties":{"updateData":{"type":"object"}},"required":["updateData"]}`
	normalizedSchema := `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"https://schemas.invalid/synthetic/normalized","type":"object","properties":{"updateData":{"type":"object"}},"required":["updateData"]}`
	brokenSchema := `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"https://schemas.invalid/synthetic/broken","type":"object","properties":{"x":{"$ref":"missing.json#/$defs/X"}}}`

	context := newContext(t,
		artifact(t, syntheticOpenAPIRef, "openapi/synthetic.yaml", DialectOpenAPI30, openAPI, ""),
		artifact(t, syntheticRequestRef, "schemas/request.json", DialectJSONSchema202012, requestSchema, ""),
		artifact(t, syntheticNormalizedRef, "schemas/normalized.json", DialectJSONSchema202012, normalizedSchema, ""),
		artifact(t, syntheticBrokenRef, "schemas/broken.json", DialectJSONSchema202012, brokenSchema, ""),
	)
	exports := map[string]RootRef{
		syntheticRequestRef:    {ArtifactRef: syntheticRequestRef, Pointer: "#"},
		syntheticNormalizedRef: {ArtifactRef: syntheticNormalizedRef, Pointer: "#"},
	}
	requestDigest, err := context.ComputeSchemaClosure(exports[syntheticRequestRef])
	if err != nil {
		t.Fatalf("request closure: %v", err)
	}
	normalizedDigest, err := context.ComputeSchemaClosure(exports[syntheticNormalizedRef])
	if err != nil {
		t.Fatalf("normalized closure: %v", err)
	}
	return &syntheticSnapshot{
		context: context,
		exports: exports,
		values: map[string]string{
			"{{OPENAPI_ARTIFACT_REF}}":     syntheticOpenAPIRef,
			"{{OPENAPI_DIGEST}}":           digestOf([]byte(openAPI)),
			"{{REQUEST_SCHEMA_REF}}":       syntheticRequestRef,
			"{{REQUEST_SCHEMA_DIGEST}}":    requestDigest.Digest,
			"{{NORMALIZED_SCHEMA_REF}}":    syntheticNormalizedRef,
			"{{NORMALIZED_SCHEMA_DIGEST}}": normalizedDigest.Digest,
		},
	}
}

func (s *syntheticSnapshot) trusted() *TrustedSnapshot {
	return NewTrustedSnapshot(s.context, s.exports)
}

func (s *syntheticSnapshot) catalogBytes(t *testing.T) []byte {
	t.Helper()
	template, err := os.ReadFile("testdata/valid-catalog.yaml.tmpl")
	if err != nil {
		t.Fatalf("read catalog template: %v", err)
	}
	text := string(template)
	replacer := make([]string, 0, len(s.values)*2)
	for placeholder, value := range s.values {
		replacer = append(replacer, placeholder, value)
	}
	return []byte(strings.NewReplacer(replacer...).Replace(text))
}

func (s *syntheticSnapshot) admit(t *testing.T, bytes []byte) *Admission {
	t.Helper()
	return AdmitCatalog(CatalogInput{
		Bytes:           bytes,
		DependencyAlias: syntheticDependency,
		SnapshotPath:    syntheticCatalogPath,
	}, s.trusted())
}

// catalogMap parses the substituted synthetic catalog into a mutable map.
func (s *syntheticSnapshot) catalogMap(t *testing.T) map[string]any {
	t.Helper()
	parsed, err := parseStrictJSONValue(s.catalogBytes(t))
	if err != nil {
		t.Fatalf("parse synthetic catalog: %v", err)
	}
	root, ok := parsed.(map[string]any)
	if !ok {
		t.Fatalf("synthetic catalog root is not an object")
	}
	return root
}

func cloneJSONValue(t *testing.T, value any) any {
	t.Helper()
	encoded := mustMarshalJSON(t, value)
	return decodeJSONValue(t, encoded)
}

func operationsOf(t *testing.T, root map[string]any) []any {
	t.Helper()
	spec, ok := root["spec"].(map[string]any)
	if !ok {
		t.Fatalf("catalog spec missing")
	}
	operations, ok := spec["operations"].([]any)
	if !ok || len(operations) == 0 {
		t.Fatalf("catalog operations missing")
	}
	return operations
}

func firstOperation(t *testing.T, root map[string]any) map[string]any {
	t.Helper()
	operation, ok := operationsOf(t, root)[0].(map[string]any)
	if !ok {
		t.Fatalf("catalog operation is not an object")
	}
	return operation
}

func firstSourceSchema(t *testing.T, root map[string]any) map[string]any {
	t.Helper()
	request := firstOperation(t, root)["request"].(map[string]any)
	sourceSchemas := request["sourceSchemas"].([]any)
	source, ok := sourceSchemas[0].(map[string]any)
	if !ok {
		t.Fatalf("source schema is not an object")
	}
	return source
}

func findDiagnostic(t *testing.T, admission *Admission, code string) Diagnostic {
	t.Helper()
	for _, diagnostic := range admission.Diagnostics {
		if diagnostic.Code == code {
			return diagnostic
		}
	}
	t.Fatalf("expected diagnostic %s, got %#v", code, admission.Diagnostics)
	return Diagnostic{}
}

func assertDiagnosticLocation(t *testing.T, diagnostic Diagnostic, pointer string) {
	t.Helper()
	if diagnostic.SourceLocation.JSONPointer != pointer {
		t.Fatalf("diagnostic %s pointer = %q, want %q", diagnostic.Code, diagnostic.SourceLocation.JSONPointer, pointer)
	}
	if diagnostic.SourceLocation.DependencyAlias != syntheticDependency {
		t.Fatalf("diagnostic dependency alias = %q", diagnostic.SourceLocation.DependencyAlias)
	}
	if diagnostic.SourceLocation.SnapshotPath != syntheticCatalogPath {
		t.Fatalf("diagnostic snapshot path = %q", diagnostic.SourceLocation.SnapshotPath)
	}
	if diagnostic.Phase != PhaseCStatic || diagnostic.Severity != SeverityError {
		t.Fatalf("diagnostic phase/severity = %s/%s", diagnostic.Phase, diagnostic.Severity)
	}
}

func TestValidSyntheticCatalogAdmission(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	bytes := snapshot.catalogBytes(t)
	admission := snapshot.admit(t, bytes)
	if !admission.Valid() {
		t.Fatalf("valid synthetic catalog rejected: %#v", admission.Diagnostics)
	}
	if admission.Catalog == nil || admission.Catalog.APIVersion != IngressCatalogAPIVersion ||
		admission.Catalog.Kind != IngressCatalogKind || admission.Catalog.Owner != "example.integration-owner" {
		t.Fatalf("catalog projection mismatch: %#v", admission.Catalog)
	}
	if len(admission.Catalog.Operations) != 1 {
		t.Fatalf("operation count = %d", len(admission.Catalog.Operations))
	}
	operation := admission.Catalog.Operations[0]
	if operation.ID != "synthetic.update-sm-context" ||
		operation.Event != (EventSelector{Source: "AMF", Service: "Nsmf_PDUSession", Operation: "UpdateSMContext"}) {
		t.Fatalf("operation projection mismatch: %#v", operation)
	}
	if operation.Protocol.OpenAPI.ArtifactRef != syntheticOpenAPIRef ||
		operation.Protocol.OpenAPI.OperationID != "SyntheticUpdateSmContext" ||
		operation.Protocol.OpenAPI.Method != "POST" ||
		operation.Protocol.OpenAPI.Path != "/synthetic/sm-contexts/{smContextRef}/modify" {
		t.Fatalf("protocol projection mismatch: %#v", operation.Protocol)
	}
	if operation.Request.Normalized.SchemaRef != syntheticNormalizedRef ||
		operation.Request.Normalized.SchemaDigest == "" || len(operation.Request.SourceSchemas) != 1 ||
		operation.Request.MaxBytes != 4194304 {
		t.Fatalf("request projection mismatch: %#v", operation.Request)
	}
	if len(admission.VerifiedSchemaClosures) != 2 {
		t.Fatalf("verified schema closures = %d", len(admission.VerifiedSchemaClosures))
	}
	if len(admission.DeclaredSchemaBindings) != 2 {
		t.Fatalf("declared schema bindings = %d", len(admission.DeclaredSchemaBindings))
	}
	// The same input must produce identical diagnostics and projections.
	again := snapshot.admit(t, bytes)
	if len(again.Diagnostics) != 0 || again.Catalog == nil ||
		again.Catalog.Operations[0].Protocol.OpenAPI.ArtifactDigest != operation.Protocol.OpenAPI.ArtifactDigest {
		t.Fatalf("admission is not deterministic")
	}
}

func TestCatalogStructuralSchemaCopiesAndNoNetwork(t *testing.T) {
	embedded := structuralSchemaBytes
	contractCopy, err := os.ReadFile("../../contracts/v1alpha1/ingresscatalog.schema.json")
	if err != nil {
		t.Fatalf("read contracts copy: %v", err)
	}
	if string(embedded) != string(contractCopy) {
		t.Fatalf("contracts/v1alpha1 schema copy differs from the embedded schema")
	}
	packageCopy, err := os.ReadFile("schemas/ingresscatalog.schema.json")
	if err != nil {
		t.Fatalf("read package schema copy: %v", err)
	}
	if string(embedded) != string(packageCopy) {
		t.Fatalf("package schema copy differs from the embedded schema")
	}
	if _, err := compiledCatalogSchema(); err != nil {
		t.Fatalf("compile structural schema: %v", err)
	}
}

func TestCatalogInvalidAPIVersionAndKind(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	for field, value := range map[string]string{
		"apiVersion": "connectionagent.io/ingresscatalog/v9",
		"kind":       "NotIngressCatalog",
	} {
		root := snapshot.catalogMap(t)
		root[field] = value
		admission := snapshot.admit(t, mustMarshalJSON(t, root))
		diagnostic := findDiagnostic(t, admission, CodeIngressCatalogInvalid)
		assertDiagnosticLocation(t, diagnostic, "#/"+field)
		if diagnostic.RegistryID != RegistryIDIngressDiagnostics ||
			diagnostic.MessageKey != ingressMessageKeys[CodeIngressCatalogInvalid] {
			t.Fatalf("ingress diagnostic metadata mismatch: %#v", diagnostic)
		}
	}
	// A non-object Catalog root is still INGRESS_CATALOG_INVALID at '#'.
	admission := snapshot.admit(t, []byte("- not\n- a\n- catalog\n"))
	findDiagnostic(t, admission, CodeIngressCatalogInvalid)
}

func TestCatalogMissingOwnerIsStructural(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	delete(root["metadata"].(map[string]any), "owner")
	admission := snapshot.admit(t, mustMarshalJSON(t, root))
	diagnostic := findDiagnostic(t, admission, CodeIngressCatalogInvalid)
	if !strings.HasPrefix(diagnostic.SourceLocation.JSONPointer, "#/metadata") {
		t.Fatalf("missing owner pointer = %q", diagnostic.SourceLocation.JSONPointer)
	}
}

func TestCatalogDuplicateExportID(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	operations := operationsOf(t, root)
	duplicate := cloneJSONValue(t, operations[0]).(map[string]any)
	duplicate["event"] = map[string]any{"source": "AMF", "service": "Nsmf_PDUSession", "operation": "DifferentOperation"}
	spec := root["spec"].(map[string]any)
	spec["operations"] = append(operations, duplicate)
	admission := snapshot.admit(t, mustMarshalJSON(t, root))
	diagnostic := findDiagnostic(t, admission, CodeIngressExportIDDuplicate)
	assertDiagnosticLocation(t, diagnostic, "#/spec/operations/1/id")
	if len(diagnostic.RelatedLocations) != 1 || diagnostic.RelatedLocations[0].JSONPointer != "#/spec/operations/0/id" {
		t.Fatalf("duplicate export related location mismatch: %#v", diagnostic.RelatedLocations)
	}
	if !strings.Contains(diagnostic.Message, "synthetic.update-sm-context") {
		t.Fatalf("duplicate export message = %q", diagnostic.Message)
	}
}

func TestCatalogDuplicateEventSelector(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	operations := operationsOf(t, root)
	duplicate := cloneJSONValue(t, operations[0]).(map[string]any)
	duplicate["id"] = "synthetic.other-export"
	spec := root["spec"].(map[string]any)
	spec["operations"] = append(operations, duplicate)
	admission := snapshot.admit(t, mustMarshalJSON(t, root))
	diagnostic := findDiagnostic(t, admission, CodeIngressEventSelectorDuplicate)
	assertDiagnosticLocation(t, diagnostic, "#/spec/operations/1/event")
	if len(diagnostic.RelatedLocations) != 1 || diagnostic.RelatedLocations[0].JSONPointer != "#/spec/operations/0/event" {
		t.Fatalf("duplicate selector related location mismatch: %#v", diagnostic.RelatedLocations)
	}
	if !strings.Contains(diagnostic.Message, "AMF/Nsmf_PDUSession/UpdateSMContext") {
		t.Fatalf("duplicate selector message = %q", diagnostic.Message)
	}
}

func TestCatalogProtocolMetadataMismatch(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	cases := []struct {
		field   string
		value   any
		pointer string
		related bool
	}{
		{"method", "PUT", "#/spec/operations/0/protocol/openapi/method", true},
		{"path", "/synthetic/other/{smContextRef}", "#/spec/operations/0/protocol/openapi/path", true},
		{"operationId", "DifferentOperation", "#/spec/operations/0/protocol/openapi/operationId", false},
	}
	for _, testCase := range cases {
		root := snapshot.catalogMap(t)
		openAPI := firstOperation(t, root)["protocol"].(map[string]any)["openapi"].(map[string]any)
		openAPI[testCase.field] = testCase.value
		admission := snapshot.admit(t, mustMarshalJSON(t, root))
		diagnostic := findDiagnostic(t, admission, CodeIngressProtocolArtifactInvalid)
		assertDiagnosticLocation(t, diagnostic, testCase.pointer)
		if testCase.related {
			if len(diagnostic.RelatedLocations) != 1 ||
				!strings.HasPrefix(diagnostic.RelatedLocations[0].JSONPointer, "#/paths/") {
				t.Fatalf("protocol related location mismatch: %#v", diagnostic.RelatedLocations)
			}
		}
	}
}

func TestCatalogProtocolArtifactIdentityUnverified(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	// Missing artifactRef in the trusted snapshot.
	root := snapshot.catalogMap(t)
	openAPI := firstOperation(t, root)["protocol"].(map[string]any)["openapi"].(map[string]any)
	openAPI["artifactRef"] = "urn:example:synthetic:missing-openapi"
	admission := snapshot.admit(t, mustMarshalJSON(t, root))
	diagnostic := findDiagnostic(t, admission, CodeDependencyIdentityUnverified)
	assertDiagnosticLocation(t, diagnostic, "#/spec/operations/0/protocol/openapi/artifactRef")
	if diagnostic.RegistryID != "" || diagnostic.MessageKey != "" {
		t.Fatalf("reused dependency diagnostic must not claim the Ingress registry: %#v", diagnostic)
	}

	// Raw-byte digest mismatch for an artifact that exists in the snapshot.
	root = snapshot.catalogMap(t)
	openAPI = firstOperation(t, root)["protocol"].(map[string]any)["openapi"].(map[string]any)
	openAPI["artifactDigest"] = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	admission = snapshot.admit(t, mustMarshalJSON(t, root))
	diagnostic = findDiagnostic(t, admission, CodeDependencyIdentityUnverified)
	assertDiagnosticLocation(t, diagnostic, "#/spec/operations/0/protocol/openapi/artifactDigest")
}

func TestCatalogMissingSchemaClosure(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	firstSourceSchema(t, root)["schemaRef"] = "urn:example:synthetic:schema:not-declared"
	admission := snapshot.admit(t, mustMarshalJSON(t, root))
	diagnostic := findDiagnostic(t, admission, CodeSchemaMissing)
	assertDiagnosticLocation(t, diagnostic, "#/spec/operations/0/request/sourceSchemas/0/schemaRef")
}

func TestCatalogSchemaDigestMismatch(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	firstSourceSchema(t, root)["schemaDigest"] = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	admission := snapshot.admit(t, mustMarshalJSON(t, root))
	diagnostic := findDiagnostic(t, admission, CodeSchemaInvalid)
	assertDiagnosticLocation(t, diagnostic, "#/spec/operations/0/request/sourceSchemas/0/schemaDigest")
}

func TestCatalogInvalidSchemaClosure(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	exports := snapshot.trusted()
	exports.SchemaExports[syntheticRequestRef] = RootRef{ArtifactRef: syntheticBrokenRef, Pointer: "#"}
	admission := AdmitCatalog(CatalogInput{
		Bytes:           snapshot.catalogBytes(t),
		DependencyAlias: syntheticDependency,
		SnapshotPath:    syntheticCatalogPath,
	}, exports)
	diagnostic := findDiagnostic(t, admission, CodeSchemaInvalid)
	assertDiagnosticLocation(t, diagnostic, "#/spec/operations/0/request/sourceSchemas/0/schemaRef")
	if !strings.Contains(diagnostic.Message, "Schema Closure Digest v1") {
		t.Fatalf("schema invalid message = %q", diagnostic.Message)
	}
}

func TestCatalogDuplicateYAMLKeysRejected(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	bytes := []byte(strings.Replace(string(snapshot.catalogBytes(t)),
		"  owner: example.integration-owner\n",
		"  owner: example.integration-owner\n  owner: example.integration-owner\n", 1))
	admission := snapshot.admit(t, bytes)
	diagnostic := findDiagnostic(t, admission, CodeYAMLParseError)
	assertDiagnosticLocation(t, diagnostic, "#")
	if diagnostic.RegistryID != "" {
		t.Fatalf("YAML_PARSE_ERROR is not Ingress-owned: %#v", diagnostic)
	}
}

func TestCatalogUnsafeIntegerRejected(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	firstOperation(t, root)["request"].(map[string]any)["maxBytes"] = int64(1) << 60
	admission := snapshot.admit(t, mustMarshalJSON(t, root))
	findDiagnostic(t, admission, CodeYAMLParseError)
}

func TestCatalogDiagnosticOrdering(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	firstSourceSchema(t, root)["schemaRef"] = "urn:example:synthetic:schema:not-declared"
	operations := operationsOf(t, root)
	duplicate := cloneJSONValue(t, operations[0]).(map[string]any)
	duplicate["event"] = map[string]any{"source": "AMF", "service": "Nsmf_PDUSession", "operation": "Different"}
	root["spec"].(map[string]any)["operations"] = append(operations, duplicate)
	admission := snapshot.admit(t, mustMarshalJSON(t, root))
	if len(admission.Diagnostics) < 2 {
		t.Fatalf("expected co-reported diagnostics, got %#v", admission.Diagnostics)
	}
	for index := 1; index < len(admission.Diagnostics); index++ {
		previous := diagnosticPrecedenceOf(admission.Diagnostics[index-1].Code)
		current := diagnosticPrecedenceOf(admission.Diagnostics[index].Code)
		if previous > current {
			t.Fatalf("diagnostics are not ordered by precedence: %#v", admission.Diagnostics)
		}
	}
	if admission.Diagnostics[0].Code != CodeIngressExportIDDuplicate {
		t.Fatalf("expected duplicate export id first, got %#v", admission.Diagnostics)
	}
}

func TestCatalogSchemaDigestFormatAndPresence(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	// Malformed digest format is a structural failure.
	root := snapshot.catalogMap(t)
	firstSourceSchema(t, root)["schemaDigest"] = "not-a-digest"
	admission := snapshot.admit(t, mustMarshalJSON(t, root))
	diagnostic := findDiagnostic(t, admission, CodeIngressCatalogInvalid)
	if !strings.Contains(diagnostic.SourceLocation.JSONPointer, "/schemaDigest") {
		t.Fatalf("malformed digest pointer = %q", diagnostic.SourceLocation.JSONPointer)
	}
	// Missing required schemaDigest is also a structural failure.
	root = snapshot.catalogMap(t)
	delete(firstSourceSchema(t, root), "schemaDigest")
	admission = snapshot.admit(t, mustMarshalJSON(t, root))
	findDiagnostic(t, admission, CodeIngressCatalogInvalid)
	// Missing protocol metadata is a structural failure.
	root = snapshot.catalogMap(t)
	delete(firstOperation(t, root)["protocol"].(map[string]any)["openapi"].(map[string]any), "method")
	admission = snapshot.admit(t, mustMarshalJSON(t, root))
	findDiagnostic(t, admission, CodeIngressCatalogInvalid)
}

func TestCatalogProjectionIsDeterministicJSON(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	admission := snapshot.admit(t, snapshot.catalogBytes(t))
	encoded, err := json.Marshal(admission.Catalog)
	if err != nil {
		t.Fatalf("marshal catalog projection: %v", err)
	}
	if !strings.Contains(string(encoded), `"operationId":"SyntheticUpdateSmContext"`) {
		t.Fatalf("unexpected projection JSON: %s", encoded)
	}
}
