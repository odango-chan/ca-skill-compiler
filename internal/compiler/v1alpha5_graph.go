package compiler

// WP5: CA-SL v1alpha5 Procedure control-flow graph (Appendix L.2.1/L.2.2/L.11).
//
// A Node-entry Wait is not a fifth business stage and not a separate Node kind.
// It is an entry control on the Node that declares it (L.1, L.2.1). This file
// models that control explicitly:
//
//   - every authored Node is a vertex;
//   - every Wait-bearing Node additionally gets one distinct entry-control
//     vertex (id "<nodeId>@entry"; '@' is outside the NodeId grammar, so the id
//     can never collide with an authored Node);
//   - every business-stage route edge whose target Node carries a Wait targets
//     that Node's entry vertex, so the Wait gate runs on every incoming path;
//   - the entry vertex has the implicit EVENT_RECEIVED control edge into the
//     Node's own first business stage and the authored TIMEOUT control edge to
//     wait.routes.TIMEOUT.
//
// EVENT_RECEIVED and TIMEOUT are control edges. Neither is a Stage Outcome, a
// Predicate PASS, Evidence or a Goal proof (L.2.1, L.11). Only a reachable final
// Postcondition PASS route directly targeting SUCCEEDED proves the Goal.
//
// Business-stage edges reuse the inherited v1alpha4 stage Outcome -> route
// semantics exactly; no new route key is introduced. Route/cycle/terminal
// diagnostics keep their inherited identities (CONTROL_FLOW_DANGLING,
// CONTROL_FLOW_CYCLE, FINAL_POSTCONDITION_MISSING,
// SUCCESS_BYPASSES_FINAL_POSTCONDITION); no v1alpha5 graph diagnostic is
// invented here. Reachability and dominator/post-dominator sets are exposed for
// WP4 (wait.output availability) and WP6 (response boundary post-dominance).

import (
	"sort"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

const (
	// v1alpha5WaitEntrySuffix marks a Wait entry-control vertex. '@' is not
	// part of the inherited NodeId grammar, so this can never collide with an
	// authored Node identifier.
	v1alpha5WaitEntrySuffix = "@entry"

	// v1alpha5GraphVirtualExit is an internal virtual exit used only to lift
	// the multi-terminal post-dominator computation into a single-exit
	// dominator computation on the reversed graph. It is never exposed.
	v1alpha5GraphVirtualExit = "\x00v1alpha5-exit"
)

// V1Alpha5GraphEdgeKind is the kind of a v1alpha5 Procedure graph edge.
type V1Alpha5GraphEdgeKind string

const (
	// V1Alpha5EdgeStageRoute is an authored business-stage route edge. Stage and
	// Outcome carry the inherited v1alpha4 stage outcome semantics.
	V1Alpha5EdgeStageRoute V1Alpha5GraphEdgeKind = "STAGE_ROUTE"
	// V1Alpha5EdgeEventReceived is the implicit Wait event-acceptance control
	// edge from a Wait entry vertex into the Node's own first business stage. It
	// is never a Stage Outcome and never a Goal proof.
	V1Alpha5EdgeEventReceived V1Alpha5GraphEdgeKind = "EVENT_RECEIVED"
	// V1Alpha5EdgeTimeout is the authored Wait TIMEOUT control edge from a Wait
	// entry vertex to wait.routes.TIMEOUT. It is never a Stage Outcome and never
	// a Goal proof.
	V1Alpha5EdgeTimeout V1Alpha5GraphEdgeKind = "TIMEOUT"
)

// v1alpha5GraphTerminals is the deterministic terminal vertex order.
var v1alpha5GraphTerminals = []string{"FAILED", "SAFE_STOPPED", "SUCCEEDED", "UNRESOLVED"}

// V1Alpha5GraphEdge is one directed Procedure graph edge. Raw is the source
// anchor used by diagnostics (route value node, TIMEOUT value node or the Wait
// mapping); it is nil for synthetic edges.
type V1Alpha5GraphEdge struct {
	From    string
	To      string
	Kind    V1Alpha5GraphEdgeKind
	Stage   string // business stage name for STAGE_ROUTE edges
	Outcome string // inherited stage route key for STAGE_ROUTE edges
	Raw     *yaml.Node
}

// V1Alpha5GraphVertex is one Procedure graph vertex. A Wait-bearing Node has
// two vertices: its business vertex (WaitEntry=false) and its entry-control
// vertex (WaitEntry=true). Terminals are terminal vertices.
type V1Alpha5GraphVertex struct {
	ID        string
	NodeID    string
	WaitEntry bool
	Terminal  bool
	Node      *V1Alpha5ProcedureNodeAST
}

// V1Alpha5ProcedureGraph is the deterministic v1alpha5 Procedure control-flow
// graph. It is built once from the parsed Procedure AST and is immutable
// afterwards; the dominator caches are filled lazily.
type V1Alpha5ProcedureGraph struct {
	procedure *V1Alpha5ProcedureAST
	start     string
	startNode string

	vertices     map[string]*V1Alpha5GraphVertex
	nodeVertices map[string]*V1Alpha5GraphVertex
	waitEntry    map[string]string
	order        []string
	edgesFrom    map[string][]V1Alpha5GraphEdge
	edgesTo      map[string][]V1Alpha5GraphEdge

	dominators     map[string]map[string]bool
	postDominators map[string]map[string]bool
}

// buildV1Alpha5ProcedureGraph builds the graph for one parsed Procedure. Nodes
// whose declared id is not a legal, non-reserved NodeId are omitted from the
// graph because the parser already reports NODE_ID_RESERVED for them; this also
// keeps a reserved id such as "SAFE_STOPPED" from colliding with a terminal
// vertex.
func buildV1Alpha5ProcedureGraph(procedure *V1Alpha5ProcedureAST) *V1Alpha5ProcedureGraph {
	graph := &V1Alpha5ProcedureGraph{
		procedure:    procedure,
		vertices:     map[string]*V1Alpha5GraphVertex{},
		nodeVertices: map[string]*V1Alpha5GraphVertex{},
		waitEntry:    map[string]string{},
		edgesFrom:    map[string][]V1Alpha5GraphEdge{},
		edgesTo:      map[string][]V1Alpha5GraphEdge{},
	}
	if procedure == nil || procedure.Raw == nil {
		return graph
	}

	for _, nodeID := range sortedStringMapKeys(procedure.ByID) {
		if !v1alpha5GraphNodeID(nodeID) {
			continue
		}
		node := procedure.ByID[nodeID]
		vertex := &V1Alpha5GraphVertex{ID: nodeID, NodeID: nodeID, Node: node}
		graph.addVertex(vertex)
		graph.nodeVertices[nodeID] = vertex
		if node != nil && node.Wait != nil {
			entryID := v1alpha5WaitEntryVertexID(nodeID)
			graph.addVertex(&V1Alpha5GraphVertex{ID: entryID, NodeID: nodeID, WaitEntry: true, Node: node})
			graph.waitEntry[nodeID] = entryID
		}
	}
	for _, terminal := range v1alpha5GraphTerminals {
		graph.addVertex(&V1Alpha5GraphVertex{ID: terminal, NodeID: terminal, Terminal: true})
	}
	graph.order = sortedStringMapKeys(graph.vertices)

	for _, nodeID := range sortedStringMapKeys(procedure.ByID) {
		node := procedure.ByID[nodeID]
		if _, isGraphNode := graph.nodeVertices[nodeID]; !isGraphNode || node == nil {
			continue
		}
		if node.Wait != nil {
			entryID := graph.waitEntry[nodeID]
			// (a) Implicit event acceptance resumes this same Node at its first
			// declared business stage.
			graph.addEdge(V1Alpha5GraphEdge{
				From: entryID, To: nodeID, Kind: V1Alpha5EdgeEventReceived, Raw: node.Wait.Raw,
			})
			// (b) The authored TIMEOUT Wait-control edge.
			if target := node.Wait.Routes.Timeout; target != "" {
				if resolved, ok := graph.resolveTarget(target); ok {
					graph.addEdge(V1Alpha5GraphEdge{
						From: entryID, To: resolved, Kind: V1Alpha5EdgeTimeout,
						Raw: child(node.Wait.Routes.Raw, "TIMEOUT"),
					})
				}
			}
		}
		// Business-stage routes: inherited v1alpha4 Outcome -> route semantics.
		// Every target Node that carries a Wait is entered through its entry
		// vertex.
		for _, stage := range v1alpha5WaitBusinessStages {
			routes := v1alpha5ProcedureStageRoutes(node, stage)
			if routes == nil {
				continue
			}
			for _, outcome := range routeKeys(routes) {
				valueNode := child(routes, outcome)
				resolved, ok := graph.resolveTarget(scalar(valueNode))
				if !ok {
					// Undeclared targets are diagnosed by CONTROL_FLOW_DANGLING;
					// the structural graph only carries resolved edges.
					continue
				}
				graph.addEdge(V1Alpha5GraphEdge{
					From: nodeID, To: resolved, Kind: V1Alpha5EdgeStageRoute,
					Stage: stage, Outcome: outcome, Raw: valueNode,
				})
			}
		}
	}

	if startNode := scalar(procedure.Start); startNode != "" {
		if _, declared := graph.nodeVertices[startNode]; declared {
			graph.startNode = startNode
			if entryID, waitBearing := graph.waitEntry[startNode]; waitBearing {
				// The Wait is evaluated after control flow selects the start
				// Node, so the start vertex is the start Node's entry vertex.
				graph.start = entryID
			} else {
				graph.start = startNode
			}
		}
	}
	return graph
}

// v1alpha5GraphNodeID reports whether a declared Node id is a legal, non-reserved
// NodeId and can therefore be represented as a graph vertex.
func v1alpha5GraphNodeID(nodeID string) bool {
	if nodeID == "" || nodeID == "context" || v4Terminals[nodeID] {
		return false
	}
	return v4NodeID.MatchString(nodeID)
}

// v1alpha5WaitEntryVertexID returns the deterministic Wait entry-control vertex
// id for a Wait-bearing Node.
func v1alpha5WaitEntryVertexID(nodeID string) string {
	return nodeID + v1alpha5WaitEntrySuffix
}

// v1alpha5ProcedureStage returns the raw stage mapping of a Node, or nil when the
// stage is not declared.
func v1alpha5ProcedureStage(node *V1Alpha5ProcedureNodeAST, stage string) *yaml.Node {
	if node == nil {
		return nil
	}
	switch stage {
	case "precondition":
		return node.Precondition
	case "operation":
		return node.Operation
	case "observation":
		return node.Observation
	case "postcondition":
		return node.Postcondition
	}
	return nil
}

// v1alpha5ProcedureStageRoutes returns the raw routes mapping of one declared
// business stage, or nil.
func v1alpha5ProcedureStageRoutes(node *V1Alpha5ProcedureNodeAST, stage string) *yaml.Node {
	return child(v1alpha5ProcedureStage(node, stage), "routes")
}

func (graph *V1Alpha5ProcedureGraph) addVertex(vertex *V1Alpha5GraphVertex) {
	if vertex == nil || vertex.ID == "" {
		return
	}
	graph.vertices[vertex.ID] = vertex
}

func (graph *V1Alpha5ProcedureGraph) addEdge(edge V1Alpha5GraphEdge) {
	if edge.From == "" || edge.To == "" {
		return
	}
	graph.edgesFrom[edge.From] = append(graph.edgesFrom[edge.From], edge)
	graph.edgesTo[edge.To] = append(graph.edgesTo[edge.To], edge)
}

// resolveTarget maps an authored route target to its graph vertex. A declared
// Wait-bearing Node resolves to its entry-control vertex so the Wait gate runs
// on every incoming path. The second result is false when the target is neither
// a permitted terminal nor a declared Node (an author error).
func (graph *V1Alpha5ProcedureGraph) resolveTarget(target string) (string, bool) {
	if target == "" {
		return "", false
	}
	if v4Terminals[target] {
		return target, true
	}
	if _, declared := graph.nodeVertices[target]; declared {
		if entryID, waitBearing := graph.waitEntry[target]; waitBearing {
			return entryID, true
		}
		return target, true
	}
	return "", false
}

// Start returns the graph entry vertex id. For a Wait-bearing start Node this is
// that Node's entry-control vertex. It is empty when the Procedure start does not
// resolve to a declared Node (already diagnosed by PROCEDURE_START_MISSING).
func (graph *V1Alpha5ProcedureGraph) Start() string {
	if graph == nil {
		return ""
	}
	return graph.start
}

// StartNode returns the authored start Node id, or empty when unresolved.
func (graph *V1Alpha5ProcedureGraph) StartNode() string {
	if graph == nil {
		return ""
	}
	return graph.startNode
}

// Vertex returns one graph vertex, or nil.
func (graph *V1Alpha5ProcedureGraph) Vertex(id string) *V1Alpha5GraphVertex {
	if graph == nil {
		return nil
	}
	return graph.vertices[id]
}

// Vertices returns every graph vertex in deterministic id order.
func (graph *V1Alpha5ProcedureGraph) Vertices() []*V1Alpha5GraphVertex {
	if graph == nil {
		return nil
	}
	vertices := make([]*V1Alpha5GraphVertex, 0, len(graph.order))
	for _, id := range graph.order {
		vertices = append(vertices, graph.vertices[id])
	}
	return vertices
}

// WaitEntryVertex returns the entry-control vertex id of a Wait-bearing Node, or
// empty when the Node does not declare a Wait.
func (graph *V1Alpha5ProcedureGraph) WaitEntryVertex(nodeID string) string {
	if graph == nil {
		return ""
	}
	return graph.waitEntry[nodeID]
}

// Edges returns every graph edge in deterministic construction order.
func (graph *V1Alpha5ProcedureGraph) Edges() []V1Alpha5GraphEdge {
	if graph == nil {
		return nil
	}
	edges := make([]V1Alpha5GraphEdge, 0)
	for _, id := range graph.order {
		edges = append(edges, graph.edgesFrom[id]...)
	}
	return edges
}

// EdgesFrom returns a copy of the outgoing edges of one vertex.
func (graph *V1Alpha5ProcedureGraph) EdgesFrom(id string) []V1Alpha5GraphEdge {
	if graph == nil {
		return nil
	}
	return append([]V1Alpha5GraphEdge(nil), graph.edgesFrom[id]...)
}

// EdgesTo returns a copy of the incoming edges of one vertex.
func (graph *V1Alpha5ProcedureGraph) EdgesTo(id string) []V1Alpha5GraphEdge {
	if graph == nil {
		return nil
	}
	return append([]V1Alpha5GraphEdge(nil), graph.edgesTo[id]...)
}

// ReachableFrom returns the set of vertices reachable from root (including root
// itself when present). An empty root returns an empty set.
func (graph *V1Alpha5ProcedureGraph) ReachableFrom(root string) map[string]bool {
	reachable := map[string]bool{}
	if graph == nil || root == "" || graph.vertices[root] == nil {
		return reachable
	}
	queue := []string{root}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if reachable[current] {
			continue
		}
		reachable[current] = true
		for _, edge := range graph.edgesFrom[current] {
			if !reachable[edge.To] {
				queue = append(queue, edge.To)
			}
		}
	}
	return reachable
}

// ReachableVertices returns the set of vertices reachable from the Procedure
// start vertex.
func (graph *V1Alpha5ProcedureGraph) ReachableVertices() map[string]bool {
	if graph == nil {
		return map[string]bool{}
	}
	return graph.ReachableFrom(graph.start)
}

// HasCycleFromStart reports whether the start-reachable graph contains a cycle.
// Wait entry vertices, EVENT_RECEIVED edges and TIMEOUT edges participate in the
// cycle check.
func (graph *V1Alpha5ProcedureGraph) HasCycleFromStart() bool {
	if graph == nil || graph.start == "" {
		return false
	}
	const (
		white = 0
		grey  = 1
		black = 2
	)
	color := map[string]int{}
	var visit func(string) bool
	visit = func(id string) bool {
		switch color[id] {
		case grey:
			return true
		case black:
			return false
		}
		color[id] = grey
		for _, edge := range graph.edgesFrom[id] {
			if visit(edge.To) {
				return true
			}
		}
		color[id] = black
		return false
	}
	return visit(graph.start)
}

// IsGoalProofEdge reports whether an edge is a reachable final Postcondition PASS
// route directly targeting SUCCEEDED. EVENT_RECEIVED and TIMEOUT control edges
// are never Goal proofs (L.2.1/L.2.2).
func (graph *V1Alpha5ProcedureGraph) IsGoalProofEdge(edge V1Alpha5GraphEdge) bool {
	if graph == nil || edge.Kind != V1Alpha5EdgeStageRoute {
		return false
	}
	if edge.Stage != "postcondition" || edge.Outcome != "PASS" || edge.To != "SUCCEEDED" {
		return false
	}
	return graph.ReachableVertices()[edge.From]
}

// Dominators returns the dominator set of every start-reachable vertex. The set
// for a vertex always contains the vertex itself. The result is deterministic
// and cached; callers must not mutate the returned maps.
func (graph *V1Alpha5ProcedureGraph) Dominators() map[string]map[string]bool {
	if graph == nil {
		return map[string]map[string]bool{}
	}
	if graph.dominators == nil {
		successors := make(map[string][]string, len(graph.edgesFrom))
		for from, edges := range graph.edgesFrom {
			for _, edge := range edges {
				successors[from] = append(successors[from], edge.To)
			}
		}
		graph.dominators = dominatorSetsForEdges(graph.start, graph.order, successors)
	}
	return graph.dominators
}

// DominatorSet returns a copy of the dominator set of one vertex.
func (graph *V1Alpha5ProcedureGraph) DominatorSet(vertex string) map[string]bool {
	return copyStringSet(graph.Dominators()[vertex])
}

// Dominates reports whether dominator dominates vertex on every path from the
// Procedure start. Unreachable vertices are dominated by nothing.
func (graph *V1Alpha5ProcedureGraph) Dominates(dominator, vertex string) bool {
	if graph == nil || dominator == "" || vertex == "" {
		return false
	}
	return graph.Dominators()[vertex][dominator]
}

// PostDominators returns the post-dominator set of every vertex that can reach a
// terminal vertex. A vertex v post-dominates u iff every path from u to any
// terminal vertex passes through v. The virtual exit used for the computation is
// not exposed. The result is deterministic and cached; callers must not mutate
// the returned maps.
func (graph *V1Alpha5ProcedureGraph) PostDominators() map[string]map[string]bool {
	if graph == nil {
		return map[string]map[string]bool{}
	}
	if graph.postDominators != nil {
		return graph.postDominators
	}
	reverse := map[string][]string{}
	for _, from := range graph.order {
		for _, edge := range graph.edgesFrom[from] {
			reverse[edge.To] = append(reverse[edge.To], from)
		}
	}
	order := append([]string(nil), graph.order...)
	order = append(order, v1alpha5GraphVirtualExit)
	exitSuccessors := make([]string, 0, len(v1alpha5GraphTerminals))
	for _, terminal := range v1alpha5GraphTerminals {
		if graph.vertices[terminal] != nil {
			exitSuccessors = append(exitSuccessors, terminal)
		}
	}
	reverse[v1alpha5GraphVirtualExit] = exitSuccessors
	sets := dominatorSetsForEdges(v1alpha5GraphVirtualExit, order, reverse)
	result := make(map[string]map[string]bool, len(sets))
	for vertex, set := range sets {
		if vertex == v1alpha5GraphVirtualExit {
			continue
		}
		exposed := make(map[string]bool, len(set))
		for dominator := range set {
			if dominator == v1alpha5GraphVirtualExit {
				continue
			}
			exposed[dominator] = true
		}
		result[vertex] = exposed
	}
	graph.postDominators = result
	return result
}

// PostDominatorSet returns a copy of the post-dominator set of one vertex.
func (graph *V1Alpha5ProcedureGraph) PostDominatorSet(vertex string) map[string]bool {
	return copyStringSet(graph.PostDominators()[vertex])
}

// PostDominates reports whether postDominator post-dominates vertex, that is
// whether every path from vertex to a terminal vertex passes through
// postDominator.
func (graph *V1Alpha5ProcedureGraph) PostDominates(postDominator, vertex string) bool {
	if graph == nil || postDominator == "" || vertex == "" {
		return false
	}
	return graph.PostDominators()[vertex][postDominator]
}

// dominatorSetsForEdges is the deterministic iterative dominator computation for
// the vertices reachable from root in a directed graph. Vertices that cannot be
// reached from root are excluded from the result.
func dominatorSetsForEdges(root string, order []string, successors map[string][]string) map[string]map[string]bool {
	result := map[string]map[string]bool{}
	if root == "" {
		return result
	}
	reachable := map[string]bool{}
	queue := []string{root}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if reachable[current] {
			continue
		}
		reachable[current] = true
		queue = append(queue, successors[current]...)
	}
	if !reachable[root] {
		return result
	}
	predecessors := map[string][]string{}
	for from, targets := range successors {
		if !reachable[from] {
			continue
		}
		for _, to := range targets {
			if !reachable[to] {
				continue
			}
			predecessors[to] = append(predecessors[to], from)
		}
	}
	for key := range predecessors {
		sort.Strings(predecessors[key])
	}
	nodes := make([]string, 0, len(reachable))
	for node := range reachable {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		if node == root {
			result[node] = map[string]bool{root: true}
			continue
		}
		set := make(map[string]bool, len(nodes))
		for _, candidate := range nodes {
			set[candidate] = true
		}
		result[node] = set
	}
	for changed := true; changed; {
		changed = false
		for _, node := range nodes {
			if node == root {
				continue
			}
			preds := predecessors[node]
			if len(preds) == 0 {
				// Only root can have no reachable predecessor; a non-root node
				// with no predecessor is unreachable and already excluded.
				continue
			}
			intersection := map[string]bool{}
			for candidate := range result[preds[0]] {
				intersection[candidate] = true
			}
			for _, pred := range preds[1:] {
				for candidate := range intersection {
					if !result[pred][candidate] {
						delete(intersection, candidate)
					}
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

func copyStringSet(set map[string]bool) map[string]bool {
	if set == nil {
		return nil
	}
	copied := make(map[string]bool, len(set))
	for key, present := range set {
		copied[key] = present
	}
	return copied
}

// validateV1Alpha5ProcedureGraph is the WP5 graph validation over one parsed
// v1alpha5 Skill. It reuses inherited diagnostic identities only:
//
//   - undeclared business-stage route target or grammatically valid undeclared
//     TIMEOUT target: CONTROL_FLOW_DANGLING;
//   - unknown response.after target: WAIT_RESPONSE_BOUNDARY_INVALID (Appendix
//     L.11);
//   - cycle through Wait entry vertices, EVENT_RECEIVED or TIMEOUT edges:
//     CONTROL_FLOW_CYCLE;
//   - only a reachable final Postcondition PASS may target SUCCEEDED:
//     FINAL_POSTCONDITION_MISSING / SUCCESS_BYPASSES_FINAL_POSTCONDITION.
//
// response.after reachability and post-dominance for a *declared* Node are WP6;
// this WP checks only that a present after value identifies a declared Node.
// Missing/invalid Wait fields are already diagnosed by the WP2 parser and are
// not re-reported here.
func validateV1Alpha5ProcedureGraph(procedurePath string, skill *V1Alpha5SkillAST) []Diagnostic {
	var diagnostics []Diagnostic
	if skill == nil || skill.Procedure == nil || skill.Procedure.Raw == nil {
		return diagnostics
	}
	procedure := skill.Procedure
	add := func(at *yaml.Node, code, message string) {
		line, column := 0, 0
		if at != nil {
			line, column = at.Line, at.Column
		}
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Phase:    "C_STATIC",
			Code:     code,
			Path:     limitDiagnosticPath(procedurePath),
			Line:     line,
			Column:   column,
			Message:  limitDiagnosticMessage(message),
		})
	}

	graph := buildV1Alpha5ProcedureGraph(procedure)
	declared := graph.nodeVertices

	// Business-stage route targets: reused inherited route semantics, exact
	// target identity only.
	for _, nodeID := range sortedStringMapKeys(procedure.ByID) {
		node := procedure.ByID[nodeID]
		if node == nil {
			continue
		}
		for _, stage := range v1alpha5WaitBusinessStages {
			routes := v1alpha5ProcedureStageRoutes(node, stage)
			if routes == nil {
				continue
			}
			for _, outcome := range routeKeys(routes) {
				valueNode := child(routes, outcome)
				target := scalar(valueNode)
				if _, ok := declared[target]; !ok && !v4Terminals[target] {
					add(valueNode, "CONTROL_FLOW_DANGLING", "Route value is neither a terminal nor an existing Node")
				}
			}
		}
	}

	// Wait-control targets: TIMEOUT must be a declared Node or exactly one of
	// the three permitted stopping terminals (grammar-level SUCCEEDED rejection
	// stays WAIT_TIMEOUT_ROUTE_INVALID); a present response.after must identify a
	// declared Node (Appendix L.11: an unknown response.after is
	// WAIT_RESPONSE_BOUNDARY_INVALID). Reachability and post-dominance of a
	// *declared* response.after remain WP6.
	for _, nodeID := range sortedStringMapKeys(procedure.ByID) {
		node := procedure.ByID[nodeID]
		if node == nil || node.Wait == nil {
			continue
		}
		if target := node.Wait.Routes.Timeout; target != "" {
			if _, ok := declared[target]; !ok && !v4Terminals[target] {
				add(child(node.Wait.Routes.Raw, "TIMEOUT"), "CONTROL_FLOW_DANGLING",
					"Wait TIMEOUT route value is neither a permitted stopping terminal nor an existing Node")
			}
		}
		if node.Wait.Response != nil && node.Wait.Response.After != "" {
			if _, ok := declared[node.Wait.Response.After]; !ok {
				add(child(node.Wait.Response.Raw, "after"), "WAIT_RESPONSE_BOUNDARY_INVALID",
					"Node '"+nodeID+"' wait.response.after does not identify one reachable response boundary on every accepted-event path.")
			}
		}
	}

	// DAG check over the expanded graph: Wait entry vertices and the implicit
	// EVENT_RECEIVED / authored TIMEOUT control edges participate.
	if graph.Start() != "" && graph.HasCycleFromStart() {
		add(procedure.Raw, "CONTROL_FLOW_CYCLE", "Procedure must be a DAG")
	}

	validateV1Alpha5GraphFinalSuccess(procedure, graph, add)
	return diagnostics
}

// validateV1Alpha5GraphFinalSuccess enforces the inherited final-success rule on
// the expanded v1alpha5 graph: a reachable SUCCEEDED target must be the direct
// target of a reachable final Postcondition PASS route. Otherwise
// FINAL_POSTCONDITION_MISSING is reported when no legal proof exists at all, and
// SUCCESS_BYPASSES_FINAL_POSTCONDITION for every reachable non-final SUCCEEDED
// route when a legal proof does exist. Event acceptance never creates a proof.
func validateV1Alpha5GraphFinalSuccess(procedure *V1Alpha5ProcedureAST, graph *V1Alpha5ProcedureGraph, add func(*yaml.Node, string, string)) {
	if graph.Start() == "" {
		return
	}
	reachable := graph.ReachableVertices()
	anySuccess, legalSuccess := false, false
	for _, from := range graph.order {
		if !reachable[from] {
			continue
		}
		for _, edge := range graph.edgesFrom[from] {
			if edge.Kind != V1Alpha5EdgeStageRoute || edge.To != "SUCCEEDED" {
				continue
			}
			anySuccess = true
			if edge.Stage == "postcondition" && edge.Outcome == "PASS" {
				legalSuccess = true
			}
		}
	}
	if anySuccess && !legalSuccess {
		add(procedure.Raw, "FINAL_POSTCONDITION_MISSING", "Only a final Postcondition PASS may request SUCCEEDED")
		return
	}
	if !legalSuccess {
		return
	}
	for _, from := range graph.order {
		if !reachable[from] {
			continue
		}
		for _, edge := range graph.edgesFrom[from] {
			if edge.To != "SUCCEEDED" || graph.IsGoalProofEdge(edge) {
				continue
			}
			add(edge.Raw, "SUCCESS_BYPASSES_FINAL_POSTCONDITION", "Non-final stage requests SUCCEEDED")
		}
	}
}
