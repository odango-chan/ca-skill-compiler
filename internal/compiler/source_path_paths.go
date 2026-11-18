package compiler

// WP6 slice: per-path availability of producer outputs, including the frozen
// COMMITTED-only recovery distinction.
//
// A producer output may only be consumed on a path where the producing stage
// actually succeeded. If a consumer is reachable from a non-SUCCEEDED edge of the
// producer, the output is not available:
//   - COMMITTED edge (proved committed, Owner did not declare a recoverable
//     success output) -> RECONCILE_OUTPUT_UNAVAILABLE;
//   - other non-SUCCEEDED edges -> SOURCE_NOT_AVAILABLE.
// Recoverable-output assignability to the original output contract remains later
// WP8 work; a declared recoverable output is not rejected here.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

// producerKeyReachability returns, per route key of the producer Node, the set of
// Nodes reachable by following that key's target (BFS over the whole graph).
func producerKeyReachability(skill *SkillAST, producerID string) map[string]map[string]bool {
	result := map[string]map[string]bool{}
	if skill == nil || skill.Procedure == nil {
		return result
	}
	graph := buildProcedureGraph(skill.Procedure)
	for _, edge := range procedureRouteEdges(skill.Procedure) {
		if edge.from != producerID {
			continue
		}
		reachable := map[string]bool{edge.to: true}
		queue := []string{edge.to}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, next := range graph.edges[current] {
				if !reachable[next] {
					reachable[next] = true
					queue = append(queue, next)
				}
			}
		}
		result[edge.key] = reachable
	}
	return result
}

func nonSucceededProducerPath(producerID, consumerID string, reachability map[string]map[string]bool) (committed bool, other bool) {
	for key, reachable := range reachability {
		if key == "SUCCEEDED" || !reachable[consumerID] {
			continue
		}
		if key == "COMMITTED" {
			committed = true
			continue
		}
		other = true
	}
	return committed, other
}

var _ = yaml.Node{}
