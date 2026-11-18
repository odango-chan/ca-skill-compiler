package ingresscatalog

// Pinned OpenAPI response-shape cross-check tests (IN-044).

import "testing"

// newMultipartResponseSnapshot is the synthetic snapshot with a pinned OpenAPI
// operation whose response declares multipart/related content.
func newMultipartResponseSnapshot(t *testing.T) *syntheticSnapshot {
	t.Helper()
	openAPI := `{"openapi":"3.0.0","info":{"title":"Synthetic Ingress API","version":"1.0.0"},"paths":{"/synthetic/sm-contexts/{smContextRef}/modify":{"post":{"operationId":"SyntheticUpdateSmContext","parameters":[{"name":"smContextRef","in":"path","required":true,"schema":{"type":"string"}}],"requestBody":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Request"}}}},"responses":{"200":{"description":"updated","content":{"multipart/related":{"schema":{"type":"object"}},"application/json":{"schema":{"$ref":"#/components/schemas/Request"}}}}}}}},"components":{"schemas":{"Request":{"type":"object","properties":{"updateData":{"type":"object"}},"required":["updateData"]}}}}`
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

// TestPinnedOpenAPIResponseMultipartRequiresMultipartMapping covers IN-044: a
// pinned OpenAPI response requiring multipart/related content while the export
// declares no multipart representation emits INGRESS_RESPONSE_MAPPING_INVALID.
func TestPinnedOpenAPIResponseMultipartRequiresMultipartMapping(t *testing.T) {
	snapshot := newMultipartResponseSnapshot(t)
	admission := snapshot.admit(t, snapshot.catalogBytes(t))
	expectIngressDiagnosticAt(t, admission, CodeIngressResponseMappingInvalid,
		"#/spec/operations/0/response/dispositions")
}

// TestPinnedOpenAPIJSONOnlyResponseHasNoMappingDiagnostic proves a JSON-only
// pinned response does not trigger the multipart cross-check.
func TestPinnedOpenAPIJSONOnlyResponseHasNoMappingDiagnostic(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	admission := snapshot.admit(t, snapshot.catalogBytes(t))
	if !admission.Valid() {
		t.Fatalf("valid synthetic catalog rejected: %#v", admission.Diagnostics)
	}
	for _, diagnostic := range admission.Diagnostics {
		if diagnostic.Code == CodeIngressResponseMappingInvalid {
			t.Fatalf("JSON-only pinned response produced %#v", diagnostic)
		}
	}
}
