package compiler

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func (a *analysis) validateProcedureBindings(s *skillEntry, proc, nodes *yaml.Node, path string) {
	outputs := map[string]*yaml.Node{}
	stepOutputs := map[string]map[string]*yaml.Node{}
	for _, nodeID := range mapKeys(nodes) {
		n := child(nodes, nodeID)
		if op := scalar(child(n, "call")); op != "" {
			outputs[nodeID] = a.tools[op].outputNode
		}
		stepOutputs[nodeID] = map[string]*yaml.Node{}
		for _, scope := range []string{"preconditions", "postconditions"} {
			for _, step := range sequence(child(child(n, scope), "steps")) {
				id := scalar(child(step, "name"))
				if op := scalar(child(step, "call")); op != "" {
					stepOutputs[nodeID][id] = a.tools[op].outputNode
				} else if rel := scalar(child(step, "script")); rel != "" {
					if b := a.files["skills/"+s.name+"/"+rel]; b != nil {
						if schema, err := parseScriptOutput(string(b)); err == nil {
							stepOutputs[nodeID][id] = schemaNode(schema)
						}
					}
				}
			}
		}
	}
	for _, nodeID := range mapKeys(nodes) {
		n := child(nodes, nodeID)
		expected := (*yaml.Node)(nil)
		if op := scalar(child(n, "call")); op != "" {
			expected = a.tools[op].inputNode
		}
		a.checkBindings(s, proc, path, child(n, "with"), expected, nodeID, outputs, stepOutputs[nodeID], stepOutputs)
		for _, scope := range []string{"preconditions", "postconditions"} {
			for _, step := range sequence(child(child(n, scope), "steps")) {
				op := scalar(child(step, "call"))
				var target *yaml.Node
				if op != "" {
					target = a.tools[op].inputNode
				}
				id := scalar(child(step, "name"))
				a.checkBindings(s, proc, path, child(step, "with"), target, nodeID, outputs, stepOutputs[nodeID], stepOutputs)
				_ = id
			}
		}
	}
}

func (a *analysis) checkBindings(s *skillEntry, proc *yaml.Node, path string, with, expected *yaml.Node, nodeID string, outputs map[string]*yaml.Node, steps map[string]*yaml.Node, allSteps map[string]map[string]*yaml.Node) {
	if with == nil {
		if expected != nil {
			for _, required := range sequence(child(expected, "required")) {
				a.errorAt(path, expected.Line, "BINDING_UNRESOLVED", "required consumer input is not bound: "+scalar(required))
			}
		}
		return
	}
	with = unalias(with)
	if with.Kind != yaml.MappingNode {
		a.errorAt(path, with.Line, "BINDING_UNSUPPORTED", "with must be a mapping")
		return
	}
	for i := 0; i+1 < len(with.Content); i += 2 {
		key, val := with.Content[i].Value, unalias(with.Content[i+1])
		line := val.Line
		from := child(val, "from")
		literal := child(val, "value")
		if (from == nil) == (literal == nil) {
			a.errorAt(path, line, "BINDING_UNSUPPORTED", "binding requires exactly one of from or value")
			continue
		}
		var source *yaml.Node
		if from != nil {
			source = a.resolveSource(s, proc, scalar(from), path, from.Line, nodeID, outputs, steps, allSteps)
		}
		if expected != nil {
			target := schemaProperty(expected, key)
			if target == nil {
				a.errorAt(path, line, "SOURCE_FIELD_UNKNOWN", "consumer input field is not declared: "+key)
				continue
			}
			if from != nil && source != nil {
				st, tt := scalar(child(source, "type")), scalar(child(target, "type"))
				if st != "" && tt != "" && st != tt {
					a.errorAt(path, line, "BINDING_TYPE_MISMATCH", fmt.Sprintf("%s produces %s but input requires %s", scalar(from), st, tt))
				}
			} else if literal != nil {
				lt := literalType(literal)
				tt := scalar(child(target, "type"))
				if lt != "" && tt != "" && lt != tt {
					a.errorAt(path, line, "BINDING_TYPE_MISMATCH", "literal binding type does not match input field "+key)
				}
			}
		}
	}
	if expected != nil {
		seen := map[string]bool{}
		for i := 0; i+1 < len(with.Content); i += 2 {
			seen[with.Content[i].Value] = true
		}
		for _, required := range sequence(child(expected, "required")) {
			name := scalar(required)
			if !seen[name] {
				a.errorAt(path, with.Line, "BINDING_UNRESOLVED", "required consumer input is not bound: "+name)
			}
		}
	}
}

func (a *analysis) resolveSource(s *skillEntry, proc *yaml.Node, ref, path string, line int, nodeID string, outputs, steps map[string]*yaml.Node, allSteps map[string]map[string]*yaml.Node) *yaml.Node {
	parts := strings.Split(ref, ".")
	if len(parts) < 2 {
		a.errorAt(path, line, "BINDING_UNSUPPORTED", "source reference must be a supported dotted path")
		return nil
	}
	var schema *yaml.Node
	offset := 0
	switch parts[0] {
	case "context":
		if len(parts) < 3 {
			break
		}
		category := parts[1]
		rel := ""
		if category == "request" {
			rel = s.requestSchema
		} else if category == "subject" || category == "facts" || category == "observations" {
			rel = s.contextSchemas[category]
			if rel == "" {
				a.errorAt(path, line, "EXTERNAL_SCHEMA_MISSING", "missing explicit Owner schema for context."+category)
				return nil
			}
		} else {
			break
		}
		if rel == "" {
			a.errorAt(path, line, "EXTERNAL_SCHEMA_MISSING", "missing explicit Skill requestSchema")
			return nil
		}
		filename := rel
		if !strings.HasPrefix(rel, "lib/") {
			filename = "skills/" + s.name + "/" + rel
		}
		if a.files[filename] == nil {
			a.errorAt(path, line, "EXTERNAL_SCHEMA_MISSING", "schema file is not imported: "+filename)
			return nil
		}
		schema = a.parseYAML(filename)
		offset = 2
	case "input":
		inputs := child(proc, "inputs")
		field := child(inputs, parts[1])
		if field == nil {
			a.errorAt(path, line, "BINDING_UNRESOLVED", "Procedure input is not declared: "+parts[1])
			return nil
		}
		if from := child(field, "from"); from != nil {
			return a.resolveSource(s, proc, scalar(from), path, line, nodeID, outputs, steps, allSteps)
		}
		a.errorAt(path, line, "BINDING_UNRESOLVED", "Procedure input has no resolvable Producer: "+parts[1])
		return nil
	case "preconditions":
		if len(parts) < 3 || parts[2] != "output" {
			break
		}
		schema = steps[parts[1]]
		offset = 3
		if schema == nil {
			a.errorAt(path, line, "STEP_SCOPE_UNKNOWN", "unknown local precondition step: "+parts[1])
			return nil
		}
	default:
		if len(parts) >= 4 && (parts[1] == "preconditions" || parts[1] == "postconditions") && parts[3] == "output" {
			schema = allSteps[parts[0]][parts[2]]
			offset = 4
			if schema == nil {
				a.errorAt(path, line, "STEP_SCOPE_UNKNOWN", "unknown nested step reference: "+strings.Join(parts[:3], "."))
				return nil
			}
		} else if len(parts) >= 2 && parts[1] == "output" {
			schema = outputs[parts[0]]
			offset = 2
			if schema == nil {
				a.errorAt(path, line, "BINDING_UNRESOLVED", "unknown node output Producer: "+parts[0])
				return nil
			}
		} else {
			a.errorAt(path, line, "BINDING_UNSUPPORTED", "unsupported source reference: "+ref)
			return nil
		}
	}
	if schema == nil {
		a.errorAt(path, line, "BINDING_UNSUPPORTED", "unsupported source reference: "+ref)
		return nil
	}
	for _, field := range parts[offset:] {
		schema = schemaProperty(schema, field)
		if schema == nil {
			a.errorAt(path, line, "SOURCE_FIELD_UNKNOWN", "Producer schema has no field "+field+" in "+ref)
			return nil
		}
	}
	return schema
}

func schemaNode(v any) *yaml.Node {
	b, e := yaml.Marshal(v)
	if e != nil {
		return nil
	}
	var doc yaml.Node
	if yaml.Unmarshal(b, &doc) != nil || len(doc.Content) == 0 {
		return nil
	}
	return doc.Content[0]
}
func schemaProperty(schema *yaml.Node, name string) *yaml.Node {
	schema = unalias(schema)
	if schema == nil {
		return nil
	}
	if scalar(child(schema, "$ref")) != "" {
		return nil
	}
	return child(child(schema, "properties"), name)
}
func literalType(n *yaml.Node) string {
	n = unalias(n)
	if n == nil {
		return ""
	}
	switch n.Tag {
	case "!!str":
		return "string"
	case "!!bool":
		return "boolean"
	case "!!int":
		return "integer"
	case "!!float":
		return "number"
	case "!!seq":
		return "array"
	case "!!map":
		return "object"
	}
	return ""
}
