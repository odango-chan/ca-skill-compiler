package compiler

// WP6 slice 3: producer output SourcePath typing, Producer dominance and
// per-path availability. Argument target typing (F.5) is a later slice.

import (
	"sort"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validateProducerSourcePaths(req BuildRequest, skill *SkillAST, identity dependencyIdentityResult, ownerLinking ownerLinkingResult, procedurePath string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if skill == nil || skill.Procedure == nil {
		return diagnostics
	}
	registry := buildSchemaRegistry(req, skill, identity)
	graph := buildProcedureGraph(skill.Procedure)
	start := scalar(skill.Procedure.Start)
	dominators := procedureDominators(graph, start)
	outputSchemas := producerOutputSchemas(skill, ownerLinking, registry)
	reachabilityCache := map[string]map[string]map[string]bool{}

	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		consumer := skill.Procedure.ByID[nodeID]
		if consumer == nil {
			continue
		}
		check := func(stage string, arguments *yaml.Node) {
			arguments = unalias(arguments)
			if arguments == nil || arguments.Kind != yaml.MappingNode {
				return
			}
			for i := 1; i < len(arguments.Content); i += 2 {
				value := unalias(arguments.Content[i])
				if value == nil || value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
					continue
				}
				if value.Style == yaml.SingleQuotedStyle || value.Style == yaml.DoubleQuotedStyle {
					continue
				}
				if !strings.HasPrefix(value.Value, "context.") {
					reachability := reachabilityCache[value.Value]
					if reachability == nil {
						reachability = producerKeyReachability(skill, value.Value)
						reachabilityCache[value.Value] = reachability
					}
					diagnostics = append(diagnostics, checkProducerPath(value, nodeID, stage, registry, outputSchemas, dominators, reachability, procedurePath)...)
				}
			}
		}
		if consumer.Operation != nil {
			check("operation", consumer.Operation.Arguments)
		}
		if consumer.Observation != nil {
			check("observation", consumer.Observation.Arguments)
		}
		for _, stageName := range []string{"precondition", "postcondition"} {
			stage := consumer.Precondition
			if stageName == "postcondition" {
				stage = consumer.Postcondition
			}
			if stage == nil {
				continue
			}
			for _, call := range stage.Predicates {
				if call == nil {
					continue
				}
				arguments := sequence(call.Arguments)
				for _, argument := range arguments {
					value := unalias(argument)
					if value == nil || value.Kind != yaml.ScalarNode || value.Tag != "!!str" ||
						value.Style == yaml.SingleQuotedStyle || value.Style == yaml.DoubleQuotedStyle {
						continue
					}
					if !strings.HasPrefix(value.Value, "context.") {
						reachability := reachabilityCache[value.Value]
						if reachability == nil {
							reachability = producerKeyReachability(skill, value.Value)
							reachabilityCache[value.Value] = reachability
						}
						diagnostics = append(diagnostics, checkProducerPath(value, nodeID, stageName, registry, outputSchemas, dominators, reachability, procedurePath)...)
					}
				}
			}
		}
	}
	sortDiagnostics(diagnostics)
	return diagnostics
}

type producerOutput struct {
	node        string
	schema      *yaml.Node
	effect      string
	recoverable bool
}

func producerOutputSchemas(skill *SkillAST, ownerLinking ownerLinkingResult, registry schemaRegistry) map[string]producerOutput {
	linked := map[string]producerOutput{}
	for _, export := range ownerLinking.Exports {
		if (export.ExportKind != "operation" && export.ExportKind != "observation") || export.OutputSchemaID == "" {
			continue
		}
		resource, problem := registry.resolveReference(export.DependencyAlias + "::" + export.OutputSchemaID)
		if problem != "" {
			continue
		}
		linked[export.DependencyAlias+"::"+export.ExportID] = producerOutput{
			schema: unalias(resource.node), effect: export.Effect,
			recoverable: export.MutationFields.RecoverableSchemaID != "",
		}
	}
	outputs := map[string]producerOutput{}
	if skill == nil || skill.Procedure == nil {
		return outputs
	}
	for nodeID, node := range skill.Procedure.ByID {
		if node == nil {
			continue
		}
		if node.Operation != nil {
			if output, ok := linked[scalar(node.Operation.Tool)]; ok {
				output.node = "operation"
				outputs[nodeID] = output
			}
		}
		if node.Observation != nil {
			if output, ok := linked[scalar(node.Observation.Tool)]; ok {
				output.node = "observation"
				outputs[nodeID] = output
			}
		}
	}
	return outputs
}

func checkProducerPath(value *yaml.Node, consumerID, consumerStage string, registry schemaRegistry, outputs map[string]producerOutput, dominators map[string]map[string]bool, reachability map[string]map[string]bool, procedurePath string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	segments := strings.Split(value.Value, ".")
	if len(segments) < 3 {
		return diagnostics
	}
	producerID := segments[0]
	producerStage := segments[1]
	output := outputs[producerID]
	if output.node != producerStage {
		return diagnostics
	}
	if output.schema == nil {
		return diagnostics
	}
	if producerID != consumerID {
		committed, other := nonSucceededProducerPath(consumerID, reachability)
		if committed && output.effect == "MUTATION" && !output.recoverable {
			addSourcePathDiagnostic(&diagnostics, "error", "C_STATIC", "RECONCILE_OUTPUT_UNAVAILABLE", procedurePath, value,
				"Producer output is only reachable through its COMMITTED path and no recoverable success output is declared")
			return diagnostics
		}
		if other {
			addSourcePathDiagnostic(&diagnostics, "error", "C_STATIC", "SOURCE_NOT_AVAILABLE", procedurePath, value,
				"Producer output is reachable through a non-SUCCEEDED path: "+value.Value)
			return diagnostics
		}
	}
	if producerID != consumerID {
		if !dominators[consumerID][producerID] {
			addSourcePathDiagnostic(&diagnostics, "error", "C_STATIC", "SOURCE_NOT_AVAILABLE", procedurePath, value,
				"Producer Node "+producerID+" does not dominate the consumer Node "+consumerID)
			return diagnostics
		}
	} else if !(consumerStage == "postcondition" || consumerStage == "observation" && producerStage == "observation") {
		addSourcePathDiagnostic(&diagnostics, "error", "C_STATIC", "SOURCE_NOT_AVAILABLE", procedurePath, value,
			"Node output is not available before its producing stage: "+value.Value)
		return diagnostics
	}
	optional := false
	current := output.schema
	for _, segment := range segments[3:] {
		if current == nil || current.Kind != yaml.MappingNode {
			addSourcePathDiagnostic(&diagnostics, "error", "C_STATIC", "SOURCE_PATH_INVALID", procedurePath, value,
				"SourcePath segment is not an object field: "+segment)
			return diagnostics
		}
		if ref := child(current, "$ref"); ref != nil {
			resource, problem := registry.resolveReference(scalar(ref))
			if problem != "" {
				return diagnostics
			}
			current = unalias(resource.node)
		}
		property := child(child(current, "properties"), segment)
		if property == nil {
			addSourcePathDiagnostic(&diagnostics, "error", "C_STATIC", "SOURCE_PATH_INVALID", procedurePath, value,
				"SourcePath segment is not a declared output Schema property: "+segment)
			return diagnostics
		}
		if !requiredProperty(current, segment) {
			optional = true
		}
		current = unalias(property)
	}
	if optional {
		addSourcePathDiagnostic(&diagnostics, "error", "C_STATIC", "SOURCE_NOT_AVAILABLE", procedurePath, value,
			"SourcePath reads an output field that is not guaranteed present: "+value.Value)
	}
	return diagnostics
}

// procedureDominators computes the dominator set of every reachable Node.
func procedureDominators(graph procedureGraph, start string) map[string]map[string]bool {
	result := map[string]map[string]bool{}
	reachable := reachableProcedureNodes(graph, start)
	predecessors := map[string][]string{}
	for node, targets := range graph.edges {
		for _, target := range targets {
			predecessors[target] = append(predecessors[target], node)
		}
	}
	nodes := make([]string, 0, len(reachable))
	for node := range reachable {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		if node == start {
			result[node] = map[string]bool{start: true}
			continue
		}
		set := map[string]bool{}
		for _, candidate := range nodes {
			set[candidate] = true
		}
		result[node] = set
	}
	changed := true
	for changed {
		changed = false
		for _, node := range nodes {
			if node == start {
				continue
			}
			intersection := map[string]bool{}
			preds := predecessors[node]
			if len(preds) == 0 {
				continue
			}
			for candidate := range result[preds[0]] {
				inAll := true
				for _, pred := range preds[1:] {
					if !result[pred][candidate] {
						inAll = false
						break
					}
				}
				if inAll && reachable[preds[0]] {
					intersection[candidate] = true
				}
			}
			intersection[node] = true
			if !sameStringSet(intersection, result[node]) {
				result[node] = intersection
				changed = true
			}
		}
	}
	return result
}

func sameStringSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if !b[key] {
			return false
		}
	}
	return true
}
