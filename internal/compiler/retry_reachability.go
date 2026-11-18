package compiler

// WP8 slice 5: retry reachability proof (CA-SL §12.8.3/§12.8.4).
//
// A retry Node may only be entered through the proven NOT_COMMITTED edge of its
// original Mutation Node. Every incoming edge to the retry Node must therefore
// originate from the original Node with route key NOT_COMMITTED; no SUCCEEDED,
// COMMITTED, OUTCOME_UNKNOWN, FAILED or bypass edge may authorize the retry.
//
// Owner-side retryability/no-side-effect proof and Runtime retry-admission
// remain Host responsibilities; the Compiler proves only the static graph rule.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

type routeEdge struct {
	from  string
	stage string
	key   string
	to    string
}

func procedureRouteEdges(procedure *ProcedureAST) []routeEdge {
	edges := make([]routeEdge, 0)
	if procedure == nil {
		return edges
	}
	for _, nodeID := range sortedStringMapKeys(procedure.ByID) {
		node := procedure.ByID[nodeID]
		if node == nil {
			continue
		}
		type stageRoutes struct {
			name   string
			routes *yaml.Node
		}
		stages := make([]stageRoutes, 0, 4)
		if node.Precondition != nil {
			stages = append(stages, stageRoutes{"precondition", node.Precondition.Routes})
		}
		if node.Operation != nil {
			stages = append(stages, stageRoutes{"operation", node.Operation.Routes})
		}
		if node.Observation != nil {
			stages = append(stages, stageRoutes{"observation", node.Observation.Routes})
		}
		if node.Postcondition != nil {
			stages = append(stages, stageRoutes{"postcondition", node.Postcondition.Routes})
		}
		for _, stage := range stages {
			routes := unalias(stage.routes)
			if routes == nil || routes.Kind != yaml.MappingNode {
				continue
			}
			for i := 0; i+1 < len(routes.Content); i += 2 {
				key := scalar(unalias(routes.Content[i]))
				target := scalar(unalias(routes.Content[i+1]))
				if key == "" || target == "" {
					continue
				}
				edges = append(edges, routeEdge{from: nodeID, stage: stage.name, key: key, to: target})
			}
		}
	}
	return edges
}

func validateRetryReachability(skill *SkillAST, procedurePath string, add func(string, *yaml.Node, string, string)) {
	if skill == nil || skill.Procedure == nil {
		return
	}
	incoming := map[string][]routeEdge{}
	for _, edge := range procedureRouteEdges(skill.Procedure) {
		incoming[edge.to] = append(incoming[edge.to], edge)
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
		if target == "" || target == nodeID {
			continue // reference validity is diagnosed by the earlier slice
		}
		edges := incoming[nodeID]
		if len(edges) == 0 {
			add(procedurePath, retryNode, "RETRY_CONTRACT_INVALID",
				"retry Node is not reachable through the original Mutation NOT_COMMITTED edge: "+target)
			continue
		}
		for _, edge := range edges {
			if edge.from != target || edge.key != "NOT_COMMITTED" {
				add(procedurePath, retryNode, "RETRY_CONTRACT_INVALID",
					"retry Node may only be entered from the original Mutation NOT_COMMITTED edge; found "+edge.from+"."+edge.key)
			}
		}
	}
}
