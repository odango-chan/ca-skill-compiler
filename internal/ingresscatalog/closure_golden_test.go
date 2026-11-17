package ingresscatalog

import (
	"os"
	"testing"
)

// TestSchemaClosureDigestGoldenVector is the normative #62 §6 conformance
// check: the digest is recomputed from rawArtifact bytes and the exact node
// set, pointers, artifact digests and JCS bytes are reproduced.
func TestSchemaClosureDigestGoldenVector(t *testing.T) {
	fixture := loadVector(t)
	context := newContext(t, vectorArtifacts(t, fixture)...)

	root := RootRef{
		ArtifactRef: fixture.Root.ArtifactRef,
		Pointer:     fixture.Root.Pointer,
	}
	result, err := context.ComputeSchemaClosure(root)
	if err != nil {
		t.Fatalf("ComputeSchemaClosure: %v", err)
	}
	if result.Digest != fixture.ExpectedDigest {
		t.Fatalf("digest mismatch:\n got %s\nwant %s\nmanifest JCS: %s",
			result.Digest, fixture.ExpectedDigest, result.JCS)
	}
	if result.Root.ArtifactRef != fixture.Root.ArtifactRef ||
		result.Root.ArtifactDigest != fixture.Root.ArtifactDigest ||
		result.Root.Pointer != fixture.Root.Pointer {
		t.Fatalf("root entry mismatch: %#v", result.Root)
	}
	expected := sortedExpectedNodes(fixture)
	if len(result.Nodes) != len(expected) {
		t.Fatalf("node count = %d, want %d: %#v", len(result.Nodes), len(expected), result.Nodes)
	}
	for index := range expected {
		actual := result.Nodes[index]
		if actual.ArtifactRef != expected[index].ArtifactRef ||
			actual.ArtifactDigest != expected[index].ArtifactDigest ||
			actual.Pointer != expected[index].Pointer {
			t.Fatalf("node[%d] identity mismatch: got %#v want %#v", index, actual, expected[index])
		}
		equal, err := jsonValuesEqual(actual.Schema, expected[index].Schema)
		if err != nil {
			t.Fatalf("compare node[%d] schema: %v", index, err)
		}
		if !equal {
			t.Fatalf("node[%d] schema mismatch: got %#v want %#v", index, actual.Schema, expected[index].Schema)
		}
	}

	goldenPath := "testdata/schema-closure-digest-v1.manifest.jcs"
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, result.JCS, 0o644); err != nil {
			t.Fatalf("write golden JCS: %v", err)
		}
	}
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden JCS manifest: %v", err)
	}
	if string(golden) != string(result.JCS) {
		t.Fatalf("JCS manifest bytes changed:\n got %s\nwant %s", result.JCS, golden)
	}
}

// TestSchemaClosureDigestInputOrderIndependent proves the digest does not
// depend on resolution-context document order.
func TestSchemaClosureDigestInputOrderIndependent(t *testing.T) {
	fixture := loadVector(t)
	artifacts := vectorArtifacts(t, fixture)
	reversed := make([]SnapshotArtifact, 0, len(artifacts))
	for index := len(artifacts) - 1; index >= 0; index-- {
		reversed = append(reversed, artifacts[index])
	}
	forward, err := newContext(t, artifacts...).ComputeSchemaClosure(RootRef{fixture.Root.ArtifactRef, fixture.Root.Pointer})
	if err != nil {
		t.Fatalf("forward closure: %v", err)
	}
	backward, err := newContext(t, reversed...).ComputeSchemaClosure(RootRef{fixture.Root.ArtifactRef, fixture.Root.Pointer})
	if err != nil {
		t.Fatalf("reverse closure: %v", err)
	}
	if forward.Digest != backward.Digest {
		t.Fatalf("digest depends on document order: %s vs %s", forward.Digest, backward.Digest)
	}
	if string(forward.JCS) != string(backward.JCS) {
		t.Fatalf("JCS bytes depend on document order")
	}
}
