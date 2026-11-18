package compiler

// WP6 slice 6: F.5 value typing for positional PredicateCall arguments against
// the owning Predicate's declared ordered parameters.

import (
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validatePredicateArgumentTypes(req BuildRequest, skill *SkillAST, identity dependencyIdentityResult, ownerLinking ownerLinkingResult, procedurePath string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if skill == nil || skill.Procedure == nil || skill.Predicates == nil || skill.Contract == nil {
		return diagnostics
	}
	registry := buildSchemaRegistry(req, skill, identity)
	contextRoot, requestRoot := sourcePathRootSchemas(skill, registry)
	producers := producerOutputSchemas(skill, ownerLinking, registry)
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil {
			continue
		}
		for _, stage := range []*PredicateStageAST{node.Precondition, node.Postcondition} {
			if stage == nil {
				continue
			}
			for _, call := range stage.Predicates {
				if call == nil {
					continue
				}
				definition := skill.Predicates.Definitions[scalar(call.Predicate)]
				if definition == nil {
					continue
				}
				parameters := sequence(definition.Parameters)
				arguments := sequence(call.Arguments)
				for index, argument := range arguments {
					if index >= len(parameters) {
						break
					}
					value := unalias(argument)
					if value == nil || value.Kind != yaml.ScalarNode || value.Tag != "!!str" ||
						value.Style == yaml.SingleQuotedStyle || value.Style == yaml.DoubleQuotedStyle {
						continue
					}
					targetSchema := predicateParameterSchema(parameters[index], registry)
					if targetSchema == nil {
						continue
					}
					sourceSchema := resolveSourcePathSchema(value.Value, contextRoot, requestRoot, producers, registry)
					if sourceSchema == nil {
						continue
					}
					reason, ok := f5Assignable(sourceSchema, targetSchema)
					if ok {
						continue
					}
					code := "PREDICATE_TYPE_MISMATCH"
					if reason == assignableNullabilityMismatch {
						code = "INPUT_NULLABILITY_MISMATCH"
					}
					if reason == assignableUnresolved {
						continue
					}
					addSourcePathDiagnostic(&diagnostics, "error", "C_STATIC", code, procedurePath, value,
						"Predicate argument value is not assignable to the declared parameter type")
				}
			}
		}
	}
	sortDiagnostics(diagnostics)
	return diagnostics
}

// predicateParameterSchema maps one `{name: type}` parameter declaration to a
// G.4-shaped target Schema node.
func predicateParameterSchema(parameter *yaml.Node, registry schemaRegistry) *yaml.Node {
	parameter = unalias(parameter)
	if parameter == nil || parameter.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(parameter.Content); i += 2 {
		declared := scalar(unalias(parameter.Content[i+1]))
		switch declared {
		case "string", "integer", "number", "boolean":
			return syntheticPrimitiveSchema(declared)
		default:
			if !strings.Contains(declared, "::") {
				return nil
			}
			resource, problem := registry.resolveReference(declared)
			if problem != "" {
				return nil
			}
			effective, problem := registry.resolveSchemaRoot(resource)
			if problem != "" {
				return nil
			}
			return unalias(effective.node)
		}
	}
	return nil
}

func syntheticPrimitiveSchema(declared string) *yaml.Node {
	return &yaml.Node{
		Kind: yaml.MappingNode, Tag: "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "type"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: declared},
		},
	}
}
