package compiler

// WP5 slice 3: §22.1 route-key eligibility matrix for linked Tool callables.
//
// Allowed keys are the union of the trusted Owner-declared outcomes and the
// applicable Runtime-reserved outcomes. Owner-declared FAILED / NOT_COMMITTED and
// every Mutation OUTCOME_UNKNOWN must be routed explicitly; COMMITTED is only
// allowed for a Mutation with a trusted Reconcile capability. Procedure Script
// routes allow SUCCEEDED/FAILED with FAILED required.
//
// Normal-outcome placement (auto-advance vs last-stage mandatory) is owned by the
// earlier procedure-stage slice; this validator skips SUCCEEDED.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validateRouteKeyEligibility(skill *SkillAST, ownerLinking ownerLinkingResult, procedurePath string, add func(string, *yaml.Node, string, string)) {
	if skill == nil || skill.Procedure == nil {
		return
	}
	linked := map[string]linkedExport{}
	for _, export := range ownerLinking.Exports {
		if export.ExportKind == "operation" || export.ExportKind == "observation" {
			linked[export.DependencyAlias+"::"+export.ExportID] = export
		}
	}
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil {
			continue
		}
		if node.Operation != nil {
			checkRouteKeys(procedurePath, node.Operation.Routes, node.Operation.Tool, node.Operation.Script, linked, add)
		}
		if node.Observation != nil {
			checkRouteKeys(procedurePath, node.Observation.Routes, node.Observation.Tool, nil, linked, add)
		}
	}
}

func checkRouteKeys(path string, routes *yaml.Node, tool, script *yaml.Node, linked map[string]linkedExport, add func(string, *yaml.Node, string, string)) {
	if routes == nil {
		return
	}
	declared := map[string]*yaml.Node{}
	for _, key := range routeKeys(routes) {
		declared[key] = child(routes, key)
	}
	allowed := map[string]bool{"SUCCEEDED": true, "FAILED": true}
	required := map[string]bool{"FAILED": true}
	if script != nil {
		// Procedure Script routes: SUCCEEDED/FAILED only, FAILED required.
	} else {
		ref := scalar(unalias(tool))
		export, ok := linked[ref]
		if !ok {
			return // unresolved links are diagnosed by the linking slices
		}
		switch export.Effect {
		case "READ":
			allowed["DENIED"] = true
			allowed["UNKNOWN"] = true
		case "MUTATION":
			allowed["DENIED"] = true
			allowed["OUTCOME_UNKNOWN"] = true
			required["OUTCOME_UNKNOWN"] = true
			if export.ReconcileTarget != "" {
				allowed["COMMITTED"] = true
			}
			for _, outcome := range export.OwnerOutcomes {
				if outcome == "NOT_COMMITTED" {
					allowed["NOT_COMMITTED"] = true
					required["NOT_COMMITTED"] = true
				}
			}
		default:
			return
		}
		if !containsString(export.OwnerOutcomes, "FAILED") {
			delete(allowed, "FAILED")
			delete(required, "FAILED")
		}
	}
	for key, node := range declared {
		if key == "SUCCEEDED" {
			continue
		}
		if !allowed[key] {
			add(path, node, "ROUTE_OUTCOME_UNKNOWN", "Route key is not eligible for the linked callable: "+key)
		}
	}
	for key := range required {
		if _, ok := declared[key]; !ok {
			add(path, routes, "ROUTE_OUTCOME_MISSING", "Required route key is missing: "+key)
		}
	}
	if export, ok := linked[scalar(unalias(tool))]; ok && script == nil && export.Effect == "MUTATION" {
		if targetNode, present := declared["OUTCOME_UNKNOWN"]; present {
			target := scalar(unalias(targetNode))
			if target != "UNRESOLVED" && target != "SAFE_STOPPED" {
				add(path, targetNode, "RECONCILE_CONTRACT_INVALID",
					"Mutation OUTCOME_UNKNOWN may only route to UNRESOLVED or SAFE_STOPPED, not "+target)
			}
		}
	}
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
