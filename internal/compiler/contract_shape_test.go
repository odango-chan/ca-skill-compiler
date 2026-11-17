package compiler

// Contract-shape guard: public outputs must validate against the frozen machine
// schemas using the ADR-selected San engine.

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

func compileContractSchema(t *testing.T, path string) *jsonschema.Schema {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema %s: %v", path, err)
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("decode schema %s: %v", path, err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err := compiler.AddResource("urn:test:contract:v1", value); err != nil {
		t.Fatalf("register schema %s: %v", path, err)
	}
	schema, err := compiler.Compile("urn:test:contract:v1")
	if err != nil {
		t.Fatalf("compile schema %s: %v", path, err)
	}
	return schema
}

func validateContractShape(t *testing.T, schemaPath string, output any) {
	t.Helper()
	schema := compileContractSchema(t, schemaPath)
	raw, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}
	var instance any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&instance); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if err := schema.Validate(instance); err != nil {
		t.Fatalf("output violates %s: %v", schemaPath, err)
	}
}

func TestInspectionMatchesFrozenSchema(t *testing.T) {
	inspection := InspectDependencies(wp3aBaseFixture(t).request(t))
	validateContractShape(t, "../../contracts/v1alpha4/dependency-inspection.schema.json", inspection)
}

func TestBuildResultMatchesFrozenSchema(t *testing.T) {
	result := Build(wp3aBaseFixture(t).request(t))
	validateContractShape(t, "../../contracts/v1alpha4/build-result.schema.json", result)
}
