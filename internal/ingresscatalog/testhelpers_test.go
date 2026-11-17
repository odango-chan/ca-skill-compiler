package ingresscatalog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"sort"
	"testing"
)

const testSnapshotBase = "https://snapshot.invalid/package/"

func digestOf(raw []byte) string {
	sum := sha256.Sum256(raw)
	return DigestPrefix + hex.EncodeToString(sum[:])
}

func mustMarshalJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return data
}

func decodeJSONValue(t *testing.T, data []byte) any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("decode JSON fixture: %v", err)
	}
	return value
}

// artifact builds one trusted snapshot artifact. When suppliedDigest is empty
// the resolver computes the digest itself.
func artifact(t *testing.T, ref, snapshotPath string, dialect Dialect, raw string, suppliedDigest string) SnapshotArtifact {
	t.Helper()
	rawBytes := []byte(raw)
	return SnapshotArtifact{
		ArtifactRef:    ref,
		SnapshotPath:   snapshotPath,
		Dialect:        dialect,
		Raw:            rawBytes,
		Parsed:         decodeJSONValue(t, rawBytes),
		ArtifactDigest: suppliedDigest,
	}
}

func newContext(t *testing.T, artifacts ...SnapshotArtifact) *ResolutionContext {
	t.Helper()
	context, err := NewResolutionContext(testSnapshotBase, artifacts)
	if err != nil {
		t.Fatalf("NewResolutionContext: %v", err)
	}
	return context
}

func wantCode(t *testing.T, err error, code ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %s, got nil", code)
	}
	if actual, ok := CodeOf(err); !ok || actual != code {
		t.Fatalf("expected error code %s, got %v (%v)", code, actual, err)
	}
}

type vectorFixture struct {
	Root struct {
		ArtifactRef    string `json:"artifactRef"`
		ArtifactDigest string `json:"artifactDigest"`
		Pointer        string `json:"pointer"`
	} `json:"root"`
	Nodes []struct {
		ArtifactRef    string `json:"artifactRef"`
		ArtifactDigest string `json:"artifactDigest"`
		Pointer        string `json:"pointer"`
		Schema         any    `json:"schema"`
	} `json:"nodes"`
	ExpectedDigest    string `json:"expectedDigest"`
	ResolutionContext struct {
		SnapshotBaseURI string `json:"snapshotBaseUri"`
		Documents       []struct {
			ArtifactRef    string `json:"artifactRef"`
			SnapshotPath   string `json:"snapshotPath"`
			DocumentURI    string `json:"documentUri"`
			ArtifactDigest string `json:"artifactDigest"`
			Dialect        string `json:"dialect"`
			Document       any    `json:"document"`
			RawArtifact    string `json:"rawArtifact"`
		} `json:"documents"`
	} `json:"resolutionContext"`
}

func loadVector(t *testing.T) vectorFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/schema-closure-digest-v1.vector.json")
	if err != nil {
		t.Fatalf("read vector: %v", err)
	}
	var fixture vectorFixture
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatalf("decode vector: %v", err)
	}
	if fixture.ResolutionContext.SnapshotBaseURI != testSnapshotBase {
		t.Fatalf("unexpected vector snapshot base: %q", fixture.ResolutionContext.SnapshotBaseURI)
	}
	return fixture
}

func vectorArtifacts(t *testing.T, fixture vectorFixture) []SnapshotArtifact {
	t.Helper()
	artifacts := make([]SnapshotArtifact, 0, len(fixture.ResolutionContext.Documents))
	for _, document := range fixture.ResolutionContext.Documents {
		artifacts = append(artifacts, SnapshotArtifact{
			ArtifactRef:    document.ArtifactRef,
			SnapshotPath:   document.SnapshotPath,
			DocumentURI:    document.DocumentURI,
			Dialect:        Dialect(document.Dialect),
			Raw:            []byte(document.RawArtifact),
			Parsed:         document.Document,
			ArtifactDigest: document.ArtifactDigest,
		})
	}
	return artifacts
}

func sortedExpectedNodes(fixture vectorFixture) []ManifestNode {
	nodes := make([]ManifestNode, 0, len(fixture.Nodes))
	for _, node := range fixture.Nodes {
		nodes = append(nodes, ManifestNode{
			ArtifactRef:    node.ArtifactRef,
			ArtifactDigest: node.ArtifactDigest,
			Pointer:        node.Pointer,
			Schema:         node.Schema,
		})
	}
	sort.Slice(nodes, func(left, right int) bool {
		if nodes[left].ArtifactRef != nodes[right].ArtifactRef {
			return nodes[left].ArtifactRef < nodes[right].ArtifactRef
		}
		return nodes[left].Pointer < nodes[right].Pointer
	})
	return nodes
}
