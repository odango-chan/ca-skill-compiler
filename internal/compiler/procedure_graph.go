package compiler

// WP5: Procedure graph reachability, UNREACHABLE_NODE warning and global
// Precondition dominance. The frozen Procedure grammar/cardinality/route checks
// already live in v1alpha4.go; this slice adds the graph proofs that need the
// reachable graph and the trusted callable Effects from WP3.

import (
	"sort"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

type procedureGraph struct {
	edges    map[string][]string
	nodeByID map[string]*V1Alpha4ProcedureNodeAST
	order    []string
}

func buildV1Alpha4ProcedureGraph(procedure *V1Alpha4ProcedureAST) procedureGraph {
	graph := procedureGraph{edges: map[string][]string{}, nodeByID: map[string]*V1Alpha4ProcedureNodeAST{}}
	if procedure == nil {
		return graph
	}
	graph.order = sortedStringMapKeys(procedure.ByID)
	for _, id := range graph.order {
		node := procedure.ByID[id]
		graph.nodeByID[id] = node
		graph.edges[id] = routeTargets(node)
	}
	return graph
}

func routeTargets(node *V1Alpha4ProcedureNodeAST) []string {
	if node == nil {
		return nil
	}
	targets := make([]string, 0)
	stages := []*yaml.Node{}
	if node.Precondition != nil {
		stages = append(stages, node.Precondition.Routes)
	}
	if node.Operation != nil {
		stages = append(stages, node.Operation.Routes)
	}
	if node.Observation != nil {
		stages = append(stages, node.Observation.Routes)
	}
	if node.Postcondition != nil {
		stages = append(stages, node.Postcondition.Routes)
	}
	for _, routes := range stages {
		for _, target := range mapKeys(routes) {
			value := scalar(child(routes, target))
			if value != "" && !v4Terminals[value] {
				targets = append(targets, value)
			}
		}
	}
	sort.Strings(targets)
	return targets
}

func reachableProcedureNodes(graph procedureGraph, start string) map[string]bool {
	reachable := map[string]bool{}
	queue := []string{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == "" || reachable[current] || graph.nodeByID[current] == nil {
			continue
		}
		reachable[current] = true
		queue = append(queue, graph.edges[current]...)
	}
	return reachable
}

// validateV1Alpha4ProcedureGraph emits UNREACHABLE_NODE/C_WARNING and
// ENTRY_PRECONDITION_BYPASSED/C_STATIC. Missing/unknown start is left to the
// existing PROCEDURE_START_MISSING check.
func validateV1Alpha4ProcedureGraph(contract *V1Alpha4ContractAST, procedure *V1Alpha4ProcedureAST, ownerLinking ownerLinkingResult, contractPath, procedurePath string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if procedure == nil || procedure.Raw == nil {
		return diagnostics
	}
	graph := buildV1Alpha4ProcedureGraph(procedure)
	start := scalar(procedure.Start)
	if graph.nodeByID[start] == nil {
		return diagnostics
	}
	reachable := reachableProcedureNodes(graph, start)
	for _, id := range graph.order {
		if !reachable[id] {
			diagnostics = append(diagnostics, dependencyDiagnostic(
				"warning", "C_WARNING", "UNREACHABLE_NODE", procedurePath,
				child(graph.nodeByID[id].Raw, "id"), "Node is unreachable from the Procedure start: "+id))
		}
	}
	gate := ""
	if contract != nil {
		if value := scalar(contract.Precondition); value != "" {
			parts := strings.Split(value, ".")
			if len(parts) == 2 {
				gate = parts[0]
			}
		}
	}
	if gate == "" || graph.nodeByID[gate] == nil || graph.nodeByID[gate].Precondition == nil {
		return diagnostics
	}
	mutations := mutationToolIdentities(ownerLinking)
	if procedureHasBypass(graph, start, gate, reachable, mutations) {
		node := contract.Precondition
		line, column := 0, 0
		if node != nil {
			line, column = node.Line, node.Column
		}
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error", Phase: "C_STATIC", Code: "ENTRY_PRECONDITION_BYPASSED",
			Path: limitDiagnosticPath(contractPath), Line: line, Column: column,
			Message: limitDiagnosticMessage("A reachable path reaches a business Mutation or SUCCEEDED without the global Precondition aggregate PASS: " + gate + ".precondition"),
		})
	}
	sortDiagnostics(diagnostics)
	return diagnostics
}

// procedureHasBypass proves whether a path from start reaches a SUCCEEDED route
// target or a trusted MUTATION callable without visiting the gate node.
func procedureHasBypass(graph procedureGraph, start, gate string, reachable map[string]bool, mutations map[string]bool) bool {
	type state struct {
		node   string
		passed bool
	}
	visited := map[state]bool{}
	queue := []state{{node: start, passed: start == gate}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true
		node := graph.nodeByID[current.node]
		if node == nil {
			continue
		}
		if !current.passed {
			if node.Operation != nil {
				identity := scalar(node.Operation.Tool)
				if mutations[identity] {
					return true
				}
			}
			if routesToSucceeded(node) {
				return true
			}
		}
		nextPassed := current.passed || current.node == gate
		for _, next := range graph.edges[current.node] {
			queue = append(queue, state{node: next, passed: nextPassed})
		}
	}
	return false
}

func routesToSucceeded(node *V1Alpha4ProcedureNodeAST) bool {
	if node == nil {
		return false
	}
	stages := []*yaml.Node{}
	if node.Precondition != nil {
		stages = append(stages, node.Precondition.Routes)
	}
	if node.Operation != nil {
		stages = append(stages, node.Operation.Routes)
	}
	if node.Observation != nil {
		stages = append(stages, node.Observation.Routes)
	}
	if node.Postcondition != nil {
		stages = append(stages, node.Postcondition.Routes)
	}
	for _, routes := range stages {
		for _, outcome := range mapKeys(routes) {
			if scalar(child(routes, outcome)) == "SUCCEEDED" {
				return true
			}
		}
	}
	return false
}

func mutationToolIdentities(ownerLinking ownerLinkingResult) map[string]bool {
	mutations := map[string]bool{}
	for _, export := range ownerLinking.Exports {
		if export.ExportKind == "operation" && export.Effect == "MUTATION" {
			mutations[export.DependencyAlias+"::"+export.ExportID] = true
		}
	}
	return mutations
}
