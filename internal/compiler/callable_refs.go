package compiler

// Qualified callable reference grammar for Procedure operation/observation tool.
// Frozen diagnostic: QUALIFIED_REFERENCE_INVALID (G.1 QualifiedCallableId).

import (
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validQualifiedCallableReference(ref string) bool {
	index := strings.Index(ref, "::")
	if index <= 0 {
		return false
	}
	return validDependencyID(ref[:index]) && validExportID(ref[index+2:])
}

func validateQualifiedCallableReferences(skill *SkillAST, procedurePath string, add func(string, *yaml.Node, string, string)) {
	if skill == nil || skill.Procedure == nil {
		return
	}
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil {
			continue
		}
		if node.Operation != nil {
			validateToolReference(node.Operation.Tool, procedurePath, add)
		}
		if node.Observation != nil {
			validateToolReference(node.Observation.Tool, procedurePath, add)
		}
	}
}

func validateToolReference(tool *yaml.Node, procedurePath string, add func(string, *yaml.Node, string, string)) {
	tool = unalias(tool)
	if tool == nil {
		return // absence is diagnosed by the existing control-flow checks
	}
	if !validQualifiedCallableReference(scalar(tool)) {
		add(procedurePath, tool, "QUALIFIED_REFERENCE_INVALID",
			"Tool reference must be an exact Alias::ExportId qualified callable identity")
	}
}
