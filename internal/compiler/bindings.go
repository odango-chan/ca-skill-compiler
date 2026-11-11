package compiler

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
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
		} else if rel := scalar(child(n, "script")); rel != "" {
			if b := a.files["skills/"+s.name+"/"+rel]; b != nil {
				if schema, err := parseScriptOutput(string(b)); err == nil {
					outputs[nodeID] = schemaNode(schema)
				}
			}
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
				if !schemasCompatible(source, target) {
					a.errorAt(path, line, "BINDING_TYPE_MISMATCH", fmt.Sprintf("Producer schema %s is not compatible with input %s", scalar(from), key))
				}
			} else if literal != nil {
				if !literalMatchesSchema(literal, target) {
					a.errorAt(path, line, "BINDING_TYPE_MISMATCH", "literal binding does not satisfy input field "+key)
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

func schemasCompatible(source, target *yaml.Node) bool {
	source, target = unalias(source), unalias(target)
	if source == nil || target == nil || !supportedStaticSchema(source) || !supportedStaticSchema(target) {
		return false
	}
	st, tt := scalar(child(source, "type")), scalar(child(target, "type"))
	if st == "" || tt == "" || st != tt {
		return false
	}
	if !schemaConstraintsCompatible(source, target) {
		return false
	}
	switch st {
	case "object":
		sp, tp := child(source, "properties"), child(target, "properties")
		if sp == nil || tp == nil {
			return false
		}
		if scalar(child(target, "additionalProperties")) == "false" && scalar(child(source, "additionalProperties")) != "false" {
			return false
		}
		for _, name := range mapKeys(tp) {
			sprop, tprop := child(sp, name), child(tp, name)
			if sprop == nil {
				if containsScalar(sequence(child(target, "required")), name) {
					return false
				}
				continue
			}
			if !schemasCompatible(sprop, tprop) {
				return false
			}
		}
		for _, name := range mapKeys(sp) {
			if child(tp, name) == nil && scalar(child(target, "additionalProperties")) == "false" {
				return false
			}
		}
		for _, req := range sequence(child(target, "required")) {
			name := scalar(req)
			if child(sp, name) == nil || !containsScalar(sequence(child(source, "required")), name) {
				return false
			}
		}
	case "array":
		return schemasCompatible(child(source, "items"), child(target, "items"))
	}
	return true
}

func schemaConstraintsCompatible(source, target *yaml.Node) bool {
	for _, key := range []string{"minLength", "minItems", "minimum"} {
		if t := schemaNumber(target, key); t != nil {
			s := schemaNumber(source, key)
			if s == nil || *s < *t {
				return false
			}
		}
	}
	for _, key := range []string{"maxLength", "maxItems", "maximum"} {
		if t := schemaNumber(target, key); t != nil {
			s := schemaNumber(source, key)
			if s == nil || *s > *t {
				return false
			}
		}
	}
	if p := scalar(child(target, "pattern")); p != "" && scalar(child(source, "pattern")) != p {
		return false
	}
	if scalar(child(target, "uniqueItems")) == "true" && scalar(child(source, "uniqueItems")) != "true" {
		return false
	}
	if c := child(target, "const"); c != nil {
		sc := child(source, "const")
		if sc == nil || !reflect.DeepEqual(nodeValue(sc), nodeValue(c)) {
			return false
		}
	}
	if enum := sequence(child(target, "enum")); len(enum) > 0 {
		values := sequence(child(source, "enum"))
		if len(values) == 0 {
			if c := child(source, "const"); c != nil {
				values = []*yaml.Node{c}
			} else {
				return false
			}
		}
		for _, v := range values {
			found := false
			for _, allowed := range enum {
				if reflect.DeepEqual(nodeValue(v), nodeValue(allowed)) {
					found = true
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

func schemaNumber(schema *yaml.Node, key string) *float64 {
	n := child(schema, key)
	if n == nil {
		return nil
	}
	v, err := strconv.ParseFloat(scalar(n), 64)
	if err != nil {
		return nil
	}
	return &v
}

func literalMatchesSchema(value, schema *yaml.Node) bool {
	value, schema = unalias(value), unalias(schema)
	if value == nil || schema == nil || !supportedStaticSchema(schema) {
		return false
	}
	typ := scalar(child(schema, "type"))
	if typ == "" || literalType(value) != typ {
		return false
	}
	if c := child(schema, "const"); c != nil && !reflect.DeepEqual(nodeValue(c), nodeValue(value)) {
		return false
	}
	if enum := sequence(child(schema, "enum")); len(enum) > 0 {
		matched := false
		for _, v := range enum {
			if reflect.DeepEqual(nodeValue(v), nodeValue(value)) {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}
	switch typ {
	case "object":
		props := child(schema, "properties")
		if props == nil || value.Kind != yaml.MappingNode {
			return false
		}
		seen := map[string]bool{}
		for i := 0; i+1 < len(value.Content); i += 2 {
			key, v := value.Content[i].Value, unalias(value.Content[i+1])
			target := child(props, key)
			if target == nil {
				return false
			}
			seen[key] = true
			if !literalMatchesSchema(v, target) {
				return false
			}
		}
		for _, req := range sequence(child(schema, "required")) {
			if !seen[scalar(req)] {
				return false
			}
		}
	case "array":
		items := child(schema, "items")
		if items == nil || value.Kind != yaml.SequenceNode {
			return false
		}
		if n, ok := numberValue(child(schema, "minItems")); ok && float64(len(value.Content)) < n {
			return false
		}
		if n, ok := numberValue(child(schema, "maxItems")); ok && float64(len(value.Content)) > n {
			return false
		}
		if scalar(child(schema, "uniqueItems")) == "true" {
			seen := map[string]bool{}
			for _, item := range value.Content {
				b, err := json.Marshal(nodeValue(item))
				if err != nil || seen[string(b)] {
					return false
				}
				seen[string(b)] = true
			}
		}
		for _, v := range value.Content {
			if !literalMatchesSchema(v, items) {
				return false
			}
		}
	case "string":
		text := value.Value
		if n, ok := numberValue(child(schema, "minLength")); ok && float64(len([]rune(text))) < n {
			return false
		}
		if n, ok := numberValue(child(schema, "maxLength")); ok && float64(len([]rune(text))) > n {
			return false
		}
		if pattern := scalar(child(schema, "pattern")); pattern != "" {
			re, err := regexp.Compile(pattern)
			if err != nil || !re.MatchString(text) {
				return false
			}
		}
	case "integer", "number":
		n, err := strconv.ParseFloat(value.Value, 64)
		if err != nil {
			return false
		}
		if bound, ok := numberValue(child(schema, "minimum")); ok && n < bound {
			return false
		}
		if bound, ok := numberValue(child(schema, "maximum")); ok && n > bound {
			return false
		}
	}
	return true
}

func numberValue(n *yaml.Node) (float64, bool) {
	if n == nil {
		return 0, false
	}
	v, err := strconv.ParseFloat(scalar(n), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}
func containsScalar(nodes []*yaml.Node, value string) bool {
	for _, n := range nodes {
		if scalar(n) == value {
			return true
		}
	}
	return false
}

func supportedStaticSchema(schema *yaml.Node) bool {
	schema = unalias(schema)
	if schema == nil || schema.Kind != yaml.MappingNode {
		return false
	}
	known := map[string]bool{"$schema": true, "$id": true, "$ref": true, "title": true, "description": true, "type": true, "properties": true, "required": true, "additionalProperties": true, "items": true, "minLength": true, "maxLength": true, "pattern": true, "minimum": true, "maximum": true, "minItems": true, "maxItems": true, "uniqueItems": true, "const": true, "enum": true}
	for _, key := range mapKeys(schema) {
		if !known[key] || key == "$ref" {
			return false
		}
	}
	return true
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
			source := scalar(from)
			if len(parts) > 2 {
				source += "." + strings.Join(parts[2:], ".")
			}
			return a.resolveSource(s, proc, source, path, line, nodeID, outputs, steps, allSteps)
		}
		a.errorAt(path, line, "BINDING_UNRESOLVED", "Procedure input has no resolvable Producer: "+parts[1])
		return nil
	case "preconditions", "postconditions":
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
