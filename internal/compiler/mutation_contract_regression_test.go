package compiler

// Regression coverage for reviewed Mutation/Reconcile fail-open paths in the
// trusted ToolCatalog contract: malformed failure classes and responsibility
// records must be rejected, and the reviewed recoverable-output path must link.

import (
	"strings"
	"testing"
)

func mutationContractRequest(t *testing.T, catalog string, procedure string) BuildRequest {
	t.Helper()
	fixture := wp3aBaseFixture(t)
	addSpecMutationProofSchemas(&fixture)
	fixture.addSnapshotFile("ca", "mutation-catalog.yaml", []byte(catalog))
	fixture.files[wp3aPredicatesPath] = []byte(specEvidenceFreePredicates)
	fixture.files[wp3aProcedurePath] = []byte(procedure)
	fixture.refreshSnapshotDigests(t)
	return wp3cRequest(t, fixture, specMutationCapabilityBundle(t))
}

func assertMutationContractRejected(t *testing.T, req BuildRequest, phrase string) {
	t.Helper()
	if !v4HasCode(Check(req), "RECONCILE_CONTRACT_INVALID") {
		t.Fatalf("malformed Mutation contract was accepted by Check: %+v", Check(req))
	}
	result := Build(req)
	if result.Success {
		t.Fatalf("malformed Mutation contract produced a successful Build")
	}
	if len(result.OutputFiles) != 0 {
		t.Fatalf("failed Build emitted deliverable output files: %d", len(result.OutputFiles))
	}
	if phrase != "" {
		found := false
		for _, diagnostic := range result.Diagnostics {
			if strings.Contains(diagnostic.Message, phrase) {
				found = true
			}
		}
		if !found {
			t.Fatalf("no diagnostic contains %q: %+v", phrase, result.Diagnostics)
		}
	}
}

const mutationOnlyProcedure = `start: mutate
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
          arguments: []
      routes:
        PASS: SUCCEEDED
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
`

func TestMutationMalformedFailureClassesRejected(t *testing.T) {
	malformed := strings.Replace(specMutationReconcileCatalogYAML, `        failureClasses:
          - id: transient-rejection
            outcome: NOT_COMMITTED
            proofSchema: urn:example:ca:commit-proof:v1
            proofPath: receipt.commit
            noSideEffect: true
            retryable: true
`, `        failureClasses:
          - broken
`, 1)
	req := mutationContractRequest(t, malformed, mutationOnlyProcedure)
	assertMutationContractRejected(t, req, "must be a mapping")
}

func TestMutationFailureClassBoolTypeRejected(t *testing.T) {
	malformed := strings.Replace(specMutationReconcileCatalogYAML, `            retryable: true`, `            retryable: "true"`, 1)
	req := mutationContractRequest(t, malformed, mutationOnlyProcedure)
	assertMutationContractRejected(t, req, "must be a YAML boolean")
}

func TestMutationMalformedResourceOutputsRejected(t *testing.T) {
	malformed := strings.Replace(specMutationReconcileCatalogYAML, `        reconcile:
          callable: example.session.reconcile
`, `        resourceOutputs:
          - broken
        reconcile:
          callable: example.session.reconcile
`, 1)
	req := mutationContractRequest(t, malformed, mutationOnlyProcedure)
	assertMutationContractRejected(t, req, "resourceOutputs[0] must be a mapping")
}

func TestMutationUnresolvedResponsibilitySchemaRejected(t *testing.T) {
	malformed := strings.Replace(specMutationReconcileCatalogYAML, `        reconcile:
          callable: example.session.reconcile
`, `        resourceOutputs:
          - path: output.state
            resourceKind: smf.registration
            responsibilitySchema: urn:example:ca:missing-responsibility:v1
        reconcile:
          callable: example.session.reconcile
`, 1)
	req := mutationContractRequest(t, malformed, mutationOnlyProcedure)
	assertMutationContractRejected(t, req, "responsibility Schema is not resolvable")
}

const resourceResponsibilitySchema = `$schema: https://json-schema.org/draft/2020-12/schema
$id: urn:example:ca:resource-responsibility:v1
type: object
additionalProperties: false
required: [source, resourceKind, owner]
properties:
  source: {type: string}
  resourceKind: {type: string}
  owner: {type: string}
`

func TestMutationValidResourceOutputsBuilds(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	addSpecMutationProofSchemas(&fixture)
	catalog := strings.Replace(specMutationReconcileCatalogYAML, `        reconcile:
          callable: example.session.reconcile
`, `        resourceOutputs:
          - path: output.state
            resourceKind: smf.registration
            responsibilitySchema: urn:example:ca:resource-responsibility:v1
        reconcile:
          callable: example.session.reconcile
`, 1)
	fixture.addSnapshotFile("ca", "mutation-catalog.yaml", []byte(catalog))
	fixture.addSnapshotFile("ca", "resource-responsibility.schema.yaml", []byte(resourceResponsibilitySchema))
	fixture.files[wp3aPredicatesPath] = []byte(specEvidenceFreePredicates)
	fixture.files[wp3aProcedurePath] = []byte(mutationOnlyProcedure)
	fixture.refreshSnapshotDigests(t)
	req := wp3cRequest(t, fixture, specMutationCapabilityBundle(t))
	result := Build(req)
	if !result.Success {
		t.Fatalf("valid resourceOutputs declaration must Build: %+v", result.Diagnostics)
	}
	if len(result.OutputFiles) == 0 {
		t.Fatal("successful Build emitted no artifact files")
	}
}

func recoveryMutationRequest(t *testing.T, catalog string, descriptor string) BuildRequest {
	t.Helper()
	fixture := wp3aBaseFixture(t)
	fixture.addSnapshotFile("ca", "mutation-catalog.yaml", []byte(catalog))
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

func TestMutationInvalidRecoverableOutputPathRejected(t *testing.T) {
	catalog := strings.Replace(recoveryMutationCatalog, `          recoverableOutputPath: output.recovered`, `          recoverableOutputPath: output.missing`, 1)
	req := recoveryMutationRequest(t, catalog, recoveryDescriptor(true))
	if !v4HasCode(Check(req), "RECONCILE_CONTRACT_INVALID") {
		t.Fatalf("invalid recoverableOutputPath accepted: %+v", Check(req))
	}
	result := Build(req)
	if result.Success || len(result.OutputFiles) != 0 {
		t.Fatalf("invalid recoverableOutputPath produced a deliverable Build: %+v", result.Diagnostics)
	}
}

func TestMutationValidRecoverableOutputBuilds(t *testing.T) {
	req := recoveryMutationRequest(t, recoveryMutationCatalog, recoveryDescriptor(true))
	result := Build(req)
	if !result.Success {
		t.Fatalf("valid recoverable-output Mutation must Build: %+v", result.Diagnostics)
	}
	if len(result.OutputFiles) == 0 {
		t.Fatal("successful Build emitted no artifact files")
	}
}
