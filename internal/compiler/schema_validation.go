package compiler

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

func validateSchemaShape(schema *yaml.Node) error {
	schema = unalias(schema)
	if schema == nil || schema.Kind != yaml.MappingNode {
		return fmt.Errorf("Schema must be an object")
	}
	if err := validateSchemaObjectKeys(schema); err != nil {
		return err
	}
	known := map[string]bool{
		"$schema": true, "$id": true, "$ref": true, "$defs": true, "definitions": true,
		"title": true, "description": true, "default": true, "examples": true,
		"type": true, "properties": true, "patternProperties": true, "required": true,
		"additionalProperties": true, "unevaluatedProperties": true, "items": true,
		"prefixItems": true, "unevaluatedItems": true, "contains": true, "minItems": true,
		"maxItems": true, "uniqueItems": true, "minLength": true, "maxLength": true,
		"minProperties": true, "maxProperties": true, "minContains": true, "maxContains": true,
		"pattern": true, "format": true, "minimum": true, "maximum": true,
		"exclusiveMinimum": true, "exclusiveMaximum": true, "multipleOf": true,
		"const": true, "enum": true, "allOf": true, "anyOf": true, "oneOf": true,
		"not": true, "if": true, "then": true, "else": true, "dependentRequired": true,
		"dependentSchemas": true, "propertyNames": true, "readOnly": true, "writeOnly": true,
		"deprecated": true, "contentEncoding": true, "contentMediaType": true,
		"contentSchema": true,
	}
	for _, key := range mapKeys(schema) {
		if !known[key] && !strings.HasPrefix(key, "x-") {
			return fmt.Errorf("unsupported Schema keyword %q", key)
		}
	}
	if ref := child(schema, "$ref"); ref != nil {
		return fmt.Errorf("unresolved $ref remains after project-local reference linking")
	}
	if typ := child(schema, "type"); typ != nil {
		allowed := map[string]bool{"null": true, "boolean": true, "object": true, "array": true, "number": true, "integer": true, "string": true}
		if unalias(typ).Kind == yaml.ScalarNode {
			if unalias(typ).Tag != "!!str" || !allowed[scalar(typ)] {
				return fmt.Errorf("unsupported type %q", scalar(typ))
			}
		} else if unalias(typ).Kind == yaml.SequenceNode {
			if len(sequence(typ)) == 0 {
				return fmt.Errorf("type array must not be empty")
			}
			seen := map[string]bool{}
			for _, item := range sequence(typ) {
				name := scalar(item)
				if unalias(item).Kind != yaml.ScalarNode || unalias(item).Tag != "!!str" || !allowed[name] || seen[name] {
					return fmt.Errorf("invalid or duplicate type %q", name)
				}
				seen[name] = true
			}
		} else {
			return fmt.Errorf("type must be a string or string array")
		}
	}
	for _, key := range []string{"properties", "patternProperties", "$defs", "definitions", "dependentSchemas"} {
		mapping := child(schema, key)
		if mapping == nil {
			continue
		}
		if unalias(mapping).Kind != yaml.MappingNode {
			return fmt.Errorf("%s must be an object", key)
		}
		if err := validateSchemaObjectKeys(mapping); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		for _, name := range mapKeys(mapping) {
			if key == "patternProperties" {
				if _, err := regexp.Compile(name); err != nil {
					return fmt.Errorf("invalid patternProperties pattern %q: %w", name, err)
				}
			}
			if err := validateSubschema(child(mapping, name)); err != nil {
				return fmt.Errorf("%s.%s: %w", key, name, err)
			}
		}
	}
	if items := child(schema, "items"); items != nil {
		if unalias(items).Kind == yaml.SequenceNode {
			return fmt.Errorf("tuple-valued items is unsupported; use prefixItems")
		}
		if err := validateSubschema(items); err != nil {
			return fmt.Errorf("items: %w", err)
		}
	}
	for _, key := range []string{"additionalProperties", "unevaluatedProperties", "unevaluatedItems", "contains", "not", "if", "then", "else", "propertyNames", "contentSchema"} {
		value := child(schema, key)
		if value == nil {
			continue
		}
		if key == "additionalProperties" || key == "unevaluatedProperties" {
			if unalias(value).Kind == yaml.ScalarNode && unalias(value).Tag == "!!bool" {
				continue
			}
		}
		if err := validateSubschema(value); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	for _, key := range []string{"allOf", "anyOf", "oneOf", "prefixItems"} {
		value := child(schema, key)
		if value == nil {
			continue
		}
		if unalias(value).Kind != yaml.SequenceNode || len(sequence(value)) == 0 {
			return fmt.Errorf("%s must be a nonempty Schema array", key)
		}
		for _, item := range sequence(value) {
			if err := validateSubschema(item); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
		}
	}
	if required := child(schema, "required"); required != nil {
		if unalias(required).Kind != yaml.SequenceNode {
			return fmt.Errorf("required must be a string array")
		}
		seen := map[string]bool{}
		for _, item := range sequence(required) {
			name := scalar(item)
			if unalias(item).Kind != yaml.ScalarNode || unalias(item).Tag != "!!str" || seen[name] {
				return fmt.Errorf("required entries must be unique strings")
			}
			seen[name] = true
		}
	}
	for _, key := range []string{"minLength", "maxLength", "minItems", "maxItems", "minProperties", "maxProperties", "minContains", "maxContains", "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf"} {
		if value := child(schema, key); value != nil {
			n, err := schemaNumericValue(value)
			if err != nil || (strings.HasPrefix(key, "min") || strings.HasPrefix(key, "max")) && n.Sign() < 0 || key == "multipleOf" && n.Sign() <= 0 {
				return fmt.Errorf("invalid numeric constraint %s", key)
			}
			if schemaCountConstraint(key) && !n.IsInt() {
				return fmt.Errorf("%s must be an integer", key)
			}
		}
	}
	for _, pair := range [][2]string{{"minLength", "maxLength"}, {"minItems", "maxItems"}, {"minProperties", "maxProperties"}, {"minContains", "maxContains"}} {
		if min, max := schemaNumber(schema, pair[0]), schemaNumber(schema, pair[1]); min != nil && max != nil && min.Cmp(max) > 0 {
			return fmt.Errorf("%s exceeds %s", pair[0], pair[1])
		}
	}
	for _, key := range []string{"pattern", "format"} {
		if value := child(schema, key); value != nil && (unalias(value).Kind != yaml.ScalarNode || unalias(value).Tag != "!!str") {
			return fmt.Errorf("%s must be a string", key)
		}
	}
	for _, key := range []string{"$schema", "$id", "title", "description", "contentEncoding", "contentMediaType"} {
		if value := child(schema, key); value != nil && (unalias(value).Kind != yaml.ScalarNode || unalias(value).Tag != "!!str") {
			return fmt.Errorf("%s must be a string", key)
		}
	}
	for _, key := range []string{"readOnly", "writeOnly", "deprecated"} {
		if value := child(schema, key); value != nil && (unalias(value).Kind != yaml.ScalarNode || unalias(value).Tag != "!!bool") {
			return fmt.Errorf("%s must be boolean", key)
		}
	}
	if dependent := child(schema, "dependentRequired"); dependent != nil {
		if unalias(dependent).Kind != yaml.MappingNode {
			return fmt.Errorf("dependentRequired must be an object")
		}
		if err := validateSchemaObjectKeys(dependent); err != nil {
			return fmt.Errorf("dependentRequired: %w", err)
		}
		for _, name := range mapKeys(dependent) {
			list := child(dependent, name)
			if unalias(list).Kind != yaml.SequenceNode {
				return fmt.Errorf("dependentRequired.%s must be a string array", name)
			}
			seen := map[string]bool{}
			for _, item := range sequence(list) {
				name := scalar(item)
				if unalias(item).Kind != yaml.ScalarNode || unalias(item).Tag != "!!str" || seen[name] {
					return fmt.Errorf("dependentRequired.%s entries must be strings", name)
				}
				seen[name] = true
			}
		}
	}
	if pattern := scalar(child(schema, "pattern")); pattern != "" {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("invalid pattern: %w", err)
		}
	}
	for _, key := range []string{"enum", "examples"} {
		value := child(schema, key)
		if value != nil && unalias(value).Kind != yaml.SequenceNode {
			return fmt.Errorf("%s must be an array", key)
		}
	}
	if value := child(schema, "default"); value != nil && !validJSONLiteral(value) {
		return fmt.Errorf("default must be a JSON value")
	}
	if examples := child(schema, "examples"); examples != nil {
		for _, value := range sequence(examples) {
			if !validJSONLiteral(value) {
				return fmt.Errorf("examples entries must be JSON values")
			}
		}
	}
	if enum := child(schema, "enum"); enum != nil && len(sequence(enum)) == 0 {
		return fmt.Errorf("enum must not be empty")
	}
	if constant := child(schema, "const"); constant != nil && !validJSONLiteral(constant) {
		return fmt.Errorf("const must be a JSON literal")
	}
	if enum := child(schema, "enum"); enum != nil {
		seen := map[string]bool{}
		for _, value := range sequence(enum) {
			if !validJSONLiteral(value) {
				return fmt.Errorf("enum values must be JSON literals")
			}
			encoded, err := json.Marshal(nodeValue(value))
			if err != nil {
				return fmt.Errorf("enum values must be JSON literals")
			}
			if seen[string(encoded)] {
				return fmt.Errorf("enum values must be unique")
			}
			seen[string(encoded)] = true
		}
	}
	if unique := child(schema, "uniqueItems"); unique != nil && (unalias(unique).Kind != yaml.ScalarNode || unalias(unique).Tag != "!!bool") {
		return fmt.Errorf("uniqueItems must be boolean")
	}
	return nil
}

func validateSchemaObjectKeys(object *yaml.Node) error {
	object = unalias(object)
	if object == nil || object.Kind != yaml.MappingNode {
		return fmt.Errorf("must be an object")
	}
	for i := 0; i+1 < len(object.Content); i += 2 {
		key := unalias(object.Content[i])
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
			return fmt.Errorf("object member names must be strings")
		}
	}
	return nil
}

func schemaNumericValue(value *yaml.Node) (*big.Rat, error) {
	value = unalias(value)
	if value == nil || value.Kind != yaml.ScalarNode || (value.Tag != "!!int" && value.Tag != "!!float") {
		return nil, fmt.Errorf("not a numeric scalar")
	}
	parsed, ok := exactRational(value.Value)
	if !ok {
		return nil, fmt.Errorf("not a finite decimal number")
	}
	return parsed, nil
}

func schemaCountConstraint(key string) bool {
	switch key {
	case "minLength", "maxLength", "minItems", "maxItems", "minProperties", "maxProperties", "minContains", "maxContains":
		return true
	default:
		return false
	}
}

func validateSubschema(schema *yaml.Node) error {
	schema = unalias(schema)
	if schema != nil && schema.Kind == yaml.ScalarNode && schema.Tag == "!!bool" {
		return nil
	}
	return validateSchemaShape(schema)
}
