package compiler

import (
	"strings"
	"testing"
)

const v4TestContract = `goal: pdu_session.initial_request_fulfilled
precondition: check.precondition
requestSchema: schemas/request.schema.yaml
contextSchema: ca-context::urn:example:host:v1
dependencies:
  ca-context: urn:example:host
  ca: urn:example:tools
capabilities: [fivegc-session-read]
`
const v4TestProcedure = `start: check
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
const v4TestPredicates = `predicates:
  test.ready:
    parameters: [{ue: string}]
    evaluator: {type: cel, expression: 'ue != ""'}
  test.complete:
    parameters: [{result: ca::urn:example:result:v1}]
    evaluator: {type: cel, expression: "result.ready == true"}
`
const v4TestSchema = `$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:request:v1alpha4
type: object
additionalProperties: false
required: [dnn]
properties:
  dnn: {type: string, minLength: 1}
`

func v4TestRequest(contract,procedure,predicates,schema string) BuildRequest {
	return BuildRequest{
		LanguageTarget:V1Alpha4Target,
		Files:[]File{
			{Path:"skills/pdu/SKILL.md",Bytes:[]byte("---
name: pdu
description: PDU v4
---
")},
			{Path:"skills/pdu/assurance/contract.yaml",Bytes:[]byte(contract)},
			{Path:"skills/pdu/assurance/procedure.yaml",Bytes:[]byte(procedure)},
			{Path:"skills/pdu/assurance/predicates.yaml",Bytes:[]byte(predicates)},
			{Path:"skills/pdu/schemas/request.schema.yaml",Bytes:[]byte(schema)},
		},
	}
}
func v4HasCode(diagnostics []Diagnostic,code string)bool {
	for _,d:=range diagnostics {
		if d.Code==code{return true}
	}
	return false
}

func TestV1Alpha4ExplicitTargetSourceShapeStillFailsClosedWithoutOwner(t *testing.T){
	req:=v4TestRequest(v4TestContract,v4TestProcedure,v4TestPredicates,v4TestSchema)
	result:=Build(req)
	if result.Success||len(result.OutputFiles)!=0 {t.Fatalf("v1alpha4 cannot claim product success without trusted Owner: %+v",result)}
	for _,code:=range []string{"DEPENDENCY_IDENTITY_UNVERIFIED","CAPABILITY_INPUT_UNTRUSTED"} {
		if !v4HasCode(result.Diagnostics,code){t.Fatalf("missing %s: %+v",code,result.Diagnostics)}
	}
	for _,d:=range result.Diagnostics {
		if d.Phase!="C_STATIC" {t.Fatalf("v1alpha4 diagnostic must expose phase: %+v",d)}
	}
}
func TestV1Alpha4RejectsLegacyAuthorFieldsAndUnknownTarget(t *testing.T){
	req:=v4TestRequest(strings.Replace(v4TestContract,"precondition: check.precondition","preconditions: [check.precondition]",1),
		strings.Replace(v4TestProcedure,"tool: ca::example.session.read","call: ca::example.session.read",1),
		v4TestPredicates,v4TestSchema)
	out:=Check(req)
	if !v4HasCode(out,"UNKNOWN_FIELD") {t.Fatalf("legacy fields should not be converted: %+v",out)}
	req.LanguageTarget="connectionagent.io/v1alpha3"
	out=Check(req)
	if !v4HasCode(out,"LANGUAGE_TARGET_UNSUPPORTED"){t.Fatalf("unknown target accepted: %+v",out)}
}
func TestV1Alpha4RejectsBypassAndCycle(t *testing.T){
	req:=v4TestRequest(v4TestContract,strings.Replace(v4TestProcedure,"PASS: SUCCEEDED","PASS: FAILED",1)+
		"  unsafe:
    operation:
      tool: ca::example.mutate
      arguments: {}
      routes:
        SUCCEEDED: SUCCEEDED
        OUTCOME_UNKNOWN: SAFE_STOPPED
",v4TestPredicates,v4TestSchema)
	if !v4HasCode(Check(req),"FINAL_POSTCONDITION_MISSING"){t.Fatal("failed to reject non-final success edge")}
	req=v4TestRequest(v4TestContract,strings.Replace(v4TestProcedure,"SUCCEEDED: final","SUCCEEDED: check",1),v4TestPredicates,v4TestSchema)
	if !v4HasCode(Check(req),"CONTROL_FLOW_CYCLE"){t.Fatal("failed to reject cycle")}
}
func TestV1Alpha4DuplicateKeyAndForbiddenSchemaKeyword(t *testing.T){
	req:=v4TestRequest(v4TestContract,strings.Replace(v4TestProcedure,"start: check","start: check
start: read",1),
		v4TestPredicates,v4TestSchema)
	if !v4HasCode(Check(req),"YAML_PARSE_ERROR"){t.Fatal("duplicate YAML key escaped guard")}
	req=v4TestRequest(v4TestContract,v4TestProcedure,v4TestPredicates,
		strings.Replace(v4TestSchema,"minLength: 1","pattern: '^[a-z]+$'",1))
	if !v4HasCode(Check(req),"SCHEMA_INVALID"){t.Fatal("G.4 unsupported pattern not rejected")}
}
