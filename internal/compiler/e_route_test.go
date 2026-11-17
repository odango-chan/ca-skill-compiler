package compiler

// E76/E84/E90/E50/E85 static conformance subset.

import (
	"strings"
	"testing"
)

func TestE76ReadStageRouteVariants(t *testing.T) {
	// Non-last READ stage with only the normal auto-advance is legal.
	legal := v4TestRequest(v4TestContract, `start: read
nodes:
  read:
    observation:
      tool: ca::example.session.read
      arguments: {ueRef: context.ueRef}
    postcondition:
      predicates:
        - predicate: test.complete
          arguments: [read.observation.output]
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`, v4TestPredicates, v4TestSchema)
	if got := Check(legal); v4HasCode(got, "ROUTE_OUTCOME_MISSING") {
		t.Fatalf("non-last READ stage without explicit normal route rejected: %+v", got)
	}

	// Last READ stage omitting the normal SUCCEEDED route is rejected.
	missingNormal := v4TestRequest(v4TestContract, `start: read
nodes:
  read:
    observation:
      tool: ca::example.session.read
      arguments: {ueRef: context.ueRef}
      routes:
        FAILED: FAILED
`, v4TestPredicates, v4TestSchema)
	if got := Check(missingNormal); !v4HasCode(got, "ROUTE_OUTCOME_MISSING") {
		t.Fatalf("last READ stage without SUCCEEDED route accepted: %+v", got)
	}

	// Owner-declared FAILED omitted from a linked READ stage is rejected.
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "        FAILED: FAILED\n        UNKNOWN: UNRESOLVED", "        UNKNOWN: UNRESOLVED")
	if got := Check(fixture.request(t)); !v4HasCode(got, "ROUTE_OUTCOME_MISSING") {
		t.Fatalf("omitted Owner-declared FAILED route accepted: %+v", got)
	}
}

func eRouteFixture(t *testing.T, catalog, procedure string) BuildRequest {
	t.Helper()
	fixture := wp3aBaseFixture(t)
	fixture.addSnapshotFile("ca", "mutation-catalog.yaml", []byte(catalog))
	fixture.addSnapshotFile("ca", "proof.schema.yaml", []byte(reconcileProofSchema))
	fixture.addSnapshotFile("ca", "receipt.schema.yaml", []byte(reconcileReceiptSchema))
	fixture.addSnapshotFile("ca", "recoverable.schema.yaml", []byte(reconcileRecoverableSchema))
	fixture.refreshSnapshotDigests(t)
	fixture.files[wp3aProcedurePath] = []byte(procedure)
	return fixture.request(t)
}

const eRouteMutationProcedure = `start: mutate
nodes:
  mutate:
    operation:
      tool: ca::example.session.mutate
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
        FAILED: FAILED
        NOT_COMMITTED: SAFE_STOPPED
        OUTCOME_UNKNOWN: UNRESOLVED
        COMMITTED: SAFE_STOPPED
`

func TestE84MutationRouteKeyVariants(t *testing.T) {
	// Owner does not publish NOT_COMMITTED: writing the key is ineligible.
	unpublished := replaceAll(reconcileCatalog, "ownerOutcomes: [SUCCEEDED, NOT_COMMITTED]", "ownerOutcomes: [SUCCEEDED]")
	req := eRouteFixture(t, unpublished, eRouteMutationProcedure)
	if got := Check(req); !v4HasCode(got, "ROUTE_OUTCOME_UNKNOWN") {
		t.Fatalf("NOT_COMMITTED route without Owner publication accepted: %+v", got)
	}

	// COMMITTED without a trusted Reconcile capability is ineligible.
	noReconcile := reconcileCatalog
	start := strings.Index(noReconcile, "      mutation:\n        commitProof:")
	end := strings.Index(noReconcile[start:], "    - kind: operation\n      id: example.session.reconcile")
	if start < 0 || end < 0 {
		t.Fatal("catalog anchors missing")
	}
	noReconcile = noReconcile[:start] + noReconcile[start+end:]
	req = eRouteFixture(t, noReconcile, eRouteMutationProcedure)
	if got := Check(req); !v4HasCode(got, "ROUTE_OUTCOME_UNKNOWN") {
		t.Fatalf("COMMITTED route without trusted Reconcile accepted: %+v", got)
	}
}

func TestE90GoalIdentifierVariants(t *testing.T) {
	invalid := []string{"ue registration verified", "ue..registration_verified"}
	for _, goal := range invalid {
		fixture := wp3aBaseFixture(t)
		fixture.replaceFile(wp3aContractPath, "goal: ue.registration_verified", "goal: "+goal)
		if got := Check(fixture.request(t)); !v4HasCode(got, "IDENTIFIER_INVALID") {
			t.Fatalf("invalid GoalId %q accepted: %+v", goal, got)
		}
	}

	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "            - inspect-registration.observation.output", "            - context")
	if got := Check(fixture.request(t)); !v4HasCode(got, "SOURCE_PATH_INVALID") {
		t.Fatalf("bare context argument accepted: %+v", got)
	}
}

func TestE50DuplicatePredicateCallMembers(t *testing.T) {
	procedure := strings.Replace(v4TestProcedure,
		"    precondition:\n      predicates:\n        - predicate: test.ready\n          arguments: [context.ueRef]",
		"    precondition:\n      predicates:\n        - predicate: test.ready\n          arguments: [context.ueRef]\n        - predicate: test.ready\n          arguments: [context.ueRef]", 1)
	request := v4TestRequest(v4TestContract, procedure, v4TestPredicates, v4TestSchema)
	if got := Check(request); v4HasCode(got, "DUPLICATE_ID") || v4HasCode(got, "PREDICATE_GROUP_INVALID") {
		t.Fatalf("duplicate PredicateId call members treated as duplicate keys: %+v", got)
	}
}

func TestE85RetryBypassAndSecondGeneration(t *testing.T) {
	bypass := strings.Replace(retryProcedureTemplate, "        NOT_COMMITTED: retry\n        SUCCEEDED: FAILED", "        NOT_COMMITTED: retry\n        SUCCEEDED: retry", 1)
	request := v4TestRequest(v4TestContract, bypass, v4TestPredicates, v4TestSchema)
	if !v4HasCode(Check(request), "RETRY_CONTRACT_INVALID") {
		t.Fatalf("bypass edge into retry Node accepted: %+v", Check(request))
	}

	secondGeneration := strings.Replace(retryProcedureTemplate, "      retryOf: mutate", "      retryOf: retry", 1)
	request = v4TestRequest(v4TestContract, secondGeneration, v4TestPredicates, v4TestSchema)
	if !v4HasCode(Check(request), "RETRY_CONTRACT_INVALID") {
		t.Fatalf("second-generation retryOf accepted: %+v", Check(request))
	}
}
