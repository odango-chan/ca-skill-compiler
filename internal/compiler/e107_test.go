package compiler

// E107 conformance subset: the trusted dedicated Reconcile READ variants that
// the static Mutation/Reconcile validator can decide. The "extra writable input
// field" variant and the "grant only the original Mutation" variant remain
// covered by capability/Schema slices or recorded as not yet fixture-backed.

import "testing"

func TestE107ReconcileReadVariants(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(string) string
	}{
		{
			name: "Reconcile callable kind observation",
			mutate: func(catalog string) string {
				return replaceAll(catalog, "- kind: operation\n      id: example.session.reconcile", "- kind: observation\n      id: example.session.reconcile")
			},
		},
		{
			name: "Reconcile callable effect MUTATION",
			mutate: func(catalog string) string {
				return replaceAll(catalog, "id: example.session.reconcile\n      effect: READ", "id: example.session.reconcile\n      effect: MUTATION")
			},
		},
		{
			name:   "delete Reconcile proofPath",
			mutate: func(catalog string) string { return replaceAll(catalog, "          proofPath: receipt.commit\n", "") },
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			req, _, _, _ := reconcileFixture(t, testCase.mutate(reconcileCatalog))
			found := false
			for _, diagnostic := range Check(req) {
				if diagnostic.Code == "RECONCILE_CONTRACT_INVALID" || diagnostic.Code == "EFFECT_MISMATCH" || diagnostic.Code == "SCHEMA_INVALID" {
					found = true
				}
			}
			if !found {
				t.Fatalf("variant %q accepted: %+v", testCase.name, Check(req))
			}
		})
	}
}
