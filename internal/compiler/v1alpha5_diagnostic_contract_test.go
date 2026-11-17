package compiler

// WP7: normative v1alpha5 static diagnostic contract tests.
//
// The frozen compiler.Diagnostic envelope deliberately carries no messageKey or
// structured sourceLocation field. The Appendix L.11.1 message keys/templates
// are therefore kept in an explicit package-local registry here and asserted
// against the actually emitted messages; the public JSON envelope is separately
// asserted unchanged (TestV1Alpha5DiagnosticEnvelopeUnchanged). No public API
// field is added silently.

import (
	"encoding/json"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// v1alpha5DiagnosticContract is the package-local L.11.1 identity mapping.
type v1alpha5DiagnosticContract struct {
	MessageKey string
	Template   string
}

// v1Alpha5NormativeDiagnostics is the complete normative Appendix L.11.1
// message-key/template registry. Templates use the exact spec placeholders
// {nodeId}, {fieldPath} and {selector}; a renderer may localize display text but
// must preserve the stable message key and (Code, Phase) identity.
var v1Alpha5NormativeDiagnostics = map[string]v1alpha5DiagnosticContract{
	"WAIT_DECLARATION_INVALID": {
		MessageKey: "wait.declaration.invalid",
		Template:   "Node '{nodeId}' has an invalid wait declaration at '{fieldPath}'.",
	},
	"WAIT_TIMEOUT_INVALID": {
		MessageKey: "wait.timeout.invalid",
		Template:   "Node '{nodeId}' has an invalid wait.timeout; expected a positive fixed ISO-8601 duration in the v1alpha5 subset.",
	},
	"WAIT_INGRESS_EXPORT_NOT_FOUND": {
		MessageKey: "wait.ingress.export.not_found",
		Template:   "Node '{nodeId}' wait selector '{selector}' does not resolve to a trusted Ingress Operation export.",
	},
	"WAIT_INGRESS_EXPORT_AMBIGUOUS": {
		MessageKey: "wait.ingress.export.ambiguous",
		Template:   "Node '{nodeId}' wait selector '{selector}' resolves to more than one trusted Ingress Operation export.",
	},
	"WAIT_CORRELATION_BINDING_INVALID": {
		MessageKey: "wait.correlation.binding.invalid",
		Template:   "Node '{nodeId}' cannot derive one exact correlation binding on every reachable entry path.",
	},
	"WAIT_TIMEOUT_ROUTE_INVALID": {
		MessageKey: "wait.timeout.route.invalid",
		Template:   "Node '{nodeId}' wait.routes must contain exactly TIMEOUT with a valid non-success target.",
	},
	"WAIT_NODE_STAGE_REQUIRED": {
		MessageKey: "wait.node.stage_required",
		Template:   "Node '{nodeId}' declares wait but has no business stage.",
	},
	"WAIT_RESPONSE_BOUNDARY_INVALID": {
		MessageKey: "wait.response.boundary.invalid",
		Template:   "Node '{nodeId}' wait.response.after does not identify one reachable response boundary on every accepted-event path.",
	},
	"WAIT_RESPONSE_MAPPING_INVALID": {
		MessageKey: "wait.response.mapping.invalid",
		Template:   "Node '{nodeId}' response boundary has no valid typed Producer-to-response mapping.",
	},
}

// v1Alpha5IngressDiagnosticRegistry is the package-local projection of the
// normative Ingress diagnostic registry v1alpha1 (connectionagent.io/
// ingress-diagnostics/v1alpha1) for the Ingress-owned codes the integrated
// v1alpha5 path can emit. The values are the exact registry messageKey/template
// strings; the compiler.Diagnostic wire envelope still carries no messageKey.
var v1Alpha5IngressDiagnosticRegistry = map[string]v1alpha5DiagnosticContract{
	"INGRESS_CATALOG_INVALID": {
		MessageKey: "ingress.catalog.invalid",
		Template:   "IngressCatalog at '{catalogPath}' is structurally invalid or declares an unsupported apiVersion.",
	},
	"INGRESS_EXPORT_ID_DUPLICATE": {
		MessageKey: "ingress.export.id.duplicate",
		Template:   "Ingress Operation export id '{exportId}' is duplicated in this trusted Catalog.",
	},
	"INGRESS_EVENT_SELECTOR_DUPLICATE": {
		MessageKey: "ingress.event.selector.duplicate",
		Template:   "Ingress event selector '{source}/{service}/{operation}' is declared by more than one export.",
	},
	"INGRESS_PROTOCOL_ARTIFACT_INVALID": {
		MessageKey: "ingress.protocol.artifact.invalid",
		Template:   "Ingress export '{exportId}' does not match the pinned protocol artifact metadata at '{fieldPath}'.",
	},
	"INGRESS_CORRELATION_INVALID": {
		MessageKey: "ingress.correlation.invalid",
		Template:   "Ingress export '{exportId}' has an invalid correlation or stale-delivery declaration at '{fieldPath}'.",
	},
	"INGRESS_RESPONSE_CONTRACT_INVALID": {
		MessageKey: "ingress.response.contract.invalid",
		Template:   "Ingress export '{exportId}' has an invalid paired-response contract at '{fieldPath}'.",
	},
	"INGRESS_RESPONSE_MAPPING_INVALID": {
		MessageKey: "ingress.response.mapping.invalid",
		Template:   "Ingress export '{exportId}' has an invalid protocol response representation or mapping at '{fieldPath}'.",
	},
	"INGRESS_BINARY_CONTENT_LIFECYCLE_INVALID": {
		MessageKey: "ingress.binary.content.lifecycle.invalid",
		Template:   "Ingress export '{exportId}' has an invalid immutable binary-content reference or lifecycle contract at '{fieldPath}'.",
	},
}

// v1alpha5NormativeDiagnosticContract resolves one diagnostic identity to its
// normative messageKey/template, whether it is a CA-SL L.11 identity or an
// Ingress-owned registry identity.
func v1alpha5NormativeDiagnosticContract(code string) (v1alpha5DiagnosticContract, bool) {
	if contract, ok := v1Alpha5NormativeDiagnostics[code]; ok {
		return contract, true
	}
	contract, ok := v1Alpha5IngressDiagnosticRegistry[code]
	return contract, ok
}

// v1Alpha5WaitImplementedDiagnosticCodes is the exact L.11 subset the integrated
// v1alpha5 static path can emit from the Appendix L.12 EW cases.
var v1Alpha5WaitImplementedDiagnosticCodes = map[string]bool{
	"WAIT_DECLARATION_INVALID":         true,
	"WAIT_TIMEOUT_INVALID":             true,
	"WAIT_INGRESS_EXPORT_NOT_FOUND":    true,
	"WAIT_INGRESS_EXPORT_AMBIGUOUS":    true,
	"WAIT_CORRELATION_BINDING_INVALID": true,
	"WAIT_TIMEOUT_ROUTE_INVALID":       true,
	"WAIT_NODE_STAGE_REQUIRED":         true,
	"WAIT_RESPONSE_BOUNDARY_INVALID":   true,
	"WAIT_RESPONSE_MAPPING_INVALID":    true,
}

// v1Alpha5ImplementedDiagnosticCodes is the union of normative identities the
// integrated v1alpha5 static path is allowed to emit (L.11 + Ingress registry).
var v1Alpha5ImplementedDiagnosticCodes = func() map[string]bool {
	codes := map[string]bool{}
	for code := range v1Alpha5NormativeDiagnostics {
		codes[code] = true
	}
	for code := range v1Alpha5IngressDiagnosticRegistry {
		codes[code] = true
	}
	return codes
}()

const v1alpha5ContractNodeID = "wait-node"

// renderV1Alpha5NormativeDiagnostic renders the canonical template with the
// exact source identifiers the compiler must use.
func renderV1Alpha5NormativeDiagnostic(code, nodeID, fieldPath, selector string) string {
	contract, ok := v1Alpha5NormativeDiagnostics[code]
	if !ok {
		return ""
	}
	message := strings.ReplaceAll(contract.Template, "{nodeId}", nodeID)
	message = strings.ReplaceAll(message, "{fieldPath}", fieldPath)
	message = strings.ReplaceAll(message, "{selector}", selector)
	return message
}

// wp7EWContractCase binds an Appendix L.12 EW case to an executable negative
// input and its exact diagnostic identity.
type wp7EWContractCase struct {
	ID        string
	Procedure string
	Code      string
	FieldPath string
	Line      int
	Column    int
}

func v1Alpha5EWContractCases() []wp7EWContractCase {
	return []wp7EWContractCase{
		{
			ID: "EW-013",
			Procedure: `start: wait-node
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
			Code:   "WAIT_NODE_STAGE_REQUIRED",
			Line:   5,
			Column: 12,
		},
		{
			ID: "EW-014",
			Procedure: `start: wait-node
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
			Code:   "WAIT_TIMEOUT_ROUTE_INVALID",
			Line:   14,
			Column: 18,
		},
		{
			ID: "EW-027",
			Procedure: `start: wait-node
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
			Code:   "WAIT_TIMEOUT_INVALID",
			Line:   12,
			Column: 16,
		},
		{
			ID: "EW-033",
			Procedure: `start: wait-node
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
			Code:   "WAIT_TIMEOUT_ROUTE_INVALID",
			Line:   8,
			Column: 12,
		},
		{
			ID: "EW-034",
			Procedure: `start: wait-node
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
			Code:   "WAIT_TIMEOUT_ROUTE_INVALID",
			Line:   15,
			Column: 9,
		},
		{
			ID: "EW-035",
			Procedure: `start: wait-node
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
			Code:      "WAIT_DECLARATION_INVALID",
			FieldPath: "wait.event.source",
			Line:      9,
			Column:    16,
		},
		{
			ID: "EW-036",
			Procedure: `start: wait-node
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
			Code:   "WAIT_TIMEOUT_INVALID",
			Line:   8,
			Column: 12,
		},
	}
}

// TestV1Alpha5DiagnosticContractMessageKeyRegistry pins the complete L.11.1
// normative registry independently of the emitted messages.
func TestV1Alpha5DiagnosticContractMessageKeyRegistry(t *testing.T) {
	want := map[string]v1alpha5DiagnosticContract{
		"WAIT_DECLARATION_INVALID": {
			"wait.declaration.invalid",
			"Node '{nodeId}' has an invalid wait declaration at '{fieldPath}'.",
		},
		"WAIT_TIMEOUT_INVALID": {
			"wait.timeout.invalid",
			"Node '{nodeId}' has an invalid wait.timeout; expected a positive fixed ISO-8601 duration in the v1alpha5 subset.",
		},
		"WAIT_INGRESS_EXPORT_NOT_FOUND": {
			"wait.ingress.export.not_found",
			"Node '{nodeId}' wait selector '{selector}' does not resolve to a trusted Ingress Operation export.",
		},
		"WAIT_INGRESS_EXPORT_AMBIGUOUS": {
			"wait.ingress.export.ambiguous",
			"Node '{nodeId}' wait selector '{selector}' resolves to more than one trusted Ingress Operation export.",
		},
		"WAIT_CORRELATION_BINDING_INVALID": {
			"wait.correlation.binding.invalid",
			"Node '{nodeId}' cannot derive one exact correlation binding on every reachable entry path.",
		},
		"WAIT_TIMEOUT_ROUTE_INVALID": {
			"wait.timeout.route.invalid",
			"Node '{nodeId}' wait.routes must contain exactly TIMEOUT with a valid non-success target.",
		},
		"WAIT_NODE_STAGE_REQUIRED": {
			"wait.node.stage_required",
			"Node '{nodeId}' declares wait but has no business stage.",
		},
		"WAIT_RESPONSE_BOUNDARY_INVALID": {
			"wait.response.boundary.invalid",
			"Node '{nodeId}' wait.response.after does not identify one reachable response boundary on every accepted-event path.",
		},
		"WAIT_RESPONSE_MAPPING_INVALID": {
			"wait.response.mapping.invalid",
			"Node '{nodeId}' response boundary has no valid typed Producer-to-response mapping.",
		},
	}
	if !reflect.DeepEqual(v1Alpha5NormativeDiagnostics, want) {
		t.Fatalf("L.11.1 message-key registry drifted from the normative spec:\n got: %#v\nwant: %#v",
			v1Alpha5NormativeDiagnostics, want)
	}
	for code := range v1Alpha5WaitImplementedDiagnosticCodes {
		if _, ok := v1Alpha5NormativeDiagnostics[code]; !ok {
			t.Fatalf("implemented diagnostic %s has no L.11.1 registry entry", code)
		}
	}
	// A Host admission diagnostic is never a C_STATIC Compiler identity.
	if _, ok := v1Alpha5NormativeDiagnostics["WAIT_DURATION_LIMIT_EXCEEDED"]; ok {
		t.Fatal("WAIT_DURATION_LIMIT_EXCEEDED is an L.11.2 Host admission identity, not a Compiler C_STATIC code")
	}

	wantIngress := map[string]v1alpha5DiagnosticContract{
		"INGRESS_CATALOG_INVALID": {
			"ingress.catalog.invalid",
			"IngressCatalog at '{catalogPath}' is structurally invalid or declares an unsupported apiVersion.",
		},
		"INGRESS_EXPORT_ID_DUPLICATE": {
			"ingress.export.id.duplicate",
			"Ingress Operation export id '{exportId}' is duplicated in this trusted Catalog.",
		},
		"INGRESS_EVENT_SELECTOR_DUPLICATE": {
			"ingress.event.selector.duplicate",
			"Ingress event selector '{source}/{service}/{operation}' is declared by more than one export.",
		},
		"INGRESS_PROTOCOL_ARTIFACT_INVALID": {
			"ingress.protocol.artifact.invalid",
			"Ingress export '{exportId}' does not match the pinned protocol artifact metadata at '{fieldPath}'.",
		},
		"INGRESS_CORRELATION_INVALID": {
			"ingress.correlation.invalid",
			"Ingress export '{exportId}' has an invalid correlation or stale-delivery declaration at '{fieldPath}'.",
		},
		"INGRESS_RESPONSE_CONTRACT_INVALID": {
			"ingress.response.contract.invalid",
			"Ingress export '{exportId}' has an invalid paired-response contract at '{fieldPath}'.",
		},
		"INGRESS_RESPONSE_MAPPING_INVALID": {
			"ingress.response.mapping.invalid",
			"Ingress export '{exportId}' has an invalid protocol response representation or mapping at '{fieldPath}'.",
		},
		"INGRESS_BINARY_CONTENT_LIFECYCLE_INVALID": {
			"ingress.binary.content.lifecycle.invalid",
			"Ingress export '{exportId}' has an invalid immutable binary-content reference or lifecycle contract at '{fieldPath}'.",
		},
	}
	if !reflect.DeepEqual(v1Alpha5IngressDiagnosticRegistry, wantIngress) {
		t.Fatalf("Ingress registry message-key projection drifted from the normative registry:\n got: %#v\nwant: %#v",
			v1Alpha5IngressDiagnosticRegistry, wantIngress)
	}
	for code := range v1Alpha5ImplementedDiagnosticCodes {
		if _, ok := v1alpha5NormativeDiagnosticContract(code); !ok {
			t.Fatalf("implemented diagnostic %s has no normative messageKey/template", code)
		}
	}
}

// TestV1Alpha5DiagnosticContractEWAnchors is the executed contract for every
// COMPILER_STATIC_IMPLEMENTED_TESTED EW row: exact identity, phase, severity,
// path, line/column source location and canonical emitted message.
func TestV1Alpha5DiagnosticContractEWAnchors(t *testing.T) {
	for _, test := range v1Alpha5EWContractCases() {
		t.Run(test.ID, func(t *testing.T) {
			diagnostics := v1alpha5CheckProcedure(t, test.Procedure)

			matches := 0
			for _, diagnostic := range diagnostics {
				if diagnostic.Code != test.Code {
					continue
				}
				matches++
				if diagnostic.Phase != "C_STATIC" {
					t.Fatalf("%s phase = %q, want C_STATIC", test.Code, diagnostic.Phase)
				}
				if diagnostic.Severity != "error" {
					t.Fatalf("%s severity = %q, want error", test.Code, diagnostic.Severity)
				}
				if diagnostic.Path != v1alpha5TestProcedurePath {
					t.Fatalf("%s path = %q, want %q", test.Code, diagnostic.Path, v1alpha5TestProcedurePath)
				}
				if diagnostic.Line != test.Line || diagnostic.Column != test.Column {
					t.Fatalf("%s location = %d:%d, want %d:%d",
						test.Code, diagnostic.Line, diagnostic.Column, test.Line, test.Column)
				}
				contract, ok := v1Alpha5NormativeDiagnostics[test.Code]
				if !ok {
					t.Fatalf("%s is not an L.11.1 normative code", test.Code)
				}
				if contract.MessageKey == "" {
					t.Fatalf("%s has an empty normative message key", test.Code)
				}
				wantMessage := renderV1Alpha5NormativeDiagnostic(
					test.Code, v1alpha5ContractNodeID, test.FieldPath, "")
				if diagnostic.Message != wantMessage {
					t.Fatalf("%s message = %q, want canonical L.11.1 template %q",
						test.Code, diagnostic.Message, wantMessage)
				}
			}
			if matches != 1 {
				t.Fatalf("%s must be emitted exactly once, got %d occurrences in %#v",
					test.Code, matches, diagnostics)
			}
		})
	}
}

// TestV1Alpha5DiagnosticContractResponseBoundaryScope pins the implemented
// syntactic slice of WAIT_RESPONSE_BOUNDARY_INVALID. It is deliberately not
// EW-019/EW-030 coverage: path reachability and accepted-path bypass are WP6.
func TestV1Alpha5DiagnosticContractResponseBoundaryScope(t *testing.T) {
	invalidAfter := []struct {
		name      string
		procedure string
	}{
		{
			name: "terminal",
			procedure: `start: wait-node
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
		},
		{
			name: "context",
			procedure: `start: wait-node
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
		},
	}
	for _, test := range invalidAfter {
		t.Run(test.name, func(t *testing.T) {
			diagnostics := v1alpha5CheckProcedure(t, test.procedure)
			diagnostic := findV1Alpha5Diagnostic(diagnostics, "WAIT_RESPONSE_BOUNDARY_INVALID")
			if diagnostic == nil {
				t.Fatalf("expected WAIT_RESPONSE_BOUNDARY_INVALID, got %#v", diagnostics)
			}
			if diagnostic.Phase != "C_STATIC" || diagnostic.Severity != "error" {
				t.Fatalf("unexpected identity: %#v", diagnostic)
			}
			if diagnostic.Line != 14 || diagnostic.Column != 16 {
				t.Fatalf("location = %d:%d, want 14:16", diagnostic.Line, diagnostic.Column)
			}
			wantMessage := renderV1Alpha5NormativeDiagnostic(
				"WAIT_RESPONSE_BOUNDARY_INVALID", v1alpha5ContractNodeID, "", "")
			if diagnostic.Message != wantMessage {
				t.Fatalf("message = %q, want canonical L.11.1 template %q",
					diagnostic.Message, wantMessage)
			}
		})
	}

	valid := `start: transfer
nodes:
  transfer:
    operation:
      tool: ca::trigger
      arguments: {}
      routes:
        SUCCEEDED: wait-node
        FAILED: FAILED
  wait-node:
    wait:
      event:
        source: SMF
        service: nsmf-pdusession
        operation: UpdateSMContext
      timeout: PT30S
      response:
        after: response-node
      routes:
        TIMEOUT: FAILED
    operation:
      tool: ca::smf.x
      arguments: {}
      routes:
        SUCCEEDED: response-node
  response-node:
    operation:
      tool: ca::response
      arguments: {}
      routes:
        SUCCEEDED: FAILED
`
	if diagnostics := v1alpha5CheckProcedure(t, valid); len(diagnostics) != 0 {
		t.Fatalf("a syntactically valid response.after NodeId must be accepted, got %#v", diagnostics)
	}
}

// TestV1Alpha5DiagnosticContractDeterminismAndOrdering asserts byte-identical
// repeated diagnostics and the shared deterministic ordering contract.
func TestV1Alpha5DiagnosticContractDeterminismAndOrdering(t *testing.T) {
	procedure := `start: wait-node
nodes:
  wait-node:
    wait:
      response: yes
    poll: true
`
	first := v1alpha5CheckProcedure(t, procedure)
	second := v1alpha5CheckProcedure(t, procedure)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("diagnostics must be deterministic:\nfirst:  %#v\nsecond: %#v", first, second)
	}
	if len(first) < 3 {
		t.Fatalf("expected several independent diagnostics, got %#v", first)
	}

	wantOrder := make([]Diagnostic, len(first))
	copy(wantOrder, first)
	sort.SliceStable(wantOrder, func(i, j int) bool {
		a, b := wantOrder[i], wantOrder[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		if a.Phase != b.Phase {
			return a.Phase < b.Phase
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Message < b.Message
	})
	if !reflect.DeepEqual(first, wantOrder) {
		t.Fatalf("diagnostics are not in deterministic (path, line, column, phase, code, message) order:\n got: %#v\nwant: %#v",
			first, wantOrder)
	}
	codes := v1alpha5SortedCodes(first)
	for _, wantCode := range []string{
		"UNKNOWN_FIELD", "WAIT_DECLARATION_INVALID", "WAIT_NODE_STAGE_REQUIRED",
		"WAIT_TIMEOUT_INVALID", "WAIT_TIMEOUT_ROUTE_INVALID",
	} {
		if !containsV1Alpha5Code(codes, wantCode) {
			t.Fatalf("expected %s in deterministic diagnostic set %#v", wantCode, codes)
		}
	}
}

func containsV1Alpha5Code(codes []string, want string) bool {
	for _, code := range codes {
		if code == want {
			return true
		}
	}
	return false
}

// TestV1Alpha5DiagnosticContractSourceLocations asserts that every EW-mapped
// static diagnostic carries a positive line/column source location and the
// Procedure author path. No EW-mapped row may emit a location-free diagnostic.
func TestV1Alpha5DiagnosticContractSourceLocations(t *testing.T) {
	cases := v1Alpha5EWContractCases()
	cases = append(cases, wp7EWContractCase{
		ID: "WAIT_RESPONSE_BOUNDARY_INVALID-slice",
		Procedure: `start: wait-node
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
        after: FAILED
      routes:
        TIMEOUT: FAILED
`,
		Code:   "WAIT_RESPONSE_BOUNDARY_INVALID",
		Line:   14,
		Column: 16,
	})
	for _, test := range cases {
		diagnostics := v1alpha5CheckProcedure(t, test.Procedure)
		diagnostic := findV1Alpha5Diagnostic(diagnostics, test.Code)
		if diagnostic == nil {
			t.Fatalf("%s: expected %s, got %#v", test.ID, test.Code, diagnostics)
		}
		if diagnostic.Line <= 0 || diagnostic.Column <= 0 {
			t.Fatalf("%s: %s must carry a positive line/column source location, got %d:%d",
				test.ID, test.Code, diagnostic.Line, diagnostic.Column)
		}
		if diagnostic.Path != v1alpha5TestProcedurePath {
			t.Fatalf("%s: %s path = %q, want %q", test.ID, test.Code,
				diagnostic.Path, v1alpha5TestProcedurePath)
		}
	}
}

// TestV1Alpha5DiagnosticContractNoInventedWaitCodes asserts the static path
// emits only real identities: no invented WAIT_* code, no L.11.2 Host code, no
// Ingress-registry code, and inherited conditions keep their exact inherited
// identities instead of a Wait alias.
func TestV1Alpha5DiagnosticContractNoInventedWaitCodes(t *testing.T) {
	procedures := make([]string, 0, len(v1Alpha5EWContractCases())+3)
	for _, test := range v1Alpha5EWContractCases() {
		procedures = append(procedures, test.Procedure)
	}
	procedures = append(procedures,
		// Unknown Wait-bearing Node field: inherited UNKNOWN_FIELD identity.
		`start: wait-node
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
		// Duplicate YAML mapping key: inherited YAML_PARSE_ERROR identity.
		`start: wait-node
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
`,
		// Wait is not a mapping: WAIT_DECLARATION_INVALID, never a new code.
		`start: wait-node
nodes:
  wait-node:
    operation:
      tool: ca::smf.x
      arguments: {}
    wait: true
`,
	)

	allowed := map[string]bool{
		// Integrated L.11 identities.
		"WAIT_DECLARATION_INVALID":         true,
		"WAIT_TIMEOUT_INVALID":             true,
		"WAIT_INGRESS_EXPORT_NOT_FOUND":    true,
		"WAIT_INGRESS_EXPORT_AMBIGUOUS":    true,
		"WAIT_CORRELATION_BINDING_INVALID": true,
		"WAIT_TIMEOUT_ROUTE_INVALID":       true,
		"WAIT_NODE_STAGE_REQUIRED":         true,
		"WAIT_RESPONSE_BOUNDARY_INVALID":   true,
		"WAIT_RESPONSE_MAPPING_INVALID":    true,
		// Inherited real identities reused by the v1alpha5 source admission and
		// trusted-linking prerequisites.
		"UNKNOWN_FIELD":                      true,
		"YAML_PARSE_ERROR":                   true,
		"PROCEDURE_START_MISSING":            true,
		"NODE_ID_RESERVED":                   true,
		"SOURCE_ROOT_INVALID":                true,
		"DEPENDENCY_IDENTITY_UNVERIFIED":     true,
		"DEPENDENCY_EXPORT_OWNER_UNVERIFIED": true,
		"DEPENDENCY_DUPLICATE":               true,
		"DEPENDENCY_ID_MISMATCH":             true,
		"DEPENDENCY_SNAPSHOT_MISSING":        true,
		"DEPENDENCY_REQUIREMENT_INVALID":     true,
		"EXPORT_ID_MISSING":                  true,
		"EXPORT_ID_AMBIGUOUS":                true,
		"SCHEMA_MISSING":                     true,
		"SCHEMA_INVALID":                     true,
		"SOURCE_PATH_INVALID":                true,
		"SOURCE_NOT_AVAILABLE":               true,
		"INPUT_TYPE_MISMATCH":                true,
		"INPUT_NULLABILITY_MISMATCH":         true,
		"CONTROL_FLOW_DANGLING":              true,
		"CONTROL_FLOW_CYCLE":                 true,
		"REQUEST_INVALID":                    true,
		"PACKAGE_PATH_INVALID":               true,
		"INPUT_LIMIT":                        true,
	}
	for code := range v1Alpha5IngressDiagnosticRegistry {
		allowed[code] = true
	}

	for _, procedure := range procedures {
		for _, diagnostic := range v1alpha5CheckProcedure(t, procedure) {
			if strings.HasPrefix(diagnostic.Code, "WAIT_") {
				if _, ok := v1Alpha5NormativeDiagnostics[diagnostic.Code]; !ok {
					t.Fatalf("invented WAIT_* code emitted: %#v", diagnostic)
				}
				if !v1Alpha5WaitImplementedDiagnosticCodes[diagnostic.Code] {
					t.Fatalf("WAIT_* code outside the implemented L.11 slice emitted: %#v", diagnostic)
				}
			}
			if strings.HasPrefix(diagnostic.Code, "INGRESS_") {
				if _, ok := v1Alpha5IngressDiagnosticRegistry[diagnostic.Code]; !ok {
					t.Fatalf("invented INGRESS_* code emitted: %#v", diagnostic)
				}
			}
			if diagnostic.Code == "WAIT_DURATION_LIMIT_EXCEEDED" {
				t.Fatal("L.11.2 Host admission code must never be emitted by the Compiler static path")
			}
			if !allowed[diagnostic.Code] {
				t.Fatalf("diagnostic code %q is not a real L.11/Ingress or inherited identity: %#v",
					diagnostic.Code, diagnostic)
			}
		}
	}

	// Inherited identities must remain their real identities, not Wait aliases.
	parseErrorDiagnostics := v1alpha5CheckProcedure(t, procedures[8])
	parseError := findV1Alpha5Diagnostic(parseErrorDiagnostics, "YAML_PARSE_ERROR")
	if parseError == nil {
		t.Fatalf("duplicate YAML key must keep YAML_PARSE_ERROR, got %#v", parseErrorDiagnostics)
	}
	if parseError.Phase != "C_STATIC" || parseError.Severity != "error" {
		t.Fatalf("YAML_PARSE_ERROR identity is wrong: %#v", parseError)
	}
	unknownFieldDiagnostics := v1alpha5CheckProcedure(t, procedures[7])
	unknownField := findV1Alpha5Diagnostic(unknownFieldDiagnostics, "UNKNOWN_FIELD")
	if unknownField == nil {
		t.Fatalf("unknown Node field must keep UNKNOWN_FIELD, got %#v", unknownFieldDiagnostics)
	}
	if unknownField.Line != 15 || unknownField.Column != 5 {
		t.Fatalf("UNKNOWN_FIELD location = %d:%d, want 15:5",
			unknownField.Line, unknownField.Column)
	}
}

// TestV1Alpha5DiagnosticEnvelopeUnchanged freezes the public JSON envelope:
// WP7 adds no messageKey/sourceLocation/registryId field silently.
func TestV1Alpha5DiagnosticEnvelopeUnchanged(t *testing.T) {
	full := Diagnostic{
		Severity: "error",
		Phase:    "C_STATIC",
		Code:     "WAIT_TIMEOUT_INVALID",
		Path:     v1alpha5TestProcedurePath,
		Line:     12,
		Column:   16,
		Message:  "message",
	}
	encoded, err := json.Marshal(full)
	if err != nil {
		t.Fatalf("marshal diagnostic: %v", err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &keys); err != nil {
		t.Fatalf("unmarshal diagnostic: %v", err)
	}
	wantKeys := []string{"severity", "phase", "code", "path", "line", "column", "message"}
	if len(keys) != len(wantKeys) {
		t.Fatalf("diagnostic JSON envelope changed: got keys %v, want %v", wp7EnvelopeKeys(keys), wantKeys)
	}
	for _, key := range wantKeys {
		if _, ok := keys[key]; !ok {
			t.Fatalf("diagnostic JSON envelope lost %q: %s", key, encoded)
		}
	}
	for _, forbidden := range []string{"messageKey", "sourceLocation", "registryId"} {
		if _, ok := keys[forbidden]; ok {
			t.Fatalf("diagnostic JSON envelope silently gained %q: %s", forbidden, encoded)
		}
	}

	// line/column stay omitempty for location-free envelope diagnostics.
	minimal, err := json.Marshal(Diagnostic{Severity: "error", Phase: "C_STATIC", Code: "REQUEST_INVALID", Message: "m"})
	if err != nil {
		t.Fatalf("marshal minimal diagnostic: %v", err)
	}
	var minimalKeys map[string]json.RawMessage
	if err := json.Unmarshal(minimal, &minimalKeys); err != nil {
		t.Fatalf("unmarshal minimal diagnostic: %v", err)
	}
	if _, ok := minimalKeys["line"]; ok {
		t.Fatalf("line must remain omitempty: %s", minimal)
	}
	if _, ok := minimalKeys["column"]; ok {
		t.Fatalf("column must remain omitempty: %s", minimal)
	}
}

func wp7EnvelopeKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// v1alpha5IntegratedContractFixtures renders one synthetic trusted-linking
// fixture per integrated diagnostic identity. Every fixture is SYNTHETIC and is
// not a real Owner artifact, deployment profile or 5GC/2GPP publication claim.
func v1alpha5IntegratedContractFixtures(t *testing.T) map[string][]Diagnostic {
	t.Helper()
	_, digests := wp3Artifacts(t)

	// Zero trusted selector match: rename the declared event operation.
	notFound := wp3Check(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			return strings.Replace(wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest),
				"operation: UpdateSMContext", "operation: UpdateSmContext", 1)
		},
	})

	// Two trusted Catalogs declare the same exact event selector.
	secondCatalog := wp3DefaultCatalogText(
		wp3Digest([]byte(wp3OpenAPIDocument)), digests[wp3RequestArtifactRef], digests[wp3NormalizedArtifactRef])
	secondCatalog = strings.Replace(secondCatalog,
		"id: nsmf-pdusession.update-sm-context", "id: nsmf-pdusession.update-sm-context.second", 1)
	ambiguous := Check(wp3BuildRequest(t, wp3FixtureOptions{
		extraFiles:     []File{{Path: "ingress/catalog-second.yaml", Bytes: []byte(secondCatalog)}},
		extraOwners:    []ExportOwner{{ExportKind: v1alpha5ExportKindIngressOperation, ExportID: "nsmf-pdusession.update-sm-context.second", Owner: wp3IntegrationOwner}},
		extraOwnersFor: wp3IntegrationDependency,
	}))

	// One incoming path bypasses the triggering Operation.
	correlation := wp3Check(t, wp3FixtureOptions{procedure: wp3BypassTriggerProcedure})

	// Export-side stale-delivery retention window is not strictly greater than
	// maximumLateArrival + clockSkewAllowance.
	ingressCorrelation := wp3Check(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			return strings.Replace(text, "tombstoneRetention: PT60S", "tombstoneRetention: PT35S", 1)
		},
	})

	// Catalog protocol metadata contradicts the pinned OpenAPI artifact.
	protocolArtifact := wp3Check(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			return strings.Replace(wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest),
				"operationId: UpdateSmContext", "operationId: NotThePinnedOperation", 1)
		},
	})

	// WP6 Skill-side mapping and boundary defects through the shared fixtures.
	responseMapping := wp6Check(t, wp6FixtureOptions{procedure: wp6BoundaryWithoutProducerProcedure})
	responseBoundary := wp6Check(t, wp6FixtureOptions{procedure: wp6SuccessBeforeBoundaryProcedure})

	// Ingress-owned response contract/mapping defects.
	responseContract := wp6Check(t, wp6FixtureOptions{response: wp6JSONBodyResponseBlockWithoutEarlyFailures})
	responseMappingIngress := wp6Check(t, wp6FixtureOptions{response: wp6JSONBodyResponseBlockMissingMapping})
	// A declared binary part larger than the bounded binary-content lifecycle is a
	// semantic lifecycle defect that still passes the structural Catalog schema.
	responseBinary := wp6Check(t, wp6FixtureOptions{response: func(d wp6Digests) string {
		block := wp6MultipartResponseBlock(d)
		return strings.Replace(block, "                    maxBytes: 4096", "                    maxBytes: 16384", 1)
	}})

	return map[string][]Diagnostic{
		"WAIT_INGRESS_EXPORT_NOT_FOUND":            notFound,
		"WAIT_INGRESS_EXPORT_AMBIGUOUS":            ambiguous,
		"WAIT_CORRELATION_BINDING_INVALID":         correlation,
		"WAIT_RESPONSE_MAPPING_INVALID":            responseMapping,
		"WAIT_RESPONSE_BOUNDARY_INVALID":           responseBoundary,
		"INGRESS_CORRELATION_INVALID":              ingressCorrelation,
		"INGRESS_PROTOCOL_ARTIFACT_INVALID":        protocolArtifact,
		"INGRESS_RESPONSE_CONTRACT_INVALID":        responseContract,
		"INGRESS_RESPONSE_MAPPING_INVALID":         responseMappingIngress,
		"INGRESS_BINARY_CONTENT_LIFECYCLE_INVALID": responseBinary,
	}
}

func v1alpha5FindEmittedDiagnostic(diagnostics []Diagnostic, code string) *Diagnostic {
	for index := range diagnostics {
		if diagnostics[index].Code == code {
			return &diagnostics[index]
		}
	}
	return nil
}

// v1alpha5TemplateLiteralParts splits a canonical template into its literal
// segments around the "{placeholder}" tokens.
func v1alpha5TemplateLiteralParts(template string) []string {
	parts := regexp.MustCompile(`\{[a-zA-Z]+\}`).Split(template, -1)
	literals := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			literals = append(literals, part)
		}
	}
	return literals
}

// TestV1Alpha5DiagnosticContractIntegratedIdentitySet asserts that every identity
// emitted by the integrated v1alpha5 static path is a real normative identity
// (L.11 or Ingress registry or inherited), that the WAIT_* canonical messages are
// exactly rendered from the L.11.1 registry, and that Ingress-owned messages
// follow the normative Ingress registry template shape.
func TestV1Alpha5DiagnosticContractIntegratedIdentitySet(t *testing.T) {
	fixtures := v1alpha5IntegratedContractFixtures(t)
	for expectedCode, diagnostics := range fixtures {
		diagnostic := v1alpha5FindEmittedDiagnostic(diagnostics, expectedCode)
		if diagnostic == nil {
			t.Fatalf("fixture for %s emitted %#v instead", expectedCode, diagnostics)
		}
		if diagnostic.Phase != "C_STATIC" || diagnostic.Severity != "error" {
			t.Fatalf("%s identity = %s/%s, want C_STATIC/error",
				expectedCode, diagnostic.Phase, diagnostic.Severity)
		}
	}

	// No emitted identity may sit outside the known normative/inherited sets.
	inherited := map[string]bool{
		"UNKNOWN_FIELD": true, "YAML_PARSE_ERROR": true, "SOURCE_ROOT_INVALID": true,
		"SCHEMA_MISSING": true, "SCHEMA_INVALID": true,
		"DEPENDENCY_IDENTITY_UNVERIFIED": true, "DEPENDENCY_EXPORT_OWNER_UNVERIFIED": true,
		"DEPENDENCY_OWNER_MISMATCH": true, "INPUT_TYPE_MISMATCH": true,
		"INPUT_NULLABILITY_MISMATCH": true, "SOURCE_NOT_AVAILABLE": true,
		"SOURCE_PATH_INVALID": true, "EXPORT_ID_MISSING": true, "EXPORT_ID_AMBIGUOUS": true,
		"CONTROL_FLOW_DANGLING": true, "CONTROL_FLOW_CYCLE": true, "REQUEST_INVALID": true,
	}
	for fixture, diagnostics := range fixtures {
		for _, diagnostic := range diagnostics {
			if !v1Alpha5ImplementedDiagnosticCodes[diagnostic.Code] && !inherited[diagnostic.Code] {
				t.Fatalf("%s fixture emitted an unknown identity: %#v", fixture, diagnostic)
			}
			contract, registered := v1alpha5NormativeDiagnosticContract(diagnostic.Code)
			if !registered {
				continue
			}
			literals := v1alpha5TemplateLiteralParts(contract.Template)
			if len(literals) == 0 {
				t.Fatalf("%s has an empty normative template", diagnostic.Code)
			}
			if !strings.HasPrefix(diagnostic.Message, literals[0]) ||
				!strings.HasSuffix(diagnostic.Message, literals[len(literals)-1]) {
				t.Fatalf("%s message %q does not follow the normative template %q",
					diagnostic.Code, diagnostic.Message, contract.Template)
			}
		}
	}

	// Exact L.11.1 messages for the newly integrated Wait identities.
	exact := []struct {
		code      string
		nodeID    string
		selector  string
		lineCheck bool
	}{
		{"WAIT_INGRESS_EXPORT_NOT_FOUND", "wait-sm-context", "AMF/Nsmf_PDUSession/UpdateSMContext", true},
		{"WAIT_INGRESS_EXPORT_AMBIGUOUS", "wait-sm-context", "AMF/Nsmf_PDUSession/UpdateSMContext", true},
		{"WAIT_CORRELATION_BINDING_INVALID", "wait-sm-context", "", true},
		{"WAIT_RESPONSE_MAPPING_INVALID", "update-sm-context", "", true},
	}
	for _, want := range exact {
		diagnostics := fixtures[want.code]
		diagnostic := v1alpha5FindEmittedDiagnostic(diagnostics, want.code)
		if diagnostic == nil {
			t.Fatalf("%s fixture emitted %#v", want.code, diagnostics)
		}
		wantMessage := renderV1Alpha5NormativeDiagnostic(want.code, want.nodeID, "", want.selector)
		if diagnostic.Message != wantMessage {
			t.Fatalf("%s message = %q, want exact L.11.1 template %q",
				want.code, diagnostic.Message, wantMessage)
		}
		if want.lineCheck && (diagnostic.Path == "" || diagnostic.Line <= 0) {
			t.Fatalf("%s must keep a source path and positive line: %#v", want.code, diagnostic)
		}
	}

	// Ingress-owned diagnostics keep the Compiler Diagnostic.path source location
	// (trusted dependency snapshot path), not an invented line/column.
	for _, code := range []string{
		"INGRESS_CORRELATION_INVALID",
		"INGRESS_PROTOCOL_ARTIFACT_INVALID",
		"INGRESS_RESPONSE_CONTRACT_INVALID",
		"INGRESS_RESPONSE_MAPPING_INVALID",
		"INGRESS_BINARY_CONTENT_LIFECYCLE_INVALID",
	} {
		diagnostic := v1alpha5FindEmittedDiagnostic(fixtures[code], code)
		if diagnostic == nil {
			t.Fatalf("%s fixture emitted %#v", code, fixtures[code])
		}
		if diagnostic.Path == "" {
			t.Fatalf("%s must carry the trusted snapshot path: %#v", code, diagnostic)
		}
	}
}
