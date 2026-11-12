package spike

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed fixtures/core-vectors.yaml
var coreFixture []byte

func LoadEmbeddedFixture() (Fixture, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(coreFixture, &doc); err != nil {
		return Fixture{}, err
	}
	v, err := YAMLJSON(&doc)
	if err != nil {
		return Fixture{}, err
	}
	m, ok := v.(map[string]any)
	if !ok {
		return Fixture{}, fmt.Errorf("embedded fixture root must be an object")
	}
	return parseFixture(m)
}
