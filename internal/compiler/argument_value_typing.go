package compiler

// WP6 slice 5: F.5 value typing from a SourcePath's source Schema to the linked
// Owner Tool Input/QueryContext argument Schema. Predicate parameter typing and
// identity-input provenance are later WP6 slices.

import (
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validateV1Alpha4ArgumentValueTypes(req BuildRequest, skill *V1Alpha4SkillAST, identity dependencyIdentityResult, ownerLinking ownerLinkingResult, procedurePath string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if skill == nil || skill.Procedure == nil || skill.Contract == nil {
		return diagnostics
	}
	registry := buildV1Alpha4SchemaRegistry(req, skill, identity)
	contextRoot, requestRoot := sourcePathRootSchemas(skill, registry)
	producerSchemas := producerOutputSchemas(skill, ownerLinking, registry)
	linked := map[string]linkedExport{}
	for _, export := range ownerLinking.Exports {
		if export.ExportKind == "operation" || export.ExportKind == "observation" {
			linked[export.DependencyAlias+"::"+export.ExportID] = export
		}
	}
	base := "skills/" + skill.Name + "/"
	_ = base
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil {
			continue
		}
		type stage struct {
			tool      string
			arguments *yaml.Node
			schemaID  string
			alias     string
		}
		stages := make([]stage, 0, 2)
		if node.Operation != nil {
			stages = append(stages, stage{tool: scalar(node.Operation.Tool), arguments: node.Operation.Arguments})
		}
		if node.Observation != nil {
			stages = append(stages, stage{tool: scalar(node.Observation.Tool), arguments: node.Observation.Arguments})
		}
		for _, current := range stages {
			export, ok := linked[current.tool]
			if !ok {
				continue
			}
			schemaID := export.InputSchemaID
			if export.ExportKind == "observation" {
				schemaID = export.QueryContextSchemaID
			}
			if schemaID == "" {
				continue
			}
			target, problem := registry.resolveReference(export.DependencyAlias + "::" + schemaID)
			if problem != "" {
				continue
			}
			arguments := unalias(current.arguments)
			if arguments == nil || arguments.Kind != yaml.MappingNode {
				continue
			}
			for i := 0; i+1 < len(arguments.Content); i += 2 {
				key := scalar(unalias(arguments.Content[i]))
				value := unalias(arguments.Content[i+1])
				if key == "" || value == nil || value.Kind != yaml.ScalarNode || value.Tag != "!!str" ||
					value.Style == yaml.SingleQuotedStyle || value.Style == yaml.DoubleQuotedStyle {
					continue // literals are a later slice; quoted strings are literals
				}
				targetProperty := child(child(unalias(target.node), "properties"), key)
				if targetProperty == nil {
					continue // key diagnostics owned by the argument-key slice
				}
				sourceSchema := resolveSourcePathSchema(value.Value, contextRoot, requestRoot, producerSchemas, registry)
				if sourceSchema == nil {
					continue
				}
				reason, ok := f5Assignable(sourceSchema, unalias(targetProperty))
				if ok {
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
					"SourcePath value is not assignable to the Owner argument Schema: "+key)
			}
		}
	}
	sortDiagnostics(diagnostics)
	return diagnostics
}

func sourcePathRootSchemas(skill *V1Alpha4SkillAST, registry schemaRegistry) (contextRoot, requestRoot *yaml.Node) {
	if skill == nil || skill.Contract == nil {
		return nil, nil
	}
	base := "skills/" + skill.Name + "/"
	if skill.Contract.RequestSchema != nil {
		if resource, ok := registry.local[base+scalar(skill.Contract.RequestSchema)]; ok {
			if effective, problem := registry.resolveSchemaRoot(resource); problem == "" {
				requestRoot = unalias(effective.node)
			}
		}
	}
	if skill.Contract.ContextSchema != nil {
		if resource, problem := registry.resolveReference(scalar(skill.Contract.ContextSchema)); problem == "" {
			if effective, problem := registry.resolveSchemaRoot(resource); problem == "" {
				contextRoot = unalias(effective.node)
			}
		}
	}
	return contextRoot, requestRoot
}

func resolveSourcePathSchema(path string, contextRoot, requestRoot *yaml.Node, producers map[string]producerOutput, registry schemaRegistry) *yaml.Node {
	if strings.HasPrefix(path, "context.request.") {
		return walkSchemaPath(requestRoot, strings.Split(strings.TrimPrefix(path, "context.request."), "."), registry)
	}
	if strings.HasPrefix(path, "context.") {
		return walkSchemaPath(contextRoot, strings.Split(strings.TrimPrefix(path, "context."), "."), registry)
	}
	segments := strings.Split(path, ".")
	if len(segments) < 3 {
		return nil
	}
	output, ok := producers[segments[0]]
	if !ok || output.node != segments[1] || segments[2] != "output" {
		return nil
	}
	return walkSchemaPath(output.schema, segments[3:], registry)
}

func walkSchemaPath(root *yaml.Node, segments []string, registry schemaRegistry) *yaml.Node {
	current := unalias(root)
	for _, segment := range segments {
		if current == nil || current.Kind != yaml.MappingNode {
			return nil
		}
		if ref := child(current, "$ref"); ref != nil {
			resource, problem := registry.resolveReference(scalar(ref))
			if problem != "" {
				return nil
			}
			current = unalias(resource.node)
		}
		property := child(child(current, "properties"), segment)
		if property == nil {
			return nil
		}
		current = unalias(property)
	}
	if current != nil && child(current, "$ref") != nil {
		if resource, problem := registry.resolveReference(scalar(child(current, "$ref"))); problem == "" {
			current = unalias(resource.node)
		}
	}
	return current
}
