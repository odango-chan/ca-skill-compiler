package ingresscatalog

import (
	"bytes"
	"encoding/json"
	"testing"
)

const testOpenAPIArtifactRef = "urn:test:openapi"

func singleOpenAPIContext(t *testing.T, raw string) *ResolutionContext {
	t.Helper()
	return newContext(t, artifact(t, testOpenAPIArtifactRef, "openapi/service.yaml", DialectOpenAPI30, raw, ""))
}

func singleJSONSchemaContext(t *testing.T, raw string) *ResolutionContext {
	t.Helper()
	return newContext(t, artifact(t, "urn:test:json-schema", "schemas/root.json", DialectJSONSchema202012, raw, ""))
}

// TestStaleDeclaredArtifactDigestRejected proves the trusted context rejects a
// supplied digest that disagrees with SHA-256 of the raw bytes (§2.1).
func TestStaleDeclaredArtifactDigestRejected(t *testing.T) {
	_, err := NewResolutionContext(testSnapshotBase, []SnapshotArtifact{
		artifact(t, testOpenAPIArtifactRef, "openapi/service.yaml", DialectOpenAPI30,
			`{"openapi":"3.0.0","components":{"schemas":{"R":{"type":"object"}}}}`,
			"sha256:0000000000000000000000000000000000000000000000000000000000000000"),
	})
	wantCode(t, err, ErrArtifactDigest)
}

// TestRawBytesChangeWithIdenticalParsedValueChangesDigest proves the digest is
// bound to the exact artifact bytes, not only the selected Schema value (§5).
func TestRawBytesChangeWithIdenticalParsedValueChangesDigest(t *testing.T) {
	fixture := loadVector(t)
	artifacts := vectorArtifacts(t, fixture)
	expected, err := newContext(t, artifacts...).ComputeSchemaClosure(RootRef{fixture.Root.ArtifactRef, fixture.Root.Pointer})
	if err != nil {
		t.Fatalf("baseline closure: %v", err)
	}
	// Reformat only the root artifact's source bytes. The parsed document value
	// stays identical, but the artifact digest and therefore the closure digest
	// must change.
	var reformatted bytes.Buffer
	if err := json.Indent(&reformatted, []byte(fixture.ResolutionContext.Documents[0].RawArtifact), "", "  "); err != nil {
		t.Fatalf("reformat root rawArtifact: %v", err)
	}
	modified := make([]SnapshotArtifact, 0, len(artifacts))
	modified = append(modified, SnapshotArtifact{
		ArtifactRef:  fixture.ResolutionContext.Documents[0].ArtifactRef,
		SnapshotPath: fixture.ResolutionContext.Documents[0].SnapshotPath,
		Dialect:      Dialect(fixture.ResolutionContext.Documents[0].Dialect),
		Raw:          []byte(reformatted.String()),
		Parsed:       fixture.ResolutionContext.Documents[0].Document,
	})
	for _, document := range fixture.ResolutionContext.Documents[1:] {
		modified = append(modified, SnapshotArtifact{
			ArtifactRef:    document.ArtifactRef,
			SnapshotPath:   document.SnapshotPath,
			DocumentURI:    document.DocumentURI,
			Dialect:        Dialect(document.Dialect),
			Raw:            []byte(document.RawArtifact),
			Parsed:         document.Document,
			ArtifactDigest: document.ArtifactDigest,
		})
	}
	context := newContext(t, modified...)
	changed, err := context.ComputeSchemaClosure(RootRef{fixture.Root.ArtifactRef, fixture.Root.Pointer})
	if err != nil {
		t.Fatalf("modified closure: %v", err)
	}
	if changed.Digest == expected.Digest {
		t.Fatalf("closure digest did not change after raw artifact bytes changed")
	}
	if changed.Root.ArtifactDigest != digestOf([]byte(reformatted.String())) {
		t.Fatalf("root artifact digest is not recomputed from raw bytes")
	}
	if len(changed.Nodes) != len(expected.Nodes) {
		t.Fatalf("node set changed after formatting-only source change")
	}
	for index := range expected.Nodes {
		equal, err := jsonValuesEqual(changed.Nodes[index].Schema, expected.Nodes[index].Schema)
		if err != nil || !equal {
			t.Fatalf("parsed Schema value changed after formatting-only source change")
		}
	}
}

// jsonBuffer is no longer used.
func TestMissingRefTargetRejected(t *testing.T) {
	context := singleOpenAPIContext(t, `{"openapi":"3.0.0","components":{"schemas":{"R":{"$ref":"missing.yaml#/definitions/Missing"}}}}`)
	_, err := context.ComputeSchemaClosure(RootRef{testOpenAPIArtifactRef, "#/components/schemas/R"})
	wantCode(t, err, ErrRefResolution)
}

func TestRefOutsideSnapshotRejected(t *testing.T) {
	context := singleJSONSchemaContext(t, `{"$defs":{"R":{"$ref":"https://elsewhere.invalid/schema.json#/$defs/X"}}}`)
	_, err := context.ComputeSchemaClosure(RootRef{"urn:test:json-schema", "#/$defs/R"})
	wantCode(t, err, ErrRefResolution)
}

func TestRelativeRefEscapesSnapshotRejected(t *testing.T) {
	context := singleJSONSchemaContext(t, `{"$defs":{"R":{"$ref":"../../../outside.json#/x"}}}`)
	_, err := context.ComputeSchemaClosure(RootRef{"urn:test:json-schema", "#/$defs/R"})
	wantCode(t, err, ErrRefResolution)
}

func TestAmbiguousIDAliasRejected(t *testing.T) {
	_, err := NewResolutionContext(testSnapshotBase, []SnapshotArtifact{
		artifact(t, "urn:test:a", "schemas/a.json", DialectJSONSchema202012,
			`{"$id":"https://schemas.invalid/duplicate","type":"object"}`, ""),
		artifact(t, "urn:test:b", "schemas/b.json", DialectJSONSchema202012,
			`{"$id":"https://schemas.invalid/duplicate","type":"object"}`, ""),
	})
	wantCode(t, err, ErrAmbiguousAlias)
}

func TestAnchorRejected(t *testing.T) {
	context := singleJSONSchemaContext(t, `{"type":"object","properties":{"x":{"$anchor":"named","type":"string"}}}`)
	_, err := context.ComputeSchemaClosure(RootRef{"urn:test:json-schema", "#"})
	wantCode(t, err, ErrUnsupportedRef)
}

func TestDynamicRefRejected(t *testing.T) {
	context := singleJSONSchemaContext(t, `{"$dynamicRef":"#node","type":"object"}`)
	_, err := context.ComputeSchemaClosure(RootRef{"urn:test:json-schema", "#"})
	wantCode(t, err, ErrUnsupportedRef)
}

func TestRecursiveRefRejected(t *testing.T) {
	context := singleJSONSchemaContext(t, `{"$recursiveRef":"#","type":"object"}`)
	_, err := context.ComputeSchemaClosure(RootRef{"urn:test:json-schema", "#"})
	wantCode(t, err, ErrUnsupportedRef)
}

func TestBooleanSchemaRootRejected(t *testing.T) {
	_, err := NewResolutionContext(testSnapshotBase, []SnapshotArtifact{
		artifact(t, "urn:test:boolean", "schemas/boolean.json", DialectJSONSchema202012, `true`, ""),
	})
	wantCode(t, err, ErrBooleanSchema)
}

func TestBooleanSubschemaRejected(t *testing.T) {
	_, err := NewResolutionContext(testSnapshotBase, []SnapshotArtifact{
		artifact(t, "urn:test:boolean", "schemas/boolean.json", DialectJSONSchema202012,
			`{"type":"object","properties":{"x":true}}`, ""),
	})
	wantCode(t, err, ErrBooleanSchema)
}

func TestOpenAPITupleItemsRejected(t *testing.T) {
	_, err := NewResolutionContext(testSnapshotBase, []SnapshotArtifact{
		artifact(t, testOpenAPIArtifactRef, "openapi/service.yaml", DialectOpenAPI30,
			`{"openapi":"3.0.0","components":{"schemas":{"R":{"type":"array","items":[{"type":"string"}]}}}}`, ""),
	})
	wantCode(t, err, ErrUnsupportedValue)
}

func TestOpenAPIReachableIDRejected(t *testing.T) {
	context := singleOpenAPIContext(t, `{"openapi":"3.0.0","components":{"schemas":{"R":{"type":"object","$id":"https://schemas.invalid/r"}}}}`)
	_, err := context.ComputeSchemaClosure(RootRef{testOpenAPIArtifactRef, "#/components/schemas/R"})
	wantCode(t, err, ErrUnsupportedRef)
}

func TestDuplicateRawMappingKeysRejected(t *testing.T) {
	_, err := NewResolutionContext(testSnapshotBase, []SnapshotArtifact{
		artifact(t, testOpenAPIArtifactRef, "openapi/service.yaml", DialectOpenAPI30,
			`{"openapi":"3.0.0","openapi":"3.0.1","components":{}}`, ""),
	})
	wantCode(t, err, ErrArtifactParse)
}

func TestUnsafeIntegerRejected(t *testing.T) {
	_, err := NewResolutionContext(testSnapshotBase, []SnapshotArtifact{
		artifact(t, testOpenAPIArtifactRef, "openapi/service.yaml", DialectOpenAPI30,
			`{"openapi":"3.0.0","components":{"schemas":{"R":{"type":"integer","maximum":9007199254740992}}}}`, ""),
	})
	wantCode(t, err, ErrUnsupportedValue)
}

func TestNonFiniteNumberRejected(t *testing.T) {
	_, err := NewResolutionContext(testSnapshotBase, []SnapshotArtifact{
		artifact(t, testOpenAPIArtifactRef, "openapi/service.yaml", DialectOpenAPI30,
			`{"openapi":"3.0.0","components":{"schemas":{"R":{"type":"number","maximum":1e999}}}}`, ""),
	})
	wantCode(t, err, ErrUnsupportedValue)
}

func TestInvalidSnapshotPathRejected(t *testing.T) {
	for _, snapshotPath := range []string{"", "/absolute.yaml", "schemas/./x.yaml", "schemas/../x.yaml", "schemas//x.yaml", "schemas\\x.yaml"} {
		_, err := NewResolutionContext(testSnapshotBase, []SnapshotArtifact{
			artifact(t, "urn:test:a", snapshotPath, DialectJSONSchema202012, `{"type":"object"}`, ""),
		})
		if err == nil {
			t.Fatalf("snapshot path %q was accepted", snapshotPath)
		}
		code, ok := CodeOf(err)
		if !ok {
			// Wrapped artifact errors are acceptable as long as they fail closed.
			continue
		}
		if code != ErrSnapshotPath && code != ErrArtifactParse {
			t.Fatalf("snapshot path %q rejected with unexpected code %s", snapshotPath, code)
		}
	}
}

func TestInvalidSnapshotBaseURIRejected(t *testing.T) {
	for _, base := range []string{"relative/package/", "https://snapshot.invalid/package", "https://snapshot.invalid/package/?q=", "https://snapshot.invalid/package/#frag"} {
		_, err := NewResolutionContext(base, nil)
		if err == nil {
			t.Fatalf("snapshot base %q was accepted", base)
		}
	}
}

func TestDocumentURIMustEqualDerivedURI(t *testing.T) {
	_, err := NewResolutionContext(testSnapshotBase, []SnapshotArtifact{
		{
			ArtifactRef:  "urn:test:a",
			SnapshotPath: "schemas/a.json",
			DocumentURI:  "https://snapshot.invalid/package/other.json",
			Dialect:      DialectJSONSchema202012,
			Raw:          []byte(`{"type":"object"}`),
			Parsed:       decodeJSONValue(t, []byte(`{"type":"object"}`)),
		},
	})
	wantCode(t, err, ErrDocumentURIMismatch)
}

func TestArtifactBindingConflictsRejected(t *testing.T) {
	first := artifact(t, "urn:test:same", "schemas/a.json", DialectJSONSchema202012, `{"type":"object"}`, "")
	duplicateRef := artifact(t, "urn:test:same", "schemas/b.json", DialectJSONSchema202012, `{"type":"object"}`, "")
	_, err := NewResolutionContext(testSnapshotBase, []SnapshotArtifact{first, duplicateRef})
	wantCode(t, err, ErrArtifactBinding)

	duplicatePath := artifact(t, "urn:test:other", "schemas/a.json", DialectJSONSchema202012, `{"type":"object"}`, "")
	_, err = NewResolutionContext(testSnapshotBase, []SnapshotArtifact{first, duplicatePath})
	wantCode(t, err, ErrArtifactBinding)
}

func TestParsedDocumentMustEqualRawParse(t *testing.T) {
	_, err := NewResolutionContext(testSnapshotBase, []SnapshotArtifact{
		{
			ArtifactRef:  "urn:test:a",
			SnapshotPath: "schemas/a.json",
			Dialect:      DialectJSONSchema202012,
			Raw:          []byte(`{"type":"object"}`),
			Parsed:       map[string]any{"type": "string"},
		},
	})
	wantCode(t, err, ErrArtifactDocument)
}

func TestRootPointerMustBeSchemaLocation(t *testing.T) {
	context := singleOpenAPIContext(t, `{"openapi":"3.0.0","info":{"title":"x"},"components":{"schemas":{"R":{"type":"object"}}}}`)
	if _, err := context.ComputeSchemaClosure(RootRef{testOpenAPIArtifactRef, "#/info"}); err == nil {
		t.Fatalf("non-Schema root pointer was accepted")
	}
	if _, err := context.ComputeSchemaClosure(RootRef{testOpenAPIArtifactRef, "#/components/schemas/Missing"}); err == nil {
		t.Fatalf("missing root pointer was accepted")
	}
}
