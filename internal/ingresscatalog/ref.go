package ingresscatalog

// resolveRefTarget performs deterministic RFC 3986 resolution of one $ref
// string against the effective base URI of the containing Schema Object,
// percent-decodes the fragment exactly once, validates RFC 6901 escaping and
// translates a nested-resource fragment pointer into the artifact-relative
// pointer stored in the digest manifest.
func (c *ResolutionContext) resolveRefTarget(artifact *resolvedArtifact, info schemaLocationInfo, rawRef string) (schemaIdentity, error) {
	if rawRef == "" {
		return schemaIdentity{}, newError(ErrRefResolution, "$ref must not be empty")
	}
	reference, err := parseURIReference(rawRef)
	if err != nil {
		return schemaIdentity{}, newError(ErrRefResolution, "invalid $ref URI-reference: "+err.Error())
	}
	base, err := parseURIReference(info.baseURI)
	if err != nil {
		return schemaIdentity{}, newError(ErrRefResolution, "containing Schema has no usable base URI")
	}
	resolved, err := resolveReferenceURI(base, reference)
	if err != nil {
		return schemaIdentity{}, newError(ErrRefResolution, "$ref cannot be resolved: "+err.Error())
	}
	fragment := ""
	if resolved.hasFragment {
		fragment = resolved.fragment
		resolved.hasFragment = false
		resolved.fragment = ""
	}
	resourceURI, err := resolved.canonicalString(false, true)
	if err != nil {
		return schemaIdentity{}, newError(ErrRefResolution, "$ref target URI cannot be canonicalized: "+err.Error())
	}
	binding, err := c.resolveResource(resourceURI)
	if err != nil {
		return schemaIdentity{}, err
	}
	decodedFragment, err := percentDecodeOnce(fragment)
	if err != nil {
		return schemaIdentity{}, newError(ErrRefResolution, "invalid $ref fragment: "+err.Error())
	}
	fragmentPointer, err := parsePointerFragment(decodedFragment)
	if err != nil {
		return schemaIdentity{}, newError(ErrRefResolution, "invalid $ref JSON Pointer fragment: "+err.Error())
	}
	targetPointer := appendPointer(binding.pointer, fragmentPointer)
	targetArtifact, ok := c.artifacts[binding.artifactRef]
	if !ok {
		return schemaIdentity{}, newError(ErrRefResolution, "resolved artifact is not in the trusted snapshot")
	}
	targetKey := canonicalPointer(targetPointer)
	if _, indexed := targetArtifact.schemaLocations[targetKey]; !indexed {
		value, resolveErr := resolvePointer(targetArtifact.document, targetPointer)
		if resolveErr != nil {
			return schemaIdentity{}, newError(ErrRefResolution, "unresolved $ref target within artifact: "+resolveErr.Error())
		}
		if _, isBoolean := value.(bool); isBoolean {
			return schemaIdentity{}, newArtifactError(ErrBooleanSchema, targetArtifact.ref, targetKey, "$ref target is a Boolean Schema")
		}
		return schemaIdentity{}, newArtifactError(ErrNonSchema, targetArtifact.ref, targetKey, "$ref target is not a Schema Object under the declared dialect")
	}
	return schemaIdentity{artifactRef: targetArtifact.ref, pointer: targetPointer}, nil
}
