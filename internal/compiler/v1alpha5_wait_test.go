package compiler

import (
	"strings"
	"testing"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func parseV1Alpha5WaitTestNode(t *testing.T, nodeYAML string) (*v1alpha5WaitAST, []Diagnostic) {
	t.Helper()
	source := "start: wait-node\nnodes:\n  wait-node:\n" + indentV1Alpha5TestYAML(nodeYAML, 4)
	root, err := yaml.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse test YAML: %v", err)
	}
	nodes := child(root, "nodes")
	node := child(nodes, "wait-node")
	return parseV1Alpha5WaitNode("skills/example/assurance/procedure.yaml", "wait-node", node)
}

func indentV1Alpha5TestYAML(value string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(strings.TrimSuffix(value, "\n"), "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n") + "\n"
}

func validV1Alpha5WaitNodeYAML() string {
	return `wait:
  event:
    source: SMF
    service: nsmf-pdusession
    operation: UpdateSMContext
  timeout: PT30S
  response:
    after: send-response
  routes:
    TIMEOUT: timeout-handler
operation:
  tool: ca::smf.session.modify
`
}

func hasV1Alpha5Diagnostic(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code && diagnostic.Phase == "C_STATIC" && diagnostic.Severity == "error" {
			return true
		}
	}
	return false
}

func TestParseV1Alpha5WaitNode(t *testing.T) {
	wait, diagnostics := parseV1Alpha5WaitTestNode(t, validV1Alpha5WaitNodeYAML())
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if wait == nil {
		t.Fatal("expected Wait AST")
	}
	if wait.Event.Source != "SMF" || wait.Event.Service != "nsmf-pdusession" || wait.Event.Operation != "UpdateSMContext" {
		t.Fatalf("unexpected exact event selector: %#v", wait.Event)
	}
	if wait.Timeout != "PT30S" {
		t.Fatalf("unexpected timeout: %q", wait.Timeout)
	}
	if wait.Response == nil || wait.Response.After != "send-response" {
		t.Fatalf("unexpected response boundary: %#v", wait.Response)
	}
	if wait.Routes.Timeout != "timeout-handler" {
		t.Fatalf("unexpected TIMEOUT target: %#v", wait.Routes)
	}
}

func TestParseV1Alpha5WaitWithoutOptionalResponse(t *testing.T) {
	source := `wait:
  event: {source: SMF, service: nsmf-pdusession, operation: UpdateSMContext}
  timeout: P1D
  routes: {TIMEOUT: FAILED}
postcondition: {}
`
	wait, diagnostics := parseV1Alpha5WaitTestNode(t, source)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if wait == nil || wait.Response != nil {
		t.Fatalf("optional response must remain absent: %#v", wait)
	}
	if wait.Timeout != "P1D" || wait.Routes.Timeout != "FAILED" {
		t.Fatalf("unexpected parsed Wait: %#v", wait)
	}
}

func TestValidV1Alpha5FixedDuration(t *testing.T) {
	valid := []string{
		"PT30S", "PT1M", "PT5M", "PT2H", "P1D", "P1DT2H30M",
		"P2DT3H4M5S", "PT2H3S", "PT2M3S", "P999999999999999999999D",
	}
	for _, value := range valid {
		if !validV1Alpha5FixedDuration(value) {
			t.Errorf("expected %q to be a valid v1alpha5 fixed duration", value)
		}
	}
}

func TestInvalidV1Alpha5FixedDuration(t *testing.T) {
	invalid := []string{
		"", "P", "PT", "P1DT", "P1M", "P1Y", "P1W",
		"PT0S", "P0D", "PT-1S", "PT+1S", "PT0.5S",
		"PT01M", "P01D", "PT1S2M", "P1D2H", "PT1MS",
		"PT 1S", "PT1S ",
	}
	for _, value := range invalid {
		if validV1Alpha5FixedDuration(value) {
			t.Errorf("expected %q to be rejected by the v1alpha5 duration grammar", value)
		}
	}
}

func TestV1Alpha5WaitNodeDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		node string
		code string
	}{
		{
			name: "missing event",
			node: `wait:
  timeout: PT30S
  routes: {TIMEOUT: FAILED}
operation: {}
`,
			code: "WAIT_DECLARATION_INVALID",
		},
		{
			name: "event selector has empty operation",
			node: `wait:
  event: {source: SMF, service: nsmf-pdusession, operation: ""}
  timeout: PT30S
  routes: {TIMEOUT: FAILED}
operation: {}
`,
			code: "WAIT_DECLARATION_INVALID",
		},
		{
			name: "event selector value is not a string",
			node: `wait:
  event: {source: 5, service: nsmf-pdusession, operation: UpdateSMContext}
  timeout: PT30S
  routes: {TIMEOUT: FAILED}
operation: {}
`,
			code: "WAIT_DECLARATION_INVALID",
		},
		{
			name: "timeout is missing",
			node: `wait:
  event: {source: SMF, service: nsmf-pdusession, operation: UpdateSMContext}
  routes: {TIMEOUT: FAILED}
operation: {}
`,
			code: "WAIT_TIMEOUT_INVALID",
		},
		{
			name: "timeout is not a YAML string",
			node: `wait:
  event: {source: SMF, service: nsmf-pdusession, operation: UpdateSMContext}
  timeout: 30
  routes: {TIMEOUT: FAILED}
operation: {}
`,
			code: "WAIT_TIMEOUT_INVALID",
		},
		{
			name: "calendar month is forbidden",
			node: `wait:
  event: {source: SMF, service: nsmf-pdusession, operation: UpdateSMContext}
  timeout: P1M
  routes: {TIMEOUT: FAILED}
operation: {}
`,
			code: "WAIT_TIMEOUT_INVALID",
		},
		{
			name: "missing routes",
			node: `wait:
  event: {source: SMF, service: nsmf-pdusession, operation: UpdateSMContext}
  timeout: PT30S
operation: {}
`,
			code: "WAIT_TIMEOUT_ROUTE_INVALID",
		},
		{
			name: "EVENT_RECEIVED route is forbidden",
			node: `wait:
  event: {source: SMF, service: nsmf-pdusession, operation: UpdateSMContext}
  timeout: PT30S
  routes: {TIMEOUT: FAILED, EVENT_RECEIVED: continue}
operation: {}
`,
			code: "WAIT_TIMEOUT_ROUTE_INVALID",
		},
		{
			name: "success terminal is forbidden",
			node: `wait:
  event: {source: SMF, service: nsmf-pdusession, operation: UpdateSMContext}
  timeout: PT30S
  routes: {TIMEOUT: SUCCEEDED}
operation: {}
`,
			code: "WAIT_TIMEOUT_ROUTE_INVALID",
		},
		{
			name: "wait-only node is forbidden",
			node: `wait:
  event: {source: SMF, service: nsmf-pdusession, operation: UpdateSMContext}
  timeout: PT30S
  routes: {TIMEOUT: FAILED}
`,
			code: "WAIT_NODE_STAGE_REQUIRED",
		},
		{
			name: "invalid response boundary",
			node: `wait:
  event: {source: SMF, service: nsmf-pdusession, operation: UpdateSMContext}
  timeout: PT30S
  response: {after: 42}
  routes: {TIMEOUT: FAILED}
operation: {}
`,
			code: "WAIT_RESPONSE_BOUNDARY_INVALID",
		},
		{
			name: "unknown Wait field uses normal unknown-field diagnostic",
			node: `wait:
  event: {source: SMF, service: nsmf-pdusession, operation: UpdateSMContext}
  timeout: PT30S
  routes: {TIMEOUT: FAILED}
  poll: true
operation: {}
`,
			code: "UNKNOWN_FIELD",
		},
		{
			name: "unknown response field uses normal unknown-field diagnostic",
			node: `wait:
  event: {source: SMF, service: nsmf-pdusession, operation: UpdateSMContext}
  timeout: PT30S
  response: {after: send-response, status: 200}
  routes: {TIMEOUT: FAILED}
operation: {}
`,
			code: "UNKNOWN_FIELD",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, diagnostics := parseV1Alpha5WaitTestNode(t, test.node)
			if !hasV1Alpha5Diagnostic(diagnostics, test.code) {
				t.Fatalf("expected %s, got %#v", test.code, diagnostics)
			}
		})
	}
}

func TestV1Alpha5WaitTimeoutTargetGrammar(t *testing.T) {
	for _, target := range []string{"FAILED", "UNRESOLVED", "SAFE_STOPPED", "timeout-handler", "Step_17"} {
		if !validV1Alpha5WaitTimeoutTarget(target) {
			t.Errorf("expected valid timeout target %q", target)
		}
	}
	for _, target := range []string{"", "SUCCEEDED", "context", "1bad", "bad target", "FAILED "} {
		if validV1Alpha5WaitTimeoutTarget(target) {
			t.Errorf("expected invalid timeout target %q", target)
		}
	}
}
