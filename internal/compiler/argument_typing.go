package compiler

// WP6 slice 4: Tool/QueryContext argument key checking against the exact linked
// Owner Input / QueryContext Schema. F.5 value typing against SourcePath source
// Schemas is a later WP6 slice.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validateArgumentKeys(req BuildRequest, skill *SkillAST, identity dependencyIdentityResult, ownerLinking ownerLinkingResult, procedurePath string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if skill == nil || skill.Procedure == nil {
		return diagnostics
	}
	registry := buildSchemaRegistry(req, skill, identity, req.development)
	linked := map[string]linkedExport{}
	for _, export := range ownerLinking.Exports {
		if export.ExportKind == "operation" || export.ExportKind == "observation" {
			linked[export.DependencyAlias+"::"+export.ExportID] = export
		}
	}
	check := func(toolRef string, arguments *yaml.Node, schemaID, dependencyAlias string) {
		if schemaID == "" {
			return
		}
		resource, problem := registry.resolveReference(dependencyAlias + "::" + schemaID)
		if problem != "" {
			// Unresolvable target Schemas stay fail-closed without duplicating
			// the Schema-identity diagnostic owned by WP3B/WP4B.
			return
		}
		target := unalias(resource.node)
		supplied := map[string]*yaml.Node{}
		arguments = unalias(arguments)
		if arguments != nil && arguments.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(arguments.Content); i += 2 {
				key := scalar(unalias(arguments.Content[i]))
				supplied[key] = unalias(arguments.Content[i+1])
			}
		}
		for _, required := range sequence(child(target, "required")) {
			name := scalar(required)
			if name == "" {
				continue
			}
			if _, ok := supplied[name]; !ok {
				addSourcePathDiagnostic(&diagnostics, "error", "C_STATIC", "INPUT_REQUIRED_MISSING", procedurePath, unalias(arguments),
					"Required Tool argument is missing: "+name)
			}
		}
		for i := 0; i+1 < len(arguments.Content); i += 2 {
			keyNode := unalias(arguments.Content[i])
			key := scalar(keyNode)
			if key == "" {
				continue
			}
			if child(child(target, "properties"), key) == nil {
				addSourcePathDiagnostic(&diagnostics, "error", "C_STATIC", "INPUT_TYPE_MISMATCH", procedurePath, unalias(arguments.Content[i+1]),
					"Tool argument is not declared by the Owner Input/QueryContext Schema: "+key)
			}
		}
	}
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil {
			continue
		}
		if node.Operation != nil {
			tool := scalar(node.Operation.Tool)
			if export, ok := linked[tool]; ok {
				check(tool, node.Operation.Arguments, export.InputSchemaID, export.DependencyAlias)
			}
		}
		if node.Observation != nil {
			tool := scalar(node.Observation.Tool)
			if export, ok := linked[tool]; ok {
				check(tool, node.Observation.Arguments, export.QueryContextSchemaID, export.DependencyAlias)
			}
		}
	}
	sortDiagnostics(diagnostics)
	return diagnostics
}
