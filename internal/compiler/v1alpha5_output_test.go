package compiler

// WP4: v1alpha5 wait.output typed Producer SourcePath tests (Appendix L.2.3,
// L.11/L.11.2, EW-012).
//
// All fixtures are SYNTHETIC static-compiler fixtures. They are not real Owner
// artifacts, deployment profiles or 5GC/3GPP publication claims. The Ingress and
// ToolCatalog Schema Closure digests are derived from the exact fixture bytes by
// the same WP1 Schema Closure Digest v1 implementation the production path uses.
// The trusted linking fixtures are reused from v1alpha5_binding_test.go (same
// package) and extended additively with a consumer Operation/Observation Input
// Schema so that statically known argument typing can be exercised.

import (
	"reflect"
	"strings"
	"testing"
)

const (
	wp4UpdateInputRef  = "urn:example:schema:wp4-update-input"
	wp4UpdateInputPath = "presentation/schemas/wp4-update-input.json"

	wp4ObserveInputRef  = "urn:example:schema:wp4-observe-input"
	wp4ObserveInputPath = "presentation/schemas/wp4-observe-input.json"

	// wp4UpdateInputSchema declares the argument the Wait-bearing Node's own
	// operation consumes. `request` is object-typed and `smContextRef` is a string,
	// which lets the tests prove both a positive and a mismatching assignment.
	wp4UpdateInputSchema = `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"https://schemas.invalid/example/wp4-update-input","type":"object","properties":{"request":{"type":"object","properties":{"smContextRef":{"type":"string"},"updateData":{"type":"object"}},"required":["smContextRef"]},"smContextRef":{"type":"string"}}}`

	// wp4ObserveInputSchema is the QueryContext Schema of the downstream
	// Observation that consumes `wait.output.smContextRef`.
	wp4ObserveInputSchema = `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"https://schemas.invalid/example/wp4-observe-input","type":"object","properties":{"request":{"type":"object"},"smContextRef":{"type":"string"}}}`

	// wp4ToolCatalog extends the synthetic WP3 ToolCatalog with the consumer
	// Operation and Observation callables. The triggering Operation is unchanged,
	// so the WP3 trigger identity stays exactly resolvable.
	// wp4DownstreamProcedure consumes wait.output from a downstream Node that is
	// dominated by the Wait-bearing Node's event-accepted business vertex.
	wp4DownstreamProcedure = `start: trigger-transfer
nodes:
  trigger-transfer:
    operation:
      tool: tools::amf.communication.n1n2_message_transfer
      arguments:
        smContextRef: "sm-context-placeholder"
      routes:
        SUCCEEDED: wait-sm-context
        FAILED: SAFE_STOPPED
        UNKNOWN: SAFE_STOPPED
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
      tool: tools::smf.session.update_from_amf
      arguments: {}
      routes:
        SUCCEEDED: consume-wait-output
        FAILED: SAFE_STOPPED
        UNKNOWN: SAFE_STOPPED
  consume-wait-output:
    observation:
      tool: tools::smf.session.observe_step17_response
      arguments:
        smContextRef: wait-sm-context.wait.output.smContextRef
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        UNKNOWN: SAFE_STOPPED
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
      tool: tools::smf.update_timeout.handle
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
        FAILED: UNRESOLVED
        UNKNOWN: SAFE_STOPPED
`

	// wp4BypassProcedure has a second path into a consumer Node that does not pass
	// the Wait-bearing Node's event acceptance, so the Wait output is not available
	// to that consumer even though the Wait Node is otherwise valid.
	wp4BypassProcedure = `start: choose
nodes:
  choose:
    precondition:
      predicates:
        - predicate: session.trigger_required
          arguments: []
      routes:
        PASS: trigger-transfer
        FAIL: bypass-consumer
        UNKNOWN: UNRESOLVED
  trigger-transfer:
    operation:
      tool: tools::amf.communication.n1n2_message_transfer
      arguments:
        smContextRef: "sm-context-placeholder"
      routes:
        SUCCEEDED: wait-sm-context
        FAILED: SAFE_STOPPED
        UNKNOWN: SAFE_STOPPED
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
      tool: tools::smf.session.update_from_amf
      arguments:
        request: wait-sm-context.wait.output
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        UNKNOWN: SAFE_STOPPED
  bypass-consumer:
    operation:
      tool: tools::smf.session.update_from_amf
      arguments:
        request: wait-sm-context.wait.output
      routes:
        SUCCEEDED: SAFE_STOPPED
        FAILED: SAFE_STOPPED
        UNKNOWN: SAFE_STOPPED
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
      tool: tools::smf.update_timeout.handle
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
        FAILED: UNRESOLVED
        UNKNOWN: SAFE_STOPPED
`
)

// wp4ToolCatalog is the base synthetic WP3 ToolCatalog with the consumer
// Operation and Observation callables added and the consumer Operation's input
// Schema replaced by the WP4 typing fixture. The triggering Operation and the
// timeout handler stay exactly resolvable.
var wp4ToolCatalog = strings.Replace(wp3ToolCatalog,
	`    - kind: operation
      id: smf.session.update_from_amf
      effect: READ
      inputSchema:
        $ref: urn:example:schema:smf-update-input
      outputSchema:
        $ref: urn:example:schema:smf-update-output
      ownerOutcomes:
        - SUCCEEDED
        - FAILED
`,
	`    - kind: operation
      id: smf.session.update_from_amf
      effect: READ
      inputSchema:
        $ref: urn:example:schema:wp4-update-input
      outputSchema:
        $ref: urn:example:schema:amf-n1n2-transfer-output
      ownerOutcomes:
        - SUCCEEDED
        - FAILED
    - kind: observation
      id: smf.session.observe_step17_response
      effect: READ
      queryContextSchema:
        $ref: urn:example:schema:wp4-observe-input
      outputSchema:
        $ref: urn:example:schema:amf-n1n2-transfer-output
      ownerOutcomes:
        - SUCCEEDED
        - FAILED
`, 1)

// wp4BuildRequest builds the WP3 trusted synthetic request plus the consumer
// Input/QueryContext Schemas needed to prove wait.output typing. The link
// artifacts themselves are unchanged synthetic fixtures.
func wp4BuildRequest(t *testing.T, options wp3FixtureOptions) BuildRequest {
	t.Helper()
	if options.toolsCatalog == "" {
		options.toolsCatalog = wp4ToolCatalog
	}
	options.extraToolsFiles = append([]File{
		{Path: wp4UpdateInputPath, Bytes: []byte(wp4UpdateInputSchema)},
		{Path: wp4ObserveInputPath, Bytes: []byte(wp4ObserveInputSchema)},
	}, options.extraToolsFiles...)
	original := options.mutateContext
	options.mutateContext = func(context *TrustedBuildContext) {
		context.ArtifactBindings = append(context.ArtifactBindings,
			TrustedArtifactBinding{ArtifactRef: wp4UpdateInputRef, DependencyID: wp3ToolsDependency, SnapshotPath: wp4UpdateInputPath, Dialect: "json-schema-2020-12"},
			TrustedArtifactBinding{ArtifactRef: wp4ObserveInputRef, DependencyID: wp3ToolsDependency, SnapshotPath: wp4ObserveInputPath, Dialect: "json-schema-2020-12"},
		)
		context.SchemaBindings = append(context.SchemaBindings,
			TrustedSchemaBinding{ExportID: wp4UpdateInputRef, ArtifactRef: wp4UpdateInputRef, Pointer: "#"},
			TrustedSchemaBinding{ExportID: wp4ObserveInputRef, ArtifactRef: wp4ObserveInputRef, Pointer: "#"},
		)
		if original != nil {
			original(context)
		}
	}
	return wp3BuildRequest(t, options)
}

func wp4Check(t *testing.T, options wp3FixtureOptions) []Diagnostic {
	t.Helper()
	return Check(wp4BuildRequest(t, options))
}

// wp4WithProcedure replaces the default Procedure in the fixture options.
func wp4WithProcedure(procedure string) wp3FixtureOptions {
	return wp3FixtureOptions{procedure: procedure}
}

// TestV1Alpha5WP4PositiveSameNodeRead is the L.2.3 positive path: a business stage
// of the Wait-bearing Node itself reads the typed wait.output value and the linked
// normalized request Schema is assignable to the Owner Operation Input Schema.
func TestV1Alpha5WP4PositiveSameNodeRead(t *testing.T) {
	diagnostics := wp4Check(t, wp3FixtureOptions{})
	if len(diagnostics) != 0 {
		t.Fatalf("same-Node wait.output read must validate, got %#v", diagnostics)
	}
}

// TestV1Alpha5WP4PositiveDownstreamDominatedRead is the L.2.3 downstream positive
// path: a consumer Node dominated by the Wait Node's event-accepted business
// vertex reads `wait.output.<property>` and the JSON Pointer property path resolves
// against the exact linked normalized request Schema.
func TestV1Alpha5WP4PositiveDownstreamDominatedRead(t *testing.T) {
	diagnostics := wp4Check(t, wp4WithProcedure(wp4DownstreamProcedure))
	if len(diagnostics) != 0 {
		t.Fatalf("downstream dominated wait.output read must validate, got %#v", diagnostics)
	}
}

// TestV1Alpha5WP4TimeoutHandlerReadIsUnavailable is EW-012: the timeout handler
// must not read the timed-out Wait's output (L.2.3).
func TestV1Alpha5WP4TimeoutHandlerReadIsUnavailable(t *testing.T) {
	procedure := strings.Replace(wp3PositiveProcedure,
		`  handle-update-timeout:
    operation:
      tool: tools::smf.update_timeout.handle
      arguments: {}`,
		`  handle-update-timeout:
    operation:
      tool: tools::smf.update_timeout.handle
      arguments:
        request: wait-sm-context.wait.output`, 1)
	diagnostics := wp4Check(t, wp4WithProcedure(procedure))
	assertWP3Diagnostics(t, diagnostics, "SOURCE_NOT_AVAILABLE")
	diagnostic := diagnostics[0]
	if diagnostic.Path != wp3ProcedurePath {
		t.Fatalf("SOURCE_NOT_AVAILABLE path = %q, want %q", diagnostic.Path, wp3ProcedurePath)
	}
	if diagnostic.Line == 0 || diagnostic.Column == 0 {
		t.Fatalf("SOURCE_NOT_AVAILABLE must be anchored at the consuming argument: %#v", diagnostic)
	}
	if !strings.Contains(diagnostic.Message, "wait-sm-context") || !strings.Contains(diagnostic.Message, "handle-update-timeout") {
		t.Fatalf("SOURCE_NOT_AVAILABLE must render both Nodes: %q", diagnostic.Message)
	}
}

// TestV1Alpha5WP4BypassPathIsUnavailable proves the branch-sensitivity rule: a
// consumer reachable through a path that does not accept the event cannot read the
// Wait output even though another path does.
func TestV1Alpha5WP4BypassPathIsUnavailable(t *testing.T) {
	diagnostics := wp4Check(t, wp4WithProcedure(wp4BypassProcedure))
	assertWP3Diagnostics(t, diagnostics, "SOURCE_NOT_AVAILABLE")
	if !strings.Contains(diagnostics[0].Message, "bypass-consumer") {
		t.Fatalf("bypass SOURCE_NOT_AVAILABLE must name the consumer Node: %q", diagnostics[0].Message)
	}
}

// TestV1Alpha5WP4UnknownProperty covers L.11.2: an undeclared Wait output Producer
// property is SOURCE_PATH_INVALID (not a Wait-specific code).
func TestV1Alpha5WP4UnknownProperty(t *testing.T) {
	procedure := strings.Replace(wp3PositiveProcedure,
		"request: wait-sm-context.wait.output", "request: wait-sm-context.wait.output.unknownField", 1)
	diagnostics := wp4Check(t, wp4WithProcedure(procedure))
	assertWP3Diagnostics(t, diagnostics, "SOURCE_PATH_INVALID")
	if !strings.Contains(diagnostics[0].Message, "unknownField") {
		t.Fatalf("SOURCE_PATH_INVALID must name the undeclared property: %q", diagnostics[0].Message)
	}
}

// TestV1Alpha5WP4NonWaitProducerRoot covers the exact producer-form rule: only a
// declared Wait-bearing Node may be the wait.output root, and the
// `<nodeId>.wait.output` form itself is required.
func TestV1Alpha5WP4NonWaitProducerRoot(t *testing.T) {
	for name, path := range map[string]string{
		"non-wait-node":       "final.wait.output",
		"undeclared-node":     "ghost-node.wait.output",
		"malformed-wait-form": "wait-sm-context.wait",
	} {
		procedure := strings.Replace(wp3PositiveProcedure, "request: wait-sm-context.wait.output",
			"request: "+path, 1)
		diagnostics := wp4Check(t, wp4WithProcedure(procedure))
		if len(diagnostics) == 0 {
			t.Fatalf("%s: expected SOURCE_PATH_INVALID, got none", name)
		}
		for _, diagnostic := range diagnostics {
			if diagnostic.Code != "SOURCE_PATH_INVALID" {
				t.Fatalf("%s: diagnostics = %#v, want only SOURCE_PATH_INVALID", name, diagnostics)
			}
		}
	}
}

// TestV1Alpha5WP4TypeMismatch covers the statically known consumer type rule:
// connecting an incompatible wait.output value to a known Owner argument Schema
// uses the existing INPUT_TYPE_MISMATCH identity, never a Wait-specific code.
func TestV1Alpha5WP4TypeMismatch(t *testing.T) {
	procedure := strings.Replace(wp3PositiveProcedure,
		"request: wait-sm-context.wait.output", "request: wait-sm-context.wait.output.smContextRef", 1)
	diagnostics := wp4Check(t, wp4WithProcedure(procedure))
	assertWP3Diagnostics(t, diagnostics, "INPUT_TYPE_MISMATCH")
	if !strings.Contains(diagnostics[0].Message, "request") {
		t.Fatalf("INPUT_TYPE_MISMATCH must name the Owner argument: %q", diagnostics[0].Message)
	}
}

// TestV1Alpha5WP4ObservationTypeMismatch covers the Observation QueryContext
// Schema side of the statically known consumer typing.
func TestV1Alpha5WP4ObservationTypeMismatch(t *testing.T) {
	procedure := strings.Replace(wp4DownstreamProcedure,
		"smContextRef: wait-sm-context.wait.output.smContextRef",
		"smContextRef: wait-sm-context.wait.output.updateData", 1)
	diagnostics := wp4Check(t, wp4WithProcedure(procedure))
	assertWP3Diagnostics(t, diagnostics, "INPUT_TYPE_MISMATCH")
}

// TestV1Alpha5WP4InvalidSegment covers the inherited SourcePath segment grammar
// for the wait.output property path.
func TestV1Alpha5WP4InvalidSegment(t *testing.T) {
	procedure := strings.Replace(wp3PositiveProcedure,
		"request: wait-sm-context.wait.output", "request: wait-sm-context.wait.output.0bad", 1)
	diagnostics := wp4Check(t, wp4WithProcedure(procedure))
	assertWP3Diagnostics(t, diagnostics, "SOURCE_PATH_INVALID")
}

// TestV1Alpha5WP4SameNodePreconditionRead is the L.2.4 author pattern: the
// Wait-bearing Node's own Precondition reads the typed wait.output value after
// event acceptance.
func TestV1Alpha5WP4SameNodePreconditionRead(t *testing.T) {
	procedure := strings.Replace(wp3PositiveProcedure,
		`      routes:
        TIMEOUT: handle-update-timeout
    operation:`,
		`      routes:
        TIMEOUT: handle-update-timeout
    precondition:
      predicates:
        - predicate: session.update_request.valid
          arguments: [wait-sm-context.wait.output]
      routes:
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
    operation:`, 1)
	diagnostics := wp4Check(t, wp4WithProcedure(procedure))
	if len(diagnostics) != 0 {
		t.Fatalf("same-Node Precondition wait.output read must validate, got %#v", diagnostics)
	}
}

// TestV1Alpha5WP4PredicateArgumentPositions proves positional PredicateCall
// arguments are SourcePath positions: the same path is accepted when available
// and rejected (unknown property) when it does not resolve in the linked Schema.
func TestV1Alpha5WP4PredicateArgumentPositions(t *testing.T) {
	replacePredicateArgument := func(argument string) string {
		return strings.Replace(wp3PositiveProcedure,
			`  final:
    postcondition:
      predicates:
        - predicate: session.goal_satisfied
          arguments: []`,
			`  final:
    postcondition:
      predicates:
        - predicate: session.wait_output_ready
          arguments: [`+argument+`]`, 1)
	}
	if diagnostics := wp4Check(t, wp4WithProcedure(replacePredicateArgument("wait-sm-context.wait.output"))); len(diagnostics) != 0 {
		t.Fatalf("available Predicate wait.output read rejected: %#v", diagnostics)
	}
	diagnostics := wp4Check(t, wp4WithProcedure(replacePredicateArgument("wait-sm-context.wait.output.unknownField")))
	assertWP3Diagnostics(t, diagnostics, "SOURCE_PATH_INVALID")
}

// TestV1Alpha5WP4Determinism proves the WP4 diagnostics are deterministic.
func TestV1Alpha5WP4Determinism(t *testing.T) {
	request := wp4BuildRequest(t, wp4WithProcedure(wp4BypassProcedure))
	first := Check(request)
	second := Check(request)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("WP4 diagnostics are not deterministic:\n first=%#v\nsecond=%#v", first, second)
	}
	if len(first) == 0 {
		t.Fatal("determinism fixture must produce a diagnostic")
	}
}
