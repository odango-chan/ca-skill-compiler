package compiler

// WP5 slice 2: Procedure stage route obligations.
//
// The frozen Procedure grammar requires: the normal Outcome of a non-last stage
// auto-advances and MUST NOT be declared in that stage's routes; the normal
// Outcome of the last declared stage MUST have an explicit route; Predicate
// stages may only declare PASS / FAIL / UNKNOWN.

import (
	"sort"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

var stageNormalOutcome = map[string]string{
	"precondition":  "PASS",
	"operation":     "SUCCEEDED",
	"observation":   "SUCCEEDED",
	"postcondition": "PASS",
}

var predicateStageOutcomes = map[string]bool{"PASS": true, "FAIL": true, "UNKNOWN": true}

func validateProcedureStageRoutes(file string, procedure *ProcedureAST, add func(string, *yaml.Node, string, string)) {
	if procedure == nil || procedure.Nodes == nil {
		return
	}
	for _, nodeID := range sortedStringMapKeys(procedure.ByID) {
		node := procedure.ByID[nodeID]
		if node == nil || node.Raw == nil || node.Raw.Kind != yaml.MappingNode {
			continue
		}
		declared := make([]string, 0, 4)
		for _, stage := range []string{"precondition", "operation", "observation", "postcondition"} {
			if child(node.Raw, stage) != nil {
				declared = append(declared, stage)
			}
		}
		for index, stage := range declared {
			body := child(node.Raw, stage)
			if body == nil {
				continue
			}
			routes := child(body, "routes")
			normal := stageNormalOutcome[stage]
			hasNormal := child(routes, normal) != nil
			isLast := index == len(declared)-1
			if !isLast && hasNormal {
				add(file, child(routes, normal), "ROUTE_NORMAL_OUTCOME_NOT_ALLOWED",
					"Normal Outcome "+normal+" of a non-last stage must auto-advance and may not be declared in routes")
			}
			if isLast && !hasNormal {
				add(file, body, "ROUTE_OUTCOME_MISSING",
					"Last declared stage must route its normal Outcome "+normal)
			}
			if stage == "precondition" || stage == "postcondition" {
				for _, outcome := range routeKeys(routes) {
					if !predicateStageOutcomes[outcome] {
						add(file, child(routes, outcome), "ROUTE_OUTCOME_UNKNOWN",
							"Predicate stage route key is not eligible: "+outcome)
					}
				}
			}
		}
	}
}

func routeKeys(routes *yaml.Node) []string {
	keys := append([]string(nil), mapKeys(routes)...)
	sort.Strings(keys)
	return keys
}
