package compiler

// WP4C: CA-SL F.5 conservative Source-Subset-Target assignability.
//
// This engine proves compatibility only when the frozen F.5 rules make it
// provable. Anything else is rejected; no coercion, default materialization or
// generic Schema evaluation is used. Callers (WP6 argument typing, WP7 Evidence)
// map the result to INPUT_TYPE_MISMATCH / INPUT_NULLABILITY_MISMATCH.
//
// Both Schemas must already have passed the G.4 allowlist. Unresolved $ref
// targets must be resolved by the caller/registry before calling this engine.

import (
	"strconv"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

type assignabilityReason string

const (
	assignableOK                  assignabilityReason = "OK"
	assignableTypeMismatch        assignabilityReason = "TYPE_MISMATCH"
	assignableNullabilityMismatch assignabilityReason = "NULLABILITY_MISMATCH"
	assignableUnresolved          assignabilityReason = "UNRESOLVED"
)

type schemaShape struct {
	types      map[string]bool
	hasTypes   bool
	nullable   bool
	constValue *yaml.Node
	enumValues []*yaml.Node
	minValue   *float64
	maxValue   *float64
	minLength  *int
	maxLength  *int
	minItems   *int
	maxItems   *int
	items      *yaml.Node
	properties map[string]*yaml.Node
	required   map[string]bool
	additional *bool
	refPresent bool
}

// schemaAssignable reports whether Source -?Target can be proven under F.5.
func schemaAssignable(source, target *yaml.Node) (assignabilityReason, bool) {
	for _, node := range []*yaml.Node{source, target} {
		if scalar(child(unalias(node), "$schema")) != g4DialectURI {
			return assignableUnresolved, false
		}
	}
	return f5Assignable(source, target)
}

func f5Assignable(source, target *yaml.Node) (assignabilityReason, bool) {
	sourceShape, ok := f5Shape(source)
	if !ok {
		return assignableUnresolved, false
	}
	targetShape, ok := f5Shape(target)
	if !ok {
		return assignableUnresolved, false
	}
	if sourceShape.refPresent || targetShape.refPresent {
		return assignableUnresolved, false
	}
	if sourceShape.nullable && !targetShape.nullable {
		return assignableNullabilityMismatch, false
	}
	if !f5TypesAcceptable(sourceShape, targetShape) {
		return assignableTypeMismatch, false
	}
	if targetShape.constValue != nil {
		if !f5SameLiteral(sourceShape.constValue, targetShape.constValue) {
			return assignableTypeMismatch, false
		}
		return assignableOK, true
	}
	if len(targetShape.enumValues) > 0 && !f5SourceValuesInTarget(sourceShape, targetShape.enumValues) {
		return assignableTypeMismatch, false
	}
	if !f5RangesImplied(sourceShape, targetShape) {
		return assignableTypeMismatch, false
	}
	if !f5ObjectAssignable(sourceShape, targetShape) {
		return assignableTypeMismatch, false
	}
	if !f5ArrayAssignable(sourceShape, targetShape) {
		return assignableTypeMismatch, false
	}
	return assignableOK, true
}

func f5Shape(node *yaml.Node) (schemaShape, bool) {
	node = unalias(node)
	shape := schemaShape{types: map[string]bool{}, properties: map[string]*yaml.Node{}, required: map[string]bool{}}
	if node == nil || node.Kind != yaml.MappingNode {
		return shape, false
	}
	if child(node, "$ref") != nil {
		shape.refPresent = true
	}
	types, hasTypes := g4DeclaredTypes(node, &[]g4Issue{})
	shape.types, shape.hasTypes = types, hasTypes
	if hasTypes {
		shape.nullable = types["null"]
	} else {
		// An untyped G.4 Schema accepts null; open values are not assignable to a
		// constrained Target unless the rules below prove otherwise.
		shape.nullable = true
	}
	if value := child(node, "const"); value != nil {
		shape.constValue = value
	}
	shape.enumValues = sequence(child(node, "enum"))
	shape.minValue = f5Number(child(node, "minimum"))
	shape.maxValue = f5Number(child(node, "maximum"))
	shape.minLength = f5Int(child(node, "minLength"))
	shape.maxLength = f5Int(child(node, "maxLength"))
	shape.minItems = f5Int(child(node, "minItems"))
	shape.maxItems = f5Int(child(node, "maxItems"))
	if items := child(node, "items"); items != nil {
		shape.items = items
	}
	if properties := child(node, "properties"); properties != nil && properties.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(properties.Content); i += 2 {
			shape.properties[scalar(properties.Content[i])] = unalias(properties.Content[i+1])
		}
	}
	for _, name := range sequence(child(node, "required")) {
		shape.required[scalar(name)] = true
	}
	if additional := child(node, "additionalProperties"); additional != nil && additional.Tag == "!!bool" {
		value := additional.Value == "true"
		shape.additional = &value
	}
	return shape, true
}

func f5TypesAcceptable(source, target schemaShape) bool {
	if !target.hasTypes {
		// A Target without a type keyword is untyped unless it constrains objects;
		// treat it as untyped per G.4 and accept any Source type except nullability.
		return true
	}
	if !source.hasTypes {
		// An untyped Source may be any type; only an untyped/any Target is provable.
		return !target.hasTypes
	}
	for sourceType := range source.types {
		if sourceType == "null" {
			continue
		}
		if target.types[sourceType] {
			continue
		}
		if sourceType == "integer" && target.types["number"] {
			continue
		}
		return false
	}
	if source.types["null"] && !target.types["null"] {
		return false
	}
	return true
}

func f5SameLiteral(a, b *yaml.Node) bool {
	a, b = unalias(a), unalias(b)
	if a == nil || b == nil || a.Kind != b.Kind || a.Tag != b.Tag {
		return false
	}
	return a.Value == b.Value
}

func f5SourceValuesInTarget(source schemaShape, targetEnum []*yaml.Node) bool {
	if source.constValue != nil {
		for _, allowed := range targetEnum {
			if f5SameLiteral(source.constValue, allowed) {
				return true
			}
		}
		return false
	}
	if len(source.enumValues) == 0 {
		// Without enum/const values the Source may contain any value of its type.
		return false
	}
	for _, sourceValue := range source.enumValues {
		found := false
		for _, allowed := range targetEnum {
			if f5SameLiteral(sourceValue, allowed) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func f5RangesImplied(source, target schemaShape) bool {
	if target.minValue != nil && (source.minValue == nil || *source.minValue < *target.minValue) {
		return false
	}
	if target.maxValue != nil && (source.maxValue == nil || *source.maxValue > *target.maxValue) {
		return false
	}
	if target.minLength != nil && (source.minLength == nil || *source.minLength < *target.minLength) {
		return false
	}
	if target.maxLength != nil && (source.maxLength == nil || *source.maxLength > *target.maxLength) {
		return false
	}
	return true
}

func f5ObjectAssignable(source, target schemaShape) bool {
	if len(target.properties) == 0 && len(target.required) == 0 && target.additional == nil {
		return true
	}
	for name := range target.required {
		if !source.required[name] {
			return false
		}
	}
	for name, sourceProperty := range source.properties {
		targetProperty, declared := target.properties[name]
		if !declared {
			if target.additional != nil && !*target.additional {
				return false
			}
			continue
		}
		if reason, ok := f5Assignable(sourceProperty, targetProperty); !ok || reason != assignableOK {
			return false
		}
	}
	return true
}

func f5ArrayAssignable(source, target schemaShape) bool {
	if target.minItems == nil && target.maxItems == nil && target.items == nil {
		return true
	}
	if target.minItems != nil && (source.minItems == nil || *source.minItems < *target.minItems) {
		return false
	}
	if target.maxItems != nil && (source.maxItems == nil || *source.maxItems > *target.maxItems) {
		return false
	}
	if target.items != nil {
		if source.items == nil {
			return false
		}
		if reason, ok := f5Assignable(source.items, target.items); !ok || reason != assignableOK {
			return false
		}
	}
	return true
}

func f5Number(node *yaml.Node) *float64 {
	node = unalias(node)
	if node == nil || (node.Tag != "!!int" && node.Tag != "!!float") {
		return nil
	}
	value, err := strconv.ParseFloat(node.Value, 64)
	if err != nil {
		return nil
	}
	return &value
}

func f5Int(node *yaml.Node) *int {
	node = unalias(node)
	if node == nil || node.Tag != "!!int" {
		return nil
	}
	value, err := strconv.Atoi(strings.TrimSpace(node.Value))
	if err != nil {
		return nil
	}
	return &value
}
