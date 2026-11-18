package compiler

// E106 conformance subset: independence-group structural linking.
// Runtime same-underlying-record UNKNOWN remains Host-side and is not emulated.
// Trusted deployment domain relations are covered by the Evidence Descriptor
// positive-path fixture; this test pins the structural independence grammar.

import (
	"testing"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func e106Predicates(evidence string) string {
	return `predicates:
  test.ready:
    parameters: [{ue: string}]
    evidence:
` + evidence + `    evaluator: {type: cel, expression: 'ue != ""'}
`
}

func e106StructuralDiagnostics(t *testing.T, evidence string) []Diagnostic {
	t.Helper()
	files := testRequest("", "", e106Predicates(evidence), "").Files
	var predicatesFile []byte
	for _, file := range files {
		if file.Path == "skills/pdu/assurance/predicates.yaml" {
			predicatesFile = file.Bytes
		}
	}
	root, err := yaml.Parse(predicatesFile)
	if err != nil {
		t.Fatalf("parse predicates: %v", err)
	}
	parsed := map[string]*yaml.Node{"skills/pdu/assurance/predicates.yaml": root}
	ast := parseSkillAST("pdu", parsed)
	diagnostics := make([]Diagnostic, 0)
	add := func(path string, node *yaml.Node, code, message string) {
		diagnostics = append(diagnostics, dependencyDiagnostic("error", "C_STATIC", code, path, node, message))
	}
	validateIndependenceGroups("skills/pdu/assurance/predicates.yaml", ast.Predicates, add)
	return diagnostics
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
	if diagnostics := e106StructuralDiagnostics(t, positive); len(diagnostics) != 0 {
		t.Fatalf("valid independence group rejected: %+v", diagnostics)
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
	if !hasCode(e106StructuralDiagnostics(t, notRequired), "EVIDENCE_GUARANTEE_LINK_INVALID") {
		t.Fatalf("independence member outside evidence.requires accepted")
	}

	singleMember := `      requires:
        ue: [ca::ue.registration-state]
      independence:
        - members:
            - parameter: ue
              guarantee: ca::ue.registration-state
`
	if !hasCode(e106StructuralDiagnostics(t, singleMember), "EVIDENCE_GUARANTEE_LINK_INVALID") {
		t.Fatalf("single-member independence group accepted")
	}
}
