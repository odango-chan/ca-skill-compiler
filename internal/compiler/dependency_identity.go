package compiler

// WP3A: DependencySnapshot identity and trust binding.
//
// For every used Contract Dependency Alias this file proves exactly one
// relation:
//
//	Contract Alias -> one ResolvedDependency -> original-byte snapshot digest
//	-> one TrustedDependencyBinding -> exact Stable Package ID
//
// It deliberately does not parse, index or expose snapshot exports. ToolCatalog
// callable linking, Owner attribution and Schema linking belong to WP3B/WP3D.
// The only normative authority is ca-skills@7f84393, §§24.1 and 34.0-34.0.2.

import (
	"sort"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

// dependencyIdentity is the verified WP3A identity tuple for one used Alias.
type dependencyIdentity struct {
	Alias           string
	MeasuredDigest  string
	StablePackageID string
}

type declaredDependency struct {
	alias       string
	packageID   string
	keyNode     *yaml.Node
	valueNode   *yaml.Node
	requirement bool
}

type dependencyReference struct {
	alias      string
	exportKind string
	exportID   string
	path       string
	node       *yaml.Node
}

type dependencyIdentityResult struct {
	Diagnostics []Diagnostic
	Identities  []dependencyIdentity
	Closure     []string
	Bindings    map[string]TrustedDependencyBinding
}

// validateV1Alpha4DependencyIdentity implements the WP3A slice for one selected
// Skill. Check and Build both reach it through validateV1Alpha4, so the identity
// check cannot diverge between the two public operations.
func validateV1Alpha4DependencyIdentity(req BuildRequest, skill *V1Alpha4SkillAST) dependencyIdentityResult {
	result := dependencyIdentityResult{
		Diagnostics: []Diagnostic{},
		Identities:  []dependencyIdentity{},
		Closure:     []string{},
		Bindings:    map[string]TrustedDependencyBinding{},
	}
	if skill == nil || skill.Contract == nil || skill.Contract.Raw == nil {
		return result
	}
	contractPath := "skills/" + skill.Name + "/assurance/contract.yaml"

	declared := collectContractDependencies(contractPath, skill.Contract.Dependencies, &result.Diagnostics)
	references := collectDependencyReferences(skill)
	used := make(map[string]bool, len(declared))
	for _, reference := range references {
		if _, exists := declared[reference.alias]; !exists {
			result.Diagnostics = append(result.Diagnostics, dependencyDiagnostic(
				"error", "C_STATIC", "DEPENDENCY_ALIAS_MISSING", reference.path, reference.node,
				"External reference uses a Dependency Alias that the selected Skill Contract does not declare: "+reference.alias))
			continue
		}
		used[reference.alias] = true
	}

	aliases := make([]string, 0, len(declared))
	for alias := range declared {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	for _, alias := range aliases {
		entry := declared[alias]
		if !used[alias] {
			result.Diagnostics = append(result.Diagnostics, dependencyDiagnostic(
				"warning", "C_WARNING", "UNUSED_DEPENDENCY", contractPath, entry.keyNode,
				"Contract declares a Dependency Alias that the selected Skill does not reference: "+alias))
			continue
		}
		if !entry.requirement {
			// The dependency value itself was already rejected with
			// DEPENDENCY_REQUIREMENT_INVALID; no identity can be inferred from it.
			continue
		}
		result.Diagnostics = append(result.Diagnostics, resolveUsedDependencyIdentity(req, entry, contractPath, &result)...)
	}

	sort.Slice(result.Identities, func(i, j int) bool { return result.Identities[i].Alias < result.Identities[j].Alias })
	sort.Strings(result.Closure)
	return result
}

// collectContractDependencies reads the Contract.dependencies mapping from the
// normalized WP2 AST. Keys are Dependency Aliases; values must be Stable Package
// ID strings. It never infers an alias from a filename, URL or display name.
func collectContractDependencies(contractPath string, dependencies *yaml.Node, diagnostics *[]Diagnostic) map[string]declaredDependency {
	declared := map[string]declaredDependency{}
	dependencies = unalias(dependencies)
	if dependencies == nil {
		return declared
	}
	if dependencies.Kind != yaml.MappingNode {
		*diagnostics = append(*diagnostics, dependencyDiagnostic(
			"error", "C_STATIC", "DEPENDENCY_REQUIREMENT_INVALID", contractPath, dependencies,
			"Contract.dependencies must be a mapping from Dependency Alias to Stable Package ID"))
		return declared
	}
	for i := 0; i+1 < len(dependencies.Content); i += 2 {
		key := unalias(dependencies.Content[i])
		value := unalias(dependencies.Content[i+1])
		if key == nil {
			continue
		}
		alias := key.Value
		if !validDependencyID(alias) {
			*diagnostics = append(*diagnostics, dependencyDiagnostic(
				"error", "C_STATIC", "IDENTIFIER_INVALID", contractPath, key,
				"Contract dependency key must satisfy the v1alpha4 Identifier grammar: "+alias))
			continue
		}
		entry := declaredDependency{alias: alias, keyNode: key, valueNode: value}
		if value == nil || value.Kind != yaml.ScalarNode || value.Tag != "!!str" || !validStablePackageID(value.Value) {
			*diagnostics = append(*diagnostics, dependencyDiagnostic(
				"error", "C_STATIC", "DEPENDENCY_REQUIREMENT_INVALID", contractPath, value,
				"Contract dependency value must be a Stable Package ID string (RFC 3986 absolute URI): "+alias))
		} else {
			entry.packageID = value.Value
			entry.requirement = true
		}
		declared[alias] = entry
	}
	return declared
}

// collectDependencyReferences scans only the frozen v1alpha4 source positions
// that can carry an external reference. It never searches arbitrary YAML strings
// for "::": callable references come from Operation/Observation tool values,
// external Schema references come from Contract.contextSchema, Predicate
// parameter types and Skill-owned Schema $ref values, and Evidence Guarantee
// references come from PredicateSet evidence requires/independence.
func collectDependencyReferences(skill *V1Alpha4SkillAST) []dependencyReference {
	references := make([]dependencyReference, 0)
	contractPath := "skills/" + skill.Name + "/assurance/contract.yaml"
	if skill.Contract != nil {
		if alias, exportID, ok := splitQualifiedReference(skill.Contract.ContextSchema); ok && validStablePackageID(exportID) {
			references = append(references, dependencyReference{alias: alias, exportKind: "schema", exportID: exportID, path: contractPath, node: unalias(skill.Contract.ContextSchema)})
		}
	}
	if skill.Procedure != nil {
		procedurePath := "skills/" + skill.Name + "/assurance/procedure.yaml"
		for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
			node := skill.Procedure.ByID[nodeID]
			if node == nil {
				continue
			}
			if node.Operation != nil {
				if alias, exportID, ok := splitQualifiedReference(node.Operation.Tool); ok && validExportID(exportID) {
					references = append(references, dependencyReference{alias: alias, exportKind: "operation", exportID: exportID, path: procedurePath, node: unalias(node.Operation.Tool)})
				}
			}
			if node.Observation != nil {
				if alias, exportID, ok := splitQualifiedReference(node.Observation.Tool); ok && validExportID(exportID) {
					references = append(references, dependencyReference{alias: alias, exportKind: "observation", exportID: exportID, path: procedurePath, node: unalias(node.Observation.Tool)})
				}
			}
		}
	}
	if skill.Predicates != nil {
		predicatesPath := "skills/" + skill.Name + "/assurance/predicates.yaml"
		for _, predicateID := range sortedStringMapKeys(skill.Predicates.Definitions) {
			predicate := skill.Predicates.Definitions[predicateID]
			if predicate == nil {
				continue
			}
			references = append(references, collectPredicateParameterReferences(predicate.Parameters, predicatesPath)...)
			references = append(references, collectEvidenceGuaranteeReferences(predicate.Evidence, predicatesPath)...)
		}
	}
	base := "skills/" + skill.Name + "/schemas/"
	for _, filename := range sortedStringMapKeys(skill.Files) {
		if !strings.HasPrefix(filename, base) || !(strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")) {
			continue
		}
		references = append(references, collectSchemaRefReferences(skill.Files[filename], filename)...)
	}
	return references
}

func collectPredicateParameterReferences(parameters *yaml.Node, predicatesPath string) []dependencyReference {
	references := make([]dependencyReference, 0)
	for _, parameter := range sequence(parameters) {
		parameter = unalias(parameter)
		if parameter == nil || parameter.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(parameter.Content); i += 2 {
			value := unalias(parameter.Content[i+1])
			if alias, exportID, ok := splitQualifiedReference(value); ok && validStablePackageID(exportID) {
				references = append(references, dependencyReference{alias: alias, exportKind: "schema", exportID: exportID, path: predicatesPath, node: value})
			}
		}
	}
	return references
}

func collectEvidenceGuaranteeReferences(evidence *yaml.Node, predicatesPath string) []dependencyReference {
	references := make([]dependencyReference, 0)
	evidence = unalias(evidence)
	if evidence == nil || evidence.Kind != yaml.MappingNode {
		return references
	}
	requires := unalias(child(evidence, "requires"))
	if requires != nil && requires.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(requires.Content); i += 2 {
			for _, guarantee := range sequence(requires.Content[i+1]) {
				if alias, exportID, ok := splitQualifiedReference(guarantee); ok && validExportID(exportID) {
					references = append(references, dependencyReference{alias: alias, exportKind: "evidenceGuarantee", exportID: exportID, path: predicatesPath, node: unalias(guarantee)})
				}
			}
		}
	}
	for _, group := range sequence(child(evidence, "independence")) {
		for _, member := range sequence(child(group, "members")) {
			if alias, exportID, ok := splitQualifiedReference(child(member, "guarantee")); ok && validExportID(exportID) {
				references = append(references, dependencyReference{alias: alias, exportKind: "evidenceGuarantee", exportID: exportID, path: predicatesPath, node: unalias(child(member, "guarantee"))})
			}
		}
	}
	return references
}

// collectSchemaRefReferences inspects only "$ref" values in a Skill-owned Schema
// document. A relative or fragment reference is not a Dependency Alias.
func collectSchemaRefReferences(root *yaml.Node, schemaPath string) []dependencyReference {
	references := make([]dependencyReference, 0)
	var walk func(*yaml.Node)
	walk = func(node *yaml.Node) {
		node = unalias(node)
		if node == nil {
			return
		}
		switch node.Kind {
		case yaml.MappingNode:
			for i := 0; i+1 < len(node.Content); i += 2 {
				key := unalias(node.Content[i])
				value := unalias(node.Content[i+1])
				if key != nil && key.Value == "$ref" {
					if alias, exportID, ok := splitQualifiedReference(value); ok && validStablePackageID(exportID) {
						references = append(references, dependencyReference{alias: alias, exportKind: "schema", exportID: exportID, path: schemaPath, node: value})
					}
				}
				walk(value)
			}
		case yaml.SequenceNode:
			for _, item := range node.Content {
				walk(item)
			}
		}
	}
	walk(root)
	return references
}

// splitQualifiedReference splits a qualified reference at the FIRST "::" exactly
// as the frozen grammar requires, so Schema URIs keep their own colons. An
// invalid Alias or empty export part is left to the later
// QUALIFIED_REFERENCE_INVALID work package instead of being guessed here.
func splitQualifiedReference(node *yaml.Node) (string, string, bool) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return "", "", false
	}
	index := strings.Index(node.Value, "::")
	if index <= 0 {
		return "", "", false
	}
	alias := node.Value[:index]
	exportID := node.Value[index+2:]
	if !validDependencyID(alias) || exportID == "" {
		return "", "", false
	}
	return alias, exportID, true
}

// resolveUsedDependencyIdentity proves the full WP3A relation for one used Alias
// and records the verified identity tuple.
func resolveUsedDependencyIdentity(req BuildRequest, entry declaredDependency, contractPath string, result *dependencyIdentityResult) []Diagnostic {
	diagnostics := make([]Diagnostic, 0, 1)
	var snapshots []ResolvedDependency
	for _, dependency := range req.ResolvedDependencies {
		if dependency.DependencyID == entry.alias {
			snapshots = append(snapshots, dependency)
		}
	}
	if len(snapshots) == 0 {
		return append(diagnostics, dependencyDiagnostic(
			"error", "C_STATIC", "DEPENDENCY_SNAPSHOT_MISSING", "", nil,
			"No supplied immutable DependencySnapshot has the used Dependency Alias as its dependencyId: "+entry.alias))
	}
	if len(snapshots) > 1 {
		return append(diagnostics, dependencyDiagnostic(
			"error", "C_STATIC", "DEPENDENCY_DUPLICATE", "", nil,
			"Multiple DependencySnapshots claim the same used Dependency Alias: "+entry.alias))
	}
	snapshot := snapshots[0]
	measuredDigest, err := ComputeDependencySnapshotDigest(snapshot.Files)
	if err != nil || measuredDigest != snapshot.Digest {
		return append(diagnostics, dependencyDiagnostic(
			"error", "C_STATIC", "DEPENDENCY_IDENTITY_UNVERIFIED", "", nil,
			"Used DependencySnapshot bytes do not match the supplied digest: "+entry.alias))
	}
	var bindings []TrustedDependencyBinding
	if req.TrustedBuildContext != nil {
		for _, binding := range req.TrustedBuildContext.DependencyBindings {
			if binding.DependencyID == entry.alias && binding.Digest == measuredDigest {
				bindings = append(bindings, binding)
			}
		}
	}
	if len(bindings) == 0 {
		return append(diagnostics, dependencyDiagnostic(
			"error", "C_STATIC", "DEPENDENCY_IDENTITY_UNVERIFIED", "", nil,
			"No trusted dependency binding matches the used Dependency Alias and measured snapshot digest: "+entry.alias))
	}
	if len(bindings) > 1 {
		return append(diagnostics, dependencyDiagnostic(
			"error", "C_STATIC", "DEPENDENCY_DUPLICATE", "", nil,
			"Multiple trusted dependency bindings match the same Alias and measured snapshot digest: "+entry.alias))
	}
	binding := bindings[0]
	if binding.StablePackageID != entry.packageID {
		return append(diagnostics, dependencyDiagnostic(
			"error", "C_STATIC", "DEPENDENCY_ID_MISMATCH", contractPath, entry.valueNode,
			"Trusted Stable Package ID differs from the Contract dependency value for Alias "+entry.alias))
	}
	result.Identities = append(result.Identities, dependencyIdentity{
		Alias:           entry.alias,
		MeasuredDigest:  measuredDigest,
		StablePackageID: entry.packageID,
	})
	result.Closure = append(result.Closure, entry.alias)
	result.Bindings[entry.alias] = binding
	return diagnostics
}

// validStablePackageID accepts an RFC 3986 absolute URI with a scheme. It rejects
// whitespace, control characters, backslashes and unresolved dot-segments; it
// never case-folds or normalizes the identity.
func validStablePackageID(value string) bool {
	colon := strings.IndexByte(value, ':')
	if colon <= 0 || !asciiLetter(value[0]) {
		return false
	}
	for i := 1; i < colon; i++ {
		c := value[i]
		if !asciiLetter(c) && !(c >= '0' && c <= '9') && c != '+' && c != '.' && c != '-' {
			return false
		}
	}
	if colon+1 >= len(value) {
		return false
	}
	for i := colon + 1; i < len(value); i++ {
		c := value[i]
		if c <= 0x20 || c == 0x7f || c == '\\' {
			return false
		}
	}
	pathPart := value[colon+1:]
	if index := strings.IndexAny(pathPart, "?#"); index >= 0 {
		pathPart = pathPart[:index]
	}
	for _, segment := range strings.Split(pathPart, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func dependencyDiagnostic(severity, phase, code, sourcePath string, node *yaml.Node, message string) Diagnostic {
	line, column := 0, 0
	if node != nil {
		line, column = node.Line, node.Column
	}
	return Diagnostic{
		Severity: severity,
		Phase:    phase,
		Code:     code,
		Path:     limitDiagnosticPath(sourcePath),
		Line:     line,
		Column:   column,
		Message:  limitDiagnosticMessage(message),
	}
}

func hasErrorDiagnostic(diagnostics []Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			return true
		}
	}
	return false
}
