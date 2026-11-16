package compiler

// WP8 slice 4: static semantic checks on the trusted Owner Mutation contract.
//
// Identity/shape requirements are checked by the earlier mutation-contract slice.
// This slice proves the referenced Schema identities resolve in the same trusted
// snapshot and that declared proof paths exist in those Schemas, that the
// dedicated Reconcile READ is a same-owner READ operation, and that
// recoverable-output definitions are paired and resolvable.
//
// Cross-snapshot Reconcile association is intentionally NOT implemented: no
// separately versioned trusted association contract exists.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

type mutationContractFields struct {
	ReceiptSchemaID        string
	CommitProofSchemaID    string
	CommitProofPath        string
	ReconcileCallable      string
	ReconcileProofSchemaID string
	ReconcileProofPath     string
	RecoverableSchemaID    string
	RecoverablePath        string
}

func parseMutationContractFields(node *yaml.Node) mutationContractFields {
	var fields mutationContractFields
	if receipt, ok := schemaRefID(child(node, "receiptSchema")); ok {
		fields.ReceiptSchemaID = receipt
	}
	mutation := unalias(child(node, "mutation"))
	if mutation == nil {
		return fields
	}
	if commit := unalias(child(mutation, "commitProof")); commit != nil {
		if id, ok := schemaRefID(unwrapNode(child(commit, "schema"))); ok {
			fields.CommitProofSchemaID = id
		}
		fields.CommitProofPath = scalar(child(commit, "path"))
	}
	if reconcile := unalias(child(mutation, "reconcile")); reconcile != nil {
		fields.ReconcileCallable = scalar(child(reconcile, "callable"))
		if id, ok := schemaRefID(unwrapNode(child(reconcile, "proofSchema"))); ok {
			fields.ReconcileProofSchemaID = id
		}
		fields.ReconcileProofPath = scalar(child(reconcile, "proofPath"))
		if id, ok := schemaRefID(unwrapNode(child(reconcile, "recoverableOutputSchema"))); ok {
			fields.RecoverableSchemaID = id
		}
		fields.RecoverablePath = scalar(child(reconcile, "recoverableOutputPath"))
	}
	return fields
}

func validateV1Alpha4ReconcileContracts(req BuildRequest, skill *V1Alpha4SkillAST, identity dependencyIdentityResult, ownerLinking ownerLinkingResult, procedurePath string, add func(string, *yaml.Node, string, string)) {
	if skill == nil || skill.Contract == nil {
		return
	}
	snapshots := map[string]ResolvedDependency{}
	for _, alias := range identity.Closure {
		for _, dependency := range req.ResolvedDependencies {
			if dependency.DependencyID == alias {
				snapshots[alias] = dependency
			}
		}
	}
	registry := buildV1Alpha4SchemaRegistry(req, skill, identity)
	trustedOwner := map[string]string{}
	for _, export := range ownerLinking.Exports {
		trustedOwner[export.DependencyAlias+"\x00"+export.ExportKind+"\x00"+export.ExportID] = export.TrustedOwner
	}
	for _, alias := range sortedStringMapKeys(snapshots) {
		index, _, _ := buildSnapshotExportIndex(snapshots[alias])
		for _, exportID := range sortedStringMapKeys(index.callables) {
			for _, callable := range index.callables[exportID] {
				if callable.Effect != "MUTATION" || callable.MutationProblem != "" {
					continue
				}
				report := func(message string) {
					add(procedurePath, nil, "RECONCILE_CONTRACT_INVALID", message)
				}
				if callable.MutationFields.ReceiptSchemaID != "" {
					if _, problem := registry.resolveReference(alias + "::" + callable.MutationFields.ReceiptSchemaID); problem != "" {
						report("Mutation receipt Schema is not resolvable in the trusted snapshot: " + callable.MutationFields.ReceiptSchemaID)
					}
				}
				if callable.MutationFields.CommitProofSchemaID != "" {
					proof, problem := registry.resolveReference(alias + "::" + callable.MutationFields.CommitProofSchemaID)
					if problem != "" {
						report("Mutation commitProof Schema is not resolvable in the trusted snapshot: " + callable.MutationFields.CommitProofSchemaID)
					} else if !schemaPathExists(unalias(proof.node), callable.MutationFields.CommitProofPath) {
						report("Mutation commitProof path is not declared by its proof Schema: " + callable.MutationFields.CommitProofPath)
					}
				}
				if callable.MutationFields.ReconcileCallable == "" {
					continue
				}
				mutationOwner := callable.TrustedOwner
				if mutationOwner == "" {
					mutationOwner = trustedOwner[alias+"\x00operation\x00"+callable.ExportID]
				}
				targets := index.callables[callable.MutationFields.ReconcileCallable]
				if len(targets) != 1 {
					report("Mutation Reconcile READ callable must resolve to exactly one same-snapshot operation: " + callable.MutationFields.ReconcileCallable)
					continue
				}
				target := targets[0]
				if target.ExportKind != "operation" || target.Effect != "READ" {
					report("Mutation Reconcile callable must be a READ operation: " + callable.MutationFields.ReconcileCallable)
				}
				targetOwner := target.TrustedOwner
				if targetOwner == "" {
					targetOwner = trustedOwner[alias+"\x00operation\x00"+target.ExportID]
				}
				if mutationOwner != "" && targetOwner != "" && mutationOwner != targetOwner {
					report("Mutation Reconcile READ must belong to the same Owner as the Mutation: " + callable.MutationFields.ReconcileCallable)
				}
				if callable.MutationFields.ReconcileProofSchemaID != "" {
					proof, problem := registry.resolveReference(alias + "::" + callable.MutationFields.ReconcileProofSchemaID)
					if problem != "" {
						report("Mutation Reconcile proof Schema is not resolvable: " + callable.MutationFields.ReconcileProofSchemaID)
					} else if !schemaPathExists(unalias(proof.node), callable.MutationFields.ReconcileProofPath) {
						report("Mutation Reconcile proofPath is not declared by its proof Schema: " + callable.MutationFields.ReconcileProofPath)
					}
				}
				if callable.MutationFields.RecoverableSchemaID != "" {
					recoverable, problem := registry.resolveReference(alias + "::" + callable.MutationFields.RecoverableSchemaID)
					if problem != "" {
						report("Mutation recoverableOutputSchema is not resolvable: " + callable.MutationFields.RecoverableSchemaID)
					} else if !schemaPathExists(unalias(recoverable.node), callable.MutationFields.RecoverablePath) {
						report("Mutation recoverableOutputPath is not declared by its Schema: " + callable.MutationFields.RecoverablePath)
					}
				}
			}
		}
	}
}

// schemaPathExists reports whether a dotted property path resolves in a G.4 object
// Schema. Arrays are not traversed; callers use object property paths.
func schemaPathExists(root *yaml.Node, path string) bool {
	if path == "" {
		return false
	}
	node := root
	for _, segment := range splitDottedPath(path) {
		properties := child(node, "properties")
		if properties == nil {
			return false
		}
		node = child(properties, segment)
		if node == nil {
			return false
		}
	}
	return true
}

func splitDottedPath(path string) []string {
	segments := make([]string, 0)
	start := 0
	for index := 0; index <= len(path); index++ {
		if index == len(path) || path[index] == '.' {
			if index > start {
				segments = append(segments, path[start:index])
			}
			start = index + 1
		}
	}
	return segments
}
