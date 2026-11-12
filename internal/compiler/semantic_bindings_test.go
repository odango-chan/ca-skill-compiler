package compiler

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func semanticTestNode(t *testing.T, source string) *yaml.Node {
	t.Helper()
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(source), &document); err != nil { t.Fatal(err) }
	if len(document.Content) != 1 { t.Fatalf("expected one YAML root: %#v", document.Content) }
	return document.Content[0]
}

func TestSemanticFieldIDRequiresExplicitNamespacedAnnotation(t *testing.T) {
	for _, tc := range []struct{ name, schema, want string; ok bool }{
		{"valid", "{type: string, x-ca-semanticId: session.subject.ue}", "session.subject.ue", true},
		{"missing", "{type: string}", "", false},
		{"empty", "{type: string, x-ca-semanticId: ''}", "", false},
		{"non-string", "{type: string, x-ca-semanticId: 7}", "", false},
		{"unqualified", "{type: string, x-ca-semanticId: ue}", "", false},
		{"empty-segment", "{type: string, x-ca-semanticId: session..ue}", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id, ok := semanticFieldID(semanticTestNode(t, tc.schema))
			if id != tc.want || ok != tc.ok { t.Fatalf("got (%q,%v), want (%q,%v)", id, ok, tc.want, tc.ok) }
		})
	}
}

func TestSemanticProcedureInputBindingCandidate(t *testing.T) {
	proc := semanticTestNode(t, "inputs: {ue: {from: context.subject.ueRef}}")
	inputSchema := semanticTestNode(t, "{type: string, x-ca-semanticId: session.subject.ue}")
	expected := semanticTestNode(t, "type: object\nrequired: [ueRef]\nproperties:\n  ueRef: {type: string, x-ca-semanticId: session.subject.ue}")
	a := &analysis{procedureInputSchemas: map[*yaml.Node]map[string]*yaml.Node{
		proc: {"ue": inputSchema},
	}}

	with := a.addSemanticProcedureInputBindings(proc, nil, expected, "skills/demo/assurance/procedure.yaml", "invoke")
	if with == nil || len(with.Content) != 2 { t.Fatalf("expected one generated mapping: %#v", with) }
	if scalar(with.Content[0]) != "ueRef" || scalar(child(with.Content[1], "from")) != "input.ue" {
		t.Fatalf("unexpected generated binding: %#v", with)
	}
}

func TestSemanticProcedureInputBindingFailsClosed(t *testing.T) {
	expected := semanticTestNode(t, "type: object\nrequired: [stateRef]\nproperties:\n  stateRef: {type: string, x-ca-semanticId: policy.current}")
	one := semanticTestNode(t, "{type: string, x-ca-semanticId: policy.current}")
	proc := semanticTestNode(t, "inputs: {state: {from: context.facts.state}}")

	t.Run("ambiguous", func(t *testing.T) {
		a := &analysis{procedureInputSchemas: map[*yaml.Node]map[string]*yaml.Node{
			proc: {"first": one, "second": one},
		}}
		if got := a.addSemanticProcedureInputBindings(proc, nil, expected, "procedure.yaml", "call"); got != nil {
			t.Fatalf("ambiguous producer must not bind: %#v", got)
		}
	})
	t.Run("same name different semantic identity", func(t *testing.T) {
		a := &analysis{procedureInputSchemas: map[*yaml.Node]map[string]*yaml.Node{
			proc: {"stateRef": semanticTestNode(t, "{type: string, x-ca-semanticId: session.other} ")},
		}}
		if got := a.addSemanticProcedureInputBindings(proc, nil, expected, "procedure.yaml", "call"); got != nil {
			t.Fatalf("same field name must not bind: %#v", got)
		}
	})
	t.Run("schema proof remains canonical", func(t *testing.T) {
		source := semanticTestNode(t, "{type: integer, x-ca-semanticId: policy.current}")
		a := &analysis{procedureInputSchemas: map[*yaml.Node]map[string]*yaml.Node{
			proc: {"state": source},
		}}
		got := a.addSemanticProcedureInputBindings(proc, nil, expected, "procedure.yaml", "call")
		if got == nil || scalar(child(got.Content[1], "from")) != "input.state" {
			t.Fatalf("semantic resolver should propose the unique identity candidate: %#v", got)
		}
		if schemasCompatible(source, schemaProperty(expected, "stateRef")) {
			t.Fatal("canonical schema checker should reject integer Producer for string Consumer")
		}
	})
}

func TestSemanticBindingPreservesExplicitMapping(t *testing.T) {
	proc := semanticTestNode(t, "inputs: {ue: {from: context.subject.ueRef}}")
	inputSchema := semanticTestNode(t, "{type: string, x-ca-semanticId: session.subject.ue}")
	expected := semanticTestNode(t, "type: object\nrequired: [ueRef]\nproperties:\n  ueRef: {type: string, x-ca-semanticId: session.subject.ue}")
	explicit := semanticTestNode(t, "ueRef: {from: input.explicitOverride}")
	a := &analysis{procedureInputSchemas: map[*yaml.Node]map[string]*yaml.Node{proc: {"ue": inputSchema}}}
	got := a.addSemanticProcedureInputBindings(proc, explicit, expected, "procedure.yaml", "call")
	if got != explicit { t.Fatal("explicit authored binding must remain untouched") }
}
