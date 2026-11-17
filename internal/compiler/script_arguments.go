package compiler

// WP9 slice 2: call-site arguments of Procedure Scripts, and the resolved
// §13.2 rule that an Expression argument is a SourcePath or scalar literal, not
// an inline object/array. Maintainer decision: inline object/array Expression
// values are SOURCE_PATH_INVALID / C_STATIC; the outer legal `arguments: {}` and
// zero-arity `arguments: []` containers stay permitted.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

// validateV1Alpha4ScriptArguments checks required/unknown keys of every
// Procedure Script call site against the Script input TypedDict projection.
// E67 maps these call-site failures to INPUT_REQUIRED_MISSING / INPUT_TYPE_MISMATCH.
func validateV1Alpha4ScriptArguments(skill *V1Alpha4SkillAST, schemas map[string]map[string]any, procedurePath string, add func(string, *yaml.Node, string, string)) {
	if skill == nil || skill.Procedure == nil || len(schemas) == 0 {
		return
	}
	base := "skills/" + skill.Name + "/"
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil || node.Operation == nil || node.Operation.Script == nil {
			continue
		}
		script := scalar(unalias(node.Operation.Script))
		if !validLocalScriptPath(script) {
			continue
		}
		inputSchema, ok := schemas[base+script]
		if !ok {
			continue
		}
		arguments := unalias(node.Operation.Arguments)
		supplied := map[string]*yaml.Node{}
		if arguments != nil && arguments.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(arguments.Content); i += 2 {
				supplied[scalar(unalias(arguments.Content[i]))] = unalias(arguments.Content[i+1])
			}
		}
		required, _ := inputSchema["required"].([]string)
		for _, name := range required {
			if _, present := supplied[name]; !present {
				add(procedurePath, unalias(arguments), "INPUT_REQUIRED_MISSING",
					"Required Procedure Script input argument is missing: "+name)
			}
		}
		properties, _ := inputSchema["properties"].(map[string]any)
		for i := 0; i+1 < len(arguments.Content); i += 2 {
			key := scalar(unalias(arguments.Content[i]))
			if key == "" {
				continue
			}
			if _, declared := properties[key]; !declared {
				add(procedurePath, unalias(arguments.Content[i+1]), "INPUT_TYPE_MISMATCH",
					"Procedure Script argument is not declared by the input TypedDict: "+key)
			}
		}
	}
}

// validateV1Alpha4InlineArgumentExpressions rejects inline object/array values
// used in Expression position for Operation/Observation arguments and for
// PredicateCall arguments.
func validateV1Alpha4InlineArgumentExpressions(skill *V1Alpha4SkillAST, procedurePath string, add func(string, *yaml.Node, string, string)) {
	if skill == nil || skill.Procedure == nil {
		return
	}
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil {
			continue
		}
		if node.Operation != nil {
			rejectInlineMapping(procedurePath, unalias(node.Operation.Arguments), add)
		}
		if node.Observation != nil {
			rejectInlineMapping(procedurePath, unalias(node.Observation.Arguments), add)
		}
		for _, stage := range []*V1Alpha4PredicateStageAST{node.Precondition, node.Postcondition} {
			if stage == nil {
				continue
			}
			for _, call := range stage.Predicates {
				if call == nil {
					continue
				}
				arguments := unalias(call.Arguments)
				if arguments == nil || arguments.Kind != yaml.SequenceNode {
					continue
				}
				for _, item := range arguments.Content {
					rejectInlineExpression(procedurePath, unalias(item), add)
				}
			}
		}
	}
}

func rejectInlineMapping(path string, arguments *yaml.Node, add func(string, *yaml.Node, string, string)) {
	if arguments == nil || arguments.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(arguments.Content); i += 2 {
		rejectInlineExpression(path, unalias(arguments.Content[i+1]), add)
	}
}

func rejectInlineExpression(path string, value *yaml.Node, add func(string, *yaml.Node, string, string)) {
	if value == nil {
		return
	}
	if value.Kind == yaml.MappingNode || value.Kind == yaml.SequenceNode {
		add(path, value, "SOURCE_PATH_INVALID",
			"An Expression argument must be a SourcePath or scalar literal; inline object/array values are invalid")
	}
}
