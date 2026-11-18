package ingresscatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/gowebpki/jcs"
)

const (
	// AlgorithmIdentifier is the Schema Closure Digest v1 algorithm identity.
	AlgorithmIdentifier = "connectionagent.io/schema-closure/v1"
	// DigestPrefix prefixes every canonical digest wire value.
	DigestPrefix = "sha256:"
)

// DomainSeparator is the ASCII binary domain separator from Schema Closure
// Digest v1: "CA-SKILL-SCHEMA-CLOSURE\0v1\0".
var DomainSeparator = []byte("CA-SKILL-SCHEMA-CLOSURE\x00v1\x00")

// RootRef identifies one exact Schema root in the trusted snapshot. Pointer is
// a canonical JSON Pointer fragment: "#" or "#/...".
type RootRef struct {
	ArtifactRef string `json:"artifactRef"`
	Pointer     string `json:"pointer"`
}

// ResolvedSchemaRoot is a validated root identity.
type ResolvedSchemaRoot struct {
	ArtifactRef    string
	ArtifactDigest string
	Pointer        string
	Schema         any
}

type schemaIdentity struct {
	artifactRef string
	pointer     jsonPointer
}

func (identity schemaIdentity) key() string {
	return identity.artifactRef + "\x00" + canonicalPointer(identity.pointer)
}

func (identity schemaIdentity) wirePointer() string {
	return canonicalPointer(identity.pointer)
}

// ManifestRoot is the root entry of the canonical Schema Closure manifest.
type ManifestRoot struct {
	ArtifactRef    string `json:"artifactRef"`
	ArtifactDigest string `json:"artifactDigest"`
	Pointer        string `json:"pointer"`
}

// ManifestNode is one closure node of the canonical Schema Closure manifest.
type ManifestNode struct {
	ArtifactRef    string `json:"artifactRef"`
	ArtifactDigest string `json:"artifactDigest"`
	Pointer        string `json:"pointer"`
	Schema         any    `json:"schema"`
}

// Manifest is the canonical Schema Closure Digest v1 manifest object.
type Manifest struct {
	Algorithm string         `json:"algorithm"`
	Root      ManifestRoot   `json:"root"`
	Nodes     []ManifestNode `json:"nodes"`
}

// ClosureResult is the complete, independently checkable result of one Schema
// Closure Digest v1 computation.
type ClosureResult struct {
	Root     ManifestRoot
	Nodes    []ManifestNode
	Manifest Manifest
	JCS      []byte
	Digest   string
}

// ResolveRoot validates and resolves one exact Schema root identity. No name
// search, fallback path, floating lookup or network access is performed.
func (c *ResolutionContext) ResolveRoot(ref RootRef) (ResolvedSchemaRoot, error) {
	identity, schema, artifact, err := c.resolveRootIdentity(ref)
	if err != nil {
		return ResolvedSchemaRoot{}, err
	}
	return ResolvedSchemaRoot{
		ArtifactRef:    identity.artifactRef,
		ArtifactDigest: artifact.digest,
		Pointer:        identity.wirePointer(),
		Schema:         schema,
	}, nil
}

func (c *ResolutionContext) resolveRootIdentity(ref RootRef) (schemaIdentity, any, *resolvedArtifact, error) {
	artifact, ok := c.artifacts[ref.ArtifactRef]
	if !ok {
		return schemaIdentity{}, nil, nil, newError(ErrRootResolution, "artifactRef '"+ref.ArtifactRef+"' is not in the trusted snapshot")
	}
	pointer, err := parseCanonicalPointer(ref.Pointer)
	if err != nil {
		return schemaIdentity{}, nil, nil, &Error{Code: ErrRootResolution, ArtifactRef: ref.ArtifactRef, Detail: err.Error()}
	}
	key := canonicalPointer(pointer)
	if _, indexed := artifact.schemaLocations[key]; !indexed {
		return schemaIdentity{}, nil, nil, newArtifactError(ErrNonSchema, artifact.ref, key, "root pointer does not identify a Schema Object")
	}
	value, err := resolvePointer(artifact.document, pointer)
	if err != nil {
		return schemaIdentity{}, nil, nil, &Error{Code: ErrMissingPointer, ArtifactRef: artifact.ref, Pointer: key, Detail: err.Error()}
	}
	if _, isBoolean := value.(bool); isBoolean {
		return schemaIdentity{}, nil, nil, newArtifactError(ErrBooleanSchema, artifact.ref, key, "Boolean Schemas are not supported")
	}
	return schemaIdentity{artifactRef: artifact.ref, pointer: pointer}, value, artifact, nil
}

// ComputeSchemaClosure computes the exact transitive Schema-reference closure
// of one root and returns the JCS manifest bytes and domain-separated digest.
func (c *ResolutionContext) ComputeSchemaClosure(rootRef RootRef) (*ClosureResult, error) {
	rootIdentity, _, _, err := c.resolveRootIdentity(rootRef)
	if err != nil {
		return nil, err
	}
	nodes := map[string]schemaIdentity{rootIdentity.key(): rootIdentity}
	queue := []schemaIdentity{rootIdentity}
	explored := make(map[string]bool, 8)
	for len(queue) > 0 {
		identity := queue[0]
		queue = queue[1:]
		key := identity.key()
		if explored[key] {
			continue
		}
		explored[key] = true
		artifact := c.artifacts[identity.artifactRef]
		value, err := resolvePointer(artifact.document, identity.pointer)
		if err != nil {
			return nil, &Error{Code: ErrMissingPointer, ArtifactRef: artifact.ref,
				Pointer: canonicalPointer(identity.pointer), Detail: err.Error()}
		}
		if err := c.walkSchemaValue(artifact, identity.pointer, value, func(target schemaIdentity) {
			targetKey := target.key()
			if _, exists := nodes[targetKey]; !exists {
				nodes[targetKey] = target
				queue = append(queue, target)
			}
		}); err != nil {
			return nil, err
		}
	}

	identities := make([]schemaIdentity, 0, len(nodes))
	for key := range nodes {
		identities = append(identities, nodes[key])
	}
	sort.Slice(identities, func(left, right int) bool {
		if identities[left].artifactRef != identities[right].artifactRef {
			return identities[left].artifactRef < identities[right].artifactRef
		}
		return identities[left].wirePointer() < identities[right].wirePointer()
	})

	manifestNodes := make([]ManifestNode, 0, len(identities))
	for _, identity := range identities {
		artifact := c.artifacts[identity.artifactRef]
		value, err := resolvePointer(artifact.document, identity.pointer)
		if err != nil {
			return nil, &Error{Code: ErrMissingPointer, ArtifactRef: artifact.ref,
				Pointer: canonicalPointer(identity.pointer), Detail: err.Error()}
		}
		manifestNodes = append(manifestNodes, ManifestNode{
			ArtifactRef:    artifact.ref,
			ArtifactDigest: artifact.digest,
			Pointer:        identity.wirePointer(),
			Schema:         value,
		})
	}
	rootNode := ManifestRoot{
		ArtifactRef:    rootIdentity.artifactRef,
		ArtifactDigest: c.artifacts[rootIdentity.artifactRef].digest,
		Pointer:        rootIdentity.wirePointer(),
	}
	manifest := Manifest{
		Algorithm: AlgorithmIdentifier,
		Root:      rootNode,
		Nodes:     manifestNodes,
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, newError(ErrClosureDigest, "manifest is not JSON-serializable: "+err.Error())
	}
	canonical, err := jcs.Transform(manifestBytes)
	if err != nil {
		return nil, newError(ErrClosureDigest, "JCS canonicalization failed: "+err.Error())
	}
	hash := sha256.New()
	hash.Write(DomainSeparator)
	hash.Write(canonical)
	digest := DigestPrefix + hex.EncodeToString(hash.Sum(nil))
	return &ClosureResult{
		Root:     rootNode,
		Nodes:    manifestNodes,
		Manifest: manifest,
		JCS:      canonical,
		Digest:   digest,
	}, nil
}

// walkSchemaValue inspects one Schema Object, reports every $ref target, and
// recursively walks Schema-valued children. Non-Schema values, Boolean Schemas,
// unsupported anchors and OpenAPI-reachable $id are rejected.
func (c *ResolutionContext) walkSchemaValue(artifact *resolvedArtifact, pointer jsonPointer, value any, onTarget func(schemaIdentity)) error {
	object, ok := value.(map[string]any)
	if !ok {
		if _, isBoolean := value.(bool); isBoolean {
			return newArtifactError(ErrBooleanSchema, artifact.ref, canonicalPointer(pointer), "Boolean Schemas are not supported")
		}
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), "Schema-valued location is not an object Schema")
	}
	locationKey := canonicalPointer(pointer)
	info, indexed := artifact.schemaLocations[locationKey]
	if !indexed {
		return newArtifactError(ErrNonSchema, artifact.ref, locationKey, "location is not a Schema Object under the declared dialect")
	}
	for _, keyword := range []string{"$anchor", "$dynamicRef", "$dynamicAnchor", "$recursiveRef"} {
		if _, exists := object[keyword]; exists {
			return newArtifactError(ErrUnsupportedRef, artifact.ref, locationKey, keyword+" is not supported by Schema Closure Digest v1")
		}
	}
	if artifact.dialect == DialectOpenAPI30 {
		if _, exists := object["$id"]; exists {
			return newArtifactError(ErrUnsupportedRef, artifact.ref, locationKey, "OpenAPI 3.0 Schema Objects must not use $id")
		}
	}
	if rawRef, exists := object["$ref"]; exists {
		ref, ok := rawRef.(string)
		if !ok {
			return newArtifactError(ErrRefResolution, artifact.ref, locationKey, "$ref must be a string")
		}
		target, err := c.resolveRefTarget(artifact, info, ref)
		if err != nil {
			return &Error{Code: errorCodeOr(err, ErrRefResolution), ArtifactRef: artifact.ref, Pointer: locationKey, Detail: err.Error()}
		}
		onTarget(target)
	}
	if artifact.dialect == DialectJSONSchema202012 {
		return forEachJSONSchemaChild(object, pointer, func(childPointer jsonPointer, child any) error {
			return c.walkSchemaValue(artifact, childPointer, child, onTarget)
		})
	}
	return forEachOpenAPISchemaChild(object, pointer, func(childPointer jsonPointer, child any) error {
		return c.walkSchemaValue(artifact, childPointer, child, onTarget)
	})
}

func errorCodeOr(err error, fallback ErrorCode) ErrorCode {
	if code, ok := CodeOf(err); ok {
		return code
	}
	return fallback
}

func forEachJSONSchemaChild(object map[string]any, pointer jsonPointer, visit func(jsonPointer, any) error) error {
	for _, keyword := range jsonSchemaMapKeywords {
		raw, exists := object[keyword]
		if !exists {
			continue
		}
		children, ok := raw.(map[string]any)
		if !ok {
			return newError(ErrSchemaDocument, keyword+" must contain an object of Schemas")
		}
		for _, name := range sortedMapKeys(children) {
			if err := visit(appendPointer(pointer, jsonPointer{keyword, name}), children[name]); err != nil {
				return err
			}
		}
	}
	for _, keyword := range jsonSchemaSingleKeywords {
		raw, exists := object[keyword]
		if !exists {
			continue
		}
		if err := visit(appendPointer(pointer, jsonPointer{keyword}), raw); err != nil {
			return err
		}
	}
	for _, keyword := range jsonSchemaArrayKeywords {
		raw, exists := object[keyword]
		if !exists {
			continue
		}
		children, ok := raw.([]any)
		if !ok {
			return newError(ErrSchemaDocument, keyword+" must be an array of Schemas")
		}
		for index, child := range children {
			if err := visit(appendPointer(pointer, jsonPointer{keyword, arrayToken(index)}), child); err != nil {
				return err
			}
		}
	}
	return nil
}

func forEachOpenAPISchemaChild(object map[string]any, pointer jsonPointer, visit func(jsonPointer, any) error) error {
	if properties, ok, err := optionalObjectField(object, "properties"); err != nil {
		return err
	} else if ok {
		for _, name := range sortedMapKeys(properties) {
			if err := visit(appendPointer(pointer, jsonPointer{"properties", name}), properties[name]); err != nil {
				return err
			}
		}
	}
	if items, exists := object["items"]; exists {
		if _, isArray := items.([]any); isArray {
			return newError(ErrUnsupportedValue, "OpenAPI 3.0 tuple-array items is invalid")
		}
		if err := visit(appendPointer(pointer, jsonPointer{"items"}), items); err != nil {
			return err
		}
	}
	if additional, exists := object["additionalProperties"]; exists {
		if _, isBoolean := additional.(bool); !isBoolean {
			if err := visit(appendPointer(pointer, jsonPointer{"additionalProperties"}), additional); err != nil {
				return err
			}
		}
	}
	if notValue, exists := object["not"]; exists {
		if err := visit(appendPointer(pointer, jsonPointer{"not"}), notValue); err != nil {
			return err
		}
	}
	for _, keyword := range []string{"allOf", "anyOf", "oneOf"} {
		raw, exists := object[keyword]
		if !exists {
			continue
		}
		children, ok := raw.([]any)
		if !ok {
			return newError(ErrSchemaDocument, keyword+" must be an array of Schemas")
		}
		for index, child := range children {
			if err := visit(appendPointer(pointer, jsonPointer{keyword, arrayToken(index)}), child); err != nil {
				return err
			}
		}
	}
	return nil
}
