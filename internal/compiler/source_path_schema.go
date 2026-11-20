package compiler

// WP6 slice 2: context/request SourcePath typing and required-value availability.
//
// A context SourcePath is resolved against the Contract contextSchema (Host Owner
// Schema) or, for context.request.*, the Skill-owned requestSchema. Every segment
// must be an explicitly declared property; a segment that is not guaranteed
// required makes the value unavailable for a required argument. Producer output
// paths, dominance and argument typing are later WP6 slices.

import (
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validateSourcePathSchemas(req BuildRequest, skill *SkillAST, identity dependencyIdentityResult, contractPath, procedurePath string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if skill == nil || skill.Contract == nil || skill.Contract.Raw == nil {
		return diagnostics
	}
	registry := buildSchemaRegistry(req, skill, identity, req.development)
	base := "skills/" + skill.Name + "/"

	var requestRoot *yaml.Node
	requestPath := ""
	if contract := skill.Contract; contract.RequestSchema != nil {
		requestPath = base + scalar(contract.RequestSchema)
		if resource, ok := registry.local[requestPath]; ok {
			requestRoot = resource.node
		}
	}
	var contextRoot *yaml.Node
	if contract := skill.Contract; contract.ContextSchema != nil {
		if resource, problem := registry.resolveReference(scalar(contract.ContextSchema)); problem == "" {
			if effective, problem := registry.resolveSchemaRoot(resource); problem == "" {
				contextRoot = effective.node
			}
		}
	}

	for _, value := range collectSourcePathArguments(skill) {
		value = unalias(value)
		if value == nil || value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
			continue
		}
		if value.Style == yaml.SingleQuotedStyle || value.Style == yaml.DoubleQuotedStyle {
			continue
		}
		path := value.Value
		if !strings.HasPrefix(path, "context.") {
			continue // producer output paths are a later slice
		}
		segments := strings.Split(path, ".")
		if len(segments) < 2 {
			continue
		}
		if segments[1] == "request" {
			if requestRoot == nil {
				if skill.Contract.RequestSchema == nil {
					addSourcePathDiagnostic(&diagnostics, "error", "C_STATIC", "REQUEST_SCHEMA_MISSING", contractPath, value,
						"SourcePath reads context.request but the Contract declares no requestSchema")
				}
				continue
			}
			walkContextPath(requestRoot, segments[2:], registry, procedurePath, value, &diagnostics)
			continue
		}
		if contextRoot == nil {
			if skill.Contract.ContextSchema == nil {
				addSourcePathDiagnostic(&diagnostics, "error", "C_STATIC", "CONTEXT_SCHEMA_UNDECLARED", contractPath, value,
					"SourcePath reads Host Context but the Contract declares no contextSchema")
			}
			continue
		}
		walkContextPath(contextRoot, segments[1:], registry, procedurePath, value, &diagnostics)
	}
	sortDiagnostics(diagnostics)
	return diagnostics
}

func walkContextPath(root *yaml.Node, segments []string, registry schemaRegistry, procedurePath string, value *yaml.Node, diagnostics *[]Diagnostic) {
	if len(segments) == 0 {
		return
	}
	current := unalias(root)
	optional := false
	for index, segment := range segments {
		if current == nil || current.Kind != yaml.MappingNode {
			addSourcePathDiagnostic(diagnostics, "error", "C_STATIC", "SOURCE_PATH_INVALID", procedurePath, value,
				"SourcePath segment is not an object field: "+segment)
			return
		}
		// Resolve an intermediate $ref root chain before reading properties.
		if ref := child(current, "$ref"); ref != nil {
			resource, problem := registry.resolveReference(scalar(ref))
			if problem != "" {
				return // unresolved Schemas are diagnosed by WP4B
			}
			current = unalias(resource.node)
			continue
		}
		properties := child(current, "properties")
		property := child(properties, segment)
		if property == nil {
			addSourcePathDiagnostic(diagnostics, "error", "C_STATIC", "SOURCE_PATH_INVALID",
				procedurePath, value, "SourcePath segment is not a declared Schema property: "+segment)
			return
		}
		if !requiredProperty(current, segment) {
			optional = true
		}
		current = unalias(property)
		if ref := child(current, "$ref"); ref != nil {
			resource, problem := registry.resolveReference(scalar(ref))
			if problem != "" {
				return
			}
			current = unalias(resource.node)
		}
		if index == len(segments)-1 {
			break
		}
	}
	if optional {
		addSourcePathDiagnostic(diagnostics, "error", "C_STATIC", "SOURCE_NOT_AVAILABLE", procedurePath, value,
			"SourcePath reads a field that is not guaranteed present on every path: "+scalar(value))
	}
}

func requiredProperty(schema *yaml.Node, name string) bool {
	for _, requiredName := range sequence(child(schema, "required")) {
		if scalar(requiredName) == name {
			return true
		}
	}
	return false
}

// collectSourcePathArguments gathers every author-provided argument that
// is a SourcePath position (named Tool/Script arguments and positional
// PredicateCall arguments).
func collectSourcePathArguments(skill *SkillAST) []*yaml.Node {
	values := make([]*yaml.Node, 0)
	if skill == nil || skill.Procedure == nil {
		return values
	}
	appendArguments := func(arguments *yaml.Node) {
		arguments = unalias(arguments)
		if arguments == nil || arguments.Kind != yaml.MappingNode {
			return
		}
		for i := 1; i < len(arguments.Content); i += 2 {
			values = append(values, unalias(arguments.Content[i]))
		}
	}
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil {
			continue
		}
		if node.Operation != nil {
			appendArguments(node.Operation.Arguments)
		}
		if node.Observation != nil {
			appendArguments(node.Observation.Arguments)
		}
		for _, stage := range []*PredicateStageAST{node.Precondition, node.Postcondition} {
			if stage == nil {
				continue
			}
			for _, call := range stage.Predicates {
				if call != nil {
					values = append(values, sequence(call.Arguments)...)
				}
			}
		}
	}
	return values
}

func addSourcePathDiagnostic(diagnostics *[]Diagnostic, severity, phase, code, sourcePath string, node *yaml.Node, message string) {
	line, column := 0, 0
	if node != nil {
		line, column = node.Line, node.Column
	}
	*diagnostics = append(*diagnostics, Diagnostic{
		Severity: severity, Phase: phase, Code: code,
		Path: limitDiagnosticPath(sourcePath), Line: line, Column: column,
		Message: limitDiagnosticMessage(message),
	})
}
