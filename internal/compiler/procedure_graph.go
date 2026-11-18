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
	nodeByID map[string]*ProcedureNodeAST
	order    []string
}

func buildProcedureGraph(procedure *ProcedureAST) procedureGraph {
	graph := procedureGraph{edges: map[string][]string{}, nodeByID: map[string]*ProcedureNodeAST{}}
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

func routeTargets(node *ProcedureNodeAST) []string {
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
			if value != "" && !reservedTerminals[value] {
				targets = append(targets, value)
			}
		}
	}
	// CA-SL v1alpha5 Appendix L.1 makes the Wait TIMEOUT route a real control-flow
	// edge of the expanded Procedure graph. A logical v1alpha4 source cannot carry
	// a wait declaration (it is rejected with UNSUPPORTED_WAIT), so admitting the
	// edge here is target-neutral.
	waitRoutes := child(child(node.Raw, "wait"), "routes")
	for _, target := range mapKeys(waitRoutes) {
		value := scalar(child(waitRoutes, target))
		if value != "" && !reservedTerminals[value] {
			targets = append(targets, value)
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

// validateProcedureGraph emits UNREACHABLE_NODE/C_WARNING and
// ENTRY_PRECONDITION_BYPASSED/C_STATIC. Missing/unknown start is left to the
// existing PROCEDURE_START_MISSING check.
func validateProcedureGraph(contract *ContractAST, procedure *ProcedureAST, ownerLinking ownerLinkingResult, contractPath, procedurePath string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if procedure == nil || procedure.Raw == nil {
		return diagnostics
	}
	graph := buildProcedureGraph(procedure)
	start := scalar(procedure.Start)
	if graph.nodeByID[start] == nil {
		return diagnostics
	}
	reachable := reachableProcedureNodes(graph, start)
	for _, id := range procedureReachableStagesWithoutIncomingEdge(graph, start, reachable) {
		diagnostics = append(diagnostics, dependencyDiagnostic(
			"error", "C_STATIC", "REACHABLE_STAGE_NO_INCOMING_EDGE", procedurePath,
			child(graph.nodeByID[id].Raw, "id"), "Reachable non-entry stage has no reachable incoming edge: "+id))
	}
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
	// An unreachable global Precondition reference is ENTRY_PRECONDITION_REF_INVALID
	// (classic E52 sixth variant), not a per-path bypass.
	if !reachable[gate] {
		return diagnostics
	}
	mutations := mutationToolIdentities(ownerLinking)
	if procedureHasBypass(graph, procedure, start, gate, mutations) {
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

// procedureHasBypass proves whether a reachable path reaches a SUCCEEDED route
// target or a trusted MUTATION callable without taking the global Precondition
// gate's PASS edge. It is edge-exact: arriving at the gate Node is not proof;
// only a PASS route from its Precondition stage (or a later stage reached by the
// implied auto-advance, which can only execute after PASS) carries the proof.
// A FAIL/UNKNOWN route out of the gate keeps the path unproven.
func procedureHasBypass(graph procedureGraph, procedure *ProcedureAST, start, gate string, mutations map[string]bool) bool {
	edgesFrom := map[string][]routeEdge{}
	for _, edge := range procedureRouteEdges(procedure) {
		edgesFrom[edge.from] = append(edgesFrom[edge.from], edge)
	}
	type state struct {
		node   string
		passed bool
	}
	visited := map[state]bool{}
	queue := []state{{node: start}}
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
		edges := edgesFrom[current.node]
		if current.node == gate {
			// The Precondition stage executes on arrival. Only the PASS route (or
			// a route out of a post-auto-advance stage) establishes the proof.
			if !current.passed {
				for _, edge := range edges {
					if edge.to == "SUCCEEDED" && !edgePassesGate(edge) {
						return true
					}
				}
			}
		} else if !current.passed {
			if node.Operation != nil {
				identity := scalar(node.Operation.Tool)
				if mutations[identity] {
					return true
				}
			}
			for _, edge := range edges {
				if edge.to == "SUCCEEDED" {
					return true
				}
			}
		}
		for _, edge := range edges {
			if edge.to == "" || reservedTerminals[edge.to] || graph.nodeByID[edge.to] == nil {
				continue
			}
			nextPassed := current.passed
			if !nextPassed && current.node == gate {
				nextPassed = edgePassesGate(edge)
			}
			queue = append(queue, state{node: edge.to, passed: nextPassed})
		}
	}
	return false
}

// edgePassesGate reports whether taking this route Edge proves the global
// Precondition aggregate PASS. Only the Precondition PASS route does; routes
// from later stages of the same Node are reachable only after the implied PASS
// auto-advance, so they also carry the proof.
func edgePassesGate(edge routeEdge) bool {
	return edge.stage != "precondition" || edge.key == "PASS"
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

// procedureReachableStagesWithoutIncomingEdge is the expanded-graph construction
// invariant (CA-SL section 22 / 24.1, E99): every reachable non-entry stage must
// have at least one incoming edge from a reachable stage. A violation means the
// graph builder produced an inconsistent analysis graph, not an author error.
func procedureReachableStagesWithoutIncomingEdge(graph procedureGraph, start string, reachable map[string]bool) []string {
	incoming := map[string]bool{}
	for from := range reachable {
		for _, to := range graph.edges[from] {
			if reachable[to] {
				incoming[to] = true
			}
		}
	}
	missing := make([]string, 0)
	for id := range reachable {
		if id == start {
			continue
		}
		if !incoming[id] {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	return missing
}
