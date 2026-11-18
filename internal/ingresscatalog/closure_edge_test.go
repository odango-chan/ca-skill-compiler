package ingresscatalog

import (
	"strings"
	"testing"
)

func TestOpenAPIAdditionalPropertiesBooleanContributesNoChild(t *testing.T) {
	for _, value := range []string{"true", "false"} {
		context := singleOpenAPIContext(t, `{"openapi":"3.0.0","components":{"schemas":{"R":{"type":"object","properties":{"x":{"type":"string"}},"additionalProperties":`+value+`}}}}`)
		result, err := context.ComputeSchemaClosure(RootRef{testOpenAPIArtifactRef, "#/components/schemas/R"})
		if err != nil {
			t.Fatalf("additionalProperties %s: %v", value, err)
		}
		if len(result.Nodes) != 1 || result.Nodes[0].Pointer != "#/components/schemas/R" {
			t.Fatalf("boolean additionalProperties must contribute no child: %#v", result.Nodes)
		}
	}
}

func TestCyclicReferencesVisitedOncePerIdentity(t *testing.T) {
	context := singleJSONSchemaContext(t,
		`{"$id":"https://schemas.invalid/cycle","$ref":"#/$defs/A","$defs":{"A":{"$ref":"#/$defs/B"},"B":{"$ref":"#/$defs/A"}}}`)
	result, err := context.ComputeSchemaClosure(RootRef{"urn:test:json-schema", "#"})
	if err != nil {
		t.Fatalf("cyclic closure: %v", err)
	}
	pointers := make([]string, 0, len(result.Nodes))
	for _, node := range result.Nodes {
		pointers = append(pointers, node.Pointer)
	}
	joined := strings.Join(pointers, ",")
	for _, want := range []string{"#", "#/$defs/A", "#/$defs/B"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("cyclic closure missing %s: %s", want, joined)
		}
	}
	if len(result.Nodes) != 3 {
		t.Fatalf("cyclic closure node count = %d, want 3: %s", len(result.Nodes), joined)
	}
}

func TestFragmentPercentEncodingAndRFC6901Escaping(t *testing.T) {
	context := singleJSONSchemaContext(t,
		`{"$defs":{"Leaf":{"type":"string"},"a/b":{"type":"number"}},"properties":{"one":{"$ref":"#/%24defs/Leaf"},"two":{"$ref":"#/$defs/a~1b"}}}`)
	result, err := context.ComputeSchemaClosure(RootRef{"urn:test:json-schema", "#"})
	if err != nil {
		t.Fatalf("fragment closure: %v", err)
	}
	pointers := make([]string, 0, len(result.Nodes))
	for _, node := range result.Nodes {
		pointers = append(pointers, node.Pointer)
	}
	joined := strings.Join(pointers, ",")
	if !strings.Contains(joined, "#/$defs/Leaf") {
		t.Fatalf("percent-encoded fragment did not canonicalize: %s", joined)
	}
	if !strings.Contains(joined, "#/$defs/a~1b") {
		t.Fatalf("RFC 6901 escaped token was not preserved: %s", joined)
	}
}

func TestPercentEncodedUnreservedFragmentIsEquivalent(t *testing.T) {
	first := singleJSONSchemaContext(t,
		`{"$defs":{"Leaf":{"type":"string"}},"$ref":"#/%24defs/Leaf"}`)
	second := singleJSONSchemaContext(t,
		`{"$defs":{"Leaf":{"type":"string"}},"$ref":"#/$defs/Leaf"}`)
	left, err := first.ComputeSchemaClosure(RootRef{"urn:test:json-schema", "#"})
	if err != nil {
		t.Fatalf("left closure: %v", err)
	}
	right, err := second.ComputeSchemaClosure(RootRef{"urn:test:json-schema", "#"})
	if err != nil {
		t.Fatalf("right closure: %v", err)
	}
	// The artifact byte digests differ because the source $ref spelling differs,
	// but the resolved node identities must be identical.
	if len(left.Nodes) != len(right.Nodes) {
		t.Fatalf("node counts differ: %d vs %d", len(left.Nodes), len(right.Nodes))
	}
	for index := range left.Nodes {
		if left.Nodes[index].Pointer != right.Nodes[index].Pointer {
			t.Fatalf("equivalent fragment spellings resolved to different pointers")
		}
	}
}

func TestDocumentURICanonicalization(t *testing.T) {
	context, err := NewResolutionContext("HTTPS://SNAPSHOT.INVALID:443/package/", []SnapshotArtifact{
		{
			ArtifactRef:  "urn:test:a",
			SnapshotPath: "schemas/~a.json",
			DocumentURI:  "https://SNAPSHOT.INVALID:443/package/schemas/%7ea.json",
			Dialect:      DialectJSONSchema202012,
			Raw:          []byte(`{"type":"object"}`),
			Parsed:       decodeJSONValue(t, []byte(`{"type":"object"}`)),
		},
	})
	if err != nil {
		t.Fatalf("canonical URI context: %v", err)
	}
	artifact, ok := context.Artifact("urn:test:a")
	if !ok {
		t.Fatalf("artifact missing")
	}
	if artifact.DocumentURI != "https://snapshot.invalid/package/schemas/~a.json" {
		t.Fatalf("document URI = %q", artifact.DocumentURI)
	}
}

func TestInvalidSnapshotBaseVariantsRejected(t *testing.T) {
	cases := []string{
		"https://user@snapshot.invalid/package/",
		"https://snapshot.invalid/package//",
		"https://snapshot.invalid/package/?query=1",
		"https://snapshot.invalid/package/#fragment",
		"https://snapshot.invalid/package/%zz",
		"https://snapshot.invalid/package/ ",
		"https://snapshot.invalid\\package\\",
	}
	for _, base := range cases {
		if _, err := NewResolutionContext(base, nil); err == nil {
			t.Fatalf("snapshot base %q was accepted", base)
		}
	}
}

func TestInvalidPointerEscapingRejected(t *testing.T) {
	context := singleJSONSchemaContext(t, `{"$defs":{"Leaf":{"type":"string"}}}`)
	for _, pointer := range []string{"#/$defs/Leaf~2", "#/$defs/Leaf~", "#not-a-pointer", "#/%zz"} {
		if _, err := context.ComputeSchemaClosure(RootRef{"urn:test:json-schema", pointer}); err == nil {
			t.Fatalf("root pointer %q was accepted", pointer)
		}
	}
}

func TestCrossArtifactRelativeReferenceResolution(t *testing.T) {
	context := newContext(t,
		artifact(t, "urn:test:root", "schemas/root.json", DialectJSONSchema202012,
			`{"$id":"https://schemas.invalid/root.json","properties":{"next":{"$ref":"child.json#/$defs/Child"}}}`, ""),
		artifact(t, "urn:test:child", "schemas/child.json", DialectJSONSchema202012,
			`{"$id":"https://schemas.invalid/child.json","$defs":{"Child":{"type":"string"}}}`, ""),
	)
	result, err := context.ComputeSchemaClosure(RootRef{"urn:test:root", "#"})
	if err != nil {
		t.Fatalf("cross-artifact closure: %v", err)
	}
	if len(result.Nodes) != 2 {
		t.Fatalf("cross-artifact node count = %d", len(result.Nodes))
	}
	found := false
	for _, node := range result.Nodes {
		if node.ArtifactRef == "urn:test:child" && node.Pointer == "#/$defs/Child" {
			found = true
		}
	}
	if !found {
		t.Fatalf("cross-artifact node missing: %#v", result.Nodes)
	}
}

func TestManifestNodesSortedByArtifactRefThenPointer(t *testing.T) {
	fixture := loadVector(t)
	context := newContext(t, vectorArtifacts(t, fixture)...)
	result, err := context.ComputeSchemaClosure(RootRef{fixture.Root.ArtifactRef, fixture.Root.Pointer})
	if err != nil {
		t.Fatalf("closure: %v", err)
	}
	for index := 1; index < len(result.Nodes); index++ {
		previous, current := result.Nodes[index-1], result.Nodes[index]
		if previous.ArtifactRef > current.ArtifactRef {
			t.Fatalf("nodes not sorted by artifactRef")
		}
		if previous.ArtifactRef == current.ArtifactRef && previous.Pointer > current.Pointer {
			t.Fatalf("nodes not sorted by pointer")
		}
	}
}
