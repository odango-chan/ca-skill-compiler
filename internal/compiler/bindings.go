package compiler

import (
	"encoding/json"
	"fmt"
	"math/big"
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
				stepKey := scope + "." + id
				if op := scalar(child(step, "call")); op != "" {
					stepOutputs[nodeID][stepKey] = a.tools[op].outputNode
				} else if rel := scalar(child(step, "script")); rel != "" {
					if b := a.files["skills/"+s.name+"/"+rel]; b != nil {
						if schema, err := parseScriptOutput(string(b)); err == nil {
							stepOutputs[nodeID][stepKey] = schemaNode(schema)
						}
					}
				}
			}
		}
	}
	// Procedure inputs are producer aliases in their own right. Validate every
	// declaration here so an unused input cannot hide a missing field or a
	// declared type that disagrees with its producer.
	inputs := child(proc, "inputs")
	for _, name := range mapKeys(inputs) {
		input := child(inputs, name)
		from := scalar(child(input, "from"))
		typ := scalar(child(input, "type"))
		if from == "" || !validProcedureInputType(typ) {
			continue // parseProcedure reports malformed declarations.
		}
		producer := a.resolveSource(s, proc, from, path, child(input, "from").Line, "", outputs, nil, stepOutputs)
		if producer != nil && scalar(child(producer, "type")) != typ && !(scalar(child(producer, "type")) == "integer" && typ == "number") {
			a.errorAt(path, input.Line, "BINDING_TYPE_MISMATCH", "Procedure input source type does not match declared type: "+name)
		}
	}
	for _, nodeID := range mapKeys(nodes) {
		n := child(nodes, nodeID)
		expected := (*yaml.Node)(nil)
		procedureInputs := false
		if op := scalar(child(n, "call")); op != "" {
			expected = a.tools[op].inputNode
		} else if ref := child(n, "procedure"); ref != nil {
			if target := a.calledProcedure(s, ref); target != nil {
				expected = procedureInputSchema(target)
				procedureInputs = true
			}
		}
		a.checkBindings(s, proc, path, child(n, "with"), expected, nodeID, outputs, stepOutputs[nodeID], stepOutputs, procedureInputs)
		for _, scope := range []string{"preconditions", "postconditions"} {
			for _, step := range sequence(child(child(n, scope), "steps")) {
				op := scalar(child(step, "call"))
				var target *yaml.Node
				if op != "" {
					target = a.tools[op].inputNode
				}
				a.checkBindings(s, proc, path, child(step, "with"), target, nodeID, outputs, stepOutputs[nodeID], stepOutputs, false)
			}
		}
	}
}

func (a *analysis) checkBindings(s *skillEntry, proc *yaml.Node, path string, with, expected *yaml.Node, nodeID string, outputs map[string]*yaml.Node, steps map[string]*yaml.Node, allSteps map[string]map[string]*yaml.Node, procedureInputs bool) {
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
		for _, field := range mapKeys(val) {
			if field != "from" && field != "value" {
				a.errorAt(path, child(val, field).Line, "BINDING_UNSUPPORTED", "unknown binding field: "+field)
			}
		}
		from := child(val, "from")
		literal := child(val, "value")
		if (from == nil) == (literal == nil) {
			a.errorAt(path, line, "BINDING_UNSUPPORTED", "binding requires exactly one of from or value")
			continue
		}
		var source *yaml.Node
		if from != nil {
			if scalar(from) == "" {
				a.errorAt(path, from.Line, "BINDING_UNSUPPORTED", "from binding must be a nonempty source reference")
				continue
			}
			source = a.resolveSource(s, proc, scalar(from), path, from.Line, nodeID, outputs, steps, allSteps)
			if expected == nil && source != nil && !projectionSchemaSupported(source) {
				a.errorAt(path, from.Line, "BINDING_UNSUPPORTED", "Producer Schema cannot be represented by the static input projection: "+scalar(from))
				source = nil
			}
		}
		if literal != nil && expected == nil && !validJSONLiteral(literal) {
			a.errorAt(path, literal.Line, "BINDING_UNSUPPORTED", "value binding must be a JSON literal")
			continue
		}
		if expected != nil {
			target := schemaProperty(expected, key)
			if target == nil {
				a.errorAt(path, line, "SOURCE_FIELD_UNKNOWN", "consumer input field is not declared: "+key)
				continue
			}
			if from != nil && source != nil {
				compatible := schemasCompatible(source, target)
				if procedureInputs {
					compatible = declaredTypeCompatible(source, target)
				}
				if !compatible {
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

func validJSONLiteral(node *yaml.Node) bool {
	node = unalias(node)
	if node == nil {
		return false
	}
	switch node.Kind {
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!str", "!!bool", "!!int", "!!float", "!!null":
			_, err := json.Marshal(nodeValue(node))
			return err == nil
		default:
			return false
		}
	case yaml.SequenceNode:
		for _, item := range node.Content {
			if !validJSONLiteral(item) {
				return false
			}
		}
		return true
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			if unalias(node.Content[i]).Tag != "!!str" || !validJSONLiteral(node.Content[i+1]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (a *analysis) calledProcedure(caller *skillEntry, ref *yaml.Node) *yaml.Node {
	name, procedure := caller.name, scalar(ref)
	if unalias(ref).Kind == yaml.MappingNode {
		if skill := scalar(child(ref, "skill")); skill != "" {
			name = skill
		}
		procedure = scalar(child(ref, "procedure"))
	}
	callee := a.skills[name]
	if callee == nil || callee.procedureAST == nil {
		return nil
	}
	return child(child(callee.procedureAST, "procedures"), procedure)
}

func procedureInputSchema(proc *yaml.Node) *yaml.Node {
	inputs := child(proc, "inputs")
	props := make(map[string]any)
	required := make([]any, 0)
	for _, name := range mapKeys(inputs) {
		typ := scalar(child(child(inputs, name), "type"))
		if !validProcedureInputType(typ) {
			continue
		}
		props[name] = map[string]any{"type": typ}
		required = append(required, name)
	}
	return schemaNode(map[string]any{"type": "object", "properties": props, "required": required})
}

func declaredTypeCompatible(source, target *yaml.Node) bool {
	sourceType, targetType := scalar(child(source, "type")), scalar(child(target, "type"))
	return schemaTypeCompatible(sourceType, targetType)
}

func schemaTypeCompatible(source, target string) bool {
	if source == "" || target == "" {
		return false
	}
	return source == target || (source == "integer" && target == "number")
}

func schemasCompatible(source, target *yaml.Node) bool {
	source, target = unalias(source), unalias(target)
	if source == nil || target == nil || !projectionSchemaSupported(source) || !projectionSchemaSupported(target) {
		return false
	}
	st, tt := scalar(child(source, "type")), scalar(child(target, "type"))
	if !schemaTypeCompatible(st, tt) {
		return false
	}
	if !schemaConstraintsCompatible(source, target) {
		return false
	}
	switch st {
	case "object":
		sp, tp := child(source, "properties"), child(target, "properties")
		sourceAdditional := child(source, "additionalProperties")
		targetAdditional := child(target, "additionalProperties")
		targetForbidsExtras := schemaBoolean(targetAdditional, false)
		sourceForbidsExtras := schemaBoolean(sourceAdditional, false)
		sourceHasAdditionalSchema := unalias(sourceAdditional) != nil && unalias(sourceAdditional).Kind == yaml.MappingNode
		targetHasAdditionalSchema := unalias(targetAdditional) != nil && unalias(targetAdditional).Kind == yaml.MappingNode
		if targetForbidsExtras && !sourceForbidsExtras {
			return false
		}
		if targetHasAdditionalSchema {
			if !sourceForbidsExtras && (!sourceHasAdditionalSchema || !schemasCompatible(sourceAdditional, targetAdditional)) {
				return false
			}
		}
		for _, name := range mapKeys(tp) {
			sprop, tprop := child(sp, name), child(tp, name)
			if sprop == nil {
				if containsScalar(sequence(child(target, "required")), name) {
					return false
				}
				if sourceForbidsExtras {
					continue
				}
				if !sourceHasAdditionalSchema || !schemasCompatible(sourceAdditional, tprop) {
					return false
				}
				continue
			}
			if !schemasCompatible(sprop, tprop) {
				return false
			}
		}
		for _, name := range mapKeys(sp) {
			if child(tp, name) == nil {
				if targetForbidsExtras {
					return false
				}
				if targetHasAdditionalSchema && !schemasCompatible(child(sp, name), targetAdditional) {
					return false
				}
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

func schemaBoolean(schema *yaml.Node, value bool) bool {
	schema = unalias(schema)
	return schema != nil && schema.Kind == yaml.ScalarNode && schema.Tag == "!!bool" && scalar(schema) == strconv.FormatBool(value)
}

func projectionSchemaSupported(schema *yaml.Node) bool {
	schema = unalias(schema)
	if schema == nil || !supportedStaticSchema(schema) {
		return false
	}
	typ := scalar(child(schema, "type"))
	switch typ {
	case "object":
		if properties := child(schema, "properties"); properties != nil {
			if unalias(properties).Kind != yaml.MappingNode {
				return false
			}
			for _, name := range mapKeys(properties) {
				if !projectionSchemaSupported(child(properties, name)) {
					return false
				}
			}
		}
		if additional := child(schema, "additionalProperties"); additional != nil {
			if unalias(additional).Kind == yaml.MappingNode && !projectionSchemaSupported(additional) {
				return false
			}
			if unalias(additional).Kind == yaml.ScalarNode && scalar(additional) != "true" && scalar(additional) != "false" {
				return false
			}
		}
		for _, required := range sequence(child(schema, "required")) {
			if scalar(required) == "" {
				return false
			}
		}
	case "array":
		items := child(schema, "items")
		if items == nil || !projectionSchemaSupported(items) {
			return false
		}
	case "string", "integer", "number", "boolean", "null":
	default:
		return false
	}
	return true
}

func schemaConstraintsCompatible(source, target *yaml.Node) bool {
	for _, key := range []string{"minLength", "minItems", "minProperties", "minimum"} {
		if t := schemaNumber(target, key); t != nil {
			s := schemaNumber(source, key)
			exclusive := schemaNumber(source, "exclusiveMinimum")
			if s == nil && (exclusive == nil || *exclusive < *t) || s != nil && *s < *t {
				return false
			}
		}
	}
	for _, key := range []string{"maxLength", "maxItems", "maxProperties", "maximum"} {
		if t := schemaNumber(target, key); t != nil {
			s := schemaNumber(source, key)
			exclusive := schemaNumber(source, "exclusiveMaximum")
			if s == nil && (exclusive == nil || *exclusive > *t) || s != nil && *s > *t {
				return false
			}
		}
	}
	if t := schemaNumber(target, "exclusiveMinimum"); t != nil {
		minimum, exclusive := schemaNumber(source, "minimum"), schemaNumber(source, "exclusiveMinimum")
		if minimum == nil && exclusive == nil || minimum != nil && *minimum <= *t || exclusive != nil && *exclusive <= *t {
			return false
		}
	}
	if t := schemaNumber(target, "exclusiveMaximum"); t != nil {
		maximum, exclusive := schemaNumber(source, "maximum"), schemaNumber(source, "exclusiveMaximum")
		if maximum == nil && exclusive == nil || maximum != nil && *maximum >= *t || exclusive != nil && *exclusive >= *t {
			return false
		}
	}
	if child(target, "multipleOf") != nil {
		s := schemaNumber(source, "multipleOf")
		if s == nil || !isIntegerMultiple(scalar(child(source, "multipleOf")), scalar(child(target, "multipleOf"))) {
			return false
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
	if value == nil || schema == nil || !validJSONLiteral(value) {
		return false
	}
	if schema.Kind == yaml.ScalarNode && schema.Tag == "!!bool" {
		return scalar(schema) == "true"
	}
	if !projectionSchemaSupported(schema) {
		return false
	}
	typ := scalar(child(schema, "type"))
	if typ == "" || !schemaTypeCompatible(literalType(value), typ) {
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
		if value.Kind != yaml.MappingNode {
			return false
		}
		seen := map[string]bool{}
		for i := 0; i+1 < len(value.Content); i += 2 {
			key, v := value.Content[i].Value, unalias(value.Content[i+1])
			target := child(props, key)
			if target == nil {
				additional := child(schema, "additionalProperties")
				if additional == nil || unalias(additional).Kind == yaml.ScalarNode && scalar(additional) == "true" {
					if !validJSONLiteral(v) {
						return false
					}
				} else if !literalMatchesSchema(v, additional) {
					return false
				}
			} else if !literalMatchesSchema(v, target) {
				return false
			}
			seen[key] = true
		}
		for _, req := range sequence(child(schema, "required")) {
			if !seen[scalar(req)] {
				return false
			}
		}
		if n, ok := numberValue(child(schema, "minProperties")); ok && float64(len(value.Content)/2) < n {
			return false
		}
		if n, ok := numberValue(child(schema, "maxProperties")); ok && float64(len(value.Content)/2) > n {
			return false
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
		if bound, ok := numberValue(child(schema, "exclusiveMinimum")); ok && n <= bound {
			return false
		}
		if bound, ok := numberValue(child(schema, "exclusiveMaximum")); ok && n >= bound {
			return false
		}
		if divisor := child(schema, "multipleOf"); divisor != nil && !isIntegerMultiple(value.Value, scalar(divisor)) {
			return false
		}
	}
	return true
}

func isIntegerMultiple(value, divisor string) bool {
	valueRat, valueOK := exactRational(value)
	divisorRat, divisorOK := exactRational(divisor)
	if !valueOK || !divisorOK || divisorRat.Sign() <= 0 {
		return false
	}
	quotient := new(big.Rat).Quo(valueRat, divisorRat)
	return quotient.IsInt()
}

func exactRational(raw string) (*big.Rat, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false
	}
	exponent := 0
	if i := strings.IndexAny(raw, "eE"); i >= 0 {
		parsed, err := strconv.Atoi(raw[i+1:])
		if err != nil || parsed < -10000 || parsed > 10000 {
			return nil, false
		}
		exponent, raw = parsed, raw[:i]
	}
	sign := ""
	if raw[0] == '-' || raw[0] == '+' {
		sign, raw = raw[:1], raw[1:]
	}
	parts := strings.Split(raw, ".")
	if len(parts) > 2 {
		return nil, false
	}
	whole, fraction := parts[0], ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	digits := whole + fraction
	scale := len(fraction) - exponent
	if digits == "" || len(digits) > 1024 || scale < -1024 || scale > 1024 {
		return nil, false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return nil, false
		}
	}
	numerator, ok := new(big.Int).SetString(digits, 10)
	if !ok {
		return nil, false
	}
	if sign == "-" {
		numerator.Neg(numerator)
	}
	denominator := big.NewInt(1)
	ten := big.NewInt(10)
	if scale > 0 {
		denominator.Exp(ten, big.NewInt(int64(scale)), nil)
	} else if scale < 0 {
		numerator.Mul(numerator, new(big.Int).Exp(ten, big.NewInt(int64(-scale)), nil))
	}
	return new(big.Rat).SetFrac(numerator, denominator), true
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
	known := map[string]bool{"$schema": true, "$id": true, "$ref": true, "title": true, "description": true, "type": true, "properties": true, "required": true, "additionalProperties": true, "items": true, "minLength": true, "maxLength": true, "pattern": true, "minimum": true, "maximum": true, "exclusiveMinimum": true, "exclusiveMaximum": true, "multipleOf": true, "minItems": true, "maxItems": true, "minProperties": true, "maxProperties": true, "uniqueItems": true, "const": true, "enum": true}
	for _, key := range mapKeys(schema) {
		if !known[key] || key == "$ref" {
			return false
		}
	}
	return true
}

func (a *analysis) resolveSource(s *skillEntry, proc *yaml.Node, ref, path string, line int, nodeID string, outputs, steps map[string]*yaml.Node, allSteps map[string]map[string]*yaml.Node) *yaml.Node {
	return a.resolveSourceWithInputStack(s, proc, ref, path, line, nodeID, outputs, steps, allSteps, map[string]bool{})
}

func (a *analysis) resolveSourceWithInputStack(s *skillEntry, proc *yaml.Node, ref, path string, line int, nodeID string, outputs, steps map[string]*yaml.Node, allSteps map[string]map[string]*yaml.Node, resolvingInputs map[string]bool) *yaml.Node {
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
		document := a.parseYAML(filename)
		if document == nil {
			return nil
		}
		expanded, err := a.expandSchemaRefs(filename, document, document)
		if err != nil {
			a.errorAt(filename, document.Line, "LIB_SCHEMA_INVALID", "cannot resolve Schema references: "+err.Error())
			return nil
		}
		if err := validateSchemaShape(expanded); err != nil {
			a.errorAt(filename, document.Line, "LIB_SCHEMA_INVALID", "invalid linked Schema: "+err.Error())
			return nil
		}
		schema = expanded
		offset = 2
	case "input":
		if resolvingInputs[parts[1]] {
			a.errorAt(path, line, "BINDING_UNRESOLVED", "cyclic Procedure input source: "+parts[1])
			return nil
		}
		inputs := child(proc, "inputs")
		field := child(inputs, parts[1])
		if field == nil {
			a.errorAt(path, line, "BINDING_UNRESOLVED", "Procedure input is not declared: "+parts[1])
			return nil
		}
		if from := child(field, "from"); from != nil {
			resolvingInputs[parts[1]] = true
			base := a.resolveSourceWithInputStack(s, proc, scalar(from), path, line, nodeID, outputs, steps, allSteps, resolvingInputs)
			delete(resolvingInputs, parts[1])
			for _, nested := range parts[2:] {
				base = schemaProperty(base, nested)
				if base == nil {
					a.errorAt(path, line, "SOURCE_FIELD_UNKNOWN", "Procedure input source has no field "+nested+": "+parts[1])
					return nil
				}
			}
			return base
		}
		a.errorAt(path, line, "BINDING_UNRESOLVED", "Procedure input has no resolvable Producer: "+parts[1])
		return nil
	case "preconditions", "postconditions":
		if len(parts) < 3 || parts[2] != "output" {
			break
		}
		schema = steps[parts[0]+"."+parts[1]]
		offset = 3
		if schema == nil {
			a.errorAt(path, line, "STEP_SCOPE_UNKNOWN", "unknown local "+parts[0]+" step: "+parts[1])
			return nil
		}
	default:
		if len(parts) >= 4 && (parts[1] == "preconditions" || parts[1] == "postconditions") && parts[3] == "output" {
			schema = allSteps[parts[0]][parts[1]+"."+parts[2]]
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
	case "!!null":
		return "null"
	}
	return ""
}
