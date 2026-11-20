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
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

type mutationFailureClass struct {
	ID            string
	Outcome       string
	ProofSchemaID string
	ProofPath     string
	NoSideEffect  bool
	Retryable     bool
}

type mutationResourceOutput struct {
	Path                   string
	ResourceKind           string
	ResponsibilitySchemaID string
}

type mutationContractFields struct {
	ReceiptSchemaID        string
	CommitProofSchemaID    string
	CommitProofPath        string
	FailureClassPath       string
	FailureClasses         []mutationFailureClass
	ResourceOutputs        []mutationResourceOutput
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
	fields.FailureClassPath = scalar(child(mutation, "failureClassPath"))
	for _, classNode := range sequence(child(mutation, "failureClasses")) {
		classNode = unalias(classNode)
		if classNode == nil || classNode.Kind != yaml.MappingNode {
			continue
		}
		class := mutationFailureClass{
			ID:           scalar(child(classNode, "id")),
			Outcome:      scalar(child(classNode, "outcome")),
			ProofPath:    scalar(child(classNode, "proofPath")),
			NoSideEffect: child(classNode, "noSideEffect") != nil && scalar(child(classNode, "noSideEffect")) == "true",
			Retryable:    child(classNode, "retryable") != nil && scalar(child(classNode, "retryable")) == "true",
		}
		if id, ok := schemaRefID(unwrapNode(child(classNode, "proofSchema"))); ok {
			class.ProofSchemaID = id
		}
		fields.FailureClasses = append(fields.FailureClasses, class)
	}
	for _, outputNode := range sequence(child(mutation, "resourceOutputs")) {
		outputNode = unalias(outputNode)
		if outputNode == nil || outputNode.Kind != yaml.MappingNode {
			continue
		}
		output := mutationResourceOutput{
			Path:         scalar(child(outputNode, "path")),
			ResourceKind: scalar(child(outputNode, "resourceKind")),
		}
		if id, ok := schemaRefID(unwrapNode(child(outputNode, "responsibilitySchema"))); ok {
			output.ResponsibilitySchemaID = id
		}
		fields.ResourceOutputs = append(fields.ResourceOutputs, output)
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

func validateReconcileContracts(req BuildRequest, skill *SkillAST, identity dependencyIdentityResult, ownerLinking ownerLinkingResult, procedurePath string, add func(string, *yaml.Node, string, string)) {
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
	registry := buildSchemaRegistry(req, skill, identity, req.development)
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
					if _, problem := registry.resolveReference(alias + "::" + callable.MutationFields.CommitProofSchemaID); problem != "" {
						report("Mutation commitProof Schema is not resolvable in the trusted snapshot: " + callable.MutationFields.CommitProofSchemaID)
					}
				}
				for _, output := range callable.MutationFields.ResourceOutputs {
					if output.ResponsibilitySchemaID == "" {
						continue // identity-level shape check already rejected this declaration
					}
					if _, problem := registry.resolveReference(alias + "::" + output.ResponsibilitySchemaID); problem != "" {
						report("Mutation resourceOutputs responsibility Schema is not resolvable: " + output.ResponsibilitySchemaID)
						continue
					}
					outputSchema, problem := registry.resolveReference(alias + "::" + callable.OutputSchemaID)
					if problem != "" {
						report("Mutation resourceOutputs path requires a resolvable Mutation output Schema: " + callable.OutputSchemaID)
						continue
					}
					if !schemaPathExists(unalias(outputSchema.node), strings.TrimPrefix(output.Path, "output.")) {
						report("Mutation resourceOutputs path is not declared by the Mutation output Schema: " + output.Path)
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
					if _, problem := registry.resolveReference(alias + "::" + callable.MutationFields.ReconcileProofSchemaID); problem != "" {
						report("Mutation Reconcile proof Schema is not resolvable: " + callable.MutationFields.ReconcileProofSchemaID)
					}
				}
				if callable.MutationFields.RecoverableSchemaID != "" {
					if _, problem := registry.resolveReference(alias + "::" + callable.MutationFields.RecoverableSchemaID); problem != "" {
						report("Mutation recoverableOutputSchema is not resolvable: " + callable.MutationFields.RecoverableSchemaID)
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
