package compiler

// WP8 slice 2: retryOf reference validity (no retry chain, target must exist and
// be a Mutation-shaped operation Node). Full retry contract compatibility and
// path-coverage analysis remain NOT_IMPLEMENTED and fail-closed.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validateV1Alpha4RetryContracts(skill *V1Alpha4SkillAST, procedurePath string, add func(string, *yaml.Node, string, string)) {
	if skill == nil || skill.Procedure == nil {
		return
	}
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil || node.Operation == nil {
			continue
		}
		retryNode := unalias(child(node.Operation.Raw, "retryOf"))
		if retryNode == nil {
			continue
		}
		target := scalar(retryNode)
		if target == "" {
			add(procedurePath, retryNode, "RETRY_CONTRACT_INVALID", "retryOf must name a declared Mutation Node")
			continue
		}
		if target == nodeID {
			add(procedurePath, retryNode, "RETRY_CONTRACT_INVALID", "retryOf must not reference the same Node")
			continue
		}
		targetNode := skill.Procedure.ByID[target]
		if targetNode == nil {
			add(procedurePath, retryNode, "RETRY_CONTRACT_INVALID", "retryOf references an undeclared Node: "+target)
			continue
		}
		if targetNode.Operation == nil {
			add(procedurePath, retryNode, "RETRY_CONTRACT_INVALID", "retryOf must reference a Mutation Node with an operation stage: "+target)
			continue
		}
		if child(targetNode.Operation.Raw, "retryOf") != nil {
			add(procedurePath, retryNode, "RETRY_CONTRACT_INVALID", "a retry Node cannot itself be retried (second-generation chain): "+target)
		}
	}
}
