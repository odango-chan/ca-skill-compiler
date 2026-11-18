package ingresscatalog

// Ingress-owned request normalization and peer profile static semantics.
//
// The released IngressCatalog v1alpha1 contract §4.3/§4.4/§4.5/§5 defines the
// request normalization mapping and peer profile reference obligations that are
// statically owned by the Compiler. This file validates them on the admitted
// catalog document with the same trusted Schema resolution context used by the
// Schema Closure Digest, so it never guesses, falls back or performs network
// resolution.
//
// Diagnostic identities:
//   - INGRESS_NORMALIZATION_INVALID             (§4.3/§4.4 mapping semantics)
//   - INGRESS_BINARY_CONTENT_LIFECYCLE_INVALID  (§4.5 binary lifecycle)
//   - INGRESS_PEER_PROFILE_REFERENCE_INVALID    (§5.1/§5.2 profile reference form)
//
// Missing/invalid source or normalized Schema references keep the existing
// SCHEMA_MISSING / SCHEMA_INVALID identities and are not re-reported here.

import (
	"strconv"
	"strings"
)

// normalizationMappingProjection is one normalized view of a declared mapping.
type normalizationMappingProjection struct {
	pointer   string
	transform string
	source    normalizationSourceProjection
	target    string
}

type normalizationSourceProjection struct {
	kind             string
	name             string
	pointer          string
	part             string
	contentIDPart    string
	contentIDPointer string
}

type normalizationPartProjection struct {
	name    string
	kind    string
	maxByte int64
}

type normalizationRepresentationProjection struct {
	pointer       string
	mediaType     string
	parts         []normalizationPartProjection
	mappings      []normalizationMappingProjection
	ignored       []normalizationSourceProjection
	minBinary     int64
	maxBinary     int64
	hasMinBinary  bool
	hasMaxBinary  bool
	hasParts      bool
	hasNormalized bool
}

// checkIngressStaticSemantics runs the catalog-level semantic checks that are not
// expressed by the structural JSON Schema: request normalization (§4.3/§4.4),
// immutable binary content lifecycle (§4.5) and peer profile reference form
// (§5.1/§5.2).
func checkIngressStaticSemantics(catalog *Catalog, root map[string]any, snapshot *TrustedSnapshot, input CatalogInput) []Diagnostic {
	if catalog == nil || root == nil {
		return nil
	}
	var diagnostics []Diagnostic
	operations := arrayField(specOf(root), "operations")
	for index, operationValue := range operations {
		operationMap, ok := operationValue.(map[string]any)
		if !ok || index >= len(catalog.Operations) {
			continue
		}
		operation := catalog.Operations[index]
		diagnostics = append(diagnostics, checkOperationNormalization(operation, operationMap, index, snapshot, input)...)
		diagnostics = append(diagnostics, checkOperationPeerProfiles(operation, operationMap, index, input)...)
	}
	return diagnostics
}

func specOf(root map[string]any) map[string]any {
	return objectField(root, "spec")
}

func invalidNormalization(operation Operation, index int, input CatalogInput, pointer string) Diagnostic {
	return ingressDiagnostic(
		CodeIngressNormalizationInvalid,
		renderMessage(ingressMessageTemplates[CodeIngressNormalizationInvalid],
			"{exportId}", operation.ID, "{fieldPath}", pointer),
		input.DependencyAlias, input.SnapshotPath, operationPointer(index, pointer))
}

func invalidBinaryLifecycle(operation Operation, index int, input CatalogInput, pointer string) Diagnostic {
	return ingressDiagnostic(
		CodeIngressBinaryContentLifecycleInvalid,
		renderMessage(ingressMessageTemplates[CodeIngressBinaryContentLifecycleInvalid],
			"{exportId}", operation.ID, "{fieldPath}", pointer),
		input.DependencyAlias, input.SnapshotPath, operationPointer(index, pointer))
}

func checkOperationNormalization(operation Operation, operationMap map[string]any, index int, snapshot *TrustedSnapshot, input CatalogInput) []Diagnostic {
	request := objectField(operationMap, "request")
	if request == nil {
		return nil
	}
	var diagnostics []Diagnostic
	accepted := stringSequence(arrayField(request, "acceptedMediaTypes"))
	sources := arrayField(request, "sourceSchemas")
	normalization := objectField(request, "normalization")
	representations := arrayField(normalization, "representations")

	// §4.3/§4.4: acceptedMediaTypes, sourceSchemas and representations media
	// types are one-to-one, deduplicated and use the normalized media-type
	// essence.
	acceptedSet := map[string]bool{}
	for _, mediaType := range accepted {
		if acceptedSet[mediaType] {
			diagnostics = append(diagnostics, invalidNormalization(operation, index, input, "/request/acceptedMediaTypes"))
			break
		}
		acceptedSet[mediaType] = true
	}
	sourceMedia := map[string]bool{}
	sourceSchemaByMedia := map[string]string{}
	for _, sourceValue := range sources {
		source, _ := sourceValue.(map[string]any)
		mediaType := stringField(source, "mediaType")
		if sourceMedia[mediaType] {
			diagnostics = append(diagnostics, invalidNormalization(operation, index, input, "/request/sourceSchemas"))
			continue
		}
		sourceMedia[mediaType] = true
		sourceSchemaByMedia[mediaType] = stringField(source, "schemaRef")
	}
	if !sameStringSet(acceptedSet, sourceMedia) {
		diagnostics = append(diagnostics, invalidNormalization(operation, index, input, "/request/sourceSchemas"))
	}
	representationMedia := map[string]bool{}
	projections := make([]normalizationRepresentationProjection, 0, len(representations))
	for representationIndex, value := range representations {
		representation, _ := value.(map[string]any)
		projection := projectRepresentation(representation, representationIndex)
		if representationMedia[projection.mediaType] {
			diagnostics = append(diagnostics, invalidNormalization(operation, index, input, projection.pointer))
			continue
		}
		representationMedia[projection.mediaType] = true
		projections = append(projections, projection)
	}
	if !sameStringSet(acceptedSet, representationMedia) {
		diagnostics = append(diagnostics, invalidNormalization(operation, index, input, "/request/normalization/representations"))
	}

	normalizedRoot, normalizedOK := lookupSchemaRoot(snapshot, operation.Request.Normalized.SchemaRef)
	sourceRoots := map[string]RootRef{}
	for mediaType, schemaRef := range sourceSchemaByMedia {
		if rootRef, ok := lookupSchemaRoot(snapshot, schemaRef); ok {
			sourceRoots[mediaType] = rootRef
		}
	}

	for _, representation := range projections {
		diagnostics = append(diagnostics, checkRepresentation(operation, index, input, representation, normalizedRoot, normalizedOK, sourceRoots[representation.mediaType], snapshot)...)
	}
	diagnostics = append(diagnostics, checkRequestBinaryContent(operation, index, input, operationMap, projections, snapshot)...)
	return diagnostics
}

func projectRepresentation(representation map[string]any, representationIndex int) normalizationRepresentationProjection {
	projection := normalizationRepresentationProjection{
		pointer:   "/request/normalization/representations/" + strconv.Itoa(representationIndex),
		mediaType: stringField(representation, "mediaType"),
	}
	for partIndex, partValue := range arrayField(representation, "parts") {
		part, _ := partValue.(map[string]any)
		projection.parts = append(projection.parts, normalizationPartProjection{
			name:    stringField(part, "name"),
			kind:    stringField(part, "kind"),
			maxByte: integerField(part, "maxBytes"),
		})
		_ = partIndex
	}
	projection.hasParts = representation["parts"] != nil
	if value, ok := representation["minBinaryParts"].(int64); ok {
		projection.minBinary, projection.hasMinBinary = value, true
	}
	if value, ok := representation["maxBinaryParts"].(int64); ok {
		projection.maxBinary, projection.hasMaxBinary = value, true
	}
	for mappingIndex, mappingValue := range arrayField(representation, "mappings") {
		mapping, _ := mappingValue.(map[string]any)
		source := objectField(mapping, "source")
		projection.mappings = append(projection.mappings, normalizationMappingProjection{
			pointer:   projection.pointer + "/mappings/" + strconv.Itoa(mappingIndex),
			transform: stringField(mapping, "transform"),
			target:    stringField(mapping, "target"),
			source: normalizationSourceProjection{
				kind:             stringField(source, "kind"),
				name:             stringField(source, "name"),
				pointer:          stringField(source, "pointer"),
				part:             stringField(source, "part"),
				contentIDPart:    stringField(objectField(source, "contentIdReference"), "part"),
				contentIDPointer: stringField(objectField(source, "contentIdReference"), "pointer"),
			},
		})
	}
	for ignoredIndex, ignoredValue := range arrayField(representation, "ignoredSources") {
		ignored, _ := ignoredValue.(map[string]any)
		source := objectField(ignored, "source")
		projection.ignored = append(projection.ignored, normalizationSourceProjection{
			kind:    stringField(source, "kind"),
			name:    stringField(source, "name"),
			pointer: stringField(source, "pointer"),
			part:    stringField(source, "part"),
		})
		_ = ignoredIndex
	}
	return projection
}

func checkRepresentation(
	operation Operation,
	operationIndex int,
	input CatalogInput,
	representation normalizationRepresentationProjection,
	normalizedRoot RootRef,
	normalizedOK bool,
	sourceRoot RootRef,
	snapshot *TrustedSnapshot,
) []Diagnostic {
	var diagnostics []Diagnostic
	invalid := func(pointer string) []Diagnostic {
		return []Diagnostic{invalidNormalization(operation, operationIndex, input, pointer)}
	}
	multipart := strings.HasPrefix(representation.mediaType, "multipart/")

	// §4.4: a multipart representation must declare parts, minBinaryParts and
	// maxBinaryParts; the bounds must be consistent.
	partNames := map[string]normalizationPartProjection{}
	for _, part := range representation.parts {
		if part.name == "" || partNames[part.name].name != "" {
			diagnostics = append(diagnostics, invalid(representation.pointer+"/parts")...)
			continue
		}
		partNames[part.name] = part
	}
	if multipart {
		if !representation.hasParts || len(representation.parts) == 0 {
			diagnostics = append(diagnostics, invalid(representation.pointer+"/parts")...)
		}
		if !representation.hasMinBinary || representation.minBinary < 1 {
			diagnostics = append(diagnostics, invalid(representation.pointer+"/minBinaryParts")...)
		}
		if !representation.hasMaxBinary || representation.maxBinary < 1 {
			diagnostics = append(diagnostics, invalid(representation.pointer+"/maxBinaryParts")...)
		}
		if representation.hasMinBinary && representation.hasMaxBinary && representation.minBinary > representation.maxBinary {
			diagnostics = append(diagnostics, invalid(representation.pointer+"/minBinaryParts")...)
		}
	}

	// §4.4: each mapping has exactly one target, no duplicate target, and the
	// declared source selector/transform combination is statically valid.
	targets := map[string]int{}
	referencedParts := map[string]bool{}
	for _, mapping := range representation.mappings {
		if previous, exists := targets[mapping.target]; exists {
			_ = previous
			diagnostics = append(diagnostics, invalid(mapping.pointer+"/target")...)
			continue
		}
		targets[mapping.target] = 1
		var targetResult SchemaPathResult
		targetOK := false
		if normalizedOK {
			if result, err := lookupDataPath(snapshot, normalizedRoot, mapping.target); err == nil {
				targetResult, targetOK = result, true
			}
		}
		if !targetOK {
			diagnostics = append(diagnostics, invalid(mapping.pointer+"/target")...)
		}
		sourceSchema, sourceOK := lookupSourceSchema(snapshot, sourceRoot, representation.mediaType, mapping.source)
		if !sourceOK {
			diagnostics = append(diagnostics, invalid(mapping.pointer+"/source")...)
		}
		if mapping.source.kind == "multipart-json" || mapping.source.kind == "multipart-binary" {
			if _, exists := partIndexByName(representation.parts, mapping.source.part); !exists {
				diagnostics = append(diagnostics, invalid(mapping.pointer+"/source/part")...)
			} else {
				referencedParts[mapping.source.part] = true
			}
		}
		switch mapping.transform {
		case "identity":
			if mapping.source.kind == "multipart-binary" {
				diagnostics = append(diagnostics, invalid(mapping.pointer+"/transform")...)
			} else if targetOK && sourceOK && !identityAssignable(schemaTypes(sourceSchema), schemaTypes(targetResult.Schema)) {
				diagnostics = append(diagnostics, invalid(mapping.pointer+"/transform")...)
			}
		case "parse-integer", "parse-number", "parse-boolean":
			if targetOK && sourceOK {
				sourceTypes := schemaTypes(sourceSchema)
				targetTypes := schemaTypes(targetResult.Schema)
				if !parseTransformAssignable(mapping.transform, sourceTypes, targetTypes) {
					diagnostics = append(diagnostics, invalid(mapping.pointer+"/transform")...)
				}
			}
		case "binary-content-ref":
			if mapping.source.kind != "multipart-binary" {
				diagnostics = append(diagnostics, invalid(mapping.pointer+"/transform")...)
			}
			if mapping.source.contentIDPart == "" || mapping.source.contentIDPointer == "" {
				diagnostics = append(diagnostics, invalid(mapping.pointer+"/source/contentIdReference")...)
			}
			if targetOK && !targetIsBinaryReference(snapshot, operation, targetResult) {
				diagnostics = append(diagnostics, invalid(mapping.pointer+"/target")...)
			}
		}
	}
	for partName, part := range partNames {
		if part.kind == "binary" && !referencedParts[partName] {
			diagnostics = append(diagnostics, invalid(representation.pointer+"/parts")...)
		}
	}

	// §4.4: every required normalized property is provided exactly once, and
	// every required source property is explicitly mapped or explicitly ignored.
	if normalizedOK {
		for _, required := range normalizedRequiredProperties(snapshot, normalizedRoot) {
			covered := mappingTargetCount(representation.mappings, required)
			if covered != 1 {
				diagnostics = append(diagnostics, invalid(representation.pointer+"/mappings")...)
			}
		}
	}
	if sourceOKRoot := sourceRootValid(sourceRoot); sourceOKRoot {
		for _, required := range schemaRequiredProperties(snapshot, sourceRoot) {
			if !requiredSourceCovered(representation, required) {
				diagnostics = append(diagnostics, invalid(representation.pointer+"/mappings")...)
			}
		}
	}

	// §4.4: ignored sources must be resolvable and must not also be mapped.
	for ignoredIndex, ignored := range representation.ignored {
		pointer := representation.pointer + "/ignoredSources/" + strconv.Itoa(ignoredIndex) + "/source"
		if _, ok := lookupSourceSchema(snapshot, sourceRoot, representation.mediaType, ignored); !ok {
			diagnostics = append(diagnostics, invalid(pointer)...)
		}
		if sourceSelectorEqual(ignored, mappingSources(representation)) {
			diagnostics = append(diagnostics, invalid(pointer)...)
		}
	}
	return diagnostics
}

func checkRequestBinaryContent(operation Operation, operationIndex int, input CatalogInput, operationMap map[string]any, representations []normalizationRepresentationProjection, snapshot *TrustedSnapshot) []Diagnostic {
	request := objectField(operationMap, "request")
	binary := objectField(request, "binaryContent")
	usesBinary := false
	for _, representation := range representations {
		for _, part := range representation.parts {
			if part.kind == "binary" {
				usesBinary = true
			}
		}
		for _, mapping := range representation.mappings {
			if mapping.source.kind == "multipart-binary" || mapping.transform == "binary-content-ref" {
				usesBinary = true
			}
		}
	}
	if !usesBinary {
		return nil
	}
	var diagnostics []Diagnostic
	invalid := func(pointer string) []Diagnostic {
		return []Diagnostic{invalidBinaryLifecycle(operation, operationIndex, input, pointer)}
	}
	if binary == nil {
		return invalid("/request/binaryContent")
	}
	referenceSchemaRef := stringField(binary, "referenceSchemaRef")
	if _, ok := lookupSchemaRoot(snapshot, referenceSchemaRef); !ok {
		diagnostics = append(diagnostics, invalid("/request/binaryContent/referenceSchemaRef")...)
	}
	if stringField(binary, "digestAlgorithm") != "sha-256" || binary["immutable"] != true || stringField(binary, "access") != "authorized-tool-only" {
		diagnostics = append(diagnostics, invalid("/request/binaryContent")...)
	}
	if integerField(binary, "maxBytes") < 1 {
		diagnostics = append(diagnostics, invalid("/request/binaryContent/maxBytes")...)
	}
	retention := stringField(binary, "retention")
	retentionSeconds, ok := ingressDurationSeconds(retention)
	if !ok || retentionSeconds <= 0 {
		diagnostics = append(diagnostics, invalid("/request/binaryContent/retention")...)
	} else if window, ok := declaredRecoveryWindowSeconds(operationMap); ok && retentionSeconds < window {
		// §4.5/IN-042: binary retention must cover the declared recovery window.
		diagnostics = append(diagnostics, invalid("/request/binaryContent/retention")...)
	}
	return diagnostics
}

// declaredRecoveryWindowSeconds returns the largest declared late-arrival /
// tombstone recovery window of the export, including the clock-skew allowance.
func declaredRecoveryWindowSeconds(operationMap map[string]any) (int64, bool) {
	stale := objectField(objectField(operationMap, "correlation"), "staleDelivery")
	if stale == nil {
		return 0, false
	}
	maximumLateArrival, okLate := ingressDurationSeconds(stringField(stale, "maximumLateArrival"))
	clockSkew, okSkew := ingressDurationSeconds(stringField(stale, "clockSkewAllowance"))
	if !okLate || !okSkew {
		return 0, false
	}
	window := maximumLateArrival + clockSkew
	for _, field := range []string{"tombstoneRetention", "discriminatorRetention"} {
		if value, ok := ingressDurationSeconds(stringField(stale, field)); ok && value > window {
			window = value
		}
	}
	return window, true
}

func checkOperationPeerProfiles(operation Operation, operationMap map[string]any, index int, input CatalogInput) []Diagnostic {
	security := objectField(operationMap, "security")
	if security == nil {
		return nil
	}
	var diagnostics []Diagnostic
	for _, field := range []string{"authenticationProfileRef", "authorizationPolicyRef"} {
		value := stringField(security, field)
		if !validPeerProfileReference(value) {
			diagnostics = append(diagnostics, ingressDiagnostic(
				CodeIngressPeerProfileReferenceInvalid,
				renderMessage(ingressMessageTemplates[CodeIngressPeerProfileReferenceInvalid],
					"{exportId}", operation.ID, "{fieldPath}", "/security/"+field),
				input.DependencyAlias, input.SnapshotPath, operationPointer(index, "/security/"+field)))
		}
	}
	return diagnostics
}

// validPeerProfileReference accepts the released "Dependency Alias::resource"
// reference form used by the Ingress contract §5.1: a non-empty identifier
// alias, the exact "::" separator and a non-empty non-secret resource identity.
// Deployment profile binding itself is Host Admission, not Compiler.
func validPeerProfileReference(value string) bool {
	if value == "" || strings.ContainsAny(value, " \t\r\n") {
		return false
	}
	parts := strings.Split(value, "::")
	if len(parts) != 2 {
		return false
	}
	alias, resource := parts[0], parts[1]
	if alias == "" || resource == "" || strings.Contains(resource, "::") {
		return false
	}
	if !validDependencyAlias(alias) {
		return false
	}
	for _, r := range resource {
		if r <= 0x20 || r == 0x7f || r == '/' || r == '\\' {
			return false
		}
	}
	return true
}

func validDependencyAlias(value string) bool {
	if value == "" || !asciiLetter(rune(value[0])) {
		return false
	}
	for _, r := range value {
		if !asciiLetter(r) && !(r >= '0' && r <= '9') && r != '.' && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func asciiLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// ingressDurationSeconds parses the fixed positive ISO-8601 duration subset used
// by the Ingress contract (P[nD][T[nH][nM][nS]]).
func ingressDurationSeconds(value string) (int64, bool) {
	if value == "" || value[0] != 'P' {
		return 0, false
	}
	rest := value[1:]
	var total int64
	inTime := false
	sawComponent := false
	for rest != "" {
		if rest[0] == 'T' {
			if inTime {
				return 0, false
			}
			inTime = true
			rest = rest[1:]
			continue
		}
		digits := 0
		for digits < len(rest) && rest[digits] >= '0' && rest[digits] <= '9' {
			digits++
		}
		if digits == 0 || digits == len(rest) {
			return 0, false
		}
		number, err := strconv.ParseInt(rest[:digits], 10, 64)
		if err != nil || number < 0 {
			return 0, false
		}
		unit := rest[digits]
		factor := int64(0)
		switch {
		case !inTime && unit == 'D':
			factor = 86400
		case inTime && unit == 'H':
			factor = 3600
		case inTime && unit == 'M':
			factor = 60
		case inTime && unit == 'S':
			factor = 1
		default:
			return 0, false
		}
		if number > (1<<62)/factor {
			return 0, false
		}
		total += number * factor
		sawComponent = true
		rest = rest[digits+1:]
	}
	return total, sawComponent
}

// lookupSchemaRoot resolves the trusted Schema root declaration of one catalog
// schemaRef. Only the caller-declared trusted export map is consulted.
func lookupSchemaRoot(snapshot *TrustedSnapshot, schemaRef string) (RootRef, bool) {
	if snapshot == nil || schemaRef == "" {
		return RootRef{}, false
	}
	root, ok := snapshot.SchemaExports[schemaRef]
	if !ok {
		return RootRef{}, false
	}
	return root, true
}

// lookupDataPath resolves one catalog data pointer (for example "/updateData")
// against a trusted Schema root.
func lookupDataPath(snapshot *TrustedSnapshot, root RootRef, dataPath string) (SchemaPathResult, error) {
	if snapshot == nil || snapshot.Context == nil {
		return SchemaPathResult{}, newError(ErrRootResolution, "no trusted resolution context")
	}
	canonical := "#"
	if dataPath != "" {
		canonical = "#" + dataPath
	}
	return snapshot.Context.LookupSchemaPath(root, canonical)
}

func lookupSourceSchema(snapshot *TrustedSnapshot, sourceRoot RootRef, mediaType string, source normalizationSourceProjection) (any, bool) {
	switch source.kind {
	case "json-body":
		return lookupSchemaForSelector(snapshot, sourceRoot, source.pointer)
	case "multipart-json":
		// The exact part Schema binding is resolved by the catalog's declared
		// part list; a part-level Schema is optional in v1alpha1, so a part
		// without a schema binding can only prove selector existence.
		return nil, source.part != ""
	case "multipart-binary":
		return nil, source.part != "" && source.contentIDPointer != ""
	case "path", "query", "header":
		return nil, source.name != ""
	default:
		return nil, false
	}
}

func lookupSchemaForSelector(snapshot *TrustedSnapshot, root RootRef, pointer string) (any, bool) {
	if root.ArtifactRef == "" {
		// A source Schema is optional for non-body selectors; an absent root
		// cannot prove anything about the selector.
		return nil, pointer == ""
	}
	result, err := lookupDataPath(snapshot, root, pointer)
	if err != nil {
		return nil, false
	}
	return result.Schema, true
}

func sourceRootValid(root RootRef) bool {
	return root.ArtifactRef != ""
}

func schemaRequiredProperties(snapshot *TrustedSnapshot, root RootRef) []string {
	value, ok := schemaRootObject(snapshot, root)
	if !ok {
		return nil
	}
	required, _ := value["required"].([]any)
	out := make([]string, 0, len(required))
	for _, entry := range required {
		if name, ok := entry.(string); ok {
			out = append(out, name)
		}
	}
	return out
}

func normalizedRequiredProperties(snapshot *TrustedSnapshot, root RootRef) []string {
	return schemaRequiredProperties(snapshot, root)
}

func schemaRootObject(snapshot *TrustedSnapshot, root RootRef) (map[string]any, bool) {
	if snapshot == nil || snapshot.Context == nil {
		return nil, false
	}
	resolved, err := snapshot.Context.ResolveRoot(root)
	if err != nil {
		return nil, false
	}
	object, ok := resolved.Schema.(map[string]any)
	if !ok {
		return nil, false
	}
	if _, isRef := object["$ref"]; isRef {
		return nil, false
	}
	return object, true
}

func schemaTypes(schema any) []string {
	object, ok := schema.(map[string]any)
	if !ok {
		return nil
	}
	switch raw := object["type"].(type) {
	case string:
		return []string{raw}
	case []any:
		out := make([]string, 0, len(raw))
		for _, entry := range raw {
			if name, ok := entry.(string); ok {
				out = append(out, name)
			}
		}
		return out
	default:
		return nil
	}
}

func identityAssignable(sourceTypes, targetTypes []string) bool {
	if len(targetTypes) == 0 {
		return true
	}
	if len(sourceTypes) == 0 {
		return true
	}
	for _, source := range sourceTypes {
		match := false
		for _, target := range targetTypes {
			if source == target || target == "number" && source == "integer" {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}

func parseTransformAssignable(transform string, sourceTypes, targetTypes []string) bool {
	if !containsType(sourceTypes, "string") {
		return false
	}
	switch transform {
	case "parse-integer":
		return containsType(targetTypes, "integer")
	case "parse-number":
		return containsType(targetTypes, "number") || containsType(targetTypes, "integer")
	case "parse-boolean":
		return containsType(targetTypes, "boolean")
	}
	return false
}

func containsType(types []string, wanted string) bool {
	for _, value := range types {
		if value == wanted {
			return true
		}
	}
	return false
}

func targetIsBinaryReference(snapshot *TrustedSnapshot, operation Operation, target SchemaPathResult) bool {
	// A binary-content-ref target must resolve to a declared Schema property; the
	// exact immutable reference Schema identity is bound by request.binaryContent
	// and validated by checkRequestBinaryContent.
	_ = snapshot
	_ = operation
	return target.ArtifactRef != "" && target.Pointer != ""
}

func partIndexByName(parts []normalizationPartProjection, name string) (int, bool) {
	if name == "" {
		return -1, false
	}
	for index, part := range parts {
		if part.name == name {
			return index, true
		}
	}
	return -1, false
}

func mappingTargetCount(mappings []normalizationMappingProjection, required string) int {
	count := 0
	wanted := "/" + required
	for _, mapping := range mappings {
		if mapping.target == wanted || strings.HasPrefix(mapping.target, wanted+"/") {
			count++
		}
	}
	return count
}

func requiredSourceCovered(representation normalizationRepresentationProjection, required string) bool {
	wanted := "/" + required
	covers := func(source normalizationSourceProjection) bool {
		if source.kind != "json-body" && source.kind != "multipart-json" {
			return false
		}
		if source.pointer == "" || source.pointer == wanted || strings.HasPrefix(source.pointer, wanted+"/") {
			return true
		}
		return false
	}
	for _, mapping := range representation.mappings {
		if covers(mapping.source) {
			return true
		}
	}
	for _, ignored := range representation.ignored {
		if covers(ignored) {
			return true
		}
	}
	return false
}

func mappingSources(representation normalizationRepresentationProjection) []normalizationSourceProjection {
	sources := make([]normalizationSourceProjection, 0, len(representation.mappings))
	for _, mapping := range representation.mappings {
		sources = append(sources, mapping.source)
	}
	return sources
}

func sourceSelectorEqual(self normalizationSourceProjection, others []normalizationSourceProjection) bool {
	for _, other := range others {
		if self.kind == other.kind && self.name == other.name && self.pointer == other.pointer && self.part == other.part {
			return true
		}
	}
	return false
}

func stringSequence(values []any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func sameStringSet(left, right map[string]bool) bool {
	if len(left) != len(right) {
		return false
	}
	for key := range left {
		if !right[key] {
			return false
		}
	}
	return true
}
