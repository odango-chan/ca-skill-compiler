package spike

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestYAMLNumbersRemainExactJSONNumbers(t *testing.T) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(`values:
  - 9007199254740993
  - 18446744073709551615
  - -18446744073709551616
  - 0.1
  - 0.3
  - 1e-7
  - 3.0e-1
  - 0xFF
  - 0o10
  - 0b10
`), &root); err != nil {
		t.Fatal(err)
	}
	value, err := YAMLJSON(&root)
	if err != nil {
		t.Fatal(err)
	}
	values := value.(map[string]any)["values"].([]any)
	want := []json.Number{"9007199254740993", "18446744073709551615", "-18446744073709551616", "0.1", "0.3", "1e-7", "3.0e-1", "255", "8", "2"}
	for i, expected := range want {
		got, ok := values[i].(json.Number)
		if !ok || got != expected {
			t.Fatalf("values[%d] = %#v, want json.Number(%q)", i, values[i], expected)
		}
	}
}

func TestYAMLToJSONRejectsNonJSONStructures(t *testing.T) {
	for name, input := range map[string]string{
		"duplicate key":  "a: 1\na: 2\n",
		"non-string key": "1: value\n",
		"timestamp":      "d: 2026-11-12\n",
		"custom tag":     "x: !thing value\n",
		"nan":            "x: .nan\n",
		"infinity":       "x: .inf\n",
	} {
		t.Run(name, func(t *testing.T) {
			var root yaml.Node
			if err := yaml.Unmarshal([]byte(input), &root); err != nil {
				t.Fatal(err)
			}
			if _, err := YAMLJSON(&root); err == nil {
				t.Fatal("expected JSON boundary rejection")
			}
		})
	}
}

func TestYAMLToJSONRejectsAliasCycle(t *testing.T) {
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	alias := &yaml.Node{Kind: yaml.AliasNode, Alias: seq}
	seq.Content = []*yaml.Node{alias}
	if _, err := YAMLJSON(seq); err == nil {
		t.Fatal("expected alias cycle rejection")
	}
}

func TestYAMLToJSONResolvesAcyclicAlias(t *testing.T) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte("base: &base {type: string}\ncopy: *base\n"), &root); err != nil {
		t.Fatal(err)
	}
	value, err := YAMLJSON(&root)
	if err != nil {
		t.Fatal(err)
	}
	m := value.(map[string]any)
	base := m["base"].(map[string]any)
	copyValue := m["copy"].(map[string]any)
	if base["type"] != "string" || copyValue["type"] != "string" {
		t.Fatalf("alias result = %#v", value)
	}
}

func TestOfficialSuiteCompatibilityFilter(t *testing.T) {
	tests := map[string]bool{
		"":           true,
		"3":          true,
		"6,<=7":      false,
		"3,<=4":      false,
		"3,7,<=2020": true,
		"=2020":      true,
		"<=2019":     false,
		"9999":       false,
	}
	for input, want := range tests {
		if got := compatible2020(input); got != want {
			t.Errorf("compatible2020(%q) = %v, want %v", input, got, want)
		}
	}
}
