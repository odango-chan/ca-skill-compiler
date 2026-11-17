package compiler

// WP2: CA-SL v1alpha5 Node-entry Wait parser tests.
//
// Every rule implemented from Appendix L.1/L.1.1/L.11 has positive and negative
// executable coverage. Diagnostics are asserted with the exact Code, Phase,
// Severity, Path, Line and Column; positive cases go through the public Check
// entry with a synthetic v1alpha5 BuildRequest.

import (
	"reflect"
	"sort"
	"testing"
)

const v1alpha5TestProcedurePath = "skills/example/assurance/procedure.yaml"

func v1alpha5TestProcedureRequest(procedureYAML string) BuildRequest {
	return BuildRequest{
		APIVersion:     BuildRequestAPIVersionV1Alpha5,
		LanguageTarget: V1Alpha5Target,
		Files: []File{
			{Path: v1alpha5TestProcedurePath, Bytes: []byte(procedureYAML)},
		},
	}
}

func v1alpha5CheckProcedure(t *testing.T, procedureYAML string) []Diagnostic {
	t.Helper()
	return Check(v1alpha5TestProcedureRequest(procedureYAML))
}

func findV1Alpha5Diagnostic(diagnostics []Diagnostic, code string) *Diagnostic {
	for index := range diagnostics {
		if diagnostics[index].Code == code {
			return &diagnostics[index]
		}
	}
	return nil
}

func assertV1Alpha5DiagnosticAt(t *testing.T, diagnostics []Diagnostic, code string, line, column int) {
	t.Helper()
	diagnostic := findV1Alpha5Diagnostic(diagnostics, code)
	if diagnostic == nil {
		t.Fatalf("expected diagnostic %s, got %#v", code, diagnostics)
	}
	if diagnostic.Severity != "error" {
		t.Fatalf("%s severity = %q, want error", code, diagnostic.Severity)
	}
	if diagnostic.Phase != "C_STATIC" {
		t.Fatalf("%s phase = %q, want C_STATIC", code, diagnostic.Phase)
	}
	if diagnostic.Path != v1alpha5TestProcedurePath {
		t.Fatalf("%s path = %q, want %q", code, diagnostic.Path, v1alpha5TestProcedurePath)
	}
	if diagnostic.Line != line || diagnostic.Column != column {
		t.Fatalf("%s location = %d:%d, want %d:%d (message: %s)",
			code, diagnostic.Line, diagnostic.Column, line, column, diagnostic.Message)
	}
}

func hasV1Alpha5Diagnostic(diagnostics []Diagnostic, code string) bool {
	return findV1Alpha5Diagnostic(diagnostics, code) != nil
}

// v1alpha5SortedCodes is a deterministic projection used by assertions; the
// wire ordering itself is owned by sortDiagnostics in the shared API layer.
func v1alpha5SortedCodes(diagnostics []Diagnostic) []string {
	codes := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	sort.Strings(codes)
	return codes
}

// validV1Alpha5WaitProcedure is a complete illustrative Procedure: one node
// without Wait and one Wait-bearing node with all four inherited business
// stages plus the optional response boundary.
func validV1Alpha5WaitProcedure() string {
	return `start: transfer-n1n2-message
nodes:
  transfer-n1n2-message:
    operation:
      tool: ca::amf.communication.n1n2_message_transfer
      arguments:
        smContextRef: create-sm-context.operation.output.smContextRef
      routes:
        SUCCEEDED: update-sm-context
        FAILED: SAFE_STOPPED
        OUTCOME_UNKNOWN: SAFE_STOPPED
  update-sm-context:
    wait:
      event:
        source: AMF
        service: Nsmf_PDUSession
        operation: UpdateSMContext
      timeout: PT30S
      response:
        after: prepare-step17-response
      routes:
        TIMEOUT: handle-update-timeout
    precondition:
      predicates:
        - predicate: session.update_request.valid
          arguments: []
      routes:
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
    operation:
      tool: ca::smf.session.update_from_amf
      arguments:
        request: update-sm-context.wait.output
      routes:
        SUCCEEDED: prepare-step17-response
        FAILED: SAFE_STOPPED
        OUTCOME_UNKNOWN: SAFE_STOPPED
  prepare-step17-response:
    observation:
      tool: ca::smf.session.observe_step17_response
      arguments:
        request: update-sm-context.wait.output
      routes:
        SUCCEEDED: handle-update-timeout
        FAILED: SAFE_STOPPED
        OUTCOME_UNKNOWN: UNRESOLVED
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

func TestV1Alpha5WaitPositiveProcedure(t *testing.T) {
	diagnostics := v1alpha5CheckProcedure(t, validV1Alpha5WaitProcedure())
	if len(diagnostics) != 0 {
		t.Fatalf("valid v1alpha5 Wait procedure must parse with zero diagnostics, got %#v", diagnostics)
	}
	// The candidate Wait syntax must never surface the frozen v1alpha4
	// rejection for a valid v1alpha5 declaration.
	if hasV1Alpha5Diagnostic(diagnostics, "UNSUPPORTED_WAIT") {
		t.Fatal("valid v1alpha5 declaration must never emit UNSUPPORTED_WAIT")
	}
}

func TestV1Alpha5WaitPositiveRoutesWithoutResponse(t *testing.T) {
	for _, target := range []string{"FAILED", "UNRESOLVED", "SAFE_STOPPED", "handle-update-timeout"} {
		procedure := `start: wait-node
nodes:
  wait-node:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: P1DT2H30M
      routes:
        TIMEOUT: ` + target + `
    operation:
      tool: ca::smf.session.update_from_amf
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
  handle-update-timeout:
    operation:
      tool: ca::smf.update_timeout.handle
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
`
		if diagnostics := v1alpha5CheckProcedure(t, procedure); len(diagnostics) != 0 {
			t.Fatalf("TIMEOUT -> %q must be accepted without response, got %#v", target, diagnostics)
		}
	}
}

func TestV1Alpha5FixedDurationLexicalSubset(t *testing.T) {
	accepted := []string{
		"PT30S", "PT1M", "PT5M", "PT2H", "P1D", "P1DT2H30M",
		"PT1H2S", "P2DT3H4M5S", "PT2M3S", "PT1H30M",
		"P999999999999999999999D", "PT999999999999999999H",
	}
	for _, value := range accepted {
		if !validV1Alpha5FixedDuration(value) {
			t.Errorf("expected %q to be an accepted v1alpha5 FixedDuration", value)
		}
	}
	rejected := []string{
		// Empty time part or no component at all.
		"", "P", "PT", "P1DT", "P1DT0S", "PT0H", "PT0M",
		// Calendar units.
		"P1M", "P1Y", "P1W", "P1D1M",
		// Zero values and leading zeros.
		"PT0S", "P0D", "PT00S", "PT01M", "P01D", "PT001H",
		// Signs and decimals.
		"PT-1S", "PT+1S", "PT0.5S", "P1.5D", "PT1.5H", "P-1D",
		// Ordering and duplicate components.
		"PT1S2M", "P1D2H", "PT1MS", "PT2H1H", "PT1S2S", "P1DT2H3M4S5M",
		// Lowercase and alternate spellings.
		"pt30s", "P1d", "PT1h", "1D", "T1H", "PT 1S", "PT1S ", " PT1S",
		// Wrong YAML-ish forms.
		"30", "PT30", "P1DT2H30",
	}
	for _, value := range rejected {
		if validV1Alpha5FixedDuration(value) {
			t.Errorf("expected %q to be rejected by the v1alpha5 FixedDuration grammar", value)
		}
	}
}

// TestV1Alpha5WaitDiagnosticAnchors asserts exact Code/line/column anchors for
// each Wait structural/lexical rule. Line numbers are 1-based; columns are
// 1-based and point at the offending key/value node or the nearest existing
// parent mapping for a missing required field.
func TestV1Alpha5WaitDiagnosticAnchors(t *testing.T) {
	tests := []struct {
		name   string
		yaml   string
		code   string
		line   int
		column int
	}{
		{
			name: "missing event anchors at wait parent",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      timeout: PT30S
      routes:
        TIMEOUT: FAILED
`,
			code: "WAIT_DECLARATION_INVALID", line: 8, column: 14,
		},
		{
			name: "event not a mapping anchors at value",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event: SMF
      timeout: PT30S
      routes:
        TIMEOUT: FAILED
`,
			code: "WAIT_DECLARATION_INVALID", line: 8, column: 14,
		},
		{
			name: "missing event source anchors at event parent",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: FAILED
`,
			code: "WAIT_DECLARATION_INVALID", line: 9, column: 16,
		},
		{
			name: "event selector wrong type anchors at value",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: 5
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: FAILED
`,
			code: "WAIT_DECLARATION_INVALID", line: 9, column: 17,
		},
		{
			name: "event selector empty string anchors at value",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: ""
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: FAILED
`,
			code: "WAIT_DECLARATION_INVALID", line: 10, column: 18,
		},
		{
			name: "missing timeout anchors at wait parent",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      routes:
        TIMEOUT: FAILED
`,
			code: "WAIT_TIMEOUT_INVALID", line: 8, column: 12,
		},
		{
			name: "non-string timeout anchors at value",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: 30
      routes:
        TIMEOUT: FAILED
`,
			code: "WAIT_TIMEOUT_INVALID", line: 12, column: 16,
		},
		{
			name: "calendar month timeout anchors at value",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: P1M
      routes:
        TIMEOUT: FAILED
`,
			code: "WAIT_TIMEOUT_INVALID", line: 12, column: 16,
		},
		{
			name: "missing routes anchors at wait parent",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
`,
			code: "WAIT_TIMEOUT_ROUTE_INVALID", line: 8, column: 12,
		},
		{
			name: "routes not a mapping anchors at value",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      routes: yes
`,
			code: "WAIT_TIMEOUT_ROUTE_INVALID", line: 13, column: 15,
		},
		{
			name: "EVENT_RECEIVED extra route key anchors at key",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: FAILED
        EVENT_RECEIVED: continue
`,
			code: "WAIT_TIMEOUT_ROUTE_INVALID", line: 15, column: 9,
		},
		{
			name: "TIMEOUT to SUCCEEDED anchors at value",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: SUCCEEDED
`,
			code: "WAIT_TIMEOUT_ROUTE_INVALID", line: 14, column: 18,
		},
		{
			name: "response missing after anchors at response parent",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      response: {}
      routes:
        TIMEOUT: FAILED
`,
			code: "WAIT_RESPONSE_BOUNDARY_INVALID", line: 13, column: 17,
		},
		{
			name: "response not a mapping anchors at value",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      response: yes
      routes:
        TIMEOUT: FAILED
`,
			code: "WAIT_RESPONSE_BOUNDARY_INVALID", line: 13, column: 17,
		},
		{
			name: "response.after terminal anchors at value",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      response:
        after: SUCCEEDED
      routes:
        TIMEOUT: FAILED
`,
			code: "WAIT_RESPONSE_BOUNDARY_INVALID", line: 14, column: 16,
		},
		{
			name: "response.after context anchors at value",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      response:
        after: context
      routes:
        TIMEOUT: FAILED
`,
			code: "WAIT_RESPONSE_BOUNDARY_INVALID", line: 14, column: 16,
		},
		{
			name: "wait-only node anchors at wait mapping",
			yaml: `start: wait-node
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
`,
			code: "WAIT_NODE_STAGE_REQUIRED", line: 5, column: 12,
		},
		{
			name: "wait not a mapping anchors at value",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait: true
`,
			code: "WAIT_DECLARATION_INVALID", line: 7, column: 11,
		},
		{
			name: "unknown Node field anchors at key",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: FAILED
    poll: true
`,
			code: "UNKNOWN_FIELD", line: 15, column: 5,
		},
		{
			name: "unknown wait field anchors at key",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: FAILED
      onTimeout: SAFE_STOPPED
`,
			code: "UNKNOWN_FIELD", line: 15, column: 7,
		},
		{
			name: "unknown event field anchors at key",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
        payloadSchema: schemas/x.json
      timeout: PT30S
      routes:
        TIMEOUT: FAILED
`,
			code: "UNKNOWN_FIELD", line: 12, column: 9,
		},
		{
			name: "unknown response field anchors at key",
			yaml: `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      response:
        after: prepare-step17-response
        status: 200
      routes:
        TIMEOUT: FAILED
`,
			code: "UNKNOWN_FIELD", line: 15, column: 9,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagnostics := v1alpha5CheckProcedure(t, test.yaml)
			assertV1Alpha5DiagnosticAt(t, diagnostics, test.code, test.line, test.column)
		})
	}
}

func TestV1Alpha5WaitDuplicateYAMLKeyIsParseError(t *testing.T) {
	procedure := `start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      timeout: PT1M
      routes:
        TIMEOUT: FAILED
`
	diagnostics := v1alpha5CheckProcedure(t, procedure)
	diagnostic := findV1Alpha5Diagnostic(diagnostics, "YAML_PARSE_ERROR")
	if diagnostic == nil {
		t.Fatalf("duplicate YAML mapping key must produce YAML_PARSE_ERROR, got %#v", diagnostics)
	}
	if diagnostic.Phase != "C_STATIC" || diagnostic.Severity != "error" {
		t.Fatalf("unexpected duplicate-key diagnostic: %#v", diagnostic)
	}
	if diagnostic.Line != 13 || diagnostic.Column != 7 {
		t.Fatalf("duplicate-key location = %d:%d, want 13:7", diagnostic.Line, diagnostic.Column)
	}
	if hasV1Alpha5Diagnostic(diagnostics, "WAIT_TIMEOUT_INVALID") {
		t.Fatal("a YAML parse failure must not fabricate Wait semantic diagnostics")
	}
}

func TestV1Alpha5WaitMissingRequiredSections(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		code string
	}{
		{
			name: "missing event",
			yaml: `start: wait-node
nodes:
  wait-node:
    wait:
      timeout: PT30S
      routes:
        TIMEOUT: FAILED
    operation:
      tool: ca::smf.x
      arguments: {}
`,
			code: "WAIT_DECLARATION_INVALID",
		},
		{
			name: "missing timeout",
			yaml: `start: wait-node
nodes:
  wait-node:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      routes:
        TIMEOUT: FAILED
    operation:
      tool: ca::smf.x
      arguments: {}
`,
			code: "WAIT_TIMEOUT_INVALID",
		},
		{
			name: "missing routes",
			yaml: `start: wait-node
nodes:
  wait-node:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
    operation:
      tool: ca::smf.x
      arguments: {}
`,
			code: "WAIT_TIMEOUT_ROUTE_INVALID",
		},
		{
			name: "missing TIMEOUT route key",
			yaml: `start: wait-node
nodes:
  wait-node:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      routes: {}
    operation:
      tool: ca::smf.x
      arguments: {}
`,
			code: "WAIT_TIMEOUT_ROUTE_INVALID",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagnostics := v1alpha5CheckProcedure(t, test.yaml)
			if !hasV1Alpha5Diagnostic(diagnostics, test.code) {
				t.Fatalf("expected %s, got %#v", test.code, diagnostics)
			}
		})
	}
}

func TestV1Alpha5WaitSelectorValuesComparedExactly(t *testing.T) {
	parsed := v1alpha5ParseTestProcedure(t, validV1Alpha5WaitProcedure())
	node := parsed.ByID["update-sm-context"]
	if node == nil || node.Wait == nil {
		t.Fatalf("expected parsed Wait on update-sm-context: %#v", node)
	}
	if node.Wait.Event.Source != "AMF" ||
		node.Wait.Event.Service != "Nsmf_PDUSession" ||
		node.Wait.Event.Operation != "UpdateSMContext" {
		t.Fatalf("selector values must be preserved exactly: %#v", node.Wait.Event)
	}
	if node.Wait.Response == nil || node.Wait.Response.After != "prepare-step17-response" {
		t.Fatalf("unexpected response boundary: %#v", node.Wait.Response)
	}
	if node.Wait.Routes.Timeout != "handle-update-timeout" {
		t.Fatalf("unexpected TIMEOUT route: %#v", node.Wait.Routes)
	}
}

func TestV1Alpha5WaitDiagnosticsAreDeterministic(t *testing.T) {
	procedure := `start: wait-node
nodes:
  wait-node:
    wait: true
    poll: true
`
	first := v1alpha5CheckProcedure(t, procedure)
	second := v1alpha5CheckProcedure(t, procedure)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("diagnostics must be deterministic:\nfirst:  %#v\nsecond: %#v", first, second)
	}
	codes := v1alpha5SortedCodes(first)
	if !reflect.DeepEqual(codes, []string{"UNKNOWN_FIELD", "WAIT_DECLARATION_INVALID", "WAIT_NODE_STAGE_REQUIRED"}) {
		t.Fatalf("unexpected diagnostic set: %#v", codes)
	}
}

func TestV1Alpha5WaitTimeoutTargetGrammar(t *testing.T) {
	for _, target := range []string{"FAILED", "UNRESOLVED", "SAFE_STOPPED", "timeout-handler", "Step_17", "n2"} {
		if !validV1Alpha5WaitTimeoutTarget(target) {
			t.Errorf("expected valid TIMEOUT target %q", target)
		}
	}
	// "UNKNOWN" is a syntactically valid NodeId; whether it identifies a
	// declared Node is the WP5 CONTROL_FLOW_DANGLING check.
	for _, target := range []string{"", "SUCCEEDED", "context", "1bad", "bad target", "FAILED "} {
		if validV1Alpha5WaitTimeoutTarget(target) {
			t.Errorf("expected invalid TIMEOUT target %q", target)
		}
	}
	if !validV1Alpha5WaitTimeoutTarget("UNKNOWN") {
		t.Error("UNKNOWN is a syntactically valid NodeId; declaration checks are deferred to WP5")
	}
}

func TestV1Alpha5WaitResponseBoundaryGrammar(t *testing.T) {
	for _, after := range []string{"prepare-step17-response", "Step_17", "n2"} {
		if !validV1Alpha5ResponseBoundaryNodeID(after) {
			t.Errorf("expected valid response.after %q", after)
		}
	}
	for _, after := range []string{"", "SUCCEEDED", "FAILED", "SAFE_STOPPED", "UNRESOLVED", "context", "1bad", "bad target"} {
		if validV1Alpha5ResponseBoundaryNodeID(after) {
			t.Errorf("expected invalid response.after %q", after)
		}
	}
}
