package compiler

// WP6 slice 1: frozen CA-SL static SourcePath grammar and Predicate arity.
//
// Accepted SourcePath forms are exactly:
//   context.<identifier>[.<identifier>...]
//   <node>.operation.output[.<identifier>...]
//   <node>.observation.output[.<identifier>...]
// Quoted strings are string literals; bare booleans/null/finite numbers are
// literals. Any other bare scalar is a SourcePath and must satisfy the grammar.
// Schema path typing, producer dominance and availability are later WP6 slices.

import (
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validateSourcePaths(procedurePath string, skill *SkillAST, add func(string, *yaml.Node, string, string)) {
	if skill == nil {
		return
	}
	nodeIDs := map[string]*ProcedureNodeAST{}
	if skill.Procedure != nil {
		for id, node := range skill.Procedure.ByID {
			nodeIDs[id] = node
		}
	}
	predicateArity := map[string]int{}
	if skill.Predicates != nil {
		for id, predicate := range skill.Predicates.Definitions {
			if predicate != nil {
				predicateArity[id] = len(sequence(predicate.Parameters))
			}
		}
	}
	checkArgument := func(value *yaml.Node) {
		value = unalias(value)
		if value == nil || value.Kind != yaml.ScalarNode {
			return // inline maps/arrays are rejected by the Procedure grammar checks
		}
		if value.Style == yaml.SingleQuotedStyle || value.Style == yaml.DoubleQuotedStyle {
			return // quoted string literal
		}
		switch value.Tag {
		case "!!bool", "!!null", "!!int", "!!float":
			return // frozen scalar literal, not a SourcePath
		case "!!str":
			if problem := checkSourcePathValue(value.Value, nodeIDs); problem != "" {
				add(procedurePath, value, "SOURCE_PATH_INVALID", problem)
			}
		}
	}
	if skill.Procedure != nil {
		for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
			node := skill.Procedure.ByID[nodeID]
			if node == nil {
				continue
			}
			for _, arguments := range []*yaml.Node{operationArguments(node), observationArguments(node)} {
				arguments = unalias(arguments)
				if arguments == nil || arguments.Kind != yaml.MappingNode {
					continue
				}
				for i := 1; i < len(arguments.Content); i += 2 {
					checkArgument(arguments.Content[i])
				}
			}
			checkPredicateCallArguments(node.Precondition, predicateArity, procedurePath, checkArgument, add)
			checkPredicateCallArguments(node.Postcondition, predicateArity, procedurePath, checkArgument, add)
		}
	}
}

func operationArguments(node *ProcedureNodeAST) *yaml.Node {
	if node.Operation == nil {
		return nil
	}
	return node.Operation.Arguments
}

func observationArguments(node *ProcedureNodeAST) *yaml.Node {
	if node.Observation == nil {
		return nil
	}
	return node.Observation.Arguments
}

func checkPredicateCallArguments(stage *PredicateStageAST, arity map[string]int, procedurePath string, checkArgument func(*yaml.Node), add func(string, *yaml.Node, string, string)) {
	if stage == nil {
		return
	}
	for _, call := range stage.Predicates {
		if call == nil {
			continue
		}
		predicateID := scalar(call.Predicate)
		arguments := sequence(call.Arguments)
		for _, argument := range arguments {
			checkArgument(argument)
		}
		if expected, known := arity[predicateID]; known && len(arguments) != expected {
			add(procedurePath, call.Raw, "PREDICATE_ARITY_MISMATCH",
				"Predicate "+predicateID+" requires "+itoa(expected)+" positional arguments, got "+itoa(len(arguments)))
		}
	}
}

// checkSourcePathValue validates the exact frozen static path grammar against
// the Procedure's declared Nodes and stages.
func checkSourcePathValue(value string, nodes map[string]*ProcedureNodeAST) string {
	segments := strings.Split(value, ".")
	if len(segments) < 2 {
		return "SourcePath must use context.<field> or <node>.<stage>.output[.<field>...]"
	}
	for _, segment := range segments {
		if !validDependencyID(segment) {
			return "SourcePath segment is not a valid Identifier: " + segment
		}
	}
	if segments[0] == "context" {
		if len(segments) < 2 {
			return "SourcePath must read an explicit context field"
		}
		return ""
	}
	if len(segments) < 3 {
		return "SourcePath root must be a declared Node output"
	}
	node := nodes[segments[0]]
	if node == nil {
		return "SourcePath root Node is not declared: " + segments[0]
	}
	switch segments[1] {
	case "operation":
		if node.Operation == nil {
			return "SourcePath root " + segments[0] + " declares no operation stage"
		}
	case "observation":
		if node.Observation == nil {
			return "SourcePath root " + segments[0] + " declares no observation stage"
		}
	case "wait":
		// CA-SL v1alpha5 Appendix L.2: the accepted-event value of a Node-entry
		// Wait is exposed as <nodeId>.wait.output in the Wait-bearing Node and, when
		// dominated by its event-accepted edge, on later paths. Whether the Node is
		// actually Wait-bearing is checked here; path availability and typed
		// producer mapping are owned by the v1alpha5 Wait passes.
		if child(node.Raw, "wait") == nil {
			return "SourcePath root " + segments[0] + " declares no wait entry"
		}
		if len(segments) < 3 || segments[2] != "output" {
			return "SourcePath root must be <node>.wait.output[.<field>...]"
		}
		return ""
	default:
		return "SourcePath root must be context or <node>.operation.output / <node>.observation.output / <node>.wait.output"
	}
	if segments[2] != "output" {
		return "SourcePath root must be <node>.<stage>.output"
	}
	return ""
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 4)
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}
