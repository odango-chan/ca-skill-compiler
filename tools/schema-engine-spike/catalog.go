package spike

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type NamedSchema struct {
	Name   string
	URI    string
	Schema any
}

type CatalogSchemas struct {
	Operations int
	Common     int
	Schemas    []NamedSchema
}

func LoadToolCatalogSchemas(path string) (CatalogSchemas, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return CatalogSchemas{}, err
	}
	var root yaml.Node
	if err := yaml.Unmarshal(b, &root); err != nil {
		return CatalogSchemas{}, err
	}
	value, err := YAMLJSON(&root)
	if err != nil {
		return CatalogSchemas{}, err
	}
	doc, ok := value.(map[string]any)
	if !ok {
		return CatalogSchemas{}, fmt.Errorf("ToolCatalog root must be an object")
	}
	result := CatalogSchemas{}
	common, _ := doc["x-common"].(map[string]any)
	for name, schema := range common {
		result.Common++
		result.Schemas = append(result.Schemas, NamedSchema{Name: "x-common." + name, URI: "https://catalog.spike.invalid/common/" + name + ".json", Schema: schema})
	}
	spec, _ := doc["spec"].(map[string]any)
	contracts, _ := spec["contracts"].([]any)
	for _, raw := range contracts {
		contract, _ := raw.(map[string]any)
		operation, _ := contract["operation"].(string)
		if operation == "" {
			return CatalogSchemas{}, fmt.Errorf("ToolCatalog operation is missing a string identity")
		}
		result.Operations++
		for _, field := range []string{"inputSchema", "outputSchema", "reconcileOutputSchema"} {
			schema, exists := contract[field]
			if !exists {
				continue
			}
			result.Schemas = append(result.Schemas, NamedSchema{Name: operation + "." + field, URI: "https://catalog.spike.invalid/operations/" + operation + "/" + field + ".json", Schema: schema})
		}
	}
	return result, nil
}
