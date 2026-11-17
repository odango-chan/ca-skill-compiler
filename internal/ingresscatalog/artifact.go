package ingresscatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// Dialect is the source Schema dialect of one trusted artifact.
type Dialect string

const (
	DialectOpenAPI30        Dialect = "openapi-3.0"
	DialectJSONSchema202012 Dialect = "json-schema-2020-12"
)

func (d Dialect) valid() bool {
	return d == DialectOpenAPI30 || d == DialectJSONSchema202012
}

// SnapshotArtifact is one immutable artifact in the caller-declared trusted
// dependency snapshot. Raw is authoritative: the resolver computes the artifact
// digest from these exact bytes and verifies that Parsed equals the parse of Raw.
// ArtifactDigest is optional; when supplied it must agree with the computed
// digest (Schema Closure Digest v1 rejects stale declared digests).
type SnapshotArtifact struct {
	ArtifactRef    string
	SnapshotPath   string
	DocumentURI    string // optional; when supplied it must equal the derived canonical URI
	Dialect        Dialect
	Raw            []byte
	Parsed         any
	ArtifactDigest string
}

// ArtifactView is a read-only projection of a resolved artifact.
type ArtifactView struct {
	ArtifactRef    string
	SnapshotPath   string
	DocumentURI    string
	ArtifactDigest string
	Dialect        Dialect
	Document       any
}

type schemaLocationInfo struct {
	baseURI      string
	resourceRoot jsonPointer
}

type resourceBinding struct {
	artifactRef string
	pointer     jsonPointer
}

type resolvedArtifact struct {
	ref          string
	snapshotPath string
	documentURI  string
	digest       string
	dialect      Dialect
	document     any
	// schemaLocations maps canonical artifact-relative pointers that identify a
	// Schema Object to the effective base URI and resource root.
	schemaLocations map[string]schemaLocationInfo
	// openAPISchemas marks Schema Object locations for OpenAPI 3.0 artifacts.
	openAPISchemas map[string]struct{}
}

// ResolutionContext is the immutable trusted resolution context for one
// snapshot. It binds ArtifactRef, SnapshotPath and canonical DocumentURI 1:1,
// computes artifact digests from raw bytes and owns all dialect-aware Schema
// location and JSON Schema resource-URI alias indexes.
type ResolutionContext struct {
	snapshotBaseURI string
	base            *uriReference
	artifacts       map[string]*resolvedArtifact
	byPath          map[string]*resolvedArtifact
	byURI           map[string]*resolvedArtifact
	resources       map[string][]resourceBinding
}

// NewResolutionContext validates and indexes one trusted snapshot.
func NewResolutionContext(snapshotBaseURI string, artifacts []SnapshotArtifact) (*ResolutionContext, error) {
	canonicalBase, err := canonicalizeDocumentURI(snapshotBaseURI)
	if err != nil {
		return nil, &Error{Code: ErrSnapshotBaseURI, Detail: err.Error()}
	}
	if !strings.HasSuffix(canonicalBase, "/") {
		return nil, &Error{Code: ErrSnapshotBaseURI, Detail: "snapshot base URI must end with '/'"}
	}
	baseReference, err := parseURIReference(canonicalBase)
	if err != nil {
		return nil, &Error{Code: ErrSnapshotBaseURI, Detail: err.Error()}
	}
	context := &ResolutionContext{
		snapshotBaseURI: canonicalBase,
		base:            baseReference,
		artifacts:       make(map[string]*resolvedArtifact, len(artifacts)),
		byPath:          make(map[string]*resolvedArtifact, len(artifacts)),
		byURI:           make(map[string]*resolvedArtifact, len(artifacts)),
		resources:       make(map[string][]resourceBinding, len(artifacts)*2),
	}
	for index, artifact := range artifacts {
		resolved, err := context.resolveArtifact(artifact)
		if err != nil {
			return nil, fmt.Errorf("artifact %d: %w", index, err)
		}
		if _, exists := context.artifacts[resolved.ref]; exists {
			return nil, newArtifactError(ErrArtifactBinding, resolved.ref, "", "duplicate artifactRef")
		}
		if other, exists := context.byPath[resolved.snapshotPath]; exists {
			return nil, newArtifactError(ErrArtifactBinding, resolved.ref, "", "snapshotPath already bound to "+other.ref)
		}
		if other, exists := context.byURI[resolved.documentURI]; exists {
			return nil, newArtifactError(ErrArtifactBinding, resolved.ref, "", "documentUri already bound to "+other.ref)
		}
		context.artifacts[resolved.ref] = resolved
		context.byPath[resolved.snapshotPath] = resolved
		context.byURI[resolved.documentURI] = resolved
		context.resources[resolved.documentURI] = append(context.resources[resolved.documentURI], resourceBinding{artifactRef: resolved.ref})
	}
	// Index Schema locations and $id aliases only after the 1:1 binding pass so
	// that alias conflicts are reported as alias conflicts.
	refs := make([]string, 0, len(context.artifacts))
	for ref := range context.artifacts {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	for _, ref := range refs {
		artifact := context.artifacts[ref]
		switch artifact.dialect {
		case DialectJSONSchema202012:
			if err := context.indexJSONSchemaArtifact(artifact); err != nil {
				return nil, err
			}
		case DialectOpenAPI30:
			if err := context.indexOpenAPIArtifact(artifact); err != nil {
				return nil, err
			}
		}
	}
	for key, bindings := range context.resources {
		distinct := dedupeBindings(bindings)
		if len(distinct) > 1 {
			return nil, newError(ErrAmbiguousAlias, "resource URI '"+key+"' is bound to more than one artifact/resource identity")
		}
		context.resources[key] = distinct
	}
	return context, nil
}

func dedupeBindings(bindings []resourceBinding) []resourceBinding {
	result := make([]resourceBinding, 0, len(bindings))
	seen := make(map[string]bool, len(bindings))
	for _, binding := range bindings {
		key := binding.artifactRef + "\x00" + canonicalPointer(binding.pointer)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, binding)
	}
	return result
}

func (c *ResolutionContext) resolveArtifact(artifact SnapshotArtifact) (*resolvedArtifact, error) {
	if artifact.ArtifactRef == "" {
		return nil, newError(ErrArtifactBinding, "artifactRef must not be empty")
	}
	if !artifact.Dialect.valid() {
		return nil, newArtifactError(ErrUnsupportedDialect, artifact.ArtifactRef, "", fmt.Sprintf("unsupported dialect %q", artifact.Dialect))
	}
	if len(artifact.Raw) == 0 {
		return nil, newArtifactError(ErrArtifactParse, artifact.ArtifactRef, "", "raw artifact bytes are empty")
	}
	encodedPath, err := encodeSnapshotPath(artifact.SnapshotPath)
	if err != nil {
		return nil, &Error{Code: ErrSnapshotPath, ArtifactRef: artifact.ArtifactRef, Detail: err.Error()}
	}
	pathReference, err := parseURIReference(encodedPath)
	if err != nil {
		return nil, &Error{Code: ErrSnapshotPath, ArtifactRef: artifact.ArtifactRef, Detail: err.Error()}
	}
	resolvedReference, err := resolveReferenceURI(c.base, pathReference)
	if err != nil {
		return nil, &Error{Code: ErrDocumentURI, ArtifactRef: artifact.ArtifactRef, Detail: err.Error()}
	}
	derivedURI, err := resolvedReference.canonicalString(true, false)
	if err != nil {
		return nil, &Error{Code: ErrDocumentURI, ArtifactRef: artifact.ArtifactRef, Detail: err.Error()}
	}
	if artifact.DocumentURI != "" {
		supplied, err := canonicalizeDocumentURI(artifact.DocumentURI)
		if err != nil {
			return nil, &Error{Code: ErrDocumentURI, ArtifactRef: artifact.ArtifactRef, Detail: err.Error()}
		}
		if supplied != derivedURI {
			return nil, &Error{Code: ErrDocumentURIMismatch, ArtifactRef: artifact.ArtifactRef,
				Detail: "supplied documentUri " + supplied + " does not equal derived " + derivedURI}
		}
	}

	document, err := parseStrictJSONValue(artifact.Raw)
	if err != nil {
		return nil, &Error{Code: errorCodeOr(err, ErrArtifactParse), ArtifactRef: artifact.ArtifactRef, Detail: err.Error()}
	}
	supplied, err := normalizeJSONCompatibleValue(artifact.Parsed)
	if err != nil {
		return nil, &Error{Code: ErrArtifactDocument, ArtifactRef: artifact.ArtifactRef, Detail: err.Error()}
	}
	equal, err := jsonValuesEqual(document, supplied)
	if err != nil {
		return nil, &Error{Code: ErrArtifactDocument, ArtifactRef: artifact.ArtifactRef, Detail: err.Error()}
	}
	if !equal {
		return nil, newArtifactError(ErrArtifactDocument, artifact.ArtifactRef, "",
			"supplied parsed document does not equal the parse of rawArtifact bytes")
	}

	sum := sha256.Sum256(artifact.Raw)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	if artifact.ArtifactDigest != "" && artifact.ArtifactDigest != digest {
		return nil, newArtifactError(ErrArtifactDigest, artifact.ArtifactRef, "",
			"declared artifactDigest does not match SHA-256 of rawArtifact bytes")
	}
	return &resolvedArtifact{
		ref:             artifact.ArtifactRef,
		snapshotPath:    artifact.SnapshotPath,
		documentURI:     derivedURI,
		digest:          digest,
		dialect:         artifact.Dialect,
		document:        document,
		schemaLocations: make(map[string]schemaLocationInfo),
		openAPISchemas:  make(map[string]struct{}),
	}, nil
}

// SnapshotBaseURI returns the canonical snapshot base URI.
func (c *ResolutionContext) SnapshotBaseURI() string { return c.snapshotBaseURI }

// Artifact returns a read-only projection of the artifact with the exact ref.
func (c *ResolutionContext) Artifact(ref string) (ArtifactView, bool) {
	artifact, ok := c.artifacts[ref]
	if !ok {
		return ArtifactView{}, false
	}
	return ArtifactView{
		ArtifactRef:    artifact.ref,
		SnapshotPath:   artifact.snapshotPath,
		DocumentURI:    artifact.documentURI,
		ArtifactDigest: artifact.digest,
		Dialect:        artifact.dialect,
		Document:       artifact.document,
	}, true
}

// ArtifactRefs returns artifact refs in deterministic UTF-8 byte order.
func (c *ResolutionContext) ArtifactRefs() []string {
	refs := make([]string, 0, len(c.artifacts))
	for ref := range c.artifacts {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return refs
}

func (c *ResolutionContext) resolveResource(key string) (resourceBinding, error) {
	bindings := c.resources[key]
	switch len(bindings) {
	case 0:
		return resourceBinding{}, newError(ErrRefResolution, "URI '"+key+"' is not mapped by any artifact in the trusted snapshot")
	case 1:
		return bindings[0], nil
	default:
		return resourceBinding{}, newError(ErrAmbiguousAlias, "URI '"+key+"' is bound to multiple resource identities")
	}
}

func (c *ResolutionContext) registerResource(key string, binding resourceBinding) {
	c.resources[key] = append(c.resources[key], binding)
}

func (a *resolvedArtifact) recordSchemaLocation(pointer jsonPointer, info schemaLocationInfo) {
	key := canonicalPointer(pointer)
	if _, exists := a.schemaLocations[key]; exists {
		return
	}
	a.schemaLocations[key] = info
}
