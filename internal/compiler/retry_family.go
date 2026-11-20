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

func validateRetryMutationFamilies(req BuildRequest, skill *SkillAST, identity dependencyIdentityResult, ownerLinking ownerLinkingResult, procedurePath string, add func(string, *yaml.Node, string, string)) {
	if skill == nil || skill.Procedure == nil {
		return
	}
	linked := map[string]linkedExport{}
	for _, export := range ownerLinking.Exports {
		if export.ExportKind == "operation" || export.ExportKind == "observation" {
			linked[export.DependencyAlias+"::"+export.ExportID] = export
		}
	}
	// Dedicated Reconcile READ exports are never ordinary business callables.
	dedicatedReconcile := map[string]bool{}
	snapshots := map[string]ResolvedDependency{}
	for _, alias := range identity.Closure {
		for _, dependency := range req.ResolvedDependencies {
			if dependency.DependencyID == alias {
				snapshots[alias] = dependency
			}
		}
	}
	for _, alias := range sortedStringMapKeys(snapshots) {
		index, _, _ := buildSnapshotExportIndex(snapshots[alias])
		for _, records := range index.callables {
			for _, record := range records {
				if record.Effect == "MUTATION" && record.MutationFields.ReconcileCallable != "" {
					dedicatedReconcile[record.MutationFields.ReconcileCallable] = true
				}
			}
		}
	}
	nodes := map[string]*ProcedureNodeAST{}
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
		edge := retryEdge{nodeID: nodeID, target: target}
		if export, ok := linked[scalar(unalias(node.Operation.Tool))]; ok && dedicatedReconcile[export.ExportID] {
			edge.reconcileTarget = export.ExportID
		}
		edges = append(edges, edge)
	}
	if len(edges) == 0 {
		return
	}
	effects := map[string]string{}
	for nodeID, node := range nodes {
		if export, ok := linked[scalar(node.Operation.Tool)]; ok {
			effects[nodeID] = export.Effect
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
	// Positive retry assurance (搂20.2 items 3-4): the retried operation must be
	// the same Owner business identity, the original Mutation must publish a
	// retryable, no-side-effect NOT_COMMITTED failure class, and the retry
	// callable must accept the original business identity Schema.
	registry := buildSchemaRegistry(req, skill, identity, req.development)
	for _, edge := range edges {
		retryNode := nodes[edge.nodeID]
		targetNode := skill.Procedure.ByID[edge.target]
		if retryNode == nil || retryNode.Operation == nil || targetNode == nil || targetNode.Operation == nil {
			continue
		}
		anchor := unalias(child(retryNode.Operation.Raw, "retryOf"))
		targetExport, targetLinked := linked[scalar(unalias(targetNode.Operation.Tool))]
		retryExport, retryLinked := linked[scalar(unalias(retryNode.Operation.Tool))]
		if !targetLinked || !retryLinked {
			continue // unresolved links are diagnosed by the linking slices
		}
		if targetExport.Effect != "MUTATION" {
			add(procedurePath, anchor, "RETRY_CONTRACT_INVALID",
				"retryOf target "+edge.target+" is not a MUTATION callable; it has Effect "+targetExport.Effect)
			continue
		}
		if retryExport.Effect != "MUTATION" {
			add(procedurePath, anchor, "RETRY_CONTRACT_INVALID",
				"retry Node "+edge.nodeID+" must call a MUTATION callable; it has Effect "+retryExport.Effect)
			continue
		}
		if targetExport.DependencyAlias != retryExport.DependencyAlias {
			add(procedurePath, anchor, "RETRY_CONTRACT_INVALID",
				"retry dispatch must not cross dependency identity: "+targetExport.DependencyAlias+" != "+retryExport.DependencyAlias)
			continue
		}
		if targetExport.TrustedOwner != retryExport.TrustedOwner || targetExport.TrustedOwner == "" {
			add(procedurePath, anchor, "RETRY_CONTRACT_INVALID",
				"retry dispatch must keep the exact trusted Owner of the original Mutation")
			continue
		}
		eligible := false
		for _, class := range targetExport.MutationFields.FailureClasses {
			if class.Outcome == "NOT_COMMITTED" && class.Retryable && class.NoSideEffect {
				eligible = true
				break
			}
		}
		if !eligible {
			add(procedurePath, anchor, "RETRY_CONTRACT_INVALID",
				"original Mutation publishes no retryable, no-side-effect NOT_COMMITTED failure class: "+targetExport.ExportID)
			continue
		}
		originalInput, originalProblem := registry.resolveReference(targetExport.DependencyAlias + "::" + targetExport.InputSchemaID)
		retryInput, retryProblem := registry.resolveReference(retryExport.DependencyAlias + "::" + retryExport.InputSchemaID)
		if originalProblem != "" || retryProblem != "" {
			add(procedurePath, anchor, "RETRY_CONTRACT_INVALID",
				"retry business-identity Input Schema is not resolvable in the trusted snapshot")
			continue
		}
		if _, ok := schemaAssignable(unalias(originalInput.node), unalias(retryInput.node)); !ok {
			add(procedurePath, anchor, "RETRY_CONTRACT_INVALID",
				"retry callable Input Schema cannot accept the original Mutation business identity")
		}
	}
}
