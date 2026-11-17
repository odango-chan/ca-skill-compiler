package compiler

// WP8 slice 3: retry family checks on top of retryOf reference validity.
//
// CA-SL: retryOf defaults to zero and each original Mutation family allows at
// most one new dispatch. Both the retried Operation and the retry Operation must
// link to MUTATION callables. Full retry contract compatibility (failure
// classes, path coverage) remains later WP8 work.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

type retryEdge struct {
	nodeID          string
	target          string
	reconcileTarget string
}

type retryViolation struct {
	edge    int
	message string
}

// retryFamilyViolations is the testable core of the retry family rule. effects
// maps Node ID to its linked Effect; a missing entry stays unproven and is not
// reported here (the linking slices own unresolved links).
func retryFamilyViolations(edges []retryEdge, effects map[string]string) []retryViolation {
	violations := make([]retryViolation, 0)
	families := map[string]int{}
	for index, edge := range edges {
		families[edge.target]++
		if families[edge.target] > 1 {
			violations = append(violations, retryViolation{edge: index, message: "an original Mutation family allows at most one retry dispatch: " + edge.target})
			continue
		}
		if effect, ok := effects[edge.target]; ok && effect != "MUTATION" {
			violations = append(violations, retryViolation{edge: index, message: "retryOf must target a MUTATION callable; " + edge.target + " has Effect " + effect})
			continue
		}
		if effect, ok := effects[edge.nodeID]; ok && effect != "MUTATION" {
			violations = append(violations, retryViolation{edge: index, message: "a retry Node must call a MUTATION callable; it has Effect " + effect})
			continue
		}
		if edge.reconcileTarget != "" {
			violations = append(violations, retryViolation{edge: index, message: "a retry Node must not use the dedicated Reconcile READ callable"})
		}
	}
	return violations
}

func validateV1Alpha4RetryMutationFamilies(skill *V1Alpha4SkillAST, ownerLinking ownerLinkingResult, procedurePath string, add func(string, *yaml.Node, string, string)) {
	if skill == nil || skill.Procedure == nil {
		return
	}
	linked := map[string]linkedExport{}
	for _, export := range ownerLinking.Exports {
		if export.ExportKind == "operation" || export.ExportKind == "observation" {
			linked[export.DependencyAlias+"::"+export.ExportID] = export
		}
	}
	nodes := map[string]*V1Alpha4ProcedureNodeAST{}
	edges := make([]retryEdge, 0)
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil || node.Operation == nil {
			continue
		}
		nodes[nodeID] = node
		retryNode := unalias(child(node.Operation.Raw, "retryOf"))
		if retryNode == nil {
			continue
		}
		target := scalar(retryNode)
		if target == "" || target == nodeID {
			continue // reference validity is diagnosed by the earlier slice
		}
		edges = append(edges, retryEdge{nodeID: nodeID, target: target})
	}
	if len(edges) == 0 {
		return
	}
	effects := map[string]string{}
	for nodeID, node := range nodes {
		if export, ok := linked[scalar(node.Operation.Tool)]; ok {
			effects[nodeID] = export.Effect
			if export.ReconcileTarget != "" {
				for index := range edges {
					if edges[index].target == nodeID {
						edges[index].reconcileTarget = export.ReconcileTarget
					}
				}
			}
		}
	}
	for _, violation := range retryFamilyViolations(edges, effects) {
		edge := edges[violation.edge]
		node := nodes[edge.nodeID]
		var anchor *yaml.Node
		if node != nil && node.Operation != nil {
			anchor = unalias(child(node.Operation.Raw, "retryOf"))
		}
		add(procedurePath, anchor, "RETRY_CONTRACT_INVALID", violation.message)
	}
}
