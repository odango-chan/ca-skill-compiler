package compiler

// §17.1 Host-recovery dual-Producer Evidence static check tests.

import (
	"strings"
	"testing"
)

const recoveryCommitProofSchema = `$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:ca:commit-proof:v1
type: object
additionalProperties: false
required: [originalInvocationId, commitState]
properties:
  originalInvocationId: {type: string, minLength: 1}
  commitState: {type: string, enum: [COMMITTED, NOT_COMMITTED]}
`

const recoveryReceiptSchema = `$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:ca:receipt:v1
type: object
additionalProperties: false
required: [commit]
properties:
  commit: {$ref: urn:example:ca:commit-proof:v1}
  failureClass: {type: string}
`

const recoveryReconcileInputSchema = `$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:ca:reconcile-input:v1
type: object
additionalProperties: false
required: [originalInvocationId]
properties:
  originalInvocationId: {type: string, minLength: 1}
`

const recoveryReconcileOutputSchema = `$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:ca:reconcile-output:v1
type: object
additionalProperties: false
properties:
  recovered: {$ref: urn:example:ca:registration:v1}
`

const recoveryMutationCatalog = `apiVersion: connectionagent.io/toolcatalog/v1alpha4
kind: ToolCatalog
metadata:
  owner: example-ue-owner
spec:
  contracts:
    - kind: operation
      id: example.session.mutate
      effect: MUTATION
      ownerOutcomes: [SUCCEEDED, FAILED, NOT_COMMITTED]
      inputSchema: {$ref: urn:example:ca:registration-query:v1}
      outputSchema: {$ref: urn:example:ca:registration:v1}
      receiptSchema: {$ref: urn:example:ca:receipt:v1}
      guarantees:
        evidence:
          provides:
            - guarantee: example.session-state
              path: output
      mutation:
        commitProof: {schema: urn:example:ca:commit-proof:v1, path: receipt.commit, bind: originalInvocationId}
        reconcile:
          callable: example.session.reconcile
          bind: originalInvocationId
          proofSchema: urn:example:ca:commit-proof:v1
          proofPath: receipt.commit
          recoverableOutputSchema: urn:example:ca:registration:v1
          recoverableOutputPath: output.recovered
    - kind: operation
      id: example.session.reconcile
      effect: READ
      ownerOutcomes: [SUCCEEDED]
      inputSchema: {$ref: urn:example:ca:reconcile-input:v1}
      outputSchema: {$ref: urn:example:ca:reconcile-output:v1}
      receiptSchema: {$ref: urn:example:ca:receipt:v1}
      guarantees:
        evidence:
          provides:
            - guarantee: example.session-state
              path: output
`

func recoveryDescriptor(includeReconcileProducer bool) string {
	block := `example.session-state:
  authority: example-ue-owner
  fact: example.session-state
  producers:
    - {owner: example-ue-owner, kind: operation, id: example.session.mutate, outcome: SUCCEEDED, path: output}
`
	if includeReconcileProducer {
		block += "    - {owner: example-ue-owner, kind: operation, id: example.session.reconcile, outcome: SUCCEEDED, path: output}\n"
	}
	return block + `  scope:
    subject: {record: scope.subject, equals: arguments.ueRef}
  correlation:
    invocation: {record: invocationId, equals: execution.invocationId}
  factValue: state
  selection: {mode: latest, orderBy: observedAt}
  freshness: {timestamp: observedAt, maxAgeMs: 10000}
`
}

const recoveryPredicates = `predicates:
  example.session.complete:
    parameters:
      - registration: ca::urn:example:ca:registration:v1
    evidence:
      requires:
        registration: [ca::example.session-state]
    evaluator:
      type: cel
      expression: "true"
`

const recoveryProcedure = `start: mutate
nodes:
  mutate:
    operation:
      tool: ca::example.session.mutate
      arguments:
        ueRef: context.ueRef
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        NOT_COMMITTED: SAFE_STOPPED
        OUTCOME_UNKNOWN: SAFE_STOPPED
  final:
    postcondition:
      predicates:
        - predicate: example.session.complete
          arguments: [mutate.operation.output]
      routes:
        PASS: SUCCEEDED
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
`

func recoveryFixture(t *testing.T, descriptor string) BuildRequest {
	t.Helper()
	fixture := wp3aBaseFixture(t)
	fixture.addSnapshotFile("ca", "mutation-catalog.yaml", []byte(recoveryMutationCatalog))
	fixture.addSnapshotFile("ca", "recovery-commit-proof.schema.yaml", []byte(recoveryCommitProofSchema))
	fixture.addSnapshotFile("ca", "recovery-receipt.schema.yaml", []byte(recoveryReceiptSchema))
	fixture.addSnapshotFile("ca", "recovery-reconcile-input.schema.yaml", []byte(recoveryReconcileInputSchema))
	fixture.addSnapshotFile("ca", "recovery-reconcile-output.schema.yaml", []byte(recoveryReconcileOutputSchema))
	fixture.addSnapshotFile("ca", "recovery-evidence.yaml", []byte(descriptor))
	fixture.files[wp3aPredicatesPath] = []byte(recoveryPredicates)
	fixture.files[wp3aProcedurePath] = []byte(recoveryProcedure)
	fixture.refreshSnapshotDigests(t)
	return wp3cRequest(t, fixture, specMutationCapabilityBundle(t))
}

// TestHostRecoveryDualProducerPositive proves a descriptor declaring both the
// normal Mutation Producer and the Reconcile READ Producer links successfully.
func TestHostRecoveryDualProducerPositive(t *testing.T) {
	diagnostics := Check(recoveryFixture(t, recoveryDescriptor(true)))
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "EVIDENCE_GUARANTEE_LINK_INVALID" {
			t.Fatalf("dual-Producer descriptor rejected: %+v", diagnostic)
		}
	}
}

// TestHostRecoveryDualProducerMissingReconcileProducer proves §17.1: consuming a
// recoverable Mutation output requires the guarantee to declare the Reconcile
// READ Producer as well.
func TestHostRecoveryDualProducerMissingReconcileProducer(t *testing.T) {
	diagnostics := Check(recoveryFixture(t, recoveryDescriptor(false)))
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "EVIDENCE_GUARANTEE_LINK_INVALID" && strings.Contains(diagnostic.Message, "Reconcile READ Producer") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing Reconcile READ Producer accepted: %+v", diagnostics)
	}
}
