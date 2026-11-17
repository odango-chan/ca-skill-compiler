package ingresscatalog

// Additive trusted-context helpers used by the v1alpha5 WP3 trigger/correlation
// slice. They do not change the WP1 admission semantics: raw bytes remain
// authoritative, the digest is always recomputed from those bytes, and a caller
// can never supply a digest through these constructors.

// NewSnapshotArtifactFromRaw builds one immutable trusted artifact from its raw
// bytes only. The document is parsed once through the single normalized YAML AST
// used by the rest of the compiler; a parse failure is left to resolution, which
// reports the deterministic structured error. No artifact digest is accepted
// from the caller: ResolutionContext always recomputes it from Raw.
func NewSnapshotArtifactFromRaw(artifactRef, snapshotPath string, dialect Dialect, raw []byte) SnapshotArtifact {
	artifact := SnapshotArtifact{
		ArtifactRef:  artifactRef,
		SnapshotPath: snapshotPath,
		Dialect:      dialect,
		Raw:          raw,
	}
	if parsed, err := parseStrictJSONValue(raw); err == nil {
		artifact.Parsed = parsed
	}
	return artifact
}

// SchemaPathResult is the exact target of one data-path lookup inside a trusted
// Schema root. Pointer is the canonical artifact-relative Schema pointer of the
// object Schema that declares the resolved property.
type SchemaPathResult struct {
	ArtifactRef string
	Pointer     string
	Schema      any
}

// LookupSchemaPath resolves one RFC 6901 data path (for example "#/smContextRef")
// against one trusted Schema root and returns the object Schema that describes
// the property. It follows exact local and cross-artifact $ref indirections
// through the same trusted resource index used by Schema Closure Digest v1, and
// every intermediate location must be a Schema Object under the declared dialect.
// It performs no name search, prefix match, composition merge or fallback: a
// property that is not declared exactly by the resolved Schema is an error.
func (c *ResolutionContext) LookupSchemaPath(root RootRef, dataPath string) (SchemaPathResult, error) {
	if c == nil {
		return SchemaPathResult{}, newError(ErrRootResolution, "no trusted resolution context")
	}
	tokens, err := parseCanonicalPointer(dataPath)
	if err != nil {
		return SchemaPathResult{}, err
	}
	identity, value, artifact, err := c.resolveRootIdentity(root)
	if err != nil {
		return SchemaPathResult{}, err
	}
	for _, token := range tokens {
		identity, value, artifact, err = c.resolveRefChain(artifact, identity, value)
		if err != nil {
			return SchemaPathResult{}, err
		}
		object, ok := value.(map[string]any)
		if !ok {
			return SchemaPathResult{}, newArtifactError(ErrNonSchema, artifact.ref, identity.wirePointer(),
				"data path traverses a non-object Schema")
		}
		properties, ok := object["properties"].(map[string]any)
		if !ok {
			return SchemaPathResult{}, newArtifactError(ErrMissingPointer, artifact.ref, identity.wirePointer(),
				"Schema declares no properties for data path token '"+token+"'")
		}
		next, exists := properties[token]
		if !exists {
			return SchemaPathResult{}, newArtifactError(ErrMissingPointer, artifact.ref, identity.wirePointer(),
				"Schema does not declare property '"+token+"'")
		}
		childPointer := appendPointer(identity.pointer, jsonPointer{"properties", token})
		if _, indexed := artifact.schemaLocations[canonicalPointer(childPointer)]; !indexed {
			return SchemaPathResult{}, newArtifactError(ErrNonSchema, artifact.ref, canonicalPointer(childPointer),
				"resolved property is not a Schema Object under the declared dialect")
		}
		identity = schemaIdentity{artifactRef: artifact.ref, pointer: childPointer}
		value = next
	}
	identity, value, artifact, err = c.resolveRefChain(artifact, identity, value)
	if err != nil {
		return SchemaPathResult{}, err
	}
	return SchemaPathResult{
		ArtifactRef: artifact.ref,
		Pointer:     identity.wirePointer(),
		Schema:      value,
	}, nil
}

// resolveRefChain follows exact "$ref" indirections from one Schema location and
// returns the first non-reference Schema identity. A reference chain longer than
// 64 hops is rejected as a cycle instead of looping.
func (c *ResolutionContext) resolveRefChain(artifact *resolvedArtifact, identity schemaIdentity, value any) (schemaIdentity, any, *resolvedArtifact, error) {
	for hops := 0; hops < 64; hops++ {
		object, ok := value.(map[string]any)
		if !ok {
			return identity, value, artifact, nil
		}
		rawRef, exists := object["$ref"]
		if !exists {
			return identity, value, artifact, nil
		}
		ref, ok := rawRef.(string)
		if !ok {
			return schemaIdentity{}, nil, nil, newArtifactError(ErrRefResolution, artifact.ref,
				identity.wirePointer(), "$ref must be a string")
		}
		locationKey := identity.wirePointer()
		info, indexed := artifact.schemaLocations[locationKey]
		if !indexed {
			return schemaIdentity{}, nil, nil, newArtifactError(ErrNonSchema, artifact.ref, locationKey,
				"reference location is not a Schema Object under the declared dialect")
		}
		target, err := c.resolveRefTarget(artifact, info, ref)
		if err != nil {
			return schemaIdentity{}, nil, nil, &Error{Code: errorCodeOr(err, ErrRefResolution),
				ArtifactRef: artifact.ref, Pointer: locationKey, Detail: err.Error()}
		}
		targetArtifact := c.artifacts[target.artifactRef]
		if targetArtifact == nil {
			return schemaIdentity{}, nil, nil, newError(ErrRefResolution, "resolved artifact is not in the trusted snapshot")
		}
		targetValue, err := resolvePointer(targetArtifact.document, target.pointer)
		if err != nil {
			return schemaIdentity{}, nil, nil, &Error{Code: ErrMissingPointer, ArtifactRef: targetArtifact.ref,
				Pointer: canonicalPointer(target.pointer), Detail: err.Error()}
		}
		identity, value, artifact = target, targetValue, targetArtifact
	}
	return schemaIdentity{}, nil, nil, newError(ErrRefResolution, "$ref chain exceeds 64 hops")
}
