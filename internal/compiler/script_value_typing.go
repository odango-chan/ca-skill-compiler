package compiler

// WP9 slice 3: F.5 value typing of Procedure Script call-site arguments against
// the Script input TypedDict Schema derived from Python annotations. Only
// SourcePath expressions are typed here; literal values are evaluated by their
// own annotation-driven rules in the argument slice.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validateV1Alpha4ScriptValueTypes(req BuildRequest, skill *V1Alpha4SkillAST, identity dependencyIdentityResult, ownerLinking ownerLinkingResult, scriptInputSchemas map[string]map[string]any, procedurePath string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if skill == nil || skill.Procedure == nil || skill.Contract == nil || len(scriptInputSchemas) == 0 {
		return diagnostics
	}
	registry := buildV1Alpha4SchemaRegistry(req, skill, identity)
	contextRoot, requestRoot := sourcePathRootSchemas(skill, registry)
	producerSchemas := producerOutputSchemas(skill, ownerLinking, registry)
	base := "skills/" + skill.Name + "/"
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil || node.Operation == nil || node.Operation.Script == nil {
			continue
		}
		script := scalar(unalias(node.Operation.Script))
		inputSchema, ok := scriptInputSchemas[base+script]
		if !ok {
			continue
		}
		target := schemaNode(inputSchema)
		if target == nil {
			continue
		}
		arguments := unalias(node.Operation.Arguments)
		if arguments == nil || arguments.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(arguments.Content); i += 2 {
			key := scalar(unalias(arguments.Content[i]))
			value := unalias(arguments.Content[i+1])
			if key == "" || value == nil || value.Kind != yaml.ScalarNode || value.Tag != "!!str" ||
				value.Style == yaml.SingleQuotedStyle || value.Style == yaml.DoubleQuotedStyle {
				continue // key diagnostics and literals are handled elsewhere
			}
			targetProperty := child(child(unalias(target), "properties"), key)
			if targetProperty == nil {
				continue
			}
			sourceSchema := resolveSourcePathSchema(value.Value, contextRoot, requestRoot, producerSchemas, registry)
			if sourceSchema == nil {
				continue
			}
			reason, assignable := f5Assignable(sourceSchema, unalias(targetProperty))
			if assignable {
				continue
			}
			code := "INPUT_TYPE_MISMATCH"
			if reason == assignableNullabilityMismatch {
				code = "INPUT_NULLABILITY_MISMATCH"
			}
			if reason == assignableUnresolved {
				continue
			}
			addSourcePathDiagnostic(&diagnostics, "error", "C_STATIC", code, procedurePath, value,
				"SourcePath value is not assignable to the Procedure Script input property: "+key)
		}
	}
	sortDiagnostics(diagnostics)
	return diagnostics
}
