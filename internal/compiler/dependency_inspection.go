package compiler

// Package-scoped snapshot export inventory shared by DependencyInspection.
//
// DependencyInspection is snapshot-scoped, not Skill-closure-scoped. It validates
// the supplied immutable snapshots and trusted dependency/Owner bindings and lists
// only exports whose identity and Owner can be established. It never resolves
// author references, evaluates Capability Grants, authorizes invocation or
// publishes dependency bytes/digests.

func inspectSnapshotExports(snapshot ResolvedDependency, binding TrustedDependencyBinding) ([]InspectedCallable, []InspectedSchema, []Diagnostic) {
	callables := make([]InspectedCallable, 0)
	schemas := make([]InspectedSchema, 0)
	diagnostics := make([]Diagnostic, 0)

	index, catalogs, ambiguity := buildSnapshotExportIndex(snapshot)
	diagnostics = append(diagnostics, ambiguity...)

	authorized := make(map[string]bool, len(binding.AuthorizedOwners))
	for _, owner := range binding.AuthorizedOwners {
		authorized[owner] = true
	}
	untrustedCatalog := make(map[string]bool, len(catalogs))
	for _, catalog := range catalogs {
		if !authorized[catalog.owner] {
			untrustedCatalog[catalog.path] = true
			diagnostics = append(diagnostics, dependencyDiagnostic(
				"error", "C_STATIC", "DEPENDENCY_OWNER_MISMATCH", "", nil,
				"ToolCatalog metadata.owner is not in the trusted authorized Owners for dependency "+snapshot.DependencyID))
		}
	}

	for _, exportID := range sortedStringMapKeys(index.callables) {
		records := index.callables[exportID]
		if len(records) > 1 {
			// EXPORT_ID_AMBIGUOUS was already emitted; ambiguous exports are omitted.
			continue
		}
		record := records[0]
		if untrustedCatalog[record.SourcePath] {
			continue
		}
		owner, ownershipDiagnostics := attributeExportOwner(record, binding, untrustedCatalog)
		if len(ownershipDiagnostics) > 0 || owner == "" {
			diagnostics = append(diagnostics, ownershipDiagnostics...)
			continue
		}
		callables = append(callables, InspectedCallable{
			DependencyID:         snapshot.DependencyID,
			Owner:                owner,
			Kind:                 record.ExportKind,
			ID:                   record.ExportID,
			Effect:               record.Effect,
			SourcePath:           record.SourcePath,
			InputSchemaID:        record.InputSchemaID,
			QueryContextSchemaID: record.QueryContextSchemaID,
			OutputSchemaID:       record.OutputSchemaID,
		})
	}

	for _, exportID := range sortedStringMapKeys(index.schemas) {
		records := index.schemas[exportID]
		if len(records) > 1 {
			continue
		}
		record := records[0]
		owner, ownershipDiagnostics := attributeExportOwner(record, binding, nil)
		if len(ownershipDiagnostics) > 0 || owner == "" {
			diagnostics = append(diagnostics, ownershipDiagnostics...)
			continue
		}
		schemas = append(schemas, InspectedSchema{
			DependencyID: snapshot.DependencyID,
			Owner:        owner,
			ID:           record.ExportID,
			SourcePath:   record.SourcePath,
		})
	}
	return callables, schemas, diagnostics
}
