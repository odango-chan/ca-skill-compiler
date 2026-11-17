package ingresscatalog

import (
	"sort"
	"strings"
)

// Schema-valued keywords traversed for JSON Schema Draft 2020-12 closure
// discovery. $defs and definitions are Schema containers; the applicator
// keywords cover every position where a Schema Object may occur in the
// 2020-12 core/applicator vocabularies that this algorithm supports. Legacy
// annotation keywords are not traversed because they cannot contain Schemas.
var jsonSchemaMapKeywords = []string{
	"$defs",
	"definitions",
	"properties",
	"patternProperties",
	"dependentSchemas",
}

var jsonSchemaSingleKeywords = []string{
	"additionalProperties",
	"unevaluatedProperties",
	"propertyNames",
	"items",
	"unevaluatedItems",
	"contains",
	"not",
	"if",
	"then",
	"else",
	"contentSchema",
}

var jsonSchemaArrayKeywords = []string{
	"prefixItems",
	"allOf",
	"anyOf",
	"oneOf",
}

// openAPIMethods is the fixed HTTP method set of an OpenAPI 3.0 Path Item.
var openAPIMethods = []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// indexJSONSchemaArtifact indexes every Schema Object location and registers
// every $id alias of one JSON Schema 2020-12 artifact. The document root of a
// JSON Schema artifact is itself a Schema Object.
func (c *ResolutionContext) indexJSONSchemaArtifact(artifact *resolvedArtifact) error {
	root, ok := artifact.document.(map[string]any)
	if !ok {
		if _, isBoolean := artifact.document.(bool); isBoolean {
			return newArtifactError(ErrBooleanSchema, artifact.ref, "#", "Boolean Schema roots are not supported")
		}
		return newArtifactError(ErrSchemaDocument, artifact.ref, "#", "JSON Schema document root must be an object Schema")
	}
	return c.indexJSONSchemaValue(artifact, nil, artifact.documentURI, nil, root)
}

func (c *ResolutionContext) indexJSONSchemaValue(artifact *resolvedArtifact, pointer jsonPointer, baseURI string, resourceRoot jsonPointer, value any) error {
	object, ok := value.(map[string]any)
	if !ok {
		if _, isBoolean := value.(bool); isBoolean {
			return newArtifactError(ErrBooleanSchema, artifact.ref, canonicalPointer(pointer), "Boolean Schemas are not supported")
		}
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), "Schema-valued location must contain an object Schema")
	}
	info := schemaLocationInfo{baseURI: baseURI, resourceRoot: resourceRoot}
	if idValue, exists := object["$id"]; exists {
		id, isString := idValue.(string)
		if !isString || id == "" {
			return newArtifactError(ErrIDAlias, artifact.ref, canonicalPointer(pointer), "$id must be a non-empty string")
		}
		if strings.ContainsRune(id, '#') {
			return newArtifactError(ErrIDAlias, artifact.ref, canonicalPointer(pointer), "$id must not contain a fragment")
		}
		idReference, err := parseURIReference(id)
		if err != nil {
			return newArtifactError(ErrIDAlias, artifact.ref, canonicalPointer(pointer), "$id is not a valid URI-reference: "+err.Error())
		}
		if idReference.hasFragment {
			return newArtifactError(ErrIDAlias, artifact.ref, canonicalPointer(pointer), "$id must not contain a fragment")
		}
		baseReference, err := parseURIReference(baseURI)
		if err != nil {
			return newArtifactError(ErrIDAlias, artifact.ref, canonicalPointer(pointer), "inherited base URI is invalid")
		}
		resolvedID, err := resolveReferenceURI(baseReference, idReference)
		if err != nil {
			return newArtifactError(ErrIDAlias, artifact.ref, canonicalPointer(pointer), "$id cannot be resolved: "+err.Error())
		}
		canonicalID, err := resolvedID.canonicalString(false, true)
		if err != nil {
			return newArtifactError(ErrIDAlias, artifact.ref, canonicalPointer(pointer), "$id cannot be canonicalized: "+err.Error())
		}
		c.registerResource(canonicalID, resourceBinding{artifactRef: artifact.ref, pointer: appendPointer(nil, pointer)})
		info = schemaLocationInfo{baseURI: canonicalID, resourceRoot: appendPointer(nil, pointer)}
	}
	artifact.recordSchemaLocation(pointer, info)

	for _, keyword := range jsonSchemaMapKeywords {
		raw, exists := object[keyword]
		if !exists {
			continue
		}
		children, ok := raw.(map[string]any)
		if !ok {
			return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), keyword+" must contain an object of Schemas")
		}
		for _, name := range sortedMapKeys(children) {
			childPointer := appendPointer(pointer, jsonPointer{keyword, name})
			if err := c.indexJSONSchemaValue(artifact, childPointer, info.baseURI, info.resourceRoot, children[name]); err != nil {
				return err
			}
		}
	}
	for _, keyword := range jsonSchemaSingleKeywords {
		raw, exists := object[keyword]
		if !exists {
			continue
		}
		childPointer := appendPointer(pointer, jsonPointer{keyword})
		if err := c.indexJSONSchemaValue(artifact, childPointer, info.baseURI, info.resourceRoot, raw); err != nil {
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
			return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), keyword+" must be an array of Schemas")
		}
		for index, child := range children {
			childPointer := appendPointer(pointer, jsonPointer{keyword, arrayToken(index)})
			if err := c.indexJSONSchemaValue(artifact, childPointer, info.baseURI, info.resourceRoot, child); err != nil {
				return err
			}
		}
	}
	return nil
}

func arrayToken(index int) string {
	// strconv is avoided here only to keep this helper minimal; indices are small.
	if index == 0 {
		return "0"
	}
	digits := make([]byte, 0, 4)
	for index > 0 {
		digits = append([]byte{byte('0' + index%10)}, digits...)
		index /= 10
	}
	return string(digits)
}

// indexOpenAPIArtifact indexes every Schema Object location of an OpenAPI 3.0
// artifact so that resolved $ref targets can be proven to be Schema Objects.
// OpenAPI 3.0 Schema Objects never establish a base URI through $id; a reachable
// $id is rejected during closure traversal.
func (c *ResolutionContext) indexOpenAPIArtifact(artifact *resolvedArtifact) error {
	root, ok := artifact.document.(map[string]any)
	if !ok {
		return newArtifactError(ErrSchemaDocument, artifact.ref, "#", "OpenAPI artifact root must be an object")
	}
	if components, ok, err := optionalObjectField(root, "components"); err != nil {
		return newArtifactError(ErrSchemaDocument, artifact.ref, "#/components", err.Error())
	} else if ok {
		if schemas, ok, err := optionalObjectField(components, "schemas"); err != nil {
			return newArtifactError(ErrSchemaDocument, artifact.ref, "#/components/schemas", err.Error())
		} else if ok {
			for _, name := range sortedMapKeys(schemas) {
				if err := c.indexOpenAPISchema(artifact, jsonPointer{"components", "schemas", name}, schemas[name]); err != nil {
					return err
				}
			}
		}
		if parameters, ok, err := optionalObjectField(components, "parameters"); err != nil {
			return newArtifactError(ErrSchemaDocument, artifact.ref, "#/components/parameters", err.Error())
		} else if ok {
			for _, name := range sortedMapKeys(parameters) {
				if err := c.walkOpenAPIParameter(artifact, jsonPointer{"components", "parameters", name}, parameters[name]); err != nil {
					return err
				}
			}
		}
		if headers, ok, err := optionalObjectField(components, "headers"); err != nil {
			return newArtifactError(ErrSchemaDocument, artifact.ref, "#/components/headers", err.Error())
		} else if ok {
			for _, name := range sortedMapKeys(headers) {
				if err := c.walkOpenAPIHeader(artifact, jsonPointer{"components", "headers", name}, headers[name]); err != nil {
					return err
				}
			}
		}
		if requestBodies, ok, err := optionalObjectField(components, "requestBodies"); err != nil {
			return newArtifactError(ErrSchemaDocument, artifact.ref, "#/components/requestBodies", err.Error())
		} else if ok {
			for _, name := range sortedMapKeys(requestBodies) {
				if err := c.walkOpenAPIRequestBody(artifact, jsonPointer{"components", "requestBodies", name}, requestBodies[name]); err != nil {
					return err
				}
			}
		}
		if responses, ok, err := optionalObjectField(components, "responses"); err != nil {
			return newArtifactError(ErrSchemaDocument, artifact.ref, "#/components/responses", err.Error())
		} else if ok {
			for _, name := range sortedMapKeys(responses) {
				if err := c.walkOpenAPIResponse(artifact, jsonPointer{"components", "responses", name}, responses[name]); err != nil {
					return err
				}
			}
		}
		if callbacks, ok, err := optionalObjectField(components, "callbacks"); err != nil {
			return newArtifactError(ErrSchemaDocument, artifact.ref, "#/components/callbacks", err.Error())
		} else if ok {
			for _, name := range sortedMapKeys(callbacks) {
				if err := c.walkOpenAPICallback(artifact, jsonPointer{"components", "callbacks", name}, callbacks[name]); err != nil {
					return err
				}
			}
		}
	}
	if paths, ok, err := optionalObjectField(root, "paths"); err != nil {
		return newArtifactError(ErrSchemaDocument, artifact.ref, "#/paths", err.Error())
	} else if ok {
		for _, path := range sortedMapKeys(paths) {
			if err := c.walkOpenAPIPathItem(artifact, jsonPointer{"paths", path}, paths[path]); err != nil {
				return err
			}
		}
	}
	return nil
}

func optionalObjectField(parent map[string]any, field string) (map[string]any, bool, error) {
	raw, exists := parent[field]
	if !exists {
		return nil, false, nil
	}
	object, ok := raw.(map[string]any)
	if !ok {
		return nil, false, newError(ErrSchemaDocument, field+" must be an object")
	}
	return object, true, nil
}

func optionalArrayField(parent map[string]any, field string) ([]any, bool, error) {
	raw, exists := parent[field]
	if !exists {
		return nil, false, nil
	}
	array, ok := raw.([]any)
	if !ok {
		return nil, false, newError(ErrSchemaDocument, field+" must be an array")
	}
	return array, true, nil
}

func (c *ResolutionContext) indexOpenAPISchema(artifact *resolvedArtifact, pointer jsonPointer, value any) error {
	object, ok := value.(map[string]any)
	if !ok {
		if _, isBoolean := value.(bool); isBoolean {
			return newArtifactError(ErrBooleanSchema, artifact.ref, canonicalPointer(pointer), "OpenAPI 3.0 Schema Object must be an object")
		}
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), "OpenAPI 3.0 Schema Object must be an object")
	}
	artifact.recordSchemaLocation(pointer, schemaLocationInfo{baseURI: artifact.documentURI})
	if properties, ok, err := optionalObjectField(object, "properties"); err != nil {
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), err.Error())
	} else if ok {
		for _, name := range sortedMapKeys(properties) {
			if err := c.indexOpenAPISchema(artifact, appendPointer(pointer, jsonPointer{"properties", name}), properties[name]); err != nil {
				return err
			}
		}
	}
	if items, exists := object["items"]; exists {
		if _, isArray := items.([]any); isArray {
			return newArtifactError(ErrUnsupportedValue, artifact.ref, canonicalPointer(pointer),
				"OpenAPI 3.0 tuple-array items is invalid")
		}
		if err := c.indexOpenAPISchema(artifact, appendPointer(pointer, jsonPointer{"items"}), items); err != nil {
			return err
		}
	}
	if additional, exists := object["additionalProperties"]; exists {
		if _, isBoolean := additional.(bool); !isBoolean {
			if err := c.indexOpenAPISchema(artifact, appendPointer(pointer, jsonPointer{"additionalProperties"}), additional); err != nil {
				return err
			}
		}
	}
	if notValue, exists := object["not"]; exists {
		if err := c.indexOpenAPISchema(artifact, appendPointer(pointer, jsonPointer{"not"}), notValue); err != nil {
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
			return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), keyword+" must be an array of Schemas")
		}
		for index, child := range children {
			if err := c.indexOpenAPISchema(artifact, appendPointer(pointer, jsonPointer{keyword, arrayToken(index)}), child); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *ResolutionContext) walkOpenAPIPathItem(artifact *resolvedArtifact, pointer jsonPointer, value any) error {
	object, ok := value.(map[string]any)
	if !ok {
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), "Path Item must be an object")
	}
	if parameters, ok, err := optionalArrayField(object, "parameters"); err != nil {
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), err.Error())
	} else if ok {
		for index, parameter := range parameters {
			if err := c.walkOpenAPIParameter(artifact, appendPointer(pointer, jsonPointer{"parameters", arrayToken(index)}), parameter); err != nil {
				return err
			}
		}
	}
	for _, method := range openAPIMethods {
		raw, exists := object[method]
		if !exists {
			continue
		}
		if err := c.walkOpenAPIOperation(artifact, appendPointer(pointer, jsonPointer{method}), raw); err != nil {
			return err
		}
	}
	return nil
}

func (c *ResolutionContext) walkOpenAPIOperation(artifact *resolvedArtifact, pointer jsonPointer, value any) error {
	object, ok := value.(map[string]any)
	if !ok {
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), "Operation must be an object")
	}
	if parameters, ok, err := optionalArrayField(object, "parameters"); err != nil {
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), err.Error())
	} else if ok {
		for index, parameter := range parameters {
			if err := c.walkOpenAPIParameter(artifact, appendPointer(pointer, jsonPointer{"parameters", arrayToken(index)}), parameter); err != nil {
				return err
			}
		}
	}
	if requestBody, exists := object["requestBody"]; exists {
		if err := c.walkOpenAPIRequestBody(artifact, appendPointer(pointer, jsonPointer{"requestBody"}), requestBody); err != nil {
			return err
		}
	}
	if responses, ok, err := optionalObjectField(object, "responses"); err != nil {
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), err.Error())
	} else if ok {
		for _, status := range sortedMapKeys(responses) {
			if err := c.walkOpenAPIResponse(artifact, appendPointer(pointer, jsonPointer{"responses", status}), responses[status]); err != nil {
				return err
			}
		}
	}
	if callbacks, ok, err := optionalObjectField(object, "callbacks"); err != nil {
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), err.Error())
	} else if ok {
		for _, name := range sortedMapKeys(callbacks) {
			if err := c.walkOpenAPICallback(artifact, appendPointer(pointer, jsonPointer{"callbacks", name}), callbacks[name]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *ResolutionContext) walkOpenAPIParameter(artifact *resolvedArtifact, pointer jsonPointer, value any) error {
	object, ok := value.(map[string]any)
	if !ok {
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), "Parameter must be an object")
	}
	if schema, exists := object["schema"]; exists {
		if err := c.indexOpenAPISchema(artifact, appendPointer(pointer, jsonPointer{"schema"}), schema); err != nil {
			return err
		}
	}
	if content, exists := object["content"]; exists {
		if err := c.walkOpenAPIContent(artifact, appendPointer(pointer, jsonPointer{"content"}), content); err != nil {
			return err
		}
	}
	return nil
}

func (c *ResolutionContext) walkOpenAPIHeader(artifact *resolvedArtifact, pointer jsonPointer, value any) error {
	object, ok := value.(map[string]any)
	if !ok {
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), "Header must be an object")
	}
	if schema, exists := object["schema"]; exists {
		if err := c.indexOpenAPISchema(artifact, appendPointer(pointer, jsonPointer{"schema"}), schema); err != nil {
			return err
		}
	}
	if content, exists := object["content"]; exists {
		if err := c.walkOpenAPIContent(artifact, appendPointer(pointer, jsonPointer{"content"}), content); err != nil {
			return err
		}
	}
	return nil
}

func (c *ResolutionContext) walkOpenAPIRequestBody(artifact *resolvedArtifact, pointer jsonPointer, value any) error {
	object, ok := value.(map[string]any)
	if !ok {
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), "Request Body must be an object")
	}
	if content, exists := object["content"]; exists {
		return c.walkOpenAPIContent(artifact, appendPointer(pointer, jsonPointer{"content"}), content)
	}
	return nil
}

func (c *ResolutionContext) walkOpenAPIResponse(artifact *resolvedArtifact, pointer jsonPointer, value any) error {
	object, ok := value.(map[string]any)
	if !ok {
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), "Response must be an object")
	}
	if content, exists := object["content"]; exists {
		if err := c.walkOpenAPIContent(artifact, appendPointer(pointer, jsonPointer{"content"}), content); err != nil {
			return err
		}
	}
	if headers, ok, err := optionalObjectField(object, "headers"); err != nil {
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), err.Error())
	} else if ok {
		for _, name := range sortedMapKeys(headers) {
			if err := c.walkOpenAPIHeader(artifact, appendPointer(pointer, jsonPointer{"headers", name}), headers[name]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *ResolutionContext) walkOpenAPIContent(artifact *resolvedArtifact, pointer jsonPointer, value any) error {
	object, ok := value.(map[string]any)
	if !ok {
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), "Content must be an object")
	}
	for _, mediaType := range sortedMapKeys(object) {
		media, ok := object[mediaType].(map[string]any)
		if !ok {
			return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), "Media Type Object must be an object")
		}
		if schema, exists := media["schema"]; exists {
			if err := c.indexOpenAPISchema(artifact, appendPointer(pointer, jsonPointer{mediaType, "schema"}), schema); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *ResolutionContext) walkOpenAPICallback(artifact *resolvedArtifact, pointer jsonPointer, value any) error {
	object, ok := value.(map[string]any)
	if !ok {
		return newArtifactError(ErrSchemaDocument, artifact.ref, canonicalPointer(pointer), "Callback must be an object")
	}
	for _, expression := range sortedMapKeys(object) {
		if err := c.walkOpenAPIPathItem(artifact, appendPointer(pointer, jsonPointer{expression}), object[expression]); err != nil {
			return err
		}
	}
	return nil
}
