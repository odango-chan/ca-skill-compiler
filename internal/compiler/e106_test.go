package compiler

// E106 conformance subset: independence-group structural linking.
// Runtime same-underlying-record UNKNOWN remains Host-side and is not emulated.

import "testing"

func e106Predicates(evidence string) string {
	return `predicates:
  test.ready:
    parameters: [{ue: string}]
    evidence:
` + evidence + `    evaluator: {type: cel, expression: 'ue != ""'}
`
}

func e106Request(t *testing.T, evidence string) BuildRequest {
	t.Helper()
	return v4TestRequest(v4TestContract, v4TestProcedure, e106Predicates(evidence), v4TestSchema)
}

func TestE106IndependenceGroupStructuralLink(t *testing.T) {
	positive := `      requires:
        ue: [ca::ue.registration-state, ca::ue.registration-freshness]
      independence:
        - members:
            - parameter: ue
              guarantee: ca::ue.registration-state
            - parameter: ue
              guarantee: ca::ue.registration-freshness
`
	if v4HasCode(Check(e106Request(t, positive)), "EVIDENCE_GUARANTEE_LINK_INVALID") {
		t.Fatalf("valid independence group rejected: %+v", Check(e106Request(t, positive)))
	}

	notRequired := `      requires:
        ue: [ca::ue.registration-state]
      independence:
        - members:
            - parameter: ue
              guarantee: ca::ue.registration-state
            - parameter: ue
              guarantee: ca::ue.not-required
`
	if !v4HasCode(Check(e106Request(t, notRequired)), "EVIDENCE_GUARANTEE_LINK_INVALID") {
		t.Fatalf("member pair outside requires accepted")
	}

	singleMember := `      requires:
        ue: [ca::ue.registration-state]
      independence:
        - members:
            - parameter: ue
              guarantee: ca::ue.registration-state
`
	if !v4HasCode(Check(e106Request(t, singleMember)), "EVIDENCE_GUARANTEE_LINK_INVALID") {
		t.Fatalf("single-member independence group accepted")
	}

	duplicate := `      requires:
        ue: [ca::ue.registration-state, ca::ue.registration-freshness]
      independence:
        - members:
            - parameter: ue
              guarantee: ca::ue.registration-state
            - parameter: ue
              guarantee: ca::ue.registration-state
`
	if !v4HasCode(Check(e106Request(t, duplicate)), "EVIDENCE_GUARANTEE_LINK_INVALID") {
		t.Fatalf("duplicate independence member accepted")
	}
}
