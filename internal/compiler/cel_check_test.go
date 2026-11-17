package compiler

// WP4D conformance fixtures: CEL subset parsing/type checking and Predicate
// parameter signatures.

import "testing"

func TestWP4DCELSubset(t *testing.T) {
	parameters := map[string]string{"a": "boolean", "b": "boolean", "x": "string", "y": "integer", "s": "string", "m": "schema"}
	valid := []string{
		`a && !b`,
		`x == "ACTIVE"`,
		`x != "IDLE" || y > 3`,
		`has(m.state) && m.state == "REGISTERED"`,
		`size(s) > 0`,
	}
	for _, expression := range valid {
		if problem := checkV1Alpha4CELExpression(expression, parameters); problem != "" {
			t.Fatalf("valid CEL expression rejected (%s): %s", expression, problem)
		}
	}
	invalid := []string{
		`[1, 2].all(v, v > 0)`,
		`y in [1, 2]`,
		`"A" + "B" == "AB"`,
		`m[0] == 1`,
		`{"a": 1} == {"a": 1}`,
		`unknownFunction(y)`,
		`y`,
		`x > 1`,
		`undeclared == 1`,
		`size(s, 1) > 0`,
	}
	for _, expression := range invalid {
		if problem := checkV1Alpha4CELExpression(expression, parameters); problem == "" {
			t.Fatalf("out-of-subset CEL expression accepted: %s", expression)
		}
	}
}

func TestWP4DCELIntegratedIntoBuild(t *testing.T) {
	valid := v4TestRequest(v4TestContract, v4TestProcedure, v4TestPredicates, v4TestSchema)
	if diagnostics := Check(valid); v4HasCode(diagnostics, "CEL_INVALID") {
		t.Fatalf("valid CEL fixture rejected: %+v", diagnostics)
	}

	cases := map[string]string{
		"non-boolean result":  `evaluator: {type: cel, expression: "1 + 2"}`,
		"macro/comprehension": `evaluator: {type: cel, expression: "[1].all(v, v > 0)"}`,
		"array index":         `evaluator: {type: cel, expression: "xs[0] == 1"}`,
	}
	for name, evaluator := range cases {
		request := v4TestRequest(v4TestContract, v4TestProcedure,
			`predicates:
  test.ready:
    parameters: [{ue: string}]
    evaluator: {type: cel, expression: 'ue != ""'}
  test.complete:
    parameters: [{result: ca::urn:example:result:v1}]
    `+evaluator+`
`, v4TestSchema)
		diagnostics := Check(request)
		if !v4HasCode(diagnostics, "CEL_INVALID") {
			t.Fatalf("%s: expected CEL_INVALID, got %+v", name, diagnostics)
		}
		found := false
		for _, diagnostic := range diagnostics {
			if diagnostic.Code == "CEL_INVALID" && diagnostic.Path != "" && diagnostic.Line > 0 && diagnostic.Column > 0 {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s: CEL_INVALID must retain the predicates.yaml source location: %+v", name, diagnostics)
		}
	}
}

func TestWP4DInvalidParameterSignature(t *testing.T) {
	request := v4TestRequest(v4TestContract, v4TestProcedure,
		`predicates:
  test.ready:
    parameters: [{ue: {type: string}}]
    evaluator: {type: cel, expression: 'ue != ""'}
  test.complete:
    parameters: [{result: ca::urn:example:result:v1}]
    evaluator: {type: cel, expression: "result.ready == true"}
`, v4TestSchema)
	diagnostics := Check(request)
	if !v4HasCode(diagnostics, "PREDICATE_GROUP_INVALID") {
		t.Fatalf("invalid parameter type was accepted: %+v", diagnostics)
	}
}
