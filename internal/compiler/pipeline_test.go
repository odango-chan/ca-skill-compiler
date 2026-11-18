package compiler

// Shared fixtures for the inherited static semantic tests.
//
// These fixtures describe ordinary (non-Wait) CA-SL v1alpha5 author source. They
// exercise the inherited package, Contract, PredicateSet, Schema, Procedure and
// assurance obligations through the single public Check/Build path.

import (
	"strings"
	"testing"
)

const testContract = `goal: pdu_session.initial_request_fulfilled
precondition: check.precondition
requestSchema: schemas/request.schema.yaml
contextSchema: ca-context::urn:example:host:v1
dependencies:
  ca-context: urn:example:host
  ca: urn:example:tools
capabilities: [fivegc-session-read]
`
const testProcedure = `start: check
nodes:
  check:
    precondition:
      predicates:
        - predicate: test.ready
          arguments: [context.ueRef]
      routes:
        PASS: read
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
  read:
    observation:
      tool: ca::example.session.read
      arguments:
        ueRef: context.ueRef
      routes:
        SUCCEEDED: final
        FAILED: FAILED
  final:
    postcondition:
      predicates:
        - predicate: test.complete
          arguments: [read.observation.output]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`
const testPredicates = `predicates:
  test.ready:
    parameters: [{ue: string}]
    evaluator: {type: cel, expression: 'ue != ""'}
  test.complete:
    parameters: [{result: ca::urn:example:result:v1}]
    evaluator: {type: cel, expression: "result.ready == true"}
`
const testSchema = `$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:request:v1
type: object
additionalProperties: false
required: [dnn]
properties:
  dnn: {type: string, minLength: 1}
`

func testRequest(contract, procedure, predicates, schema string) BuildRequest {
	return BuildRequest{
		APIVersion:           BuildRequestAPIVersion,
		LanguageTarget:       V1Alpha5Target,
		ResolvedDependencies: []ResolvedDependency{},
		Files: []File{
			{Path: "skills/pdu/SKILL.md", Bytes: []byte(`---
name: pdu
description: PDU fixture
---
`)},
			{Path: "skills/pdu/assurance/contract.yaml", Bytes: []byte(contract)},
			{Path: "skills/pdu/assurance/procedure.yaml", Bytes: []byte(procedure)},
			{Path: "skills/pdu/assurance/predicates.yaml", Bytes: []byte(predicates)},
			{Path: "skills/pdu/schemas/request.schema.yaml", Bytes: []byte(schema)},
		},
	}
}

func hasCode(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func TestInheritedPipelineStillFailsClosedWithoutTrustedOwner(t *testing.T) {
	req := testRequest(testContract, testProcedure, testPredicates, testSchema)
	result := Build(req)
	if result.Success || len(result.OutputFiles) != 0 {
		t.Fatalf("a Skill cannot claim product success without trusted Owner: %+v", result)
	}
	if !hasCode(result.Diagnostics, "DEPENDENCY_SNAPSHOT_MISSING") {
		t.Fatalf("missing DEPENDENCY_SNAPSHOT_MISSING: %+v", result.Diagnostics)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Phase != "C_STATIC" {
			t.Fatalf("diagnostic must expose phase: %+v", diagnostic)
		}
	}
}

func TestInheritedPipelineRejectsLegacyAuthorFieldsAndUnknownTarget(t *testing.T) {
	req := testRequest(strings.Replace(testContract, "precondition: check.precondition", "preconditions: [check.precondition]", 1),
		strings.Replace(testProcedure, "tool: ca::example.session.read", "call: ca::example.session.read", 1),
		testPredicates, testSchema)
	out := Check(req)
	if !hasCode(out, "UNKNOWN_FIELD") {
		t.Fatalf("legacy fields should not be converted: %+v", out)
	}
	req.LanguageTarget = "connectionagent.io/v1alpha3"
	out = Check(req)
	if !hasCode(out, "LANGUAGE_TARGET_UNSUPPORTED") {
		t.Fatalf("unknown target accepted: %+v", out)
	}
}

func TestInheritedPipelineRejectsBypassAndCycle(t *testing.T) {
	// A reachable non-final SUCCEEDED request with no reachable legal final
	// PASS edge is FINAL_POSTCONDITION_MISSING.
	procedure := strings.Replace(testProcedure, "        SUCCEEDED: final", "        SUCCEEDED: SUCCEEDED", 1)
	procedure = strings.Replace(procedure, "        PASS: SUCCEEDED", "        PASS: FAILED", 1)
	req := testRequest(testContract, procedure, testPredicates, testSchema)
	if !hasCode(Check(req), "FINAL_POSTCONDITION_MISSING") {
		t.Fatal("failed to reject non-final success edge")
	}
	// An unreachable illegal SUCCEEDED edge is only UNREACHABLE_NODE; it neither
	// creates a reachable success request nor proves the Goal (spec H.0 row 4).
	unreachable := testRequest(testContract, strings.Replace(testProcedure, "PASS: SUCCEEDED", "PASS: FAILED", 1)+
		`  unsafe:
    operation:
      tool: ca::example.mutate
      arguments: {}
      routes:
        SUCCEEDED: SUCCEEDED
        OUTCOME_UNKNOWN: SAFE_STOPPED
`, testPredicates, testSchema)
	diagnostics := Check(unreachable)
	if hasCode(diagnostics, "FINAL_POSTCONDITION_MISSING") || hasCode(diagnostics, "SUCCESS_BYPASSES_FINAL_POSTCONDITION") {
		t.Fatalf("unreachable success edge treated as a reachable success request: %+v", diagnostics)
	}
	if !hasCode(diagnostics, "UNREACHABLE_NODE") {
		t.Fatalf("unreachable Node not reported: %+v", diagnostics)
	}
	req = testRequest(testContract, strings.Replace(testProcedure, "SUCCEEDED: final", "SUCCEEDED: check", 1), testPredicates, testSchema)
	if !hasCode(Check(req), "CONTROL_FLOW_CYCLE") {
		t.Fatal("failed to reject cycle")
	}
}

func TestInheritedPipelineRejectsDuplicateKeyAndForbiddenSchemaKeyword(t *testing.T) {
	req := testRequest(testContract, strings.Replace(testProcedure, "start: check", "start: check\nstart: read", 1),
		testPredicates, testSchema)
	if !hasCode(Check(req), "YAML_PARSE_ERROR") {
		t.Fatal("duplicate YAML key escaped guard")
	}
	req = testRequest(testContract, testProcedure, testPredicates,
		strings.Replace(testSchema, "minLength: 1", "pattern: '^[a-z]+$'", 1))
	if !hasCode(Check(req), "SCHEMA_INVALID") {
		t.Fatal("G.4 unsupported pattern not rejected")
	}
}
