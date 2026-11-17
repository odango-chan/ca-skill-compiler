package compiler

// WP5: CA-SL v1alpha5 Procedure control-flow graph tests.
//
// Coverage: Wait entry-control vertices, EVENT_RECEIVED/TIMEOUT control edges,
// route-target identity (CONTROL_FLOW_DANGLING), DAG including TIMEOUT edges and
// entry vertices (CONTROL_FLOW_CYCLE), final-Postcondition-PASS Goal proof,
// entry dominance, dominator/post-dominator algorithms, multi-Wait Procedures,
// determinism and the public Check positive path.

import (
	"reflect"
	"sort"
	"testing"
)

// v1alpha5MultiWaitProcedure has two sequential Wait-bearing Nodes and a final
// Postcondition PASS Goal proof. Everything is declared, so it must validate
// with zero diagnostics.
func v1alpha5MultiWaitProcedure() string {
	return `start: trigger-transfer
nodes:
  trigger-transfer:
    operation:
      tool: ca::amf.communication.n1n2_message_transfer
      arguments: {}
      routes:
        SUCCEEDED: wait-sm-context
        FAILED: SAFE_STOPPED
        OUTCOME_UNKNOWN: SAFE_STOPPED
  wait-sm-context:
    wait:
      event:
        source: AMF
        service: Nsmf_PDUSession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: handle-update-timeout
    operation:
      tool: ca::smf.session.update_from_amf
      arguments: {}
      routes:
        SUCCEEDED: wait-n4
        FAILED: SAFE_STOPPED
        OUTCOME_UNKNOWN: SAFE_STOPPED
  wait-n4:
    wait:
      event:
        source: SMF
        service: Nsmf_PDUSession
        operation: UpdateSMContext
      timeout: PT10S
      routes:
        TIMEOUT: handle-update-timeout
    operation:
      tool: ca::upn.session.modify
      arguments: {}
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        OUTCOME_UNKNOWN: SAFE_STOPPED
  final:
    postcondition:
      predicates:
        - predicate: session.goal_satisfied
          arguments: []
      routes:
        PASS: SUCCEEDED
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
  handle-update-timeout:
    operation:
      tool: ca::smf.update_timeout.handle
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
        FAILED: UNRESOLVED
        OUTCOME_UNKNOWN: SAFE_STOPPED
`
}

func v1alpha5FindGraphEdge(graph *V1Alpha5ProcedureGraph, from, to string, kind V1Alpha5GraphEdgeKind) *V1Alpha5GraphEdge {
	for _, edge := range graph.EdgesFrom(from) {
		if edge.To == to && edge.Kind == kind {
			copied := edge
			return &copied
		}
	}
	return nil
}

func assertV1Alpha5GraphDiagnostic(t *testing.T, diagnostics []Diagnostic, code string) *Diagnostic {
	t.Helper()
	diagnostic := findV1Alpha5Diagnostic(diagnostics, code)
	if diagnostic == nil {
		t.Fatalf("expected diagnostic %s, got %#v", code, diagnostics)
	}
	if diagnostic.Severity != "error" || diagnostic.Phase != "C_STATIC" {
		t.Fatalf("%s identity = %s/%s, want error/C_STATIC", code, diagnostic.Severity, diagnostic.Phase)
	}
	if diagnostic.Path != v1alpha5TestProcedurePath {
		t.Fatalf("%s path = %q, want %q", code, diagnostic.Path, v1alpha5TestProcedurePath)
	}
	return diagnostic
}

func TestV1Alpha5GraphWaitEntryVerticesAndControlEdges(t *testing.T) {
	procedure := v1alpha5ParseTestProcedure(t, validV1Alpha5WaitProcedure())
	graph := buildV1Alpha5ProcedureGraph(procedure)

	if graph.Start() != "transfer-n1n2-message" {
		t.Fatalf("graph start = %q, want transfer-n1n2-message", graph.Start())
	}
	entryID := graph.WaitEntryVertex("update-sm-context")
	if entryID != "update-sm-context@entry" {
		t.Fatalf("Wait entry vertex = %q, want update-sm-context@entry", entryID)
	}
	if vertex := graph.Vertex(entryID); vertex == nil || !vertex.WaitEntry || vertex.NodeID != "update-sm-context" {
		t.Fatalf("unexpected entry vertex: %#v", vertex)
	}
	if vertex := graph.Vertex("update-sm-context"); vertex == nil || vertex.WaitEntry {
		t.Fatalf("Wait-bearing Node must keep a distinct business vertex: %#v", vertex)
	}
	if entry := graph.WaitEntryVertex("transfer-n1n2-message"); entry != "" {
		t.Fatalf("non-Wait Node must not have an entry vertex, got %q", entry)
	}

	// Four authored Nodes + one Wait entry vertex + four terminals.
	if got, want := len(graph.Vertices()), 9; got != want {
		t.Fatalf("vertex count = %d, want %d", got, want)
	}
	for _, terminal := range v1alpha5GraphTerminals {
		vertex := graph.Vertex(terminal)
		if vertex == nil || !vertex.Terminal {
			t.Fatalf("missing terminal vertex %q: %#v", terminal, vertex)
		}
	}

	// Entry vertex: implicit EVENT_RECEIVED into the Node's own first business
	// stage, plus the authored TIMEOUT edge.
	eventEdge := v1alpha5FindGraphEdge(graph, entryID, "update-sm-context", V1Alpha5EdgeEventReceived)
	if eventEdge == nil {
		t.Fatalf("missing implicit EVENT_RECEIVED edge from %s: %#v", entryID, graph.EdgesFrom(entryID))
	}
	if eventEdge.Stage != "" || eventEdge.Outcome != "" {
		t.Fatalf("EVENT_RECEIVED is not a Stage Outcome: %#v", eventEdge)
	}
	timeoutEdge := v1alpha5FindGraphEdge(graph, entryID, "handle-update-timeout", V1Alpha5EdgeTimeout)
	if timeoutEdge == nil {
		t.Fatalf("missing TIMEOUT edge from %s: %#v", entryID, graph.EdgesFrom(entryID))
	}

	// Every inter-node route edge that targets the Wait-bearing Node targets its
	// entry vertex, never the business vertex directly.
	if edge := v1alpha5FindGraphEdge(graph, "transfer-n1n2-message", entryID, V1Alpha5EdgeStageRoute); edge == nil ||
		edge.Stage != "operation" || edge.Outcome != "SUCCEEDED" {
		t.Fatalf("route into a Wait-bearing Node must target its entry vertex: %#v", graph.EdgesFrom("transfer-n1n2-message"))
	}
	for _, edge := range graph.Edges() {
		if edge.Kind == V1Alpha5EdgeStageRoute && edge.To == "update-sm-context" {
			t.Fatalf("stage route bypassed the Wait entry vertex: %#v", edge)
		}
	}
}

func TestV1Alpha5GraphWaitEntryDominance(t *testing.T) {
	procedure := v1alpha5ParseTestProcedure(t, validV1Alpha5WaitProcedure())
	graph := buildV1Alpha5ProcedureGraph(procedure)
	entryID := graph.WaitEntryVertex("update-sm-context")

	// The entry vertex dominates the Node's business stages.
	if !graph.Dominates(entryID, "update-sm-context") {
		t.Fatalf("%s must dominate update-sm-context", entryID)
	}
	// Every incoming edge to the business vertex is the event edge; every
	// incoming edge to the entry vertex is an ordinary route/start edge.
	incoming := graph.EdgesTo("update-sm-context")
	if len(incoming) != 1 || incoming[0].From != entryID || incoming[0].Kind != V1Alpha5EdgeEventReceived {
		t.Fatalf("business vertex must only be entered through the event edge: %#v", incoming)
	}
	for _, edge := range graph.EdgesTo(entryID) {
		if edge.Kind != V1Alpha5EdgeStageRoute {
			t.Fatalf("unexpected non-route incoming edge to the entry vertex: %#v", edge)
		}
	}

	// The TIMEOUT branch cannot reach the Node or its business stages on a valid
	// DAG.
	timeoutReachable := graph.ReachableFrom("handle-update-timeout")
	if timeoutReachable["update-sm-context"] || timeoutReachable[entryID] {
		t.Fatalf("TIMEOUT branch must not reach the Wait-bearing Node: %#v", timeoutReachable)
	}
}

func TestV1Alpha5GraphWaitBearingStartNode(t *testing.T) {
	procedure := v1alpha5ParseTestProcedure(t, `start: wait-node
nodes:
  wait-node:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: SAFE_STOPPED
    operation:
      tool: ca::smf.x
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
`)
	graph := buildV1Alpha5ProcedureGraph(procedure)
	if graph.Start() != "wait-node@entry" {
		t.Fatalf("start = %q, want the start Node's Wait entry vertex", graph.Start())
	}
	if graph.StartNode() != "wait-node" {
		t.Fatalf("start Node = %q, want wait-node", graph.StartNode())
	}
	if edge := v1alpha5FindGraphEdge(graph, "wait-node@entry", "wait-node", V1Alpha5EdgeEventReceived); edge == nil {
		t.Fatalf("Wait-bearing start Node needs its implicit event edge: %#v", graph.EdgesFrom("wait-node@entry"))
	}
}

func TestV1Alpha5GraphPositiveProcedureHasZeroDiagnostics(t *testing.T) {
	for name, procedure := range map[string]string{
		"single wait": validV1Alpha5WaitProcedure(),
		"multi wait":  v1alpha5MultiWaitProcedure(),
		"wait start": `start: wait-node
nodes:
  wait-node:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: SAFE_STOPPED
    operation:
      tool: ca::smf.x
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
`,
	} {
		if diagnostics := v1alpha5CheckProcedure(t, procedure); len(diagnostics) != 0 {
			t.Fatalf("%s: valid v1alpha5 Procedure must have zero diagnostics, got %#v", name, diagnostics)
		}
	}
}

func TestV1Alpha5GraphTimeoutEdgeCycle(t *testing.T) {
	cases := map[string]string{
		"timeout re-enters a preceding Node": `start: a
nodes:
  a:
    operation:
      tool: ca::amf.a
      arguments: {}
      routes:
        SUCCEEDED: wait-node
  wait-node:
    wait:
      event:
        source: AMF
        service: Nsmf_PDUSession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: a
    operation:
      tool: ca::smf.b
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
`,
		"timeout self-target": `start: wait-node
nodes:
  wait-node:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: wait-node
    operation:
      tool: ca::smf.x
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
`,
	}
	for name, procedure := range cases {
		t.Run(name, func(t *testing.T) {
			diagnostics := v1alpha5CheckProcedure(t, procedure)
			diagnostic := assertV1Alpha5GraphDiagnostic(t, diagnostics, "CONTROL_FLOW_CYCLE")
			if diagnostic.Line != 1 {
				t.Fatalf("CONTROL_FLOW_CYCLE should anchor at the Procedure root, got %d:%d", diagnostic.Line, diagnostic.Column)
			}
			// The TIMEOUT target is a declared Node, so no route-target error.
			if hasV1Alpha5Diagnostic(diagnostics, "WAIT_TIMEOUT_ROUTE_INVALID") ||
				hasV1Alpha5Diagnostic(diagnostics, "CONTROL_FLOW_DANGLING") {
				t.Fatalf("unexpected target diagnostics for a declared cycle target: %#v", diagnostics)
			}
		})
	}
}

func TestV1Alpha5GraphDanglingTimeoutTarget(t *testing.T) {
	procedure := `start: wait-node
nodes:
  wait-node:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: missing-timeout-handler
    operation:
      tool: ca::smf.x
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
`
	diagnostics := v1alpha5CheckProcedure(t, procedure)
	// TIMEOUT target grammar accepts a NodeId, so the declaration check is WP5's
	// inherited CONTROL_FLOW_DANGLING, anchored at the TIMEOUT value.
	assertV1Alpha5DiagnosticAt(t, diagnostics, "CONTROL_FLOW_DANGLING", 11, 18)
	if hasV1Alpha5Diagnostic(diagnostics, "WAIT_TIMEOUT_ROUTE_INVALID") {
		t.Fatalf("a well-formed but undeclared NodeId target is not a Wait grammar error: %#v", diagnostics)
	}
}

func TestV1Alpha5GraphDanglingResponseAfterTarget(t *testing.T) {
	procedure := `start: wait-node
nodes:
  wait-node:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      response:
        after: missing-boundary
      routes:
        TIMEOUT: SAFE_STOPPED
    operation:
      tool: ca::smf.x
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
`
	diagnostics := v1alpha5CheckProcedure(t, procedure)
	// WP5 checks only declaration here; reachability/post-dominance is WP6.
	assertV1Alpha5DiagnosticAt(t, diagnostics, "CONTROL_FLOW_DANGLING", 11, 16)
	if hasV1Alpha5Diagnostic(diagnostics, "WAIT_RESPONSE_BOUNDARY_INVALID") {
		t.Fatalf("a syntactically valid after value must not be a WP2 boundary-shape error: %#v", diagnostics)
	}
}

func TestV1Alpha5GraphDanglingBusinessRouteTarget(t *testing.T) {
	procedure := `start: a
nodes:
  a:
    operation:
      tool: ca::smf.x
      arguments: {}
      routes:
        SUCCEEDED: missing-successor
`
	diagnostics := v1alpha5CheckProcedure(t, procedure)
	assertV1Alpha5DiagnosticAt(t, diagnostics, "CONTROL_FLOW_DANGLING", 8, 20)
}

func TestV1Alpha5GraphFinalSuccessRequiresPostconditionPASS(t *testing.T) {
	t.Run("non-final SUCCEEDED route is FINAL_POSTCONDITION_MISSING", func(t *testing.T) {
		procedure := `start: a
nodes:
  a:
    operation:
      tool: ca::smf.x
      arguments: {}
      routes:
        SUCCEEDED: SUCCEEDED
`
		diagnostics := v1alpha5CheckProcedure(t, procedure)
		assertV1Alpha5GraphDiagnostic(t, diagnostics, "FINAL_POSTCONDITION_MISSING")
	})

	t.Run("final Postcondition PASS is a legal Goal proof", func(t *testing.T) {
		procedure := `start: a
nodes:
  a:
    postcondition:
      predicates:
        - predicate: session.goal_satisfied
          arguments: []
      routes:
        PASS: SUCCEEDED
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
`
		if diagnostics := v1alpha5CheckProcedure(t, procedure); len(diagnostics) != 0 {
			t.Fatalf("a final Postcondition PASS edge must be accepted, got %#v", diagnostics)
		}
	})

	t.Run("illegal success next to a legal proof is a bypass", func(t *testing.T) {
		procedure := `start: a
nodes:
  a:
    operation:
      tool: ca::smf.x
      arguments: {}
      routes:
        SUCCEEDED: SUCCEEDED
    postcondition:
      predicates:
        - predicate: session.goal_satisfied
          arguments: []
      routes:
        PASS: SUCCEEDED
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
`
		diagnostics := v1alpha5CheckProcedure(t, procedure)
		assertV1Alpha5GraphDiagnostic(t, diagnostics, "SUCCESS_BYPASSES_FINAL_POSTCONDITION")
		if hasV1Alpha5Diagnostic(diagnostics, "FINAL_POSTCONDITION_MISSING") {
			t.Fatalf("a legitimate Goal proof exists, so the missing-proof code is wrong: %#v", diagnostics)
		}
	})

	t.Run("unreachable illegal success is not a reachable Goal proof", func(t *testing.T) {
		procedure := `start: a
nodes:
  a:
    postcondition:
      predicates:
        - predicate: session.goal_satisfied
          arguments: []
      routes:
        PASS: SUCCEEDED
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
  isolated:
    operation:
      tool: ca::smf.y
      arguments: {}
      routes:
        SUCCEEDED: SUCCEEDED
`
		if diagnostics := v1alpha5CheckProcedure(t, procedure); len(diagnostics) != 0 {
			t.Fatalf("unreachable Nodes are outside the reachable Goal proof graph, got %#v", diagnostics)
		}
	})
}

func TestV1Alpha5GraphEventReceivedIsNotGoalProof(t *testing.T) {
	// A Wait-bearing Node whose accepted-event path never reaches a final
	// Postcondition PASS proves nothing, even though the implicit event edge was
	// taken.
	procedure := `start: wait-node
nodes:
  wait-node:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: SAFE_STOPPED
    operation:
      tool: ca::smf.x
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
`
	diagnostics := v1alpha5CheckProcedure(t, procedure)
	if len(diagnostics) != 0 {
		t.Fatalf("valid Wait Procedure must have zero diagnostics, got %#v", diagnostics)
	}
	graph := buildV1Alpha5ProcedureGraph(v1alpha5ParseTestProcedure(t, procedure))
	eventEdges, proofEdges := 0, 0
	for _, edge := range graph.Edges() {
		if edge.Kind == V1Alpha5EdgeEventReceived || edge.Kind == V1Alpha5EdgeTimeout {
			if graph.IsGoalProofEdge(edge) {
				t.Fatalf("control edge must never be a Goal proof: %#v", edge)
			}
		}
		if edge.Kind == V1Alpha5EdgeEventReceived {
			eventEdges++
			if edge.To == "SUCCEEDED" {
				t.Fatalf("EVENT_RECEIVED must never target the success terminal: %#v", edge)
			}
		}
		if graph.IsGoalProofEdge(edge) {
			proofEdges++
		}
	}
	if eventEdges != 1 {
		t.Fatalf("expected one implicit EVENT_RECEIVED edge, got %d", eventEdges)
	}
	if proofEdges != 0 {
		t.Fatalf("event receipt alone must not produce a Goal proof, got %d", proofEdges)
	}

	// In the positive multi-Wait Procedure only the final Postcondition PASS edge
	// is a Goal proof.
	multi := buildV1Alpha5ProcedureGraph(v1alpha5ParseTestProcedure(t, v1alpha5MultiWaitProcedure()))
	proofs := 0
	for _, edge := range multi.Edges() {
		if multi.IsGoalProofEdge(edge) {
			proofs++
			if edge.Stage != "postcondition" || edge.Outcome != "PASS" || edge.To != "SUCCEEDED" {
				t.Fatalf("unexpected Goal proof edge: %#v", edge)
			}
		}
	}
	if proofs != 1 {
		t.Fatalf("expected exactly one final Postcondition PASS Goal proof, got %d", proofs)
	}
}

func TestV1Alpha5GraphDominatorAndPostDominatorAlgorithm(t *testing.T) {
	// a -> {b, c}; b -> d; c -> d; d -> SAFE_STOPPED.
	procedure := `start: a
nodes:
  a:
    operation:
      tool: ca::smf.a
      arguments: {}
      routes:
        SUCCEEDED: b
        FAILED: c
  b:
    operation:
      tool: ca::smf.b
      arguments: {}
      routes:
        SUCCEEDED: d
  c:
    operation:
      tool: ca::smf.c
      arguments: {}
      routes:
        SUCCEEDED: d
  d:
    operation:
      tool: ca::smf.d
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
`
	graph := buildV1Alpha5ProcedureGraph(v1alpha5ParseTestProcedure(t, procedure))

	dominatorAssertions := []struct {
		dominator string
		node      string
		want      bool
	}{
		{"a", "a", true},
		{"a", "b", true},
		{"a", "c", true},
		{"a", "d", true},
		{"b", "d", false},
		{"c", "d", false},
		{"d", "a", false},
		{"b", "c", false},
	}
	for _, assertion := range dominatorAssertions {
		if got := graph.Dominates(assertion.dominator, assertion.node); got != assertion.want {
			t.Fatalf("Dominates(%q, %q) = %v, want %v", assertion.dominator, assertion.node, got, assertion.want)
		}
	}

	postDominatorAssertions := []struct {
		postDominator string
		node          string
		want          bool
	}{
		{"d", "b", true},
		{"d", "c", true},
		{"d", "a", true},
		{"SAFE_STOPPED", "d", true},
		{"SAFE_STOPPED", "a", true},
		{"a", "b", false},
		{"b", "a", false},
		{"a", "SAFE_STOPPED", false},
	}
	for _, assertion := range postDominatorAssertions {
		if got := graph.PostDominates(assertion.postDominator, assertion.node); got != assertion.want {
			t.Fatalf("PostDominates(%q, %q) = %v, want %v", assertion.postDominator, assertion.node, got, assertion.want)
		}
	}

	// The virtual exit used by the post-dominator computation is never exposed.
	for vertex, set := range graph.PostDominators() {
		for dominator := range set {
			if dominator == v1alpha5GraphVirtualExit {
				t.Fatalf("virtual exit leaked into PostDominators()[%q]", vertex)
			}
		}
	}

	// Unreachable vertices are dominated by nothing.
	if graph.Dominates("a", "unreachable") {
		t.Fatal("an unreachable vertex must not be dominated")
	}
	if graph.PostDominates("a", "unreachable") {
		t.Fatal("an unreachable vertex must not be post-dominated")
	}
}

func TestV1Alpha5GraphMultiWaitProcedure(t *testing.T) {
	procedure := v1alpha5ParseTestProcedure(t, v1alpha5MultiWaitProcedure())
	graph := buildV1Alpha5ProcedureGraph(procedure)

	for _, nodeID := range []string{"wait-sm-context", "wait-n4"} {
		entryID := graph.WaitEntryVertex(nodeID)
		if entryID == "" {
			t.Fatalf("missing Wait entry vertex for %q", nodeID)
		}
		if !graph.Dominates(entryID, nodeID) {
			t.Fatalf("%s must dominate %s", entryID, nodeID)
		}
		if edge := v1alpha5FindGraphEdge(graph, entryID, nodeID, V1Alpha5EdgeEventReceived); edge == nil {
			t.Fatalf("%s missing implicit event edge", nodeID)
		}
		if edge := v1alpha5FindGraphEdge(graph, entryID, "handle-update-timeout", V1Alpha5EdgeTimeout); edge == nil {
			t.Fatalf("%s missing TIMEOUT edge", nodeID)
		}
	}
	// The second Wait is entered through the first Wait Node's business stage;
	// the route must land on the second entry vertex.
	if edge := v1alpha5FindGraphEdge(graph, "wait-sm-context", "wait-n4@entry", V1Alpha5EdgeStageRoute); edge == nil ||
		edge.Outcome != "SUCCEEDED" {
		t.Fatalf("second Wait must be entered through its entry vertex: %#v", graph.EdgesFrom("wait-sm-context"))
	}
}

func TestV1Alpha5GraphDiagnosticsAreDeterministic(t *testing.T) {
	procedure := `start: a
nodes:
  a:
    operation:
      tool: ca::smf.a
      arguments: {}
      routes:
        SUCCEEDED: missing-node
        FAILED: SAFE_STOPPED
  wait-node:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      response:
        after: missing-boundary
      routes:
        TIMEOUT: missing-handler
    operation:
      tool: ca::smf.b
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
`
	first := v1alpha5CheckProcedure(t, procedure)
	second := v1alpha5CheckProcedure(t, procedure)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("graph diagnostics must be deterministic:\nfirst:  %#v\nsecond: %#v", first, second)
	}

	// Deterministic graph evidence: rebuilding yields the same vertices/edges.
	firstGraph := buildV1Alpha5ProcedureGraph(v1alpha5ParseTestProcedure(t, procedure))
	secondGraph := buildV1Alpha5ProcedureGraph(v1alpha5ParseTestProcedure(t, procedure))
	if !reflect.DeepEqual(firstGraph.Edges(), secondGraph.Edges()) {
		t.Fatal("graph edges must be deterministic")
	}
	if !reflect.DeepEqual(firstGraph.Vertices(), secondGraph.Vertices()) {
		t.Fatal("graph vertices must be deterministic")
	}

	codes := make([]string, 0, len(first))
	for _, diagnostic := range first {
		codes = append(codes, diagnostic.Code)
	}
	sort.Strings(codes)
	want := []string{"CONTROL_FLOW_DANGLING", "CONTROL_FLOW_DANGLING", "CONTROL_FLOW_DANGLING"}
	if !reflect.DeepEqual(codes, want) {
		t.Fatalf("diagnostic codes = %#v, want %#v", codes, want)
	}
}

// TestV1Alpha5GraphUnreachableWaitNodeIsNotDiagnosedOnItsOwn records the WP5
// scope boundary: reachability is computed and exposed for later WPs, but WP5
// does not invent an UNREACHABLE_NODE diagnostic and does not change the
// inherited diagnostic set.
func TestV1Alpha5GraphUnreachableWaitNodeIsNotDiagnosedOnItsOwn(t *testing.T) {
	procedure := `start: a
nodes:
  a:
    operation:
      tool: ca::smf.a
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
  unborn-wait:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: SAFE_STOPPED
    operation:
      tool: ca::smf.b
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
`
	if diagnostics := v1alpha5CheckProcedure(t, procedure); len(diagnostics) != 0 {
		t.Fatalf("WP5 must not invent an unreachable-Node diagnostic, got %#v", diagnostics)
	}
	graph := buildV1Alpha5ProcedureGraph(v1alpha5ParseTestProcedure(t, procedure))
	if graph.ReachableVertices()["unborn-wait@entry"] {
		t.Fatal("unreachable Wait entry vertex must not be reachable from start")
	}
	if graph.ReachableVertices()["unborn-wait"] {
		t.Fatal("unreachable Wait business vertex must not be reachable from start")
	}
}
