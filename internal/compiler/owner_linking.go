package compiler

// WP3B: ToolCatalog export identity and trusted Owner attribution.
//
// For every DependencySnapshot already proven by WP3A for the selected Skill,
// this file indexes the exact callable and Schema exports and attributes each
// actually referenced export to a trusted Owner. It adds no language semantics,
// no diagnostic code and no Owner format of its own. Capability membership,
// Schema type checking, Evidence descriptors and DependencyInspection remain in
// WP3C/WP4/WP7/WP3D and stay fail-closed.

import (
	"sort"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

const (
	toolCatalogAPIVersion = "connectionagent.io/toolcatalog/v1alpha4"
	toolCatalogKind       = "ToolCatalog"
)

// linkedExport is the identity metadata WP3B proves for one actually referenced
// export. It is retained for WP3C/WP3D and never published as a DependencyInspection.
type linkedExport struct {
	DependencyAlias      string
	ExportKind           string
	ExportID             string
	TrustedOwner         string
	SelfDeclaredOwner    string
	SourcePath           string
	InputSchemaID        string
	QueryContextSchemaID string
	OutputSchemaID       string
	ReceiptSchemaID      string
	OwnerOutcomes        []string
	Effect               string
	// ReconcileTarget is the same-snapshot dedicated Reconcile READ ExportId
	// declared by a Mutation contract. It is never an ordinary author callable.
	ReconcileTarget string
	// MutationProblem records the first missing/invalid Mutation-contract identity
	// requirement (WP8 slice 1); empty means the identity requirements are present.
	MutationProblem string
	// MutationFields carries the trusted Mutation-contract identities used by the
	// deeper WP8 static checks (receipt/commit-proof/Reconcile/recoverable).
	MutationFields mutationContractFields
	// EvidenceProvides lists the F.1 guarantees.evidence.provides entries published
	// by this callable, used by static Predicate Evidence satisfiability checks.
	EvidenceProvides []evidenceProvide
}

type ownerLinkingResult struct {
	Diagnostics []Diagnostic
	Exports     []linkedExport
}

type snapshotExportIndex struct {
	callables map[string][]linkedExport
	schemas   map[string][]linkedExport
	evidence  map[string][]evidenceDescriptor
}

type catalogDocument struct {
	owner     string
	path      string
	callables []linkedExport
}

func validateV1Alpha4OwnerLinking(req BuildRequest, skill *V1Alpha4SkillAST, identity dependencyIdentityResult) ownerLinkingResult {
	result := ownerLinkingResult{Diagnostics: []Diagnostic{}, Exports: []linkedExport{}}
	if skill == nil || skill.Contract == nil || skill.Contract.Raw == nil {
		return result
	}
	snapshots := make(map[string]ResolvedDependency, len(identity.Closure))
	for _, alias := range identity.Closure {
		for _, dependency := range req.ResolvedDependencies {
			if dependency.DependencyID == alias {
				snapshots[alias] = dependency
			}
		}
	}
	references := make([]dependencyReference, 0)
	for _, reference := range collectDependencyReferences(skill) {
		if _, verified := snapshots[reference.alias]; verified {
			references = append(references, reference)
		}
	}
	if len(references) == 0 {
		return result
	}
	// Deterministic diagnostic order independent of reference scan order.
	sort.SliceStable(references, func(i, j int) bool {
		a, b := references[i], references[j]
		if a.path != b.path {
			return a.path < b.path
		}
		if a.node.Line != b.node.Line {
			return a.node.Line < b.node.Line
		}
		if a.node.Column != b.node.Column {
			return a.node.Column < b.node.Column
		}
		if a.exportKind != b.exportKind {
			return a.exportKind < b.exportKind
		}
		return a.exportID < b.exportID
	})

	indexes := make(map[string]snapshotExportIndex, len(snapshots))
	dedicatedReconcile := make(map[string]map[string]bool, len(snapshots))
	catalogOwners := make(map[string]map[string]bool) // alias -> catalog path -> untrusted already reported
	for _, alias := range sortedStringMapKeys(snapshots) {
		index, catalogDocuments, ambiguity := buildSnapshotExportIndex(snapshots[alias])
		indexes[alias] = index
		if len(ambiguity) > 0 {
			result.Diagnostics = append(result.Diagnostics, ambiguity...)
		}
		binding := identity.Bindings[alias]
		authorized := make(map[string]bool, len(binding.AuthorizedOwners))
		for _, owner := range binding.AuthorizedOwners {
			authorized[owner] = true
		}
		flagged := make(map[string]bool, len(catalogDocuments))
		for _, catalog := range catalogDocuments {
			if !authorized[catalog.owner] {
				flagged[catalog.path] = true
				result.Diagnostics = append(result.Diagnostics, dependencyDiagnostic(
					"error", "C_STATIC", "DEPENDENCY_OWNER_MISMATCH", "", nil,
					"ToolCatalog metadata.owner is not in the trusted authorized Owners for dependency alias "+alias))
			}
		}
		catalogOwners[alias] = flagged
		dedicated := make(map[string]bool)
		for _, records := range index.callables {
			for _, record := range records {
				if record.ReconcileTarget != "" {
					dedicated[record.ReconcileTarget] = true
				}
			}
		}
		dedicatedReconcile[alias] = dedicated
	}
	seenExports := make(map[string]bool, len(references))
	for _, reference := range references {
		alias := reference.alias
		binding := identity.Bindings[alias]
		index := indexes[alias]
		if reference.exportKind == "operation" && dedicatedReconcile[alias][reference.exportID] {
			result.Diagnostics = append(result.Diagnostics, dependencyDiagnostic(
				"error", "C_STATIC", "RECONCILE_CONTRACT_INVALID", reference.path, reference.node,
				"A dedicated same-snapshot Reconcile READ callable cannot be called as an ordinary operation.tool: "+reference.exportID))
			continue
		}
		var matched linkedExport
		switch reference.exportKind {
		case "operation", "observation":
			records := index.callables[reference.exportID]
			if len(records) > 1 {
				// EXPORT_ID_AMBIGUOUS was already emitted while indexing.
				continue
			}
			if len(records) == 0 {
				result.Diagnostics = append(result.Diagnostics, dependencyDiagnostic(
					"error", "C_STATIC", "EXPORT_ID_MISSING", reference.path, reference.node,
					"ToolCatalog callable export does not exist exactly once in the selected snapshot: "+reference.exportID))
				continue
			}
			matched = records[0]
			if reference.exportKind == "observation" && matched.Effect != "READ" {
				result.Diagnostics = append(result.Diagnostics, dependencyDiagnostic(
					"error", "C_STATIC", "EXPORT_ID_MISSING", reference.path, reference.node,
					"ToolCatalog callable export does not exist with the required kind and Effect: "+reference.exportID))
				continue
			}
			if matched.ExportKind != reference.exportKind {
				result.Diagnostics = append(result.Diagnostics, dependencyDiagnostic(
					"error", "C_STATIC", "EXPORT_ID_MISSING", reference.path, reference.node,
					"ToolCatalog callable export does not exist with the required kind: "+reference.exportID))
				continue
			}
		case "schema":
			records := index.schemas[reference.exportID]
			if len(records) > 1 {
				// EXPORT_ID_AMBIGUOUS was already emitted while indexing.
				continue
			}
			if len(records) == 0 {
				result.Diagnostics = append(result.Diagnostics, dependencyDiagnostic(
					"error", "C_STATIC", "SCHEMA_MISSING", reference.path, reference.node,
					"Dependency snapshot does not contain exactly one Schema with the referenced $id: "+reference.exportID))
				continue
			}
			matched = records[0]
		case "evidenceGuarantee":
			// The Evidence Descriptor authority is the trusted Owner of the
			// guarantee; missing/ambiguous descriptors are diagnosed by the
			// positive-path Evidence Descriptor linking slice.
			matched = linkedExport{ExportKind: "evidenceGuarantee", ExportID: reference.exportID}
			if descriptors := index.evidence[reference.exportID]; len(descriptors) == 1 {
				matched.SelfDeclaredOwner = descriptors[0].Owner
			}
		default:
			continue
		}
		owner, ownershipDiagnostics := attributeExportOwner(matched, binding, catalogOwners[alias])
		result.Diagnostics = append(result.Diagnostics, ownershipDiagnostics...)
		if len(ownershipDiagnostics) > 0 && owner == "" {
			continue
		}
		matched.DependencyAlias = alias
		matched.ExportKind = reference.exportKind
		matched.ExportID = reference.exportID
		matched.TrustedOwner = owner
		key := alias + "\x00" + reference.exportKind + "\x00" + reference.exportID
		if !seenExports[key] {
			seenExports[key] = true
			result.Exports = append(result.Exports, matched)
		}
	}
	sort.Slice(result.Exports, func(i, j int) bool {
		a, b := result.Exports[i], result.Exports[j]
		if a.DependencyAlias != b.DependencyAlias {
			return a.DependencyAlias < b.DependencyAlias
		}
		if a.ExportKind != b.ExportKind {
			return a.ExportKind < b.ExportKind
		}
		return a.ExportID < b.ExportID
	})
	sortDiagnostics(result.Diagnostics)
	return result
}

// attributeExportOwner applies the frozen single-Owner / multi-Owner attribution
// rules. It never infers an Owner from a path, prefix or display name.
func attributeExportOwner(export linkedExport, binding TrustedDependencyBinding, untrustedCatalogOwners map[string]bool) (string, []Diagnostic) {
	diagnostics := make([]Diagnostic, 0, 1)
	authorized := make(map[string]bool, len(binding.AuthorizedOwners))
	for _, owner := range binding.AuthorizedOwners {
		authorized[owner] = true
	}
	var explicit []ExportOwner
	for _, exportOwner := range binding.ExportOwners {
		if exportOwner.ExportKind == export.ExportKind && exportOwner.ExportID == export.ExportID {
			explicit = append(explicit, exportOwner)
		}
	}
	if len(explicit) > 1 {
		return "", []Diagnostic{dependencyDiagnostic(
			"error", "C_STATIC", "DEPENDENCY_EXPORT_OWNER_UNVERIFIED", "", nil,
			"Multiple trusted export-owner records claim the same export identity: "+export.ExportKind+" "+export.ExportID)}
	}
	if len(binding.AuthorizedOwners) > 1 {
		if len(explicit) == 0 {
			return "", []Diagnostic{dependencyDiagnostic(
				"error", "C_STATIC", "DEPENDENCY_EXPORT_OWNER_UNVERIFIED", "", nil,
				"Multi-Owner snapshot has no trusted export-owner attribution for the referenced export: "+export.ExportKind+" "+export.ExportID)}
		}
		owner := explicit[0].Owner
		if !authorized[owner] {
			return "", []Diagnostic{dependencyDiagnostic(
				"error", "C_STATIC", "DEPENDENCY_OWNER_MISMATCH", "", nil,
				"Trusted export Owner is not in the authorized Owner set: "+owner)}
		}
		if export.SelfDeclaredOwner != "" && export.SelfDeclaredOwner != owner {
			return "", []Diagnostic{dependencyDiagnostic(
				"error", "C_STATIC", "DEPENDENCY_OWNER_MISMATCH", "", nil,
				"Export self-declared Owner differs from the trusted attribution: "+export.ExportID)}
		}
		return owner, diagnostics
	}
	owner := binding.AuthorizedOwners[0]
	if len(explicit) == 1 && explicit[0].Owner != owner {
		return "", []Diagnostic{dependencyDiagnostic(
			"error", "C_STATIC", "DEPENDENCY_OWNER_MISMATCH", "", nil,
			"Trusted export-owner record conflicts with the single authorized Owner: "+export.ExportID)}
	}
	if export.SelfDeclaredOwner != "" && export.SelfDeclaredOwner != owner && !untrustedCatalogOwners[export.SourcePath] {
		return "", []Diagnostic{dependencyDiagnostic(
			"error", "C_STATIC", "DEPENDENCY_OWNER_MISMATCH", "", nil,
			"Export self-declared Owner differs from the single authorized Owner: "+export.ExportID)}
	}
	return owner, diagnostics
}

// buildSnapshotExportIndex parses the ToolCatalog and Schema documents of one
// verified snapshot through the shared WP2 AST. Malformed identity envelopes are
// not indexed; a referenced identity then fails with EXPORT_ID_MISSING or
// SCHEMA_MISSING. Full Owner-contract validation remains a later work package.
func buildSnapshotExportIndex(snapshot ResolvedDependency) (snapshotExportIndex, []catalogDocument, []Diagnostic) {
	index := snapshotExportIndex{callables: map[string][]linkedExport{}, schemas: map[string][]linkedExport{}, evidence: map[string][]evidenceDescriptor{}}
	diagnostics := make([]Diagnostic, 0)
	catalogs := make([]catalogDocument, 0)
	files := make([]File, len(snapshot.Files))
	copy(files, snapshot.Files)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	for _, file := range files {
		if !strings.HasSuffix(file.Path, ".yaml") && !strings.HasSuffix(file.Path, ".yml") && !strings.HasSuffix(file.Path, ".json") {
			continue
		}
		root, err := yaml.Parse(file.Bytes)
		if err != nil {
			continue
		}
		if catalog, ok := parseToolCatalog(root, file.Path); ok && catalog != nil {
			catalogs = append(catalogs, *catalog)
			for _, callable := range catalog.callables {
				index.callables[callable.ExportID] = append(index.callables[callable.ExportID], callable)
			}
			continue
		}
		if export, ok := parseSchemaExport(root, file.Path); ok {
			index.schemas[export.ExportID] = append(index.schemas[export.ExportID], export)
			continue
		}
		if descriptors, ok := parseEvidenceDescriptors(root); ok {
			for exportID, list := range descriptors {
				for position := range list {
					list[position].SnapshotPath = file.Path
				}
				index.evidence[exportID] = append(index.evidence[exportID], list...)
			}
		}
	}
	callableIDs := sortedStringMapKeys(index.callables)
	for _, exportID := range callableIDs {
		if len(index.callables[exportID]) > 1 {
			diagnostics = append(diagnostics, dependencyDiagnostic(
				"error", "C_STATIC", "EXPORT_ID_AMBIGUOUS", "", nil,
				"Duplicate ToolCatalog callable ID in the selected snapshot: "+exportID))
		}
	}
	schemaIDs := sortedStringMapKeys(index.schemas)
	for _, exportID := range schemaIDs {
		if len(index.schemas[exportID]) > 1 {
			diagnostics = append(diagnostics, dependencyDiagnostic(
				"error", "C_STATIC", "EXPORT_ID_AMBIGUOUS", "", nil,
				"Duplicate Schema $id in the selected snapshot: "+exportID))
		}
	}
	return index, catalogs, diagnostics
}

func parseToolCatalog(root *yaml.Node, sourcePath string) (*catalogDocument, bool) {
	root = unalias(root)
	if root == nil || root.Kind != yaml.MappingNode {
		return nil, false
	}
	apiVersion := scalar(child(root, "apiVersion"))
	if apiVersion == "" {
		return nil, false
	}
	if apiVersion != toolCatalogAPIVersion || scalar(child(root, "kind")) != toolCatalogKind {
		// Unknown/unsupported imported-contract versions are not indexed; a
		// referenced export then fails with the frozen missing-export identity.
		return nil, false
	}
	metadata := child(root, "metadata")
	owner := scalar(child(metadata, "owner"))
	if owner == "" || len([]rune(owner)) > maxOwnerIDRunes {
		return nil, false
	}
	contracts := sequence(child(child(root, "spec"), "contracts"))
	if len(contracts) == 0 {
		return nil, false
	}
	catalog := &catalogDocument{owner: owner, path: sourcePath, callables: make([]linkedExport, 0, len(contracts))}
	for _, contract := range contracts {
		callable, ok := parseToolCatalogCallable(contract, owner, sourcePath)
		if !ok {
			// Malformed identity envelope: skip only this callable; the frozen
			// missing-export diagnostic applies if a Skill references it.
			continue
		}
		catalog.callables = append(catalog.callables, callable)
	}
	return catalog, true
}

func parseToolCatalogCallable(node *yaml.Node, owner, sourcePath string) (linkedExport, bool) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return linkedExport{}, false
	}
	kind := scalar(child(node, "kind"))
	if kind != "operation" && kind != "observation" {
		return linkedExport{}, false
	}
	exportID := scalar(child(node, "id"))
	if !validExportID(exportID) || len(exportID) > maxExportIDRunes {
		return linkedExport{}, false
	}
	effect := scalar(child(node, "effect"))
	if effect != "READ" && effect != "MUTATION" {
		return linkedExport{}, false
	}
	if kind == "observation" && effect != "READ" {
		return linkedExport{}, false
	}
	outcomes, ok := validOwnerOutcomes(sequence(child(node, "ownerOutcomes")))
	if !ok {
		return linkedExport{}, false
	}
	outputSchemaID, ok := schemaRefID(child(node, "outputSchema"))
	if !ok {
		return linkedExport{}, false
	}
	export := linkedExport{
		ExportKind: kind, ExportID: exportID, Effect: effect,
		SelfDeclaredOwner: owner, SourcePath: sourcePath,
		OutputSchemaID: outputSchemaID, OwnerOutcomes: outcomes,
	}
	switch kind {
	case "operation":
		inputSchemaID, ok := schemaRefID(child(node, "inputSchema"))
		if !ok {
			return linkedExport{}, false
		}
		export.InputSchemaID = inputSchemaID
		if receipt, ok := schemaRefID(child(node, "receiptSchema")); ok {
			export.ReceiptSchemaID = receipt
		}
		// F.1: Tool Operation exports may publish Evidence guarantees through the
		// same guarantees.evidence.provides logical field as Observations.
		export.EvidenceProvides = parseEvidenceProvides(child(node, "guarantees"))
		if effect == "MUTATION" {
			// G.2.1: the same-snapshot dedicated Reconcile READ callable.
			if reconcile := child(child(node, "mutation"), "reconcile"); reconcile != nil {
				if target := scalar(child(reconcile, "callable")); validExportID(target) && len(target) <= maxExportIDRunes {
					export.ReconcileTarget = target
				}
			}
			export.MutationFields = parseMutationContractFields(node)
			export.MutationProblem = mutationContractProblem(node)
		}
	case "observation":
		export.EvidenceProvides = parseEvidenceProvides(child(node, "guarantees"))
		queryContextSchemaID, ok := schemaRefID(child(node, "queryContextSchema"))
		if !ok {
			return linkedExport{}, false
		}
		export.QueryContextSchemaID = queryContextSchemaID
	}
	return export, true
}

func validOwnerOutcomes(outcomes []*yaml.Node) ([]string, bool) {
	if len(outcomes) == 0 || len(outcomes) > 3 {
		return nil, false
	}
	seen := map[string]bool{}
	values := make([]string, 0, len(outcomes))
	hasSucceeded := false
	for _, outcome := range outcomes {
		value := scalar(outcome)
		if seen[value] {
			return nil, false
		}
		switch value {
		case "SUCCEEDED":
			hasSucceeded = true
		case "FAILED", "NOT_COMMITTED":
		default:
			return nil, false
		}
		seen[value] = true
		values = append(values, value)
	}
	if !hasSucceeded {
		return nil, false
	}
	return values, true
}

func schemaRefID(node *yaml.Node) (string, bool) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return "", false
	}
	id := scalar(child(node, "$ref"))
	if !validStablePackageID(id) {
		return "", false
	}
	return id, true
}

func parseSchemaExport(root *yaml.Node, sourcePath string) (linkedExport, bool) {
	root = unalias(root)
	if root == nil || root.Kind != yaml.MappingNode {
		return linkedExport{}, false
	}
	if scalar(child(root, "$schema")) != "https://json-schema.org/draft/2020-12/schema" {
		return linkedExport{}, false
	}
	id := scalar(child(root, "$id"))
	if !validStablePackageID(id) {
		return linkedExport{}, false
	}
	return linkedExport{ExportKind: "schema", ExportID: id, SourcePath: sourcePath}, true
}

func validExportID(value string) bool {
	if value == "" {
		return false
	}
	for _, segment := range strings.Split(value, ".") {
		if segment == "" || !asciiLetter(segment[0]) {
			return false
		}
		for i := 1; i < len(segment); i++ {
			c := segment[i]
			if !asciiLetter(c) && !(c >= '0' && c <= '9') && c != '_' && c != '-' {
				return false
			}
		}
	}
	return true
}
