package compiler

// WP8 slice 1: Mutation contract identity requirements from the trusted
// ToolCatalog. Full Reconcile execution semantics and Runtime settlement remain
// out of scope; violations use the frozen RECONCILE_CONTRACT_INVALID identity.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func mutationContractProblem(node *yaml.Node) string {
	receipt := child(node, "receiptSchema")
	if _, ok := schemaRefID(receipt); !ok {
		return "Mutation callable must publish receiptSchema.$ref to an exact Schema $id"
	}
	mutation := unalias(child(node, "mutation"))
	if mutation == nil || mutation.Kind != yaml.MappingNode {
		return "Mutation callable must publish a mutation contract"
	}
	commitProof := unalias(child(mutation, "commitProof"))
	if commitProof == nil || commitProof.Kind != yaml.MappingNode {
		return "Mutation commitProof is missing"
	}
	if _, ok := schemaRefID(unwrapNode(child(commitProof, "schema"))); !ok {
		return "Mutation commitProof.schema must be an exact Schema $id"
	}
	if scalar(child(commitProof, "path")) == "" {
		return "Mutation commitProof.path is required"
	}
	if scalar(child(commitProof, "bind")) != "originalInvocationId" {
		return "Mutation commitProof.bind must be originalInvocationId"
	}
	reconcile := unalias(child(mutation, "reconcile"))
	if reconcile == nil {
		return ""
	}
	if reconcile.Kind != yaml.MappingNode {
		return "Mutation reconcile must be a mapping"
	}
	if !validExportID(scalar(child(reconcile, "callable"))) {
		return "Mutation reconcile.callable must be an exact callable ExportId"
	}
	if scalar(child(reconcile, "bind")) != "originalInvocationId" {
		return "Mutation reconcile.bind must be originalInvocationId"
	}
	if _, ok := schemaRefID(unwrapNode(child(reconcile, "proofSchema"))); !ok {
		return "Mutation reconcile.proofSchema must be an exact Schema $id"
	}
	if scalar(child(reconcile, "proofPath")) == "" {
		return "Mutation reconcile.proofPath is required"
	}
	hasRecoverableSchema := scalar(child(reconcile, "recoverableOutputSchema")) != ""
	hasRecoverablePath := scalar(child(reconcile, "recoverableOutputPath")) != ""
	if hasRecoverableSchema != hasRecoverablePath {
		return "recoverableOutputSchema and recoverableOutputPath must be paired"
	}
	return ""
}

// unwrapNode adapts a scalar string carrying a Schema $id to the mapping shape
// accepted by schemaRefID.
func unwrapNode(node *yaml.Node) *yaml.Node {
	node = unalias(node)
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!str" || node.Value == "" {
		return nil
	}
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "$ref"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: node.Value},
	}}
}

func validateV1Alpha4MutationContracts(req BuildRequest, skill *V1Alpha4SkillAST, identity dependencyIdentityResult) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if skill == nil {
		return diagnostics
	}
	snapshots := make(map[string]ResolvedDependency, len(identity.Closure))
	for _, alias := range identity.Closure {
		for _, dependency := range req.ResolvedDependencies {
			if dependency.DependencyID == alias {
				snapshots[alias] = dependency
			}
		}
	}
	for _, alias := range sortedStringMapKeys(snapshots) {
		index, _, _ := buildSnapshotExportIndex(snapshots[alias])
		for _, exportID := range sortedStringMapKeys(index.callables) {
			for _, export := range index.callables[exportID] {
				if export.Effect != "MUTATION" || export.MutationProblem == "" {
					continue
				}
				diagnostics = append(diagnostics, dependencyDiagnostic(
					"error", "C_STATIC", "RECONCILE_CONTRACT_INVALID", "", nil,
					"Trusted Mutation contract is incomplete for callable "+export.ExportID+": "+export.MutationProblem))
			}
		}
	}
	sortDiagnostics(diagnostics)
	return diagnostics
}
