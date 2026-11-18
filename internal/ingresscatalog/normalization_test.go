package ingresscatalog

// Ingress-owned request normalization, binary lifecycle and peer profile static
// semantics tests. Every fixture starts from the synthetic valid catalog and
// changes exactly one declaration (Appendix E.6 minimal-change method).

import (
	"strings"
	"testing"
)

func (s *syntheticSnapshot) admitMap(t *testing.T, root map[string]any) *Admission {
	t.Helper()
	return s.admit(t, mustMarshalJSON(t, root))
}

func requestOfFirstOperation(t *testing.T, root map[string]any) map[string]any {
	t.Helper()
	request, ok := firstOperation(t, root)["request"].(map[string]any)
	if !ok {
		t.Fatal("catalog request is not an object")
	}
	return request
}

func firstSourceSchemaDigest(t *testing.T, root map[string]any) string {
	t.Helper()
	return stringField(firstSourceSchema(t, root), "schemaDigest")
}

func firstRepresentation(t *testing.T, root map[string]any) map[string]any {
	t.Helper()
	normalization := requestOfFirstOperation(t, root)["normalization"].(map[string]any)
	representations := normalization["representations"].([]any)
	representation, ok := representations[0].(map[string]any)
	if !ok {
		t.Fatal("catalog representation is not an object")
	}
	return representation
}

func firstMapping(t *testing.T, root map[string]any) map[string]any {
	t.Helper()
	mappings := firstRepresentation(t, root)["mappings"].([]any)
	mapping, ok := mappings[0].(map[string]any)
	if !ok {
		t.Fatal("catalog mapping is not an object")
	}
	return mapping
}

func expectIngressDiagnosticAt(t *testing.T, admission *Admission, code, pointer string) {
	t.Helper()
	if admission.Valid() {
		t.Fatalf("catalog must be rejected with %s", code)
	}
	for _, diagnostic := range admission.Diagnostics {
		if diagnostic.Code != code {
			continue
		}
		if diagnostic.SourceLocation.JSONPointer == pointer {
			assertDiagnosticLocation(t, diagnostic, pointer)
			return
		}
	}
	t.Fatalf("expected %s at %q, got %#v", code, pointer, admission.Diagnostics)
}

// multipartRequestFixture turns the synthetic catalog into a structurally valid
// multipart request whose JSON part maps to /updateData and whose binary part is
// exposed through binary-content-ref.
func multipartRequestFixture(t *testing.T, root map[string]any) {
	t.Helper()
	sourceDigest := firstSourceSchemaDigest(t, root)
	request := requestOfFirstOperation(t, root)
	request["acceptedMediaTypes"] = []any{"multipart/related"}
	request["sourceSchemas"] = []any{map[string]any{
		"mediaType":    "multipart/related",
		"schemaRef":    syntheticRequestRef,
		"schemaDigest": sourceDigest,
	}}
	request["normalization"] = map[string]any{
		"representations": []any{map[string]any{
			"mediaType": "multipart/related",
			"parts": []any{
				map[string]any{
					"name": "json-root", "mediaType": "application/json", "kind": "json",
					"required": true, "schemaRef": syntheticRequestRef, "schemaDigest": sourceDigest,
					"maxBytes": int64(4096),
				},
				map[string]any{
					"name": "n1-sm", "mediaType": "application/octet-stream", "kind": "binary",
					"required": true, "maxBytes": int64(4096),
				},
			},
			"minBinaryParts": int64(1),
			"maxBinaryParts": int64(1),
			"mappings": []any{
				map[string]any{
					"source":    map[string]any{"kind": "multipart-json", "part": "json-root", "pointer": ""},
					"target":    "/updateData",
					"transform": "identity",
				},
				map[string]any{
					"source": map[string]any{
						"kind": "multipart-binary", "part": "n1-sm",
						"contentIdReference": map[string]any{"part": "json-root", "pointer": "/n1SmMsg"},
					},
					"target":    "/updateData",
					"transform": "binary-content-ref",
				},
			},
		}},
	}
	request["binaryContent"] = map[string]any{
		"referenceSchemaRef": syntheticNormalizedRef,
		"digestAlgorithm":    "sha-256",
		"immutable":          true,
		"access":             "authorized-tool-only",
		"retention":          "PT120S",
		"maxBytes":           int64(4096),
	}
}

// IN-039: acceptedMediaTypes, sourceSchemas and normalization.representations
// media types must be one-to-one.
func TestNormalizationMediaTypeCorrespondence(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	firstRepresentation(t, root)["mediaType"] = "application/xml"
	admission := snapshot.admitMap(t, root)
	expectIngressDiagnosticAt(t, admission, CodeIngressNormalizationInvalid,
		"#/spec/operations/0/request/normalization/representations")
}

// IN-014/IN-039: duplicate source media types are rejected.
func TestNormalizationDuplicateSourceMediaType(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	request := requestOfFirstOperation(t, root)
	sources := request["sourceSchemas"].([]any)
	sources = append(sources, map[string]any{
		"mediaType":    "application/json",
		"schemaRef":    syntheticNormalizedRef,
		"schemaDigest": firstSourceSchemaDigest(t, root),
	})
	request["sourceSchemas"] = sources
	admission := snapshot.admitMap(t, root)
	expectIngressDiagnosticAt(t, admission, CodeIngressNormalizationInvalid,
		"#/spec/operations/0/request/sourceSchemas")
}

// IN-015: every required normalized property must be provided exactly once.
func TestNormalizationRequiredNormalizedPropertyUncovered(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	firstMapping(t, root)["target"] = "/unrelatedTarget"
	admission := snapshot.admitMap(t, root)
	expectIngressDiagnosticAt(t, admission, CodeIngressNormalizationInvalid,
		"#/spec/operations/0/request/normalization/representations/0/mappings")
}

// IN-059: a dangling normalized target is invalid.
func TestNormalizationDanglingTarget(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	firstMapping(t, root)["target"] = "/notInNormalizedSchema"
	admission := snapshot.admitMap(t, root)
	expectIngressDiagnosticAt(t, admission, CodeIngressNormalizationInvalid,
		"#/spec/operations/0/request/normalization/representations/0/mappings/0/target")
}

// IN-059: a transform inconsistent with the target property type is invalid.
func TestNormalizationTransformTargetTypeMismatch(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	firstMapping(t, root)["transform"] = "parse-integer"
	admission := snapshot.admitMap(t, root)
	expectIngressDiagnosticAt(t, admission, CodeIngressNormalizationInvalid,
		"#/spec/operations/0/request/normalization/representations/0/mappings/0/transform")
}

// IN-059: duplicate mapping targets are invalid even when the source selectors
// differ (the structural schema cannot express this).
func TestNormalizationDuplicateTarget(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	representation := firstRepresentation(t, root)
	mappings := representation["mappings"].([]any)
	mappings = append(mappings, map[string]any{
		"source":    map[string]any{"kind": "json-body", "pointer": "/otherInput"},
		"target":    "/updateData",
		"transform": "identity",
	})
	representation["mappings"] = mappings
	admission := snapshot.admitMap(t, root)
	expectIngressDiagnosticAt(t, admission, CodeIngressNormalizationInvalid,
		"#/spec/operations/0/request/normalization/representations/0/mappings/1/target")
}

// IN-014: declared multipart binary-part bounds must be consistent.
func TestNormalizationMultipartBoundsInconsistent(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	multipartRequestFixture(t, root)
	representation := firstRepresentation(t, root)
	representation["minBinaryParts"] = int64(2)
	representation["maxBinaryParts"] = int64(1)
	admission := snapshot.admitMap(t, root)
	expectIngressDiagnosticAt(t, admission, CodeIngressNormalizationInvalid,
		"#/spec/operations/0/request/normalization/representations/0/minBinaryParts")
}

// IN-016: the immutable binary-content reference Schema must resolve in the
// trusted snapshot.
func TestBinaryContentReferenceSchemaUnresolved(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	multipartRequestFixture(t, root)
	requestOfFirstOperation(t, root)["binaryContent"].(map[string]any)["referenceSchemaRef"] = "urn:example:missing:binary-schema"
	admission := snapshot.admitMap(t, root)
	expectIngressDiagnosticAt(t, admission, CodeIngressBinaryContentLifecycleInvalid,
		"#/spec/operations/0/request/binaryContent/referenceSchemaRef")
}

// IN-042: binary retention must cover the declared late-arrival recovery window.
func TestBinaryContentRetentionBelowRecoveryWindow(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	multipartRequestFixture(t, root)
	requestOfFirstOperation(t, root)["binaryContent"].(map[string]any)["retention"] = "PT10S"
	stale := objectField(objectField(firstOperation(t, root), "correlation"), "staleDelivery")
	stale["maximumLateArrival"] = "PT30S"
	stale["clockSkewAllowance"] = "PT5S"
	stale["tombstoneRetention"] = "PT60S"
	admission := snapshot.admitMap(t, root)
	expectIngressDiagnosticAt(t, admission, CodeIngressBinaryContentLifecycleInvalid,
		"#/spec/operations/0/request/binaryContent/retention")
}

// IN-058: a malformed peer profile reference is rejected; the released
// alias::resource form is accepted.
func TestPeerProfileReferenceInvalid(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := snapshot.catalogMap(t)
	security := firstOperation(t, root)["security"].(map[string]any)
	security["authenticationProfileRef"] = "not-a-qualified-reference"
	admission := snapshot.admitMap(t, root)
	expectIngressDiagnosticAt(t, admission, CodeIngressPeerProfileReferenceInvalid,
		"#/spec/operations/0/security/authenticationProfileRef")

	valid := newSyntheticSnapshot(t)
	validRoot := valid.catalogMap(t)
	validSecurity := firstOperation(t, validRoot)["security"].(map[string]any)
	validSecurity["authenticationProfileRef"] = "deployment-auth::amf-peer"
	validSecurity["authorizationPolicyRef"] = "deployment-auth::nsmf-update-sm-context"
	if admission := valid.admitMap(t, validRoot); !admission.Valid() {
		t.Fatalf("qualified peer profile references must be admitted, got %#v", admission.Diagnostics)
	}
}

// The normalization and peer profile validators must not report diagnostics for
// the valid synthetic catalog.
func TestValidCatalogHasNoStaticSemanticDiagnostics(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	admission := snapshot.admit(t, snapshot.catalogBytes(t))
	if !admission.Valid() {
		t.Fatalf("valid catalog rejected: %#v", admission.Diagnostics)
	}
	for _, diagnostic := range admission.Diagnostics {
		if strings.HasPrefix(diagnostic.Code, "INGRESS_NORMALIZATION_INVALID") ||
			strings.HasPrefix(diagnostic.Code, "INGRESS_PEER_PROFILE_REFERENCE_INVALID") {
			t.Fatalf("valid catalog produced %#v", diagnostic)
		}
	}
}
