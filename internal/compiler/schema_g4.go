package compiler

// WP4A: CA-SL G.4 JSON Schema dialect allowlist and structural consistency.
//
// The selected San v6.0.3 engine is a generic Draft 2020-12 engine; it does not
// define the CA-SL accepted keyword set. This file enforces the CA-SL-owned G.4
// allowlist before any generic engine may see a Schema. Reference closure and
// F.5 assignability are separate WP4 slices.

import (
	"strconv"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

const g4DialectURI = "https://json-schema.org/draft/2020-12/schema"

var g4Types = map[string]bool{
	"string": true, "integer": true, "number": true, "boolean": true,
	"object": true, "array": true, "null": true,
}

var g4Annotations = map[string]bool{
	"$schema": true, "$id": true, "title": true, "description": true,
	"default": true, "examples": true,
}

type g4Issue struct {
	node    *yaml.Node
	message string
}

// validateG4Schema validates one participating CA-SL data-Schema root. It rejects
// every keyword outside the frozen G.4 allowlist and structurally inconsistent
// combinations. It never resolves or evaluates a Schema.
func validateG4Schema(root *yaml.Node) []g4Issue {
	root = unalias(root)
	if root == nil || root.Kind != yaml.MappingNode {
		return []g4Issue{{node: root, message: "Schema root must be a mapping with an explicit Draft 2020-12 dialect"}}
	}
	issues := make([]g4Issue, 0)
	if dialect := scalar(child(root, "$schema")); dialect != g4DialectURI {
		issues = append(issues, g4Issue{node: root, message: "Schema root must declare $schema: " + g4DialectURI})
	}
	issues = append(issues, validateG4Subschema(root)...)
	return issues
}

func validateG4Subschema(node *yaml.Node) []g4Issue {
	node = unalias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return []g4Issue{{node: node, message: "Schema value must be a mapping"}}
	}
	issues := make([]g4Issue, 0)
	propertyNames := map[string]bool{}
	declaredTypes, hasTypes := g4DeclaredTypes(node, &issues)
	hasRef := false
	minItems, maxItems := (*int)(nil), (*int)(nil)
	minLength, maxLength := (*int)(nil), (*int)(nil)
	minValue, maxValue := (*float64)(nil), (*float64)(nil)
	exclusiveMin, exclusiveMax := (*float64)(nil), (*float64)(nil)
	hasProperties := false
	hasItems := false

	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := unalias(node.Content[i])
		valueNode := unalias(node.Content[i+1])
		key := ""
		if keyNode != nil {
			key = keyNode.Value
		}
		if key == "" || keyNode.Kind != yaml.ScalarNode || keyNode.Tag != "!!str" {
			issues = append(issues, g4Issue{node: keyNode, message: "Schema keyword must be a string"})
			continue
		}
		switch key {
		case "$schema":
			if scalar(valueNode) != g4DialectURI {
				issues = append(issues, g4Issue{node: valueNode, message: "Only the Draft 2020-12 dialect may be declared"})
			}
		case "$id":
			id := scalar(valueNode)
			if id == "" || strings.Contains(id, "#") || !validStablePackageID(id) {
				issues = append(issues, g4Issue{node: valueNode, message: "Schema $id must be an absolute URI without a fragment"})
			}
		case "title", "description":
			if valueNode == nil || valueNode.Kind != yaml.ScalarNode || valueNode.Tag != "!!str" {
				issues = append(issues, g4Issue{node: valueNode, message: key + " must be a string annotation"})
			}
		case "default", "examples":
			// Annotation only; never materialized by the Compiler.
		case "$ref":
			hasRef = true
			ref := scalar(valueNode)
			if !validG4Reference(ref) {
				issues = append(issues, g4Issue{node: valueNode, message: "Schema $ref must be an absolute Schema identity without a fragment"})
			}
		case "type":
			// Declared types were already parsed by g4DeclaredTypes.
		case "properties":
			if valueNode == nil || valueNode.Kind != yaml.MappingNode {
				issues = append(issues, g4Issue{node: valueNode, message: "properties must be a mapping of property names to Schemas"})
				continue
			}
			hasProperties = true
			for j := 0; j+1 < len(valueNode.Content); j += 2 {
				propertyName := unalias(valueNode.Content[j])
				if propertyName == nil || propertyName.Kind != yaml.ScalarNode {
					issues = append(issues, g4Issue{node: propertyName, message: "Property name must be a scalar"})
					continue
				}
				propertyNames[propertyName.Value] = true
				issues = append(issues, validateG4Subschema(valueNode.Content[j+1])...)
			}
		case "required":
			if valueNode == nil || valueNode.Kind != yaml.SequenceNode {
				issues = append(issues, g4Issue{node: valueNode, message: "required must be a sequence of unique property names"})
				continue
			}
			seen := map[string]bool{}
			for _, requiredName := range valueNode.Content {
				name := scalar(requiredName)
				if requiredName == nil || requiredName.Kind != yaml.ScalarNode || name == "" || seen[name] {
					issues = append(issues, g4Issue{node: requiredName, message: "required entries must be unique nonempty strings"})
					continue
				}
				seen[name] = true
			}
		case "additionalProperties":
			if valueNode == nil || valueNode.Kind != yaml.ScalarNode || valueNode.Tag != "!!bool" || (valueNode.Value != "true" && valueNode.Value != "false") {
				issues = append(issues, g4Issue{node: valueNode, message: "additionalProperties must be the boolean true or false"})
			}
		case "items":
			if valueNode == nil || valueNode.Kind != yaml.MappingNode {
				issues = append(issues, g4Issue{node: valueNode, message: "items must be one Schema, not a tuple array"})
				continue
			}
			hasItems = true
			issues = append(issues, validateG4Subschema(valueNode)...)
		case "minItems", "maxItems":
			value, ok := g4NonNegativeInteger(valueNode)
			if !ok {
				issues = append(issues, g4Issue{node: valueNode, message: key + " must be a non-negative integer"})
				continue
			}
			if key == "minItems" {
				minItems = &value
			} else {
				maxItems = &value
			}
		case "minLength", "maxLength":
			value, ok := g4NonNegativeInteger(valueNode)
			if !ok {
				issues = append(issues, g4Issue{node: valueNode, message: key + " must be a non-negative integer"})
				continue
			}
			if key == "minLength" {
				minLength = &value
			} else {
				maxLength = &value
			}
		case "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum":
			value, ok := g4Number(valueNode)
			if !ok {
				issues = append(issues, g4Issue{node: valueNode, message: key + " must be a finite number"})
				continue
			}
			switch key {
			case "minimum":
				minValue = &value
			case "maximum":
				maxValue = &value
			case "exclusiveMinimum":
				exclusiveMin = &value
			case "exclusiveMaximum":
				exclusiveMax = &value
			}
		case "enum":
			if valueNode == nil || valueNode.Kind != yaml.SequenceNode || len(valueNode.Content) == 0 {
				issues = append(issues, g4Issue{node: valueNode, message: "enum must be a nonempty unique value list"})
				continue
			}
			seen := map[string]bool{}
			for _, enumValue := range valueNode.Content {
				key := g4LiteralKey(enumValue)
				if seen[key] {
					issues = append(issues, g4Issue{node: enumValue, message: "enum values must be unique"})
				}
				seen[key] = true
				if !g4LiteralAccepted(enumValue, declaredTypes, hasTypes, minValue, maxValue, exclusiveMin, exclusiveMax, minLength, maxLength) {
					issues = append(issues, g4Issue{node: enumValue, message: "enum value is not accepted by the declared type and constraints"})
				}
			}
		case "const":
			if !g4LiteralAccepted(valueNode, declaredTypes, hasTypes, minValue, maxValue, exclusiveMin, exclusiveMax, minLength, maxLength) {
				issues = append(issues, g4Issue{node: valueNode, message: "const value is not accepted by the declared type and constraints"})
			}
		default:
			if g4Annotations[key] {
				continue
			}
			issues = append(issues, g4Issue{node: keyNode, message: "Unsupported G.4 Schema keyword " + key})
		}
	}

	if hasRef {
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := scalar(unalias(node.Content[i]))
			if key == "$ref" {
				continue
			}
			if !g4Annotations[key] || key == "default" || key == "examples" {
				issues = append(issues, g4Issue{node: unalias(node.Content[i]), message: "$ref must be an independent reference object with annotations only"})
				break
			}
		}
	}
	if minItems != nil && maxItems != nil && *minItems > *maxItems {
		issues = append(issues, g4Issue{node: node, message: "minItems must not exceed maxItems"})
	}
	if minLength != nil && maxLength != nil && *minLength > *maxLength {
		issues = append(issues, g4Issue{node: node, message: "minLength must not exceed maxLength"})
	}
	if minValue != nil && maxValue != nil && *minValue > *maxValue {
		issues = append(issues, g4Issue{node: node, message: "minimum must not exceed maximum"})
	}
	if exclusiveMin != nil && exclusiveMax != nil && *exclusiveMin >= *exclusiveMax {
		issues = append(issues, g4Issue{node: node, message: "exclusiveMinimum must be less than exclusiveMaximum"})
	}
	if hasTypes {
		has := func(t string) bool { return declaredTypes[t] }
		if (minItems != nil || maxItems != nil) && !has("array") {
			issues = append(issues, g4Issue{node: node, message: "array length constraints require type array"})
		}
		if (minLength != nil || maxLength != nil) && !has("string") {
			issues = append(issues, g4Issue{node: node, message: "string length constraints require type string"})
		}
		if (minValue != nil || maxValue != nil || exclusiveMin != nil || exclusiveMax != nil) && !has("number") && !has("integer") {
			issues = append(issues, g4Issue{node: node, message: "numeric range constraints require type integer or number"})
		}
		if (hasProperties || len(propertyNames) > 0) && !has("object") {
			issues = append(issues, g4Issue{node: node, message: "properties require type object"})
		}
		if hasItems && !has("array") {
			issues = append(issues, g4Issue{node: node, message: "items requires type array"})
		}
	}
	if required := unalias(child(node, "required")); required != nil && required.Kind == yaml.SequenceNode {
		for _, requiredName := range required.Content {
			name := scalar(requiredName)
			if name != "" && !propertyNames[name] {
				issues = append(issues, g4Issue{node: requiredName, message: "required name is not a declared property: " + name})
			}
		}
	}
	return issues
}

func g4DeclaredTypes(node *yaml.Node, issues *[]g4Issue) (map[string]bool, bool) {
	value := unalias(child(node, "type"))
	if value == nil {
		return nil, false
	}
	types := map[string]bool{}
	switch value.Kind {
	case yaml.ScalarNode:
		name := value.Value
		if value.Tag != "!!str" || !g4Types[name] {
			*issues = append(*issues, g4Issue{node: value, message: "type must be a supported single type or exactly [T, \"null\"]"})
			return nil, false
		}
		types[name] = true
	case yaml.SequenceNode:
		if len(value.Content) != 2 {
			*issues = append(*issues, g4Issue{node: value, message: "type union is limited to exactly [T, \"null\"]"})
			return nil, false
		}
		nullCount := 0
		for _, item := range value.Content {
			name := scalar(item)
			if item == nil || item.Kind != yaml.ScalarNode || item.Tag != "!!str" || !g4Types[name] || types[name] {
				*issues = append(*issues, g4Issue{node: item, message: "type union is limited to exactly [T, \"null\"]"})
				return nil, false
			}
			if name == "null" {
				nullCount++
			}
			types[name] = true
		}
		if nullCount != 1 {
			*issues = append(*issues, g4Issue{node: value, message: "type union is limited to exactly [T, \"null\"]"})
			return nil, false
		}
	default:
		*issues = append(*issues, g4Issue{node: value, message: "type must be a supported single type or exactly [T, \"null\"]"})
		return nil, false
	}
	return types, true
}

func validG4Reference(ref string) bool {
	if ref == "" || strings.Contains(ref, "#") {
		return false
	}
	if index := strings.Index(ref, "::"); index > 0 {
		alias, uri := ref[:index], ref[index+2:]
		return validDependencyID(alias) && validStablePackageID(uri)
	}
	return validStablePackageID(ref)
}

func g4NonNegativeInteger(node *yaml.Node) (int, bool) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!int" {
		return 0, false
	}
	value, err := strconv.Atoi(node.Value)
	if err != nil || value < 0 {
		return 0, false
	}
	return value, true
}

func g4Number(node *yaml.Node) (float64, bool) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.ScalarNode || (node.Tag != "!!int" && node.Tag != "!!float") {
		return 0, false
	}
	value, err := strconv.ParseFloat(node.Value, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func g4LiteralKey(node *yaml.Node) string {
	node = unalias(node)
	if node == nil {
		return ""
	}
	return node.Tag + "\x00" + node.Value
}

func g4LiteralAccepted(node *yaml.Node, declaredTypes map[string]bool, hasTypes bool, minValue, maxValue, exclusiveMin, exclusiveMax *float64, minLength, maxLength *int) bool {
	node = unalias(node)
	if node == nil {
		return false
	}
	literalType := ""
	switch node.Kind {
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!str":
			literalType = "string"
		case "!!int":
			literalType = "integer"
		case "!!float":
			literalType = "number"
		case "!!bool":
			literalType = "boolean"
		case "!!null":
			literalType = "null"
		default:
			return false
		}
	case yaml.MappingNode:
		literalType = "object"
	case yaml.SequenceNode:
		literalType = "array"
	default:
		return false
	}
	if hasTypes {
		acceptedType := declaredTypes[literalType] || (literalType == "integer" && declaredTypes["number"])
		if !acceptedType {
			return false
		}
	}
	if literalType == "string" {
		length := len([]rune(node.Value))
		if minLength != nil && length < *minLength {
			return false
		}
		if maxLength != nil && length > *maxLength {
			return false
		}
	}
	if literalType == "integer" || literalType == "number" {
		value, err := strconv.ParseFloat(node.Value, 64)
		if err != nil {
			return false
		}
		if minValue != nil && value < *minValue {
			return false
		}
		if maxValue != nil && value > *maxValue {
			return false
		}
		if exclusiveMin != nil && value <= *exclusiveMin {
			return false
		}
		if exclusiveMax != nil && value >= *exclusiveMax {
			return false
		}
	}
	return true
}

// validateV1Alpha4RequestSchemaRoot enforces the section 11.5 / G.6 root shape for a
// Skill-owned Request Schema. Root $ref closure resolution belongs to the WP4
// registry slice and is intentionally not guessed here.
func validateV1Alpha4RequestSchemaRoot(file string, root *yaml.Node, add func(string, *yaml.Node, string, string)) {
	root = unalias(root)
	if root == nil || root.Kind != yaml.MappingNode {
		add(file, root, "SCHEMA_INVALID", "Request Schema root must be a mapping")
		return
	}
	if child(root, "$ref") != nil {
		return
	}
	typeNode := child(root, "type")
	if scalar(typeNode) != "object" {
		add(file, root, "SCHEMA_INVALID", "Request Schema effective root must be an object Schema")
		return
	}
	properties := child(root, "properties")
	if properties == nil || properties.Kind != yaml.MappingNode {
		add(file, root, "SCHEMA_INVALID", "Request Schema root fields must be declared by properties")
		return
	}
	additional := child(root, "additionalProperties")
	if scalar(additional) != "false" || additional.Tag != "!!bool" {
		add(file, root, "SCHEMA_INVALID", "Request Schema root must be closed with additionalProperties: false")
	}
}
