package spike

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Fixture struct {
	Draft     string
	Resources map[string]any
	Cases     []Case
}

type Case struct {
	ID      string
	Root    string
	Mode    string
	Valid   []any
	Invalid []any
}

func LoadFixture(path string) (Fixture, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Fixture{}, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return Fixture{}, err
	}
	v, err := YAMLJSON(&doc)
	if err != nil {
		return Fixture{}, err
	}
	m, ok := v.(map[string]any)
	if !ok {
		return Fixture{}, fmt.Errorf("fixture root must be an object")
	}
	return parseFixture(m)
}

func parseFixture(m map[string]any) (Fixture, error) {
	draft, _ := m["draft"].(string)
	resources, ok := m["resources"].(map[string]any)
	if !ok {
		return Fixture{}, fmt.Errorf("fixture resources must be an object")
	}
	entries, ok := m["cases"].([]any)
	if !ok {
		return Fixture{}, fmt.Errorf("fixture cases must be an array")
	}
	f := Fixture{Draft: draft, Resources: resources}
	for i, raw := range entries {
		cm, ok := raw.(map[string]any)
		if !ok {
			return Fixture{}, fmt.Errorf("case %d must be an object", i)
		}
		c := Case{}
		c.ID, _ = cm["id"].(string)
		c.Root, _ = cm["root"].(string)
		c.Mode, _ = cm["mode"].(string)
		c.Valid, _ = cm["valid"].([]any)
		c.Invalid, _ = cm["invalid"].([]any)
		if c.ID == "" || c.Root == "" {
			return Fixture{}, fmt.Errorf("case %d requires id and root", i)
		}
		if _, exists := resources[c.Root]; !exists {
			return Fixture{}, fmt.Errorf("case %s root %s is not in registry", c.ID, c.Root)
		}
		f.Cases = append(f.Cases, c)
	}
	return f, nil
}
