package compiler

import (
	"regexp"

	"gopkg.in/yaml.v3"
)

const semanticFieldIDKeyword = "x-ca-semanticId"

var semanticFieldIDPattern = regexp.MustCompile(`^[a-z][A-Za-z0-9]*(\.[a-z][A-Za-z0-9]*)+$`)

// semanticFieldID reads an owner-declared identity annotation from one
// already-resolved Schema property. It never derives identity from a field name.
func semanticFieldID(schema *yaml.Node) (string, bool) {
	value := child(unalias(schema), semanticFieldIDKeyword)
	if value == nil {
		return "", false
	}
	value = unalias(value)
	if value.Kind != yaml.ScalarNode || value.Tag != "!!str" || !semanticFieldIDPattern.MatchString(value.Value) {
		return "", false
	}
	return value.Value, true
}

// addSemanticProcedureInputBindings generates only missing required Operation
// inputs from uniquely matching, owner-annotated Procedure input Producers.
// The returned map is an in-memory candidate overlay: checkBindings remains the
// single authority for source resolution, CFG availability and Schema proof.
func (a *analysis) addSemanticProcedureInputBindings(proc *yaml.Node, with, expected *yaml.Node, path, nodeID string) *yaml.Node {
	if expected == nil || unalias(expected).Kind != yaml.MappingNode {
		return with
	}
	inputSchemas := a.procedureInputSchemas[proc]
	if len(inputSchemas) == 0 {
		return with
	}

	existing := map[string]bool{}
	if with != nil && unalias(with).Kind == yaml.MappingNode {
		for i := 0; i+1 < len(with.Content); i += 2 {
			existing[scalar(with.Content[i])] = true
		}
	}

	type producer struct {
		name string
		schema *yaml.Node
	}
	producers := make(map[string][]producer)
	for _, name := range sortedStringMapKeys(inputSchemas) {
		schema := inputSchemas[name]
		id, ok := semanticFieldID(schema)
		if !ok {
			continue
		}
		producers[id] = append(producers[id], producer{name: name, schema: schema})
	}

	var generated *yaml.Node
	for _, required := range sequence(child(expected, "required")) {
		name := scalar(required)
		if name == "" || existing[name] {
			continue
		}
		target := schemaProperty(expected, name)
		id, ok := semanticFieldID(target)
		if !ok {
			continue
		}
		candidates := producers[id]
		if len(candidates) != 1 {
			// No candidate and ambiguous candidates both fail closed. The normal
			// required-input check emits BINDING_UNRESOLVED; no path is guessed.
			continue
		}
		candidate := candidates[0]
		if !schemasCompatible(candidate.schema, target) {
			// Let the canonical binding validator report the missing required
			// input. Never bypass its richer compatibility proof.
			continue
		}
		if generated == nil {
			generated = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			if with != nil {
				generated.Line, generated.Column = with.Line, with.Column
				generated.Content = append(generated.Content, with.Content...)
			}
		}
		generated.Content = append(generated.Content,
			stringNode(name),
			&yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
				stringNode("from"), stringNode("input." + candidate.name),
			}},
		)
		existing[name] = true
	}
	if generated != nil {
		return generated
	}
	return with
}
