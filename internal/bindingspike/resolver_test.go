package bindingspike

import "testing"

// Names match the 11 repeated fields in the current MAS Procedure/ToolCatalog.
// Semantic IDs are hypothetical proposed annotations, NOT existing contracts.
var mas = []struct{ alias, input, typ, semantic string }{
	{"ue", "ueRef", "string", "session.ue"},
	{"ingress", "ingressRef", "string", "invocation.ingress"},
	{"lifecycle", "lifecycleRef", "string", "invocation.lifecycle"},
	{"pduSessionId", "pduSessionId", "integer", "session.psi"},
	{"pti", "pti", "integer", "session.pti"},
	{"requestType", "requestType", "integer", "request.type"},
	{"dnn", "dnn", "string", "session.dnn"},
	{"sNssai", "sNssai", "object", "session.snssai"},
	{"servingPlmn", "servingPlmn", "object", "session.plmn"},
	{"requestedPduSessionType", "requestedPduSessionType", "string", "session.requestedType"},
	{"requestedSscMode", "requestedSscMode", "integer", "session.requestedSsc"},
}

func TestMASContextFanout(t *testing.T) {
	var ps []Producer
	var cs []Consumer
	for _, f := range mas {
		ps = append(ps, Producer{ID: "context." + f.alias, Field: Field{Name: f.alias, SemanticID: f.semantic, Type: f.typ}, SourcePath: "input." + f.alias})
	}
	for i := 0; i < 16; i++ {
		n := string(rune('a' + i))
		inputs := make([]Field, 0, len(mas))
		for _, f := range mas {
			inputs = append(inputs, Field{Name: f.input, SemanticID: f.semantic, Type: f.typ, Required: true})
		}
		cs = append(cs, Consumer{NodeID: n, Operation: "mas.sample", Inputs: inputs})
		for j := range ps { ps[j].AvailableAt = append(ps[j].AvailableAt, n) }
	}
	got := Resolve(ps, cs)
	if len(got.Diagnostics) != 0 || len(got.Bindings) != 176 {
		t.Fatalf("want 176 bindings; got %d and diagnostics %+v", len(got.Bindings), got.Diagnostics)
	}
}

func TestFiveGCContextAliases(t *testing.T) {
	ps := []Producer{
		{"subject.ue", Field{Name: "ue-ref", SemanticID: "session.ue", Type: "string"}, "input.ue-ref", []string{"udm"}},
		{"facts.lifecycle", Field{Name: "lifecycle-ref", SemanticID: "invocation.lifecycle", Type: "string"}, "input.lifecycle-ref", []string{"udm"}},
		{"facts.ingress", Field{Name: "ingress-ref", SemanticID: "invocation.ingress", Type: "string"}, "input.ingress-ref", []string{"udm"}},
	}
	cs := []Consumer{{NodeID: "udm", Operation: "example.read", Inputs: []Field{
		{Name: "ueRef", SemanticID: "session.ue", Type: "string", Required: true},
		{Name: "lifecycleRef", SemanticID: "invocation.lifecycle", Type: "string", Required: true},
		{Name: "ingressRef", SemanticID: "invocation.ingress", Type: "string", Required: true},
	}}}
	got := Resolve(ps, cs)
	if len(got.Bindings) != 3 || len(got.Diagnostics) != 0 { t.Fatalf("unexpected result %+v", got) }
}

func TestFailClosed(t *testing.T) {
	base := Producer{"p1", Field{Name: "stateRef", SemanticID: "policy.current", Type: "string"}, "create-policy.output.stateRef", []string{"next"}}
	input := Field{Name: "policyStateRef", SemanticID: "policy.current", Type: "string", Required: true}
	cases := []struct { name string; ps []Producer; in Field; want string }{
		{"ambiguous", []Producer{base, {"p2", base.Field, "update-policy.output.stateRef", []string{"next"}}}, input, "BINDING_AMBIGUOUS"},
		{"wrong type", []Producer{{"p", Field{Name: "policyStateRef", SemanticID: "policy.current", Type: "integer"}, "p.output", []string{"next"}}}, input, "BINDING_TYPE_MISMATCH"},
		{"nullable", []Producer{{"p", Field{Name: "policyStateRef", SemanticID: "policy.current", Type: "string", Nullable: true}, "p.output", []string{"next"}}}, input, "BINDING_TYPE_MISMATCH"},
		{"bypass branch", []Producer{{"p", base.Field, "branch-a.output.stateRef", []string{"branch-a"}}}, input, "BINDING_SOURCE_UNAVAILABLE"},
		{"same name wrong semantics", []Producer{{"p", Field{Name: "policyStateRef", SemanticID: "session.other", Type: "string"}, "p.output", []string{"next"}}}, input, "BINDING_UNRESOLVED"},
		{"missing semantic identity", []Producer{base}, Field{Name: "policyStateRef", Type: "string", Required: true}, "INVALID_CONTRACT"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Resolve(tc.ps, []Consumer{{NodeID: "next", Operation: "mas.op", Inputs: []Field{tc.in}}})
			if len(got.Bindings) != 0 || len(got.Diagnostics) != 1 || got.Diagnostics[0].Code != tc.want {
				t.Fatalf("expected %s: %+v", tc.want, got)
			}
		})
	}
}

func TestOptionalAndDeterministicOrder(t *testing.T) {
	ps := []Producer{
		{"b", Field{SemanticID: "b", Type: "integer"}, "req.b", []string{"node"}},
		{"a", Field{SemanticID: "a", Type: "string"}, "req.a", []string{"node"}},
	}
	cs := []Consumer{{NodeID: "node", Inputs: []Field{
		{Name: "z", SemanticID: "b", Type: "integer", Required: true},
		{Name: "a", SemanticID: "a", Type: "string", Required: true},
		{Name: "missing", SemanticID: "optional", Type: "string"},
	}}}
	got := Resolve(ps, cs)
	if len(got.Bindings) != 2 || len(got.Diagnostics) != 0 || got.Bindings[0].Target != "node.a" || got.Bindings[1].Target != "node.z" {
		t.Fatalf("unexpected %+v", got)
	}
}
