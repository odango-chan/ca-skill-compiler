package compiler

// WP8 slice 4 conformance: trusted Mutation/Reconcile Schema semantics.

import (
	"testing"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

const reconcileCatalog = `apiVersion: connectionagent.io/toolcatalog/v1alpha4
kind: ToolCatalog
metadata:
  owner: example-ue-owner
spec:
  contracts:
    - kind: operation
      id: example.session.mutate
      effect: MUTATION
      ownerOutcomes: [SUCCEEDED, NOT_COMMITTED]
      inputSchema: {$ref: urn:example:ca:registration-query:v1}
      outputSchema: {$ref: urn:example:ca:registration:v1}
      receiptSchema: {$ref: urn:example:ca:receipt:v1}
      mutation:
        commitProof: {schema: urn:example:ca:proof:v1, path: receipt.commit, bind: originalInvocationId}
        reconcile:
          callable: example.session.reconcile
          bind: originalInvocationId
          proofSchema: urn:example:ca:proof:v1
          proofPath: receipt.commit
          recoverableOutputSchema: urn:example:ca:recoverable:v1
          recoverableOutputPath: output
    - kind: operation
      id: example.session.reconcile
      effect: READ
      ownerOutcomes: [SUCCEEDED]
      inputSchema: {$ref: urn:example:ca:registration-query:v1}
      outputSchema: {$ref: urn:example:ca:registration:v1}
      receiptSchema: {$ref: urn:example:ca:receipt:v1}
`

const reconcileProofSchema = `$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:ca:proof:v1
type: object
properties:
  receipt:
    type: object
    properties:
      commit: {type: string}
`

const reconcileReceiptSchema = `$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:ca:receipt:v1
type: object
properties:
  commit: {type: string}
`

const reconcileRecoverableSchema = `$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:ca:recoverable:v1
type: object
properties:
  output: {type: object}
`

func reconcileFixture(t *testing.T, catalog string) (BuildRequest, *V1Alpha4SkillAST, dependencyIdentityResult, ownerLinkingResult) {
	t.Helper()
	fixture := wp3aBaseFixture(t)
	fixture.addSnapshotFile("ca", "mutation-catalog.yaml", []byte(catalog))
	fixture.addSnapshotFile("ca", "proof.schema.yaml", []byte(reconcileProofSchema))
	fixture.addSnapshotFile("ca", "receipt.schema.yaml", []byte(reconcileReceiptSchema))
	fixture.addSnapshotFile("ca", "recoverable.schema.yaml", []byte(reconcileRecoverableSchema))
	fixture.refreshSnapshotDigests(t)
	req := fixture.request(t)
	ast := wp3aParseSkillAST(t, req, wp3aSkillName)
	identity := validateV1Alpha4DependencyIdentity(req, ast)
	linking := validateV1Alpha4OwnerLinking(req, ast, identity)
	return req, ast, identity, linking
}

func reconcileDiagnostics(t *testing.T, req BuildRequest, ast *V1Alpha4SkillAST, identity dependencyIdentityResult, linking ownerLinkingResult) []Diagnostic {
	t.Helper()
	diagnostics := make([]Diagnostic, 0)
	add := func(path string, node *yaml.Node, code, message string) {
		diagnostics = append(diagnostics, dependencyDiagnostic("error", "C_STATIC", code, path, node, message))
	}
	validateV1Alpha4ReconcileContracts(req, ast, identity, linking, wp3aProcedurePath, add)
	return diagnostics
}

func TestReconcileContractPositive(t *testing.T) {
	req, ast, identity, linking := reconcileFixture(t, reconcileCatalog)
	if diagnostics := reconcileDiagnostics(t, req, ast, identity, linking); len(diagnostics) != 0 {
		t.Fatalf("valid Mutation/Reconcile contract rejected: %+v", diagnostics)
	}
}

func TestReconcileCommitProofPathUnresolved(t *testing.T) {
	catalog := replaceAll(reconcileCatalog, "path: receipt.commit", "path: receipt.missing")
	req, ast, identity, linking := reconcileFixture(t, catalog)
	if len(reconcileDiagnostics(t, req, ast, identity, linking)) == 0 {
		t.Fatalf("unresolved commitProof path accepted")
	}
}

func TestReconcileCallableMustBeReadOperation(t *testing.T) {
	catalog := replaceAll(reconcileCatalog, "id: example.session.reconcile\n      effect: READ", "id: example.session.reconcile\n      effect: MUTATION")
	req, ast, identity, linking := reconcileFixture(t, catalog)
	diagnostics := reconcileDiagnostics(t, req, ast, identity, linking)
	if len(diagnostics) == 0 {
		t.Fatalf("MUTATION Reconcile callable accepted")
	}
}

func TestReconcileMissingTargetRejected(t *testing.T) {
	catalog := replaceAll(reconcileCatalog, "callable: example.session.reconcile", "callable: example.session.absent")
	req, ast, identity, linking := reconcileFixture(t, catalog)
	if len(reconcileDiagnostics(t, req, ast, identity, linking)) == 0 {
		t.Fatalf("missing Reconcile target accepted")
	}
}

func replaceAll(source, old, replacement string) string {
	out := make([]byte, 0, len(source))
	for {
		index := indexOf(source, old)
		if index < 0 {
			out = append(out, source...)
			return string(out)
		}
		out = append(out, source[:index]...)
		out = append(out, replacement...)
		source = source[index+len(old):]
	}
}

func indexOf(source, needle string) int {
	for i := 0; i+len(needle) <= len(source); i++ {
		if source[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
