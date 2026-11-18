package compiler

// WP7 slice 3: static Predicate Evidence satisfiability.
//
// A PredicateCall supplying an argument for a parameter with declared
// evidence.requires must be able to statically satisfy every required guarantee:
// the argument must resolve to a producer Node whose linked Owner callable
// publishes that exact guarantee (same dependency alias and guarantee ID) with a
// path covering the referenced output. Request/Host data, literals, and
// producers without the published guarantee are PREDICATE_EVIDENCE_UNSATISFIABLE.
//
// This uses only the frozen F.1 guarantees.evidence.provides logical field; it
// does not invent an Evidence Descriptor physical format.

import (
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

type evidenceProvide struct {
	ID   string
	Path string
}

func parseEvidenceProvides(guarantees *yaml.Node) []evidenceProvide {
	guarantees = unalias(guarantees)
	if guarantees == nil {
		return nil
	}
	provides := make([]evidenceProvide, 0)
	for _, entry := range sequence(child(child(guarantees, "evidence"), "provides")) {
		entry = unalias(entry)
		if entry == nil || entry.Kind != yaml.MappingNode {
			continue
		}
		id := scalar(child(entry, "guarantee"))
		if id == "" {
			continue
		}
		provides = append(provides, evidenceProvide{ID: id, Path: scalar(child(entry, "path"))})
	}
	return provides
}

func validateV1Alpha4PredicateEvidence(req BuildRequest, skill *V1Alpha4SkillAST, ownerLinking ownerLinkingResult, procedurePath string, add func(string, *yaml.Node, string, string)) {
	if skill == nil || skill.Procedure == nil || skill.Predicates == nil {
		return
	}
	linked := map[string]linkedExport{}
	for _, export := range ownerLinking.Exports {
		linked[export.DependencyAlias+"::"+export.ExportID] = export
	}
	// Node -> linked callable export for its operation/observation stage.
	nodeProducer := map[string]linkedExport{}
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil {
			continue
		}
		var tool *yaml.Node
		if node.Operation != nil {
			tool = node.Operation.Tool
		}
		if node.Observation != nil {
			tool = node.Observation.Tool
		}
		if tool == nil {
			continue
		}
		if export, ok := linked[scalar(unalias(tool))]; ok {
			nodeProducer[nodeID] = export
		}
	}
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil {
			continue
		}
		for _, stage := range []*V1Alpha4PredicateStageAST{node.Precondition, node.Postcondition} {
			if stage == nil {
				continue
			}
			for _, call := range stage.Predicates {
				if call == nil {
					continue
				}
				predicateID := scalar(unalias(call.Predicate))
				predicate := skill.Predicates.Definitions[predicateID]
				if predicate == nil || predicate.Evidence == nil {
					continue
				}
				requires := unalias(child(predicate.Evidence, "requires"))
				if requires == nil || requires.Kind != yaml.MappingNode {
					continue
				}
				parameterIndex := predicateParameterIndex(predicate.Parameters)
				arguments := sequence(call.Arguments)
				declaredRequires := map[string]bool{}
				for i := 0; i+1 < len(requires.Content); i += 2 {
					declaredRequires[scalar(unalias(requires.Content[i]))] = true
				}
				for i := 0; i+1 < len(requires.Content); i += 2 {
					parameter := scalar(unalias(requires.Content[i]))
					index, ok := parameterIndex[parameter]
					if !ok || index >= len(arguments) {
						continue
					}
					argument := unalias(arguments[index])
					for _, guarantee := range sequence(requires.Content[i+1]) {
						alias, guaranteeID, ok := splitQualifiedReference(guarantee)
						if !ok {
							continue // qualified-reference syntax is diagnosed elsewhere
						}
						if !argumentCanProvide(argument, alias, guaranteeID, nodeProducer) {
							add(procedurePath, argument, "PREDICATE_EVIDENCE_UNSATISFIABLE",
								"Predicate argument cannot statically satisfy required Evidence guarantee "+alias+"::"+guaranteeID)
						}
					}
				}
				// §17.1: a parameter without any declared guarantee may only be
				// bound to exempt Request/Host/literal sources at every call point.
				for parameter, index := range parameterIndex {
					if declaredRequires[parameter] || index >= len(arguments) {
						continue
					}
					argument := unalias(arguments[index])
					if argumentNeedsEvidence(argument, skill) {
						add(procedurePath, argument, "PREDICATE_EVIDENCE_REQUIRED",
							"Predicate parameter "+parameter+" is bound to an external Producer output without a declared Evidence guarantee")
					}
				}
			}
		}
	}
}

// argumentNeedsEvidence reports whether one Predicate argument directly reads a
// non-exempt external Producer output (§17.1 source exemption table).
func argumentNeedsEvidence(argument *yaml.Node, skill *V1Alpha4SkillAST) bool {
	argument = unalias(argument)
	if argument == nil || argument.Kind != yaml.ScalarNode || argument.Tag != "!!str" {
		return false
	}
	if argument.Style == yaml.SingleQuotedStyle || argument.Style == yaml.DoubleQuotedStyle {
		return false
	}
	value := argument.Value
	if value == "context.request" || strings.HasPrefix(value, "context.request.") ||
		value == "context.ueRef" || value == "context.correlation" || strings.HasPrefix(value, "context.correlation.") {
		return false
	}
	segments := strings.Split(value, ".")
	if len(segments) < 3 {
		return false
	}
	if skill == nil || skill.Procedure == nil {
		return false
	}
	node := skill.Procedure.ByID[segments[0]]
	if node == nil {
		return false
	}
	switch segments[1] {
	case "operation", "observation":
		return node.Operation != nil || node.Observation != nil
	case "script":
		return false
	}
	return false
}
func predicateParameterIndex(parameters *yaml.Node) map[string]int {
	index := map[string]int{}
	position := 0
	for _, parameter := range sequence(parameters) {
		parameter = unalias(parameter)
		if parameter == nil || parameter.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(parameter.Content); i += 2 {
			if name := scalar(unalias(parameter.Content[i])); name != "" {
				index[name] = position
			}
		}
		position++
	}
	return index
}

func argumentCanProvide(argument *yaml.Node, alias, guaranteeID string, nodeProducer map[string]linkedExport) bool {
	argument = unalias(argument)
	if argument == nil || argument.Kind != yaml.ScalarNode || argument.Tag != "!!str" {
		return false
	}
	if argument.Style == yaml.SingleQuotedStyle || argument.Style == yaml.DoubleQuotedStyle {
		return false // literal values provide no authenticated guarantee
	}
	value := argument.Value
	if strings.HasPrefix(value, "context.") || strings.HasPrefix(value, "request.") {
		return false // Request/Host data does not carry an Owner Evidence guarantee
	}
	segments := strings.Split(value, ".")
	if len(segments) < 3 {
		return false
	}
	export, ok := nodeProducer[segments[0]]
	if !ok || export.DependencyAlias != alias {
		return false
	}
	referencedPath := strings.Join(segments[2:], ".")
	for _, provide := range export.EvidenceProvides {
		if provide.ID != guaranteeID {
			continue
		}
		if provide.Path == "" || provide.Path == referencedPath || strings.HasPrefix(referencedPath, provide.Path+".") {
			return true
		}
	}
	return false
}
