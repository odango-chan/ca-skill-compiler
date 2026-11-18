package compiler

// WP7 slice 1: Evidence requirement parameter binding (risk-reduced).
//
// Only the frozen rule that an Evidence requirement must reference a declared
// Predicate parameter is enforced here (PREDICATE_EVIDENCE_REQUIRED). Descriptor
// content, Producer/Scope/Correlation/Freshness and trusted relation checks are
// NOT implemented because no Evidence Descriptor physical format is frozen; they
// remain fail-closed.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validateEvidenceRequirements(predicatesPath string, source *PredicateSetAST, add func(string, *yaml.Node, string, string)) {
	if source == nil {
		return
	}
	for _, predicateID := range sortedStringMapKeys(source.Definitions) {
		predicate := source.Definitions[predicateID]
		if predicate == nil || predicate.Evidence == nil {
			continue
		}
		declared := map[string]bool{}
		for _, parameter := range sequence(predicate.Parameters) {
			parameter = unalias(parameter)
			if parameter == nil || parameter.Kind != yaml.MappingNode {
				continue
			}
			for i := 0; i+1 < len(parameter.Content); i += 2 {
				if name := scalar(unalias(parameter.Content[i])); name != "" {
					declared[name] = true
				}
			}
		}
		requires := unalias(child(predicate.Evidence, "requires"))
		if requires != nil && requires.Kind == yaml.MappingNode {
			validateEvidenceGuaranteeReferences(requires, predicatesPath, add)
			for i := 0; i+1 < len(requires.Content); i += 2 {
				name := scalar(unalias(requires.Content[i]))
				if name == "" {
					continue
				}
				if !declared[name] {
					add(predicatesPath, unalias(requires.Content[i]), "PREDICATE_EVIDENCE_REQUIRED",
						"Evidence requirement references a parameter that the Predicate does not declare: "+name)
				}
			}
		}
		for _, group := range sequence(child(predicate.Evidence, "independence")) {
			for _, member := range sequence(child(group, "members")) {
				if guarantee := unalias(child(member, "guarantee")); guarantee != nil && !validEvidenceGuaranteeReference(guarantee) {
					add(predicatesPath, guarantee, "QUALIFIED_REFERENCE_INVALID",
						"Evidence independence guarantee must be an exact Alias::GuaranteeId reference")
				}
				name := scalar(child(member, "parameter"))
				if name != "" && !declared[name] {
					add(predicatesPath, unalias(child(member, "parameter")), "PREDICATE_EVIDENCE_REQUIRED",
						"Independence member references a parameter that the Predicate does not declare: "+name)
				}
			}
		}
	}
}

// validateEvidenceGuaranteeReferences checks only the frozen qualified-reference
// syntax of each declared guarantee id (CA-SL G.1 / Appendix H). Descriptor
// content and trusted relations remain WP7 fail-closed work.
func validateEvidenceGuaranteeReferences(requires *yaml.Node, predicatesPath string, add func(string, *yaml.Node, string, string)) {
	for i := 0; i+1 < len(requires.Content); i += 2 {
		for _, guarantee := range sequence(requires.Content[i+1]) {
			node := unalias(guarantee)
			if node == nil {
				continue
			}
			if !validEvidenceGuaranteeReference(node) {
				add(predicatesPath, node, "QUALIFIED_REFERENCE_INVALID",
					"Evidence guarantee must be an exact Alias::GuaranteeId reference")
			}
		}
	}
}

func validEvidenceGuaranteeReference(node *yaml.Node) bool {
	alias, exportID, ok := splitQualifiedReference(node)
	return ok && validDependencyID(alias) && validExportID(exportID)
}
