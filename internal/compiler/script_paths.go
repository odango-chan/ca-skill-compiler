package compiler

// WP9 slice 1: Procedure Script LocalScriptPath grammar and file presence.
// Python CST/TypedDict analysis remains NOT_IMPLEMENTED and fail-closed.

import (
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validLocalScriptPath(value string) bool {
	const prefix = "scripts/"
	const suffix = ".py"
	if !strings.HasPrefix(value, prefix) || !strings.HasSuffix(value, suffix) {
		return false
	}
	trimmed := strings.TrimSuffix(strings.TrimPrefix(value, prefix), suffix)
	if trimmed == "" {
		return false
	}
	for _, segment := range strings.Split(trimmed, "/") {
		if !validDottedFileSegment(segment) {
			return false
		}
	}
	return true
}

func validDottedFileSegment(segment string) bool {
	parts := strings.Split(segment, ".")
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if !validDependencyID(part) {
			return false
		}
	}
	return true
}

func validateV1Alpha4ScriptPaths(skill *V1Alpha4SkillAST, files map[string][]byte, procedurePath string, add func(string, *yaml.Node, string, string)) {
	if skill == nil || skill.Procedure == nil {
		return
	}
	base := "skills/" + skill.Name + "/"
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil || node.Operation == nil || node.Operation.Script == nil {
			continue
		}
		scriptNode := unalias(node.Operation.Script)
		script := scalar(scriptNode)
		if !validLocalScriptPath(script) {
			add(procedurePath, scriptNode, "SCRIPT_CONTRACT_INVALID",
				"operation.script must be a LocalScriptPath under scripts/** ending in .py")
			continue
		}
		if _, ok := files[base+script]; !ok {
			add(procedurePath, scriptNode, "SCRIPT_CONTRACT_INVALID",
				"Referenced Procedure Script file is missing from the Skill package: "+script)
		}
	}
}
