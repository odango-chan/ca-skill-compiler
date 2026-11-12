package bindingspike

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// This test reads the source fixtures already pinned by Compiler PR #5.
// SemanticIDs are experimental annotations supplied here, not inferred from
// YAML property names and not yet part of an approved Tool Owner contract.
type sourceProcedure struct {
	Procedures map[string]struct {
		Inputs map[string]struct {
			Type string `yaml:"type"`
			From string `yaml:"from"`
		} `yaml:"inputs"`
		Nodes map[string]struct {
			Call string `yaml:"call"`
			With map[string]struct {
				From string `yaml:"from"`
			} `yaml:"with"`
		} `yaml:"nodes"`
	} `yaml:"procedures"`
}

type sourceCatalog struct {
	Spec struct {
		Contracts []struct {
			Operation string `yaml:"operation"`
			InputSchema struct {
				Required   []string `yaml:"required"`
				Properties map[string]struct {
					Type string `yaml:"type"`
				} `yaml:"properties"`
			} `yaml:"inputSchema"`
		} `yaml:"contracts"`
	} `yaml:"spec"`
}

func fixtureYAML(t *testing.T, relative string, target any) {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Join("..", "..", "examples", "ca-agent-demo-project", relative))
	if err != nil { t.Fatalf("read fixture %s: %v", relative, err) }
	if err := yaml.Unmarshal(bytes, target); err != nil { t.Fatalf("parse fixture %s: %v", relative, err) }
}

func TestPinnedMASProcedureAgainstProposedToolCatalog(t *testing.T) {
	var proc sourceProcedure
	fixtureYAML(t, "skills/establish-pdu-session-mas/assurance/procedure.yaml", &proc)
	var catalog sourceCatalog
	fixtureYAML(t, "lib/tools/mas-proposed.tool.yaml", &catalog)
	p, ok := proc.Procedures["establish-session"]
	if !ok { t.Fatal("MAS establish-session procedure missing") }

	specs := make(map[string]struct {
		Type string
		ID string
	})
	for _, f := range mas {
		if p.Inputs[f.alias].Type != f.typ { t.Fatalf("MAS input %s type changed", f.alias) }
		if p.Inputs[f.alias].From == "" { t.Fatalf("MAS input %s lost context provenance", f.alias) }
		specs[f.input] = struct{ Type, ID string }{f.typ, f.semantic}
	}
	operationSchemas := make(map[string]map[string]struct{ Type string })
	for _, c := range catalog.Spec.Contracts {
		props := make(map[string]struct{ Type string })
		required := make(map[string]bool)
		for _, name := range c.InputSchema.Required { required[name] = true }
		for inputName, spec := range specs {
			x, exists := c.InputSchema.Properties[inputName]
			if exists && required[inputName] && x.Type != spec.Type {
				t.Fatalf("MAS %s input %s schema type %s != %s", c.Operation, inputName, x.Type, spec.Type)
			}
			if exists && required[inputName] { props[inputName] = struct{ Type string }{x.Type} }
		}
		operationSchemas[c.Operation] = props
	}

	var producers []Producer
	var consumers []Consumer
	observedMappings := make(map[string]string)
	for _, f := range mas {
		producers = append(producers, Producer{
			ID: "context." + f.alias,
			Field: Field{Name: f.alias, SemanticID: f.semantic, Type: f.typ},
			SourcePath: "input." + f.alias,
		})
	}
	calls := 0
	for nodeID, node := range p.Nodes {
		if node.Call == "" { continue }
		calls++
		schema, ok := operationSchemas[node.Call]
		if !ok { t.Fatalf("missing proposed MAS contract for %s", node.Call) }
		inputs := make([]Field, 0, len(mas))
		for _, f := range mas {
			if _, ok := schema[f.input]; !ok { t.Fatalf("MAS %s lacks required schema field %s", node.Call, f.input) }
			exp := "input." + f.alias
			if node.With[f.input].From != exp {
				t.Fatalf("MAS %s.%s expected existing explicit mapping %s, got %s", nodeID, f.input, exp, node.With[f.input].From)
			}
			inputs = append(inputs, Field{Name: f.input, SemanticID: f.semantic, Type: f.typ, Required: true})
			observedMappings[nodeID+"."+f.input] = exp
		}
		consumers = append(consumers, Consumer{NodeID: nodeID, Operation: node.Call, Inputs: inputs})
		for i := range producers { producers[i].AvailableAt = append(producers[i].AvailableAt, nodeID) }
	}
	if calls != 16 { t.Fatalf("MAS fixture changed: want 16 calls, got %d", calls) }
	result := Resolve(producers, consumers)
	if len(result.Diagnostics) != 0 || len(result.Bindings) != 176 { t.Fatalf("MAS generated %d bindings / diagnostics %+v", len(result.Bindings), result.Diagnostics) }
	for _, b := range result.Bindings {
		if observedMappings[b.Target] != b.Source { t.Fatalf("MAS binding drift: %s -> %s, want %s", b.Target, b.Source, observedMappings[b.Target]) }
	}
}

// The traditional 5GC input aliases differ from the actual Tool parameters.
// This checks a real current Procedure node and approved 5GC ToolCatalog,
// but the semantic annotations remain intentionally fixture-only.
func TestPinnedFiveGCProcedureAgainstToolCatalog(t *testing.T) {
	var proc sourceProcedure
	fixtureYAML(t, "skills/establish-pdu-session/assurance/procedure.yaml", &proc)
	var catalog sourceCatalog
	fixtureYAML(t, "lib/tools/createdsession.tool.yaml", &catalog)
	p := proc.Procedures["ue-requested-pdu-session-establishment"]
	n, ok := p.Nodes["nudm-sdm-get"]
	if !ok || n.Call != "sbi.udm.sm_data.get" { t.Fatalf("5GC UDM call missing or changed: %+v", n) }
	fields := []struct{ alias, input, typ, id string }{
		{"ue-ref", "ueRef", "string", "session.subject.ue"},
		{"dnn", "dnn", "string", "session.dnn"},
		{"slice", "sNssai", "object", "session.sNssai"},
		{"plmn", "servingPlmn", "object", "session.servingPlmn"},
	}
	var schemaTypes map[string]string
	var schemaRequired []string
	for _, c := range catalog.Spec.Contracts {
		if c.Operation == n.Call {
			schemaTypes = make(map[string]string)
			for name, property := range c.InputSchema.Properties { schemaTypes[name] = property.Type }
			schemaRequired = c.InputSchema.Required
			break
		}
	}
	if schemaTypes == nil { t.Fatal("UDM tool schema missing") }
	required := make(map[string]bool)
	for _, key := range schemaRequired { required[key] = true }
	var ps []Producer
	var inputs []Field
	for _, f := range fields {
		if p.Inputs[f.alias].Type != f.typ || p.Inputs[f.alias].From == "" { t.Fatalf("5GC input alias %s inconsistent", f.alias) }
		if !required[f.input] || schemaTypes[f.input] != f.typ { t.Fatalf("5GC UDM contract input %s inconsistent", f.input) }
		if n.With[f.input].From != "input."+f.alias { t.Fatalf("5GC existing source mismatch: %s", f.input) }
		ps = append(ps, Producer{ID: "context."+f.alias, Field: Field{Name: f.alias, SemanticID: f.id, Type: f.typ}, SourcePath: "input."+f.alias, AvailableAt: []string{"nudm-sdm-get"}})
		inputs = append(inputs, Field{Name: f.input, SemanticID: f.id, Type: f.typ, Required: true})
	}
	got := Resolve(ps, []Consumer{{NodeID: "nudm-sdm-get", Operation: n.Call, Inputs: inputs}})
	if len(got.Diagnostics) != 0 || len(got.Bindings) != 4 { t.Fatalf("5GC binding mismatch %+v", got) }
	for _, b := range got.Bindings { if n.With[b.Target[len("nudm-sdm-get."):]].From != b.Source { t.Fatalf("generated %s does not match authored source %s", b.Target, b.Source) } }
}
