package compiler

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func (a *analysis) validateProcedureBindings(s *skillEntry, proc, nodes *yaml.Node, path string) {
	parsedProcedure := s.procedureASTFor(proc)
	outputs, stepOutputs := a.procedureProducerSchemas(s, proc, nodes, parsedProcedure)
	for _, nodeID := range mapKeys(nodes) {
		n := child(nodes, nodeID)
		parsedNode := ProcedureNodeAST{}
		if parsedProcedure != nil {
			parsedNode = parsedProcedure.Nodes[nodeID]
		}
		expected := (*yaml.Node)(nil)
		if parsedNode.Action == ActionCall {
			op := parsedNode.Reference
			expected = a.tools[op].inputNode
		} else if parsedNode.Action == ActionProcedure {
			ref := child(n, "procedure")
			if target := a.calledProcedure(s, ref); target != nil {
				expected = procedureInputSchema(target, a.procedureInputSchemas[target])
			}
		} else if parsedNode.Action == ActionVerify && child(n, "with") != nil {
			expected = a.contextConsumerSchema(s, "subject", path, child(n, "with").Line)
		}
		a.checkBindings(s, proc, path, child(n, "with"), expected, nodeID, "main", -1, outputs, stepOutputs[nodeID], stepOutputs)
		for _, scope := range []string{"preconditions", "postconditions"} {
			for stepIndex, step := range sequence(child(child(n, scope), "steps")) {
				op := scalar(child(step, "call"))
				var target *yaml.Node
				if op != "" {
					target = a.tools[op].inputNode
				} else if child(step, "verify") != nil && child(step, "with") != nil {
					target = a.contextConsumerSchema(s, "subject", path, child(step, "with").Line)
				}
				a.checkBindings(s, proc, path, child(step, "with"), target, nodeID, scope, stepIndex, outputs, stepOutputs[nodeID], stepOutputs)
			}
		}
	}
}

func (a *analysis) procedureProducerSchemas(s *skillEntry, proc, nodes *yaml.Node, parsedProcedure *ProcedureAST) (map[string]*yaml.Node, map[string]map[string]*yaml.Node) {
	outputs := map[string]*yaml.Node{}
	stepOutputs := map[string]map[string]*yaml.Node{}
	for _, nodeID := range mapKeys(nodes) {
		parsedNode := ProcedureNodeAST{}
		if parsedProcedure != nil {
			parsedNode = parsedProcedure.Nodes[nodeID]
		}
		switch parsedNode.Action {
		case ActionCall:
			op := parsedNode.Reference
			outputs[nodeID] = a.tools[op].outputNode
		case ActionScript:
			rel := parsedNode.Reference
			if b := a.files["skills/"+s.name+"/"+rel]; b != nil {
				if schema, err := parseScriptOutput(string(b)); err == nil {
					outputs[nodeID] = schemaNode(schema)
				}
			}
		case ActionReconcile:
			targetID := parsedNode.Reference
			if operation := scalar(child(child(nodes, targetID), "call")); operation != "" {
				outputs[nodeID] = a.tools[operation].reconcileOutputNode
			}
		}
		stepOutputs[nodeID] = map[string]*yaml.Node{}
		if parsedProcedure != nil {
			for _, scope := range []struct {
				name  string
				steps []ProcedureStepAST
			}{{"preconditions", parsedNode.Preconditions}, {"postconditions", parsedNode.Postconditions}} {
				for _, step := range scope.steps {
					id := step.Name
					if id == "" {
						continue
					}
					stepKey := scope.name + "." + id
					if step.Action == ActionCall {
						op := step.Reference
						stepOutputs[nodeID][stepKey] = a.tools[op].outputNode
					} else if step.Action == ActionScript {
						rel := step.Reference
						if b := a.files["skills/"+s.name+"/"+rel]; b != nil {
							if schema, err := parseScriptOutput(string(b)); err == nil {
								stepOutputs[nodeID][stepKey] = schemaNode(schema)
							}
						}
					}
				}
			}
		}
	}
	return outputs, stepOutputs
}

func (a *analysis) prepareProcedureInputSchemas(s *skillEntry, proc *yaml.Node) {
	parsedProcedure := s.procedureASTFor(proc)
	if parsedProcedure == nil {
		return
	}
	nodes := child(proc, "nodes")
	outputs, stepOutputs := a.procedureProducerSchemas(s, proc, nodes, parsedProcedure)
	resolved := make(map[string]*yaml.Node, len(parsedProcedure.Inputs))
	for _, name := range sortedStringMapKeys(parsedProcedure.Inputs) {
		input := parsedProcedure.Inputs[name]
		if input.From == "" || input.Type != "" && !validProcedureInputType(input.Type) {
			continue
		}
		producer := a.resolveSource(s, proc, input.From, s.procedure, input.Span.Line, "", outputs, nil, stepOutputs)
		resolved[name] = producer
		if producer == nil || input.Type == "" {
			continue
		}
		types, ok := schemaTypeNames(child(producer, "type"))
		if !ok {
			a.errorAt(s.procedure, input.Span.Line, "BINDING_UNSUPPORTED", "Procedure input type assertion cannot be checked against an untyped Producer Schema: "+name)
		} else if !schemaTypeSetsCompatible(types, []string{input.Type}) {
			a.errorAt(s.procedure, input.Span.Line, "BINDING_TYPE_MISMATCH", "Procedure input source type does not match declared type: "+name)
		}
	}
	a.procedureInputSchemas[proc] = resolved
}

func (a *analysis) contextConsumerSchema(s *skillEntry, category, diagnosticPath string, line int) *yaml.Node {
	if s.ast.Contract == nil {
		a.errorAt(diagnosticPath, line, "EXTERNAL_SCHEMA_MISSING", "missing parsed SkillContract for Predicate Evaluation Context")
		return nil
	}
	rel := s.ast.Contract.ContextSchemas[category]
	if rel == "" {
		a.errorAt(diagnosticPath, line, "EXTERNAL_SCHEMA_MISSING", "Predicate input projection requires context.schemas."+category)
		return nil
	}
	if a.files[rel] == nil {
		a.errorAt(diagnosticPath, line, "EXTERNAL_SCHEMA_MISSING", "Predicate consumer Owner Schema is not imported: "+rel)
		return nil
	}
	document := a.parseYAML(rel)
	if document == nil {
		return nil
	}
	expanded, err := a.expandSchemaRefs(rel, document, document)
	if err != nil {
		a.errorAt(rel, document.Line, "LIB_SCHEMA_INVALID", "cannot resolve Predicate context Schema references: "+err.Error())
		return nil
	}
	if err := validateSchemaShape(expanded); err != nil {
		a.errorAt(rel, document.Line, "LIB_SCHEMA_INVALID", "invalid Predicate context Schema: "+err.Error())
		return nil
	}
	return expanded
}

func (a *analysis) checkBindings(s *skillEntry, proc *yaml.Node, path string, with, expected *yaml.Node, nodeID, phase string, stepIndex int, outputs map[string]*yaml.Node, steps map[string]*yaml.Node, allSteps map[string]map[string]*yaml.Node) {
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
		keyNode := unalias(with.Content[i])
		key := keyNode.Value
		val := unalias(with.Content[i+1])
		if keyNode.Kind != yaml.ScalarNode || keyNode.Tag != "!!str" || key == "" {
			a.errorAt(path, keyNode.Line, "BINDING_UNSUPPORTED", "with binding field names must be nonempty strings")
			continue
		}
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
			if unalias(from).Kind != yaml.ScalarNode || unalias(from).Tag != "!!str" || scalar(from) == "" {
				a.errorAt(path, from.Line, "BINDING_UNSUPPORTED", "from binding must be a nonempty source reference")
				continue
			}
			source = a.resolveSource(s, proc, scalar(from), path, from.Line, nodeID, outputs, steps, allSteps)
			if source != nil {
				if available, reason := bindingSourceAvailable(proc, scalar(from), nodeID, phase, stepIndex); !available {
					a.errorAt(path, from.Line, "BINDING_SOURCE_UNAVAILABLE", reason)
					source = nil
				}
			}
			// Procedure Script inputs have no separately authored consumer
			// Schema: the with: keys define the invocation projection, and the
			// exact Producer path above is the source contract. When an action
			// has no available consumer Schema, do not reject a valid Producer
			// solely because its value constraints are outside the compatibility
			// checker; there is no consumer constraint to compare here. A missing
			// authoritative Owner Schema is diagnosed independently.
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
				if !projectionSchemaSupported(source) || !projectionSchemaSupported(target) {
					a.errorAt(path, line, "BINDING_UNSUPPORTED", "Producer or consumer Schema uses constraints unsupported by static binding validation: "+scalar(from))
					continue
				}
				if schemaImplicationUnsupported(source, target) {
					a.errorAt(path, line, "BINDING_UNSUPPORTED", "Producer-to-consumer Schema implication requires unsupported string-pattern analysis: "+scalar(from))
					continue
				}
				if !schemasCompatible(source, target) {
					a.errorAt(path, line, "BINDING_TYPE_MISMATCH", fmt.Sprintf("Producer schema %s is not compatible with input %s", scalar(from), key))
				}
			} else if literal != nil {
				if !projectionSchemaSupported(target) {
					a.errorAt(path, line, "BINDING_UNSUPPORTED", "consumer Schema uses constraints unsupported by static literal validation: "+key)
					continue
				}
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

// schemaImplicationUnsupported separates an unproved regular-expression
// implication from a proven type/constraint mismatch. Draft 2020-12 patterns
// are not a proof engine for minLength/maxLength or another pattern.
func schemaImplicationUnsupported(source, target *yaml.Node) bool {
	source, target = unalias(source), unalias(target)
	if source == nil || target == nil || source.Kind != yaml.MappingNode || target.Kind != yaml.MappingNode {
		return false
	}
	sourceTypes, sourceOK := schemaTypeNames(child(source, "type"))
	targetTypes, targetOK := schemaTypeNames(child(target, "type"))
	targetMayBeString := !targetOK || schemaTypeSetHas(targetTypes, "string")
	if sourceOK && targetMayBeString && schemaTypeSetHas(sourceTypes, "string") {
		sourcePattern, targetPattern := scalar(child(source, "pattern")), scalar(child(target, "pattern"))
		if targetPattern != "" && sourcePattern != targetPattern {
			return true
		}
		if sourcePattern != "" {
			if targetMin := schemaNumber(target, "minLength"); targetMin != nil {
				sourceMin := schemaNumber(source, "minLength")
				if sourceMin == nil || sourceMin.Cmp(targetMin) < 0 {
					return true
				}
			}
			if targetMax := schemaNumber(target, "maxLength"); targetMax != nil {
				sourceMax := schemaNumber(source, "maxLength")
				if sourceMax == nil || sourceMax.Cmp(targetMax) > 0 {
					return true
				}
			}
		}
	}
	for _, name := range mapKeys(child(target, "properties")) {
		if sourceProp := child(child(source, "properties"), name); sourceProp != nil && schemaImplicationUnsupported(sourceProp, child(child(target, "properties"), name)) {
			return true
		}
	}
	return false
}

// nodeOutputDominates reports whether every declared control-flow path from
// Procedure.start to consumer visits producer first. Source map order is not
// execution order; only explicit node outcome edges participate.
func nodeOutputDominates(proc *yaml.Node, producer, consumer string) bool {
	nodes := child(proc, "nodes")
	if child(nodes, producer) == nil || child(nodes, consumer) == nil {
		return false
	}
	start := scalar(child(proc, "start"))
	if producer == start {
		return true
	}
	if producer == consumer || start == "" {
		return false
	}
	// Search for any path to consumer that avoids producer. A cycle-safe DFS
	// is sufficient for this safety property and avoids relying on map order.
	stack := []string{start}
	visited := map[string]bool{}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == producer || visited[current] {
			continue
		}
		if current == consumer {
			return false
		}
		visited[current] = true
		stack = appendOutcomeTargets(stack, child(nodes, current), visited)
	}
	// If consumer is unreachable even without excluding producer, do not claim
	// a control-flow ordering. (Normally caught by the graph validator too.)
	stack = []string{start}
	visited = map[string]bool{}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == consumer {
			return true
		}
		if visited[current] {
			continue
		}
		visited[current] = true
		stack = appendOutcomeTargets(stack, child(nodes, current), nil)
	}
	return false
}

func appendOutcomeTargets(stack []string, node *yaml.Node, visited map[string]bool) []string {
	for _, on := range []*yaml.Node{child(node, "on"), child(child(node, "preconditions"), "on"), child(child(node, "postconditions"), "on")} {
		if on == nil || unalias(on).Kind != yaml.MappingNode {
			continue
		}
		for i := 1; i < len(unalias(on).Content); i += 2 {
			target := scalar(unalias(on).Content[i])
			if target != "" && (visited == nil || !visited[target]) {
				stack = append(stack, target)
			}
		}
	}
	return stack
}

func bindingSourceAvailable(proc *yaml.Node, ref, consumerNode, consumerPhase string, consumerStep int) (bool, string) {
	return bindingSourceAvailableThroughInputs(proc, ref, consumerNode, consumerPhase, consumerStep, map[string]bool{})
}

func bindingSourceAvailableThroughInputs(proc *yaml.Node, ref, consumerNode, consumerPhase string, consumerStep int, resolvingInputs map[string]bool) (bool, string) {
	parts := strings.Split(ref, ".")
	if len(parts) >= 2 && parts[0] == "input" {
		name := parts[1]
		input := child(child(proc, "inputs"), name)
		if input == nil {
			return true, "" // The source resolver reports an undeclared Procedure input.
		}
		from := scalar(child(input, "from"))
		if from == "" || resolvingInputs[name] {
			return true, "" // Missing and cyclic aliases are diagnosed by source resolution.
		}
		resolvingInputs[name] = true
		return bindingSourceAvailableThroughInputs(proc, from, consumerNode, consumerPhase, consumerStep, resolvingInputs)
	}
	sourceNode, sourcePhase, sourceStep := "", "", -1
	if len(parts) >= 2 && parts[1] == "output" {
		sourceNode = parts[0]
	} else if len(parts) >= 4 && (parts[1] == "preconditions" || parts[1] == "postconditions") && parts[3] == "output" {
		sourceNode, sourcePhase = parts[0], parts[1]
		sourceStep = namedStepIndex(proc, sourceNode, sourcePhase, parts[2])
	} else if len(parts) >= 3 && (parts[0] == "preconditions" || parts[0] == "postconditions") && parts[2] == "output" {
		sourceNode, sourcePhase = consumerNode, parts[0]
		sourceStep = namedStepIndex(proc, sourceNode, sourcePhase, parts[1])
	}
	if sourceNode == "" || (sourcePhase != "" && sourceStep < 0) {
		return true, "" // Other source errors are reported by the reference resolver.
	}
	if sourceNode != consumerNode {
		if nodeOutputDominates(proc, sourceNode, consumerNode) {
			return true, ""
		}
		return false, "producer node output is not available on every path to this consumer: " + sourceNode
	}
	if sourcePhase == "" {
		if consumerPhase == "postconditions" {
			return true, ""
		}
		return false, "a node output is available only after its main action, in postconditions or a later node"
	}
	if sourcePhase == "postconditions" {
		if consumerPhase == "postconditions" && sourceStep < consumerStep {
			return true, ""
		}
		return false, "postcondition step output is available only to later postcondition steps or later nodes"
	}
	if consumerPhase == "main" || consumerPhase == "postconditions" || (consumerPhase == "preconditions" && sourceStep < consumerStep) {
		return true, ""
	}
	return false, "precondition step output is available only to later steps, the node action, postconditions, or later nodes"
}

func namedStepIndex(proc *yaml.Node, nodeID, phase, name string) int {
	node := child(child(proc, "nodes"), nodeID)
	for index, step := range sequence(child(child(node, phase), "steps")) {
		if scalar(child(step, "name")) == name {
			return index
		}
	}
	return -1
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

func procedureInputSchema(proc *yaml.Node, sources map[string]*yaml.Node) *yaml.Node {
	inputs := child(proc, "inputs")
	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	root.Content = append(root.Content, stringNode("type"), stringNode("object"))
	properties := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	required := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, name := range mapKeys(inputs) {
		producer := sources[name]
		if producer == nil {
			// A declared Procedure input type is an assertion about its resolved
			// Producer, never a substitute contract. Keep an unresolved Producer
			// unconstrained here; the missing source is diagnosed independently.
			producer = schemaNode(map[string]any{})
		}
		properties.Content = append(properties.Content, stringNode(name), producer)
		required.Content = append(required.Content, stringNode(name))
	}
	root.Content = append(root.Content, stringNode("properties"), properties, stringNode("required"), required)
	return root
}

func stringNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func schemaTypeCompatible(source, target string) bool {
	if source == "" || target == "" {
		return false
	}
	return source == target || (source == "integer" && target == "number")
}

func schemasCompatible(source, target *yaml.Node) bool {
	source, target = unalias(source), unalias(target)
	if source == nil || target == nil {
		return false
	}
	// Boolean Draft 2020-12 Schemas denote the universal set (true) and the
	// empty set (false), respectively.
	if schemaBoolean(source, false) {
		return true
	}
	if schemaBoolean(target, true) {
		return true
	}
	if schemaBoolean(source, true) {
		return schemaIsUnconstrained(target)
	}
	if schemaBoolean(target, false) {
		return false
	}
	// An unconstrained JSON Schema accepts every JSON value. Imported catalogs
	// commonly use this for optional extension fields; requiring a type
	// annotation there would reject safe Producer-to-Consumer assignments.
	if schemaIsUnconstrained(target) {
		return true
	}
	if schemaIsUnconstrained(source) || !projectionSchemaSupported(source) || !projectionSchemaSupported(target) {
		return false
	}
	if values, finite := finiteSchemaValues(source); finite {
		for _, value := range values {
			if literalMatchesSchema(value, source) && !literalMatchesSchema(value, target) {
				return false
			}
		}
		return true
	}
	sourceTypes, sourceOK := schemaTypeNames(child(source, "type"))
	targetTypes, targetOK := schemaTypeNames(child(target, "type"))
	if !sourceOK {
		return false
	}
	if !targetOK {
		// An omitted target type leaves type-specific assertions in force only
		// for Producers of that instance type. Compare those assertions against
		// each explicit Producer type below.
		targetTypes = sourceTypes
	} else if !schemaTypeSetsCompatible(sourceTypes, targetTypes) {
		return false
	}
	for _, sourceType := range sourceTypes {
		if !schemaConstraintsCompatible(source, target, sourceType) {
			return false
		}
	}
	if schemaTypeSetHas(sourceTypes, "object") {
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
	}
	if schemaTypeSetHas(sourceTypes, "array") {
		sourceItems, targetItems := child(source, "items"), child(target, "items")
		// Draft 2020-12 defaults an omitted `items` keyword to true. A
		// Producer with unconstrained items cannot prove a typed Consumer,
		// while any Producer item contract satisfies a Consumer that omits it.
		if sourceItems == nil {
			return targetItems == nil || schemaIsUnconstrained(targetItems)
		}
		if targetItems == nil {
			return true
		}
		return schemasCompatible(sourceItems, targetItems)
	}
	return true
}

func finiteSchemaValues(schema *yaml.Node) ([]*yaml.Node, bool) {
	if constant := child(schema, "const"); constant != nil {
		return []*yaml.Node{constant}, true
	}
	if values := child(schema, "enum"); values != nil {
		return sequence(values), true
	}
	return nil, false
}

func schemaTypeNames(node *yaml.Node) ([]string, bool) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.ScalarNode && node.Kind != yaml.SequenceNode {
		return nil, false
	}
	items := []*yaml.Node{node}
	if node.Kind == yaml.SequenceNode {
		items = sequence(node)
	}
	if len(items) == 0 {
		return nil, false
	}
	seen := make(map[string]bool, len(items))
	names := make([]string, 0, len(items))
	for _, item := range items {
		item = unalias(item)
		if item == nil || item.Kind != yaml.ScalarNode || item.Tag != "!!str" || seen[item.Value] {
			return nil, false
		}
		seen[item.Value] = true
		names = append(names, item.Value)
	}
	return names, true
}

func schemaTypeFits(source, target string) bool {
	return source == target || source == "integer" && target == "number"
}

func schemaTypeSetHas(types []string, name string) bool {
	for _, typ := range types {
		if typ == name {
			return true
		}
	}
	return false
}

func schemaTypeSetsCompatible(source, target []string) bool {
	for _, sourceType := range source {
		matched := false
		for _, targetType := range target {
			if schemaTypeFits(sourceType, targetType) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func schemaBoolean(schema *yaml.Node, value bool) bool {
	schema = unalias(schema)
	return schema != nil && schema.Kind == yaml.ScalarNode && schema.Tag == "!!bool" && scalar(schema) == strconv.FormatBool(value)
}

func projectionSchemaSupported(schema *yaml.Node) bool {
	schema = unalias(schema)
	if schema == nil {
		return false
	}
	if schema.Kind == yaml.ScalarNode && schema.Tag == "!!bool" {
		return true
	}
	if schemaIsUnconstrained(schema) {
		return true
	}
	if !supportedStaticSchema(schema) {
		return false
	}
	types, ok := schemaTypeNames(child(schema, "type"))
	if !ok {
		// Draft 2020-12 permits type-specific assertions without a type. Check
		// nested schemas recursively; the keyword assertions themselves are
		// evaluated against each concrete value's type.
		for _, name := range mapKeys(child(schema, "properties")) {
			if !projectionSchemaSupported(child(child(schema, "properties"), name)) {
				return false
			}
		}
		if additional := child(schema, "additionalProperties"); additional != nil && unalias(additional).Kind == yaml.MappingNode && !projectionSchemaSupported(additional) {
			return false
		}
		if items := child(schema, "items"); items != nil && !projectionSchemaSupported(items) {
			return false
		}
		return true
	}
	for _, typ := range types {
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
			if items != nil && !projectionSchemaSupported(items) {
				return false
			}
		case "string", "integer", "number", "boolean", "null":
		default:
			return false
		}
	}
	return true
}

// schemaIsUnconstrained recognizes JSON Schemas containing only annotations
// and dialect metadata. An empty Schema means "any JSON value"; it is not the
// same as an object Schema with an open property map.
func schemaIsUnconstrained(schema *yaml.Node) bool {
	schema = unalias(schema)
	if schema != nil && schema.Kind == yaml.ScalarNode && schema.Tag == "!!bool" {
		return scalar(schema) == "true"
	}
	if schema == nil || schema.Kind != yaml.MappingNode {
		return false
	}
	annotations := map[string]bool{
		"$schema": true, "$id": true,
		"title": true, "description": true, "default": true, "examples": true,
		"deprecated": true, "readOnly": true, "writeOnly": true,
		"format": true, "contentEncoding": true, "contentMediaType": true,
	}
	for _, key := range mapKeys(schema) {
		if !annotations[key] {
			return false
		}
	}
	return true
}

func schemaConstraintsCompatible(source, target *yaml.Node, typ string) bool {
	var lowerKeys, upperKeys []string
	switch typ {
	case "string":
		lowerKeys, upperKeys = []string{"minLength"}, []string{"maxLength"}
	case "array":
		lowerKeys, upperKeys = []string{"minItems"}, []string{"maxItems"}
	case "object":
		lowerKeys, upperKeys = []string{"minProperties"}, []string{"maxProperties"}
	case "integer", "number":
		lowerKeys, upperKeys = []string{"minimum"}, []string{"maximum"}
	}
	for _, key := range lowerKeys {
		if t := schemaNumber(target, key); t != nil {
			s := sourceConstraintBound(source, key, typ)
			weaker := s == nil || s.Cmp(t) < 0
			if typ == "integer" || typ == "number" {
				exclusive := schemaNumber(source, "exclusiveMinimum")
				weaker = weaker && (exclusive == nil || exclusive.Cmp(t) < 0)
			}
			if weaker {
				return false
			}
		}
	}
	for _, key := range upperKeys {
		if t := schemaNumber(target, key); t != nil {
			s := sourceConstraintBound(source, key, typ)
			weaker := s == nil || s.Cmp(t) > 0
			if typ == "integer" || typ == "number" {
				exclusive := schemaNumber(source, "exclusiveMaximum")
				weaker = weaker && (exclusive == nil || exclusive.Cmp(t) > 0)
			}
			if weaker {
				return false
			}
		}
	}
	if typ == "integer" || typ == "number" {
		if t := schemaNumber(target, "exclusiveMinimum"); t != nil {
			minimum, exclusive := schemaNumber(source, "minimum"), schemaNumber(source, "exclusiveMinimum")
			if (minimum == nil || minimum.Cmp(t) <= 0) && (exclusive == nil || exclusive.Cmp(t) < 0) {
				return false
			}
		}
		if t := schemaNumber(target, "exclusiveMaximum"); t != nil {
			maximum, exclusive := schemaNumber(source, "maximum"), schemaNumber(source, "exclusiveMaximum")
			if (maximum == nil || maximum.Cmp(t) >= 0) && (exclusive == nil || exclusive.Cmp(t) > 0) {
				return false
			}
		}
		if child(target, "multipleOf") != nil {
			s := schemaNumber(source, "multipleOf")
			if s == nil || !isIntegerMultiple(scalar(child(source, "multipleOf")), scalar(child(target, "multipleOf"))) {
				return false
			}
		}
	}
	if typ == "string" {
		if p := scalar(child(target, "pattern")); p != "" && scalar(child(source, "pattern")) != p {
			return false
		}
	}
	if typ == "array" && scalar(child(target, "uniqueItems")) == "true" && scalar(child(source, "uniqueItems")) != "true" {
		return false
	}
	if c := child(target, "const"); c != nil {
		sc := child(source, "const")
		if sc == nil || !jsonValuesEqual(sc, c) {
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
				if jsonValuesEqual(v, allowed) {
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

func sourceConstraintBound(schema *yaml.Node, key, typ string) *big.Rat {
	bound := schemaNumber(schema, key)
	if typ != "object" {
		return bound
	}
	var inferred *big.Rat
	switch key {
	case "minProperties":
		inferred = new(big.Rat).SetInt64(int64(len(sequence(child(schema, "required")))))
	case "maxProperties":
		if schemaBoolean(child(schema, "additionalProperties"), false) {
			inferred = new(big.Rat).SetInt64(int64(len(mapKeys(child(schema, "properties")))))
		}
	}
	if inferred == nil {
		return bound
	}
	if bound == nil {
		return inferred
	}
	if key == "minProperties" && inferred.Cmp(bound) > 0 || key == "maxProperties" && inferred.Cmp(bound) < 0 {
		return inferred
	}
	return bound
}

func schemaNumber(schema *yaml.Node, key string) *big.Rat {
	n := child(schema, key)
	if n == nil {
		return nil
	}
	v, ok := exactRational(scalar(n))
	if !ok {
		return nil
	}
	return v
}

func literalMatchesSchema(value, schema *yaml.Node) bool {
	value, schema = unalias(value), unalias(schema)
	if value == nil || schema == nil || !validJSONLiteral(value) {
		return false
	}
	if schemaIsUnconstrained(schema) {
		return true
	}
	if schema.Kind == yaml.ScalarNode && schema.Tag == "!!bool" {
		return scalar(schema) == "true"
	}
	if !projectionSchemaSupported(schema) {
		return false
	}
	types, ok := schemaTypeNames(child(schema, "type"))
	if !ok {
		typ := literalType(value)
		if typ == "" {
			return false
		}
		types = []string{typ}
	}
	if !schemaTypeSetAllowsLiteral(types, literalType(value)) {
		return false
	}
	if c := child(schema, "const"); c != nil && !jsonValuesEqual(c, value) {
		return false
	}
	if enum := sequence(child(schema, "enum")); len(enum) > 0 {
		matched := false
		for _, v := range enum {
			if jsonValuesEqual(v, value) {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}
	switch literalType(value) {
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
		if n, ok := numberValue(child(schema, "minProperties")); ok && new(big.Rat).SetInt64(int64(len(value.Content)/2)).Cmp(n) < 0 {
			return false
		}
		if n, ok := numberValue(child(schema, "maxProperties")); ok && new(big.Rat).SetInt64(int64(len(value.Content)/2)).Cmp(n) > 0 {
			return false
		}
	case "array":
		items := child(schema, "items")
		if value.Kind != yaml.SequenceNode {
			return false
		}
		if n, ok := numberValue(child(schema, "minItems")); ok && new(big.Rat).SetInt64(int64(len(value.Content))).Cmp(n) < 0 {
			return false
		}
		if n, ok := numberValue(child(schema, "maxItems")); ok && new(big.Rat).SetInt64(int64(len(value.Content))).Cmp(n) > 0 {
			return false
		}
		if scalar(child(schema, "uniqueItems")) == "true" {
			for i, item := range value.Content {
				for _, previous := range value.Content[:i] {
					if jsonValuesEqual(previous, item) {
						return false
					}
				}
			}
		}
		if items != nil {
			for _, v := range value.Content {
				if !literalMatchesSchema(v, items) {
					return false
				}
			}
		}
	case "string":
		text := value.Value
		if n, ok := numberValue(child(schema, "minLength")); ok && new(big.Rat).SetInt64(int64(len([]rune(text)))).Cmp(n) < 0 {
			return false
		}
		if n, ok := numberValue(child(schema, "maxLength")); ok && new(big.Rat).SetInt64(int64(len([]rune(text)))).Cmp(n) > 0 {
			return false
		}
		if pattern := scalar(child(schema, "pattern")); pattern != "" {
			re, err := regexp.Compile(pattern)
			if err != nil || !re.MatchString(text) {
				return false
			}
		}
	case "integer", "number":
		n, ok := exactRational(value.Value)
		if !ok {
			return false
		}
		if bound, ok := numberValue(child(schema, "minimum")); ok && n.Cmp(bound) < 0 {
			return false
		}
		if bound, ok := numberValue(child(schema, "maximum")); ok && n.Cmp(bound) > 0 {
			return false
		}
		if bound, ok := numberValue(child(schema, "exclusiveMinimum")); ok && n.Cmp(bound) <= 0 {
			return false
		}
		if bound, ok := numberValue(child(schema, "exclusiveMaximum")); ok && n.Cmp(bound) >= 0 {
			return false
		}
		if divisor := child(schema, "multipleOf"); divisor != nil && !isIntegerMultiple(value.Value, scalar(divisor)) {
			return false
		}
	}
	return true
}

func schemaTypeSetAllowsLiteral(types []string, literal string) bool {
	for _, typ := range types {
		if schemaTypeFits(literal, typ) {
			return true
		}
	}
	return false
}

// JSON Schema treats numbers mathematically, so integer and decimal spellings
// with the same exact value compare equal even when Go decodes them into
// different numeric types. Compare recursively to preserve that rule inside
// const and enum arrays and objects as well.
func jsonValuesEqual(left, right *yaml.Node) bool {
	left, right = unalias(left), unalias(right)
	if left == nil || right == nil || left.Kind != right.Kind {
		return left == nil && right == nil
	}
	if left.Kind == yaml.ScalarNode {
		leftNumber := left.Tag == "!!int" || left.Tag == "!!float"
		rightNumber := right.Tag == "!!int" || right.Tag == "!!float"
		if leftNumber || rightNumber {
			if !leftNumber || !rightNumber {
				return false
			}
			l, lok := exactRational(left.Value)
			r, rok := exactRational(right.Value)
			return lok && rok && l.Cmp(r) == 0
		}
		if left.Tag == "!!null" || right.Tag == "!!null" {
			return left.Tag == "!!null" && right.Tag == "!!null"
		}
		if left.Tag == "!!bool" && right.Tag == "!!bool" {
			l, lok := strconv.ParseBool(left.Value)
			r, rok := strconv.ParseBool(right.Value)
			return lok == nil && rok == nil && l == r
		}
		return left.Tag == right.Tag && left.Value == right.Value
	}
	if len(left.Content) != len(right.Content) {
		return false
	}
	if left.Kind == yaml.SequenceNode {
		for i := range left.Content {
			if !jsonValuesEqual(left.Content[i], right.Content[i]) {
				return false
			}
		}
		return true
	}
	if left.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(left.Content); i += 2 {
			key := unalias(left.Content[i])
			other := child(right, key.Value)
			if other == nil || !jsonValuesEqual(left.Content[i+1], other) {
				return false
			}
		}
		return true
	}
	return false
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

func numberValue(n *yaml.Node) (*big.Rat, bool) {
	if n == nil {
		return nil, false
	}
	return exactRational(scalar(n))
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
	known := map[string]bool{
		"$schema": true, "$id": true, "$ref": true,
		"title": true, "description": true, "default": true, "examples": true,
		"readOnly": true, "writeOnly": true, "deprecated": true,
		"format": true, "contentEncoding": true, "contentMediaType": true,
		"type": true, "properties": true, "required": true, "additionalProperties": true,
		"items": true, "minLength": true, "maxLength": true, "pattern": true,
		"minimum": true, "maximum": true, "exclusiveMinimum": true, "exclusiveMaximum": true,
		"multipleOf": true, "minItems": true, "maxItems": true,
		"minProperties": true, "maxProperties": true, "uniqueItems": true,
		"const": true, "enum": true,
	}
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
		contract := s.ast.Contract
		if contract == nil {
			a.errorAt(path, line, "EXTERNAL_SCHEMA_MISSING", "missing parsed SkillContract for context source")
			return nil
		}
		if category == "request" {
			rel = contract.RequestSchema
		} else if category == "subject" || category == "facts" || category == "observations" || category == "evidence" {
			rel = contract.ContextSchemas[category]
			if rel == "" {
				a.errorAt(path, line, "EXTERNAL_SCHEMA_MISSING", "missing explicit Owner schema for context."+category)
				return nil
			}
		} else if category == "constraints" {
			a.errorAt(path, line, "EXTERNAL_SCHEMA_MISSING", "no Skill Language Owner schema field is defined for context.constraints")
			return nil
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
				node := child(child(proc, "nodes"), parts[0])
				if node == nil {
					a.errorAt(path, line, "BINDING_UNRESOLVED", "unknown node output Producer: "+parts[0])
				} else if child(node, "procedure") != nil {
					a.errorAt(path, line, "BINDING_UNRESOLVED", "Procedure node has no declared output Schema: "+parts[0])
				} else {
					a.errorAt(path, line, "BINDING_UNRESOLVED", "node action has no declared output Schema: "+parts[0])
				}
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
	// Round-trip through JSON so json.Number values emitted by the Script
	// parser remain JSON numeric instances instead of YAML strings.
	b, e := json.Marshal(v)
	if e != nil {
		return nil
	}
	node, err := yaml.ParseValue(b)
	if err != nil {
		return nil
	}
	return node
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
		if value, ok := exactRational(n.Value); ok && value.IsInt() {
			return "integer"
		}
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
