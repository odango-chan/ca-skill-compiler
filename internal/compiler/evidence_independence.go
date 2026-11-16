package compiler

// WP7 slice 4 / E106: structural independence-group linking.
//
// Each independence group must contain at least two distinct (parameter,
// guarantee) pairs that are both declared in evidence.requires. Structural
// violations report the frozen EVIDENCE_GUARANTEE_LINK_INVALID; the runtime
// same-underlying-record UNKNOWN outcome is Host-side and not emulated here.

import (
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validateV1Alpha4IndependenceGroups(predicatesPath string, source *V1Alpha4PredicateSetAST, add func(string, *yaml.Node, string, string)) {
	if source == nil {
		return
	}
	for _, predicateID := range sortedStringMapKeys(source.Definitions) {
		predicate := source.Definitions[predicateID]
		if predicate == nil || predicate.Evidence == nil {
			continue
		}
		requires := unalias(child(predicate.Evidence, "requires"))
		required := map[string]map[string]bool{}
		if requires != nil && requires.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(requires.Content); i += 2 {
				parameter := scalar(unalias(requires.Content[i]))
				if parameter == "" {
					continue
				}
				values := map[string]bool{}
				for _, guarantee := range sequence(requires.Content[i+1]) {
					if text := normalizedReference(guarantee); text != "" {
						values[text] = true
					}
				}
				required[parameter] = values
			}
		}
		for _, group := range sequence(child(predicate.Evidence, "independence")) {
			members := sequence(child(group, "members"))
			if len(members) < 2 {
				add(predicatesPath, unalias(group), "EVIDENCE_GUARANTEE_LINK_INVALID",
					"Evidence independence group must contain at least two (parameter, guarantee) pairs")
				continue
			}
			seen := map[string]bool{}
			for _, member := range members {
				parameter := scalar(child(member, "parameter"))
				guarantee := normalizedReference(child(member, "guarantee"))
				if parameter == "" || guarantee == "" {
					add(predicatesPath, unalias(member), "EVIDENCE_GUARANTEE_LINK_INVALID",
						"Evidence independence member must declare exactly one parameter and guarantee pair")
					continue
				}
				pair := parameter + "\x00" + guarantee
				if seen[pair] {
					add(predicatesPath, unalias(member), "EVIDENCE_GUARANTEE_LINK_INVALID",
						"Evidence independence member duplicates another member pair")
					continue
				}
				seen[pair] = true
				if !required[parameter][guarantee] {
					add(predicatesPath, unalias(member), "EVIDENCE_GUARANTEE_LINK_INVALID",
						"Evidence independence member pair is not declared by evidence.requires")
				}
			}
		}
	}
}

func normalizedReference(node *yaml.Node) string {
	node = unalias(node)
	if node == nil {
		return ""
	}
	return strings.TrimSpace(scalar(node))
}
