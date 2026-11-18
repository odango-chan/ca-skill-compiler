package ingresscatalog

// IN-027 / IN-061 structural response-contract presence tests.
//
// The released normative IngressCatalog v1alpha1 structural Schema makes the
// paired-response deadline-policy reference and durable delivery-recovery
// declaration mandatory; the Compiler-owned static gate for a missing
// declaration is that structural admission identity. Contradictory declarations
// that the structural Schema cannot express are covered by
// INGRESS_RESPONSE_CONTRACT_INVALID checks (duplicate disposition id/status,
// early-failure policy, selector coverage).

import "testing"

func pairedResponseFixture(t *testing.T, root map[string]any) map[string]any {
	t.Helper()
	response := map[string]any{
		"mode":              "paired",
		"deadlinePolicyRef": "deployment-policy::synthetic-response-deadline",
		"deliveryRecovery": map[string]any{
			"mode":                   "protocol-replay",
			"requestIdentity":        "synthetic-request-identity",
			"lostChannelDisposition": "explicit-failure-response",
		},
		"dispositions": []any{map[string]any{
			"id": "updated", "status": int64(200), "body": "none",
		}},
	}
	operation := firstOperation(t, root)
	operation["response"] = response
	return root
}

func expectCatalogInvalidAtResponse(t *testing.T, admission *Admission) {
	t.Helper()
	if admission.Valid() {
		t.Fatalf("paired response contract defect accepted")
	}
	for _, diagnostic := range admission.Diagnostics {
		if diagnostic.Code == CodeIngressCatalogInvalid && len(diagnostic.SourceLocation.JSONPointer) >= len("#/spec/operations/0/response") &&
			diagnostic.SourceLocation.JSONPointer[:len("#/spec/operations/0/response")] == "#/spec/operations/0/response" {
			return
		}
	}
	t.Fatalf("expected INGRESS_CATALOG_INVALID under the response object, got %#v", admission.Diagnostics)
}

// IN-027: a paired response missing the deadline-policy declaration is rejected.
func TestPairedResponseMissingDeadlinePolicyRejected(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := pairedResponseFixture(t, snapshot.catalogMap(t))
	delete(firstOperation(t, root)["response"].(map[string]any), "deadlinePolicyRef")
	expectCatalogInvalidAtResponse(t, snapshot.admitMap(t, root))
}

// IN-061: a paired response missing the durable delivery-recovery declaration is
// rejected.
func TestPairedResponseMissingDeliveryRecoveryRejected(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := pairedResponseFixture(t, snapshot.catalogMap(t))
	delete(firstOperation(t, root)["response"].(map[string]any), "deliveryRecovery")
	expectCatalogInvalidAtResponse(t, snapshot.admitMap(t, root))
}

// IN-061: a paired response that declares no dispositions is rejected.
func TestPairedResponseMissingDispositionsRejected(t *testing.T) {
	snapshot := newSyntheticSnapshot(t)
	root := pairedResponseFixture(t, snapshot.catalogMap(t))
	response := firstOperation(t, root)["response"].(map[string]any)
	delete(response, "dispositions")
	expectCatalogInvalidAtResponse(t, snapshot.admitMap(t, root))
}
