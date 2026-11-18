package compiler

// Shared spec-conformant Mutation proof Schemas for the positive-path fixtures.

import (
	"strings"
	"testing"
)

const specCommitProofSchemaYAML = `$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:ca:commit-proof:v1
type: object
additionalProperties: false
required: [originalInvocationId, commitState]
properties:
  originalInvocationId: {type: string, minLength: 1}
  commitState: {type: string, enum: [COMMITTED, NOT_COMMITTED]}
`

const specReceiptSchemaYAML = `$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:ca:receipt:v1
type: object
additionalProperties: false
required: [commit]
properties:
  commit: {$ref: urn:example:ca:commit-proof:v1}
  failureClass: {type: string}
`

const specReconcileInputSchemaYAML = `$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:ca:reconcile-input:v1
type: object
additionalProperties: false
required: [originalInvocationId]
properties:
  originalInvocationId: {type: string, minLength: 1}
`

// addSpecMutationProofSchemas adds the spec-conformant proof, receipt and
// Reconcile input Schemas to the trusted ca snapshot.
func addSpecMutationProofSchemas(fixture *wp3aFixture) {
	fixture.addSnapshotFile("ca", "commit-proof.schema.yaml", []byte(specCommitProofSchemaYAML))
	fixture.addSnapshotFile("ca", "receipt.schema.yaml", []byte(specReceiptSchemaYAML))
	fixture.addSnapshotFile("ca", "reconcile-input.schema.yaml", []byte(specReconcileInputSchemaYAML))
}

// specMutationContractBlock is the spec-conformant Mutation contract fragment.
const specMutationContractBlock = `      receiptSchema: {$ref: urn:example:ca:receipt:v1}
      mutation:
        commitProof: {schema: urn:example:ca:commit-proof:v1, path: receipt.commit, bind: originalInvocationId}
`

// specMutationReconcileBlock extends the contract with a spec-conformant
// Reconcile READ declaration.
const specMutationReconcileBlock = `      receiptSchema: {$ref: urn:example:ca:receipt:v1}
      mutation:
        commitProof: {schema: urn:example:ca:commit-proof:v1, path: receipt.commit, bind: originalInvocationId}
        failureClassPath: receipt.failureClass
        failureClasses:
          - id: transient-rejection
            outcome: NOT_COMMITTED
            proofSchema: urn:example:ca:commit-proof:v1
            proofPath: receipt.commit
            noSideEffect: true
            retryable: true
        reconcile:
          callable: example.session.reconcile
          bind: originalInvocationId
          proofSchema: urn:example:ca:commit-proof:v1
          proofPath: receipt.commit
`

// specMutationReconcileCatalogYAML is a spec-conformant trusted ToolCatalog with
// one MUTATION callable and its dedicated same-Owner Reconcile READ.
const specMutationReconcileCatalogYAML = `apiVersion: connectionagent.io/toolcatalog/v1alpha4
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
` + specMutationReconcileBlock + `    - kind: operation
      id: example.session.reconcile
      effect: READ
      ownerOutcomes: [SUCCEEDED, FAILED]
      inputSchema: {$ref: urn:example:ca:reconcile-input:v1}
      outputSchema: {$ref: urn:example:ca:registration:v1}
      receiptSchema: {$ref: urn:example:ca:receipt:v1}
`

const specEvidenceFreePredicates = `predicates:
  example.session.complete:
    parameters: []
    evaluator:
      type: cel
      expression: "true"
`

const specMutationRetryProcedure = `start: mutate
nodes:
  mutate:
    operation:
      tool: ca::example.session.mutate
      arguments:
        ueRef: context.ueRef
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        NOT_COMMITTED: retry
        OUTCOME_UNKNOWN: SAFE_STOPPED
  retry:
    operation:
      tool: ca::example.session.mutate
      arguments:
        ueRef: context.ueRef
      retryOf: mutate
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        NOT_COMMITTED: SAFE_STOPPED
        OUTCOME_UNKNOWN: SAFE_STOPPED
  final:
    postcondition:
      predicates:
        - predicate: example.session.complete
          arguments: []
      routes:
        PASS: SUCCEEDED
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
`

// specMutationCapabilityBundle adds the MUTATION and Reconcile READ members to
// the granted capability of the base capability fixture.
func specMutationCapabilityBundle(t *testing.T) map[string][]byte {
	t.Helper()
	bundle := wp3cBaseBundle(t)
	content := string(bundle["capability-contracts.yaml"])
	content = strings.Replace(content, "          scopePolicyRef: deployment.registration-read", `          scopePolicyRef: deployment.registration-read
        - owner: example-ue-owner
          id: example.session.mutate
          kind: operation
          effect: MUTATION
          scopePolicyRef: deployment.registration-read
        - owner: example-ue-owner
          id: example.session.reconcile
          kind: operation
          effect: READ
          scopePolicyRef: deployment.registration-read`, 1)
	bundle["capability-contracts.yaml"] = []byte(content)
	return bundle
}

// TestV1Alpha4MutationReconcileRetryBuildsArtifact proves the positive path: a
// fully spec-conformant Mutation + Reconcile + retry Skill passes Check and
// produces a deterministic Artifact now that the Mutation/Retry feature gates
// are removed.
func TestV1Alpha4MutationReconcileRetryBuildsArtifact(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	addSpecMutationProofSchemas(&fixture)
	fixture.addSnapshotFile("ca", "mutation-catalog.yaml", []byte(specMutationReconcileCatalogYAML))
	fixture.files[wp3aPredicatesPath] = []byte(specEvidenceFreePredicates)
	fixture.files[wp3aProcedurePath] = []byte(specMutationRetryProcedure)
	fixture.refreshSnapshotDigests(t)
	req := wp3cRequest(t, fixture, specMutationCapabilityBundle(t))
	for _, diagnostic := range Check(req) {
		if diagnostic.Severity == "error" {
			t.Fatalf("spec-conformant Mutation/Reconcile/retry Skill rejected: %+v", diagnostic)
		}
	}
	result := Build(req)
	if !result.Success {
		t.Fatalf("Mutation/Reconcile/retry Build failed: %+v", result.Diagnostics)
	}
	if len(result.OutputFiles) == 0 {
		t.Fatal("successful Build emitted no artifact files")
	}
	second := Build(req)
	if !second.Success || len(second.OutputFiles) != len(result.OutputFiles) {
		t.Fatalf("repeated Build diverged: %+v", second)
	}
}

// TestV1Alpha4RetryRequiresRetryableFailureClass proves the retry assurance
// static rule: a retryOf edge needs the original Mutation to publish a
// retryable, no-side-effect NOT_COMMITTED failure class.
func TestV1Alpha4RetryRequiresRetryableFailureClass(t *testing.T) {
	catalog := strings.Replace(specMutationReconcileCatalogYAML, "            retryable: true", "            retryable: false", 1)
	fixture := wp3aBaseFixture(t)
	addSpecMutationProofSchemas(&fixture)
	fixture.addSnapshotFile("ca", "mutation-catalog.yaml", []byte(catalog))
	fixture.files[wp3aPredicatesPath] = []byte(specEvidenceFreePredicates)
	fixture.files[wp3aProcedurePath] = []byte(specMutationRetryProcedure)
	fixture.refreshSnapshotDigests(t)
	req := wp3cRequest(t, fixture, specMutationCapabilityBundle(t))
	if !v4HasCode(Check(req), "RETRY_CONTRACT_INVALID") {
		t.Fatalf("retry without a retryable NOT_COMMITTED failure class accepted: %+v", Check(req))
	}
}

// TestV1Alpha4ReconcileRequiresGrantedCapability proves the dedicated Reconcile
// READ must be covered by a granted Capability independent of the Mutation grant.
func TestV1Alpha4ReconcileRequiresGrantedCapability(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	addSpecMutationProofSchemas(&fixture)
	fixture.addSnapshotFile("ca", "mutation-catalog.yaml", []byte(specMutationReconcileCatalogYAML))
	fixture.files[wp3aPredicatesPath] = []byte(specEvidenceFreePredicates)
	fixture.files[wp3aProcedurePath] = []byte(specMutationRetryProcedure)
	fixture.refreshSnapshotDigests(t)
	bundle := specMutationCapabilityBundle(t)
	content := strings.Replace(string(bundle["capability-contracts.yaml"]), "        - owner: example-ue-owner\n          id: example.session.reconcile\n          kind: operation\n          effect: READ\n          scopePolicyRef: deployment.registration-read\n", "", 1)
	bundle["capability-contracts.yaml"] = []byte(content)
	req := wp3cRequest(t, fixture, bundle)
	if !v4HasCode(Check(req), "RECONCILE_CONTRACT_INVALID") {
		t.Fatalf("Reconcile READ without its own granted Capability accepted: %+v", Check(req))
	}
}
