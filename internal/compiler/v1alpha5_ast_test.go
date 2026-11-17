package compiler

// WP2: CA-SL v1alpha5 Procedure AST integration tests.
//
// The Wait declaration must be parsed inside the real Procedure pass, with raw
// nodes pointing into the one normalized YAML parse result and source spans for
// later graph/response work packages.

import (
	"testing"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func v1alpha5ParseTestProcedure(t *testing.T, procedureYAML string) *V1Alpha5ProcedureAST {
	t.Helper()
	root, err := yaml.Parse([]byte(procedureYAML))
	if err != nil {
		t.Fatalf("parse test procedure YAML: %v", err)
	}
	files := map[string]*yaml.Node{v1alpha5TestProcedurePath: root}
	skill, diagnostics := parseV1Alpha5SkillAST("example", files)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected parse diagnostics: %#v", diagnostics)
	}
	if skill == nil || skill.Procedure == nil {
		t.Fatal("expected parsed v1alpha5 Procedure AST")
	}
	return skill.Procedure
}

func TestV1Alpha5ProcedureASTParsesWaitInsideProcedurePass(t *testing.T) {
	procedure := v1alpha5ParseTestProcedure(t, validV1Alpha5WaitProcedure())

	if procedure.Span.Path != v1alpha5TestProcedurePath || procedure.Span.Line != 1 {
		t.Fatalf("unexpected Procedure span: %#v", procedure.Span)
	}
	if scalar(procedure.Start) != "transfer-n1n2-message" {
		t.Fatalf("unexpected start: %q", scalar(procedure.Start))
	}
	if len(procedure.ByID) != 4 {
		t.Fatalf("expected 4 parsed Nodes, got %d: %#v", len(procedure.ByID), procedure.ByID)
	}

	plain := procedure.ByID["transfer-n1n2-message"]
	if plain == nil || plain.Wait != nil {
		t.Fatalf("non-Wait Node must have no Wait AST: %#v", plain)
	}
	if plain.Operation == nil || child(plain.Operation, "tool") == nil {
		t.Fatalf("expected raw operation stage node: %#v", plain.Operation)
	}

	waitNode := procedure.ByID["update-sm-context"]
	if waitNode == nil || waitNode.Wait == nil {
		t.Fatalf("expected parsed Wait AST: %#v", waitNode)
	}
	if !waitNode.HasBusinessStage() {
		t.Fatal("Wait-bearing Node must expose its inherited business stages")
	}
	if waitNode.Precondition == nil || waitNode.Operation == nil {
		t.Fatalf("expected declared stage raw nodes: %#v", waitNode)
	}
	if waitNode.Observation != nil || waitNode.Postcondition != nil {
		t.Fatalf("undeclared stages must stay nil: %#v", waitNode)
	}
	if waitNode.Wait.Raw != child(waitNode.Raw, "wait") {
		t.Fatal("Wait Raw must point into the same normalized Procedure parse result")
	}
	if waitNode.Wait.Span.Path != v1alpha5TestProcedurePath {
		t.Fatalf("unexpected Wait span path: %#v", waitNode.Wait.Span)
	}
	if waitNode.Wait.Timeout != "PT30S" {
		t.Fatalf("unexpected timeout: %q", waitNode.Wait.Timeout)
	}
	if waitNode.Wait.Response == nil || waitNode.Wait.Response.After != "prepare-step17-response" {
		t.Fatalf("unexpected response boundary: %#v", waitNode.Wait.Response)
	}
	if waitNode.Wait.Routes.Timeout != "handle-update-timeout" {
		t.Fatalf("unexpected TIMEOUT route: %#v", waitNode.Wait.Routes)
	}
}

func TestV1Alpha5ProcedureASTSpansAreSourcePositioned(t *testing.T) {
	procedureYAML := `start: wait-node
nodes:
  wait-node:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: FAILED
    operation:
      tool: ca::smf.x
      arguments: {}
`
	procedure := v1alpha5ParseTestProcedure(t, procedureYAML)
	node := procedure.ByID["wait-node"]
	if node == nil {
		t.Fatal("expected parsed wait-node")
	}
	if node.Span.Path != v1alpha5TestProcedurePath || node.Span.Line != 4 || node.Span.Column <= 0 {
		t.Fatalf("unexpected Node span: %#v", node.Span)
	}
	if node.Wait == nil || node.Wait.Span.Line != 5 || node.Wait.Span.Column <= 0 {
		t.Fatalf("unexpected Wait span: %#v", node.Wait)
	}
	if node.Wait.Event.Span.Line != 6 || node.Wait.Event.Span.Column <= 0 {
		t.Fatalf("unexpected event span: %#v", node.Wait.Event.Span)
	}
	if node.Wait.Routes.Span.Line != 11 || node.Wait.Routes.Span.Column <= 0 {
		t.Fatalf("unexpected routes span: %#v", node.Wait.Routes.Span)
	}
}

func TestV1Alpha5WaitResponseWithoutAfterUsesDefaultBoundary(t *testing.T) {
	// Appendix L.1: WaitResponse = map(after?: NodeId). Appendix L.8.1: when
	// wait.response.after is absent the response boundary is the Wait-bearing
	// Node's own completion, so a present response mapping without "after" is
	// syntactically valid.
	procedureYAML := `start: wait-node
nodes:
  wait-node:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      response: {}
      routes:
        TIMEOUT: FAILED
    operation:
      tool: ca::smf.x
      arguments: {}
`
	diagnostics := v1alpha5CheckProcedure(t, procedureYAML)
	if len(diagnostics) != 0 {
		t.Fatalf("response without after must be valid, got %#v", diagnostics)
	}
	if hasV1Alpha5Diagnostic(diagnostics, "WAIT_RESPONSE_BOUNDARY_INVALID") {
		t.Fatal("omitted response.after must not be reported as WAIT_RESPONSE_BOUNDARY_INVALID")
	}
	procedure := v1alpha5ParseTestProcedure(t, procedureYAML)
	wait := procedure.ByID["wait-node"].Wait
	if wait == nil || wait.Response == nil {
		t.Fatalf("expected parsed response mapping: %#v", wait)
	}
	if wait.Response.After != "" {
		t.Fatalf("omitted after must leave the default boundary (empty After), got %q", wait.Response.After)
	}
}

func TestV1Alpha5ProcedureRootRequiresDeclaredStart(t *testing.T) {
	procedureYAML := `start: missing-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
`
	diagnostics := v1alpha5CheckProcedure(t, procedureYAML)
	assertV1Alpha5DiagnosticAt(t, diagnostics, "PROCEDURE_START_MISSING", 1, 8)
}

func TestV1Alpha5ProcedureUnknownRootField(t *testing.T) {
	procedureYAML := `start: wait-node
goal: session.established
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
`
	diagnostics := v1alpha5CheckProcedure(t, procedureYAML)
	assertV1Alpha5DiagnosticAt(t, diagnostics, "UNKNOWN_FIELD", 2, 1)
}

func TestV1Alpha5ProcedureReservedNodeID(t *testing.T) {
	procedureYAML := `start: SAFE_STOPPED
nodes:
  SAFE_STOPPED:
    operation:
      tool: ca::smf.x
      arguments: {}
`
	diagnostics := v1alpha5CheckProcedure(t, procedureYAML)
	if !hasV1Alpha5Diagnostic(diagnostics, "NODE_ID_RESERVED") {
		t.Fatalf("expected NODE_ID_RESERVED, got %#v", diagnostics)
	}
}

func TestV1Alpha5ProcedureMissingProcedureFile(t *testing.T) {
	req := BuildRequest{
		APIVersion:     BuildRequestAPIVersionV1Alpha5,
		LanguageTarget: V1Alpha5Target,
		Files: []File{
			{Path: "skills/example/SKILL.md", Bytes: []byte("# example\n")},
		},
	}
	diagnostics := Check(req)
	if !hasV1Alpha5Diagnostic(diagnostics, "SOURCE_ROOT_INVALID") {
		t.Fatalf("expected SOURCE_ROOT_INVALID for a missing procedure source, got %#v", diagnostics)
	}
}

func TestV1Alpha5ProcedureMultipleSkillsAreIsolated(t *testing.T) {
	// Two Skill packages parsed through their own AST passes: only the second one
	// has an invalid Wait, and diagnostics must be reported against the owning
	// package path. This asserts the WP2/WP5 AST layer only; the public Check
	// end-to-end path is covered by the WP3 trusted-fixture tests.
	procedures := map[string]string{
		"skills/alpha/assurance/procedure.yaml": `start: a
nodes:
  a:
    operation:
      tool: ca::smf.x
      arguments: {}
`,
		"skills/beta/assurance/procedure.yaml": `start: b
nodes:
  b:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT0S
      routes:
        TIMEOUT: FAILED
    operation:
      tool: ca::smf.x
      arguments: {}
`,
	}
	diagnostics := make([]Diagnostic, 0)
	for _, name := range []string{"alpha", "beta"} {
		path := "skills/" + name + "/assurance/procedure.yaml"
		root, err := yaml.Parse([]byte(procedures[path]))
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		files := map[string]*yaml.Node{path: root}
		_, skillDiagnostics := parseV1Alpha5SkillAST(name, files)
		diagnostics = append(diagnostics, skillDiagnostics...)
	}
	sortDiagnostics(diagnostics)
	if len(diagnostics) != 1 {
		t.Fatalf("expected exactly one diagnostic, got %#v", diagnostics)
	}
	if diagnostics[0].Code != "WAIT_TIMEOUT_INVALID" || diagnostics[0].Path != "skills/beta/assurance/procedure.yaml" {
		t.Fatalf("unexpected diagnostic: %#v", diagnostics[0])
	}
}
