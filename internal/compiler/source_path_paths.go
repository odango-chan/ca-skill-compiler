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

import "strings"

// producerKeyReachability tracks routes that leave a Node before its producing
// stage succeeds. A later stage can only run after the producer succeeds.
func producerKeyReachability(skill *SkillAST, sourcePath string) map[string]map[string]bool {
	result := map[string]map[string]bool{}
	if skill == nil || skill.Procedure == nil {
		return result
	}
	producerID, rest, _ := strings.Cut(sourcePath, ".")
	producerStage, _, _ := strings.Cut(rest, ".")
	graph := buildProcedureGraph(skill.Procedure)
	for _, edge := range procedureRouteEdges(skill.Procedure) {
		if edge.from != producerID {
			continue
		}
		key := edge.key
		switch {
		case edge.stage == producerStage:
			if key == stageNormalOutcome[producerStage] {
				continue
			}
		case edge.stage == "precondition" || (edge.stage == "operation" && producerStage == "observation"):
			// An earlier-stage exit bypasses this producer. Even an earlier
			// operation's COMMITTED edge provides no observation output.
			key = ""
		default:
			continue
		}
		reachable := result[key]
		if reachable == nil {
			reachable = map[string]bool{}
			result[key] = reachable
		}
		if reachable[edge.to] {
			continue
		}
		reachable[edge.to] = true
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
	}
	return result
}

func nonSucceededProducerPath(consumerID string, reachability map[string]map[string]bool) (committed bool, other bool) {
	for key, reachable := range reachability {
		if !reachable[consumerID] {
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
