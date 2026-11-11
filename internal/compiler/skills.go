package compiler

import (
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type skillEntry struct {
	name           string
	contract       string
	contractAST    *yaml.Node
	procedure      string
	procedureAST   *yaml.Node
	calls          map[string]int
	scripts        map[string]int
	preds          map[string]int
	procedures     map[string]bool
	allowed        map[string]bool
	entry          string
	nodes          map[string]map[string]*yaml.Node
	requestSchema  string
	contextSchemas map[string]string
	procedureRefs  map[string]map[string]int
	libraryFiles   map[string]bool
	schemaRoots    map[string]bool
	deps           map[string]int
	procedureCalls map[string]map[string]int
}

func sortedStringMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (a *analysis) parseSkills() {
	names := make(map[string]bool)
	for filename := range a.files {
		if !strings.HasPrefix(filename, "skills/") {
			continue
		}
		p := strings.Split(filename, "/")
		if len(p) >= 3 {
			names[p[1]] = true
		}
	}
	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	for _, name := range sorted {
		base := "skills/" + name + "/"
		s := &skillEntry{name: name, contract: base + "assurance/contract.yaml", procedure: base + "assurance/procedure.yaml",
			calls: make(map[string]int), scripts: make(map[string]int), deps: make(map[string]int), preds: make(map[string]int), procedureCalls: make(map[string]map[string]int),
			procedures: make(map[string]bool), allowed: make(map[string]bool), nodes: make(map[string]map[string]*yaml.Node), procedureRefs: make(map[string]map[string]int), libraryFiles: make(map[string]bool), schemaRoots: make(map[string]bool)}
		a.skills[name] = s
		if _, ok := a.files[base+"SKILL.md"]; !ok {
			a.errorAt(base+"SKILL.md", 0, "SKILL_ENTRY_MISSING", "SKILL.md is required")
		}
		for _, filename := range []string{s.contract, s.procedure} {
			if _, ok := a.files[filename]; !ok {
				a.errorAt(filename, 0, "SOURCE_MISSING", "required Skill source missing")
			}
		}
		if _, ok := a.files[s.contract]; ok {
			a.parseContract(s)
		}
		if filename := base + "assurance/predicates.yaml"; a.files[filename] != nil {
			a.parsePredicates(s, filename)
		}
		if _, ok := a.files[s.procedure]; ok {
			a.parseProcedure(s)
		}
		a.validateContractPredicates(s)
	}
	// Resolve call-site bindings only after every Skill's Procedure AST is available.
	for _, name := range sorted {
		s := a.skills[name]
		if s == nil || s.procedureAST == nil {
			continue
		}
		for _, pname := range mapKeys(child(s.procedureAST, "procedures")) {
			proc := child(child(s.procedureAST, "procedures"), pname)
			a.validateProcedureBindings(s, proc, child(proc, "nodes"), s.procedure)
		}
	}
	if len(a.skills) == 0 {
		a.errorAt("skills/", 0, "NO_SKILLS", "project contains no Skill")
	}
}

func (a *analysis) parsePredicates(s *skillEntry, filename string) {
	root := a.parseYAML(filename)
	if root == nil {
		return
	}
	if unalias(root).Kind != yaml.MappingNode {
		a.errorAt(filename, root.Line, "PREDICATE_SOURCE_INCOMPLETE", "PredicateSet must be a mapping")
		return
	}
	for _, field := range mapKeys(root) {
		if field != "predicates" {
			a.errorAt(filename, child(root, field).Line, "PREDICATE_SOURCE_INCOMPLETE", "unknown PredicateSet field: "+field)
		}
	}
	predicates := child(root, "predicates")
	if predicates == nil || unalias(predicates).Kind != yaml.MappingNode {
		a.errorAt(filename, root.Line, "PREDICATE_SOURCE_INCOMPLETE", "PredicateSet requires a predicates mapping")
		return
	}
	a.validateStringMappingKeys(filename, predicates, "Predicate IDs", "PREDICATE_SOURCE_INCOMPLETE")
	for _, id := range mapKeys(predicates) {
		predicate := child(predicates, id)
		s.preds[id] = 1
		if unalias(predicate).Kind != yaml.MappingNode {
			a.errorAt(filename, predicate.Line, "PREDICATE_SOURCE_INCOMPLETE", "Predicate definition must be a mapping: "+id)
			continue
		}
		for _, field := range mapKeys(predicate) {
			if field != "evaluator" && field != "observations" {
				a.errorAt(filename, child(predicate, field).Line, "PREDICATE_SOURCE_INCOMPLETE", "unsupported Predicate field: "+field)
			}
		}
		evaluator := child(predicate, "evaluator")
		evaluatorTypeNode := child(evaluator, "type")
		evaluatorType := scalar(evaluatorTypeNode)
		complete := false
		if evaluator != nil && unalias(evaluator).Kind == yaml.MappingNode {
			for _, field := range mapKeys(evaluator) {
				allowed := field == "type"
				switch evaluatorType {
				case "cel":
					allowed = allowed || field == "expression"
				case "script":
					allowed = allowed || field == "ref"
				}
				if !allowed {
					a.errorAt(filename, child(evaluator, field).Line, "PREDICATE_SOURCE_INCOMPLETE", "unsupported "+evaluatorType+" evaluator field: "+field)
				}
			}
			if evaluatorTypeNode == nil || evaluatorTypeNode.Kind != yaml.ScalarNode || evaluatorTypeNode.Tag != "!!str" {
				a.errorAt(filename, evaluator.Line, "PREDICATE_SOURCE_INCOMPLETE", "Predicate evaluator type must be a string")
			}
			switch evaluatorType {
			case "cel":
				expression := child(evaluator, "expression")
				complete = expression != nil && expression.Kind == yaml.ScalarNode && expression.Tag == "!!str" && expression.Value != ""
			case "script":
				ref := child(evaluator, "ref")
				complete = ref != nil && ref.Kind == yaml.ScalarNode && ref.Tag == "!!str" && ref.Value != ""
			}
		}
		if !complete {
			a.errorAt(filename, predicate.Line, "PREDICATE_SOURCE_INCOMPLETE", "Predicate evaluator requires cel expression or script ref: "+id)
		}
		observations := child(predicate, "observations")
		if observations != nil {
			if unalias(observations).Kind != yaml.MappingNode {
				a.errorAt(filename, observations.Line, "PREDICATE_SOURCE_INCOMPLETE", "Predicate observations must be a mapping: "+id)
				continue
			}
			a.validateStringMappingKeys(filename, observations, "Predicate observation names", "PREDICATE_SOURCE_INCOMPLETE")
			for _, name := range mapKeys(observations) {
				observation := child(observations, name)
				if unalias(observation).Kind != yaml.MappingNode {
					a.errorAt(filename, observation.Line, "PREDICATE_SOURCE_INCOMPLETE", "Predicate observation requires a reference: "+id+"/"+name)
					continue
				}
				for _, field := range mapKeys(observation) {
					if field != "ref" && field != "scope" && field != "freshness" {
						a.errorAt(filename, child(observation, field).Line, "PREDICATE_SOURCE_INCOMPLETE", "unsupported Predicate observation field: "+field)
					}
				}
				ref := child(observation, "ref")
				if ref == nil || ref.Kind != yaml.ScalarNode || ref.Tag != "!!str" || ref.Value == "" {
					a.errorAt(filename, observation.Line, "PREDICATE_SOURCE_INCOMPLETE", "Predicate observation ref must be a nonempty string: "+id+"/"+name)
				}
				for _, field := range []string{"scope", "freshness"} {
					value := child(observation, field)
					if value != nil && (value.Kind != yaml.ScalarNode || value.Tag != "!!str" || value.Value == "") {
						a.errorAt(filename, value.Line, "PREDICATE_SOURCE_INCOMPLETE", "Predicate observation "+field+" must be a nonempty string: "+id+"/"+name)
					}
				}
			}
		}
	}
}

func (a *analysis) validateContractPredicates(s *skillEntry) {
	spec := child(s.contractAST, "spec")
	if spec == nil {
		return
	}
	for _, group := range []string{"preconditions", "postconditions", "applicability"} {
		n := child(spec, group)
		if n == nil {
			continue
		}
		if unalias(n).Kind != yaml.SequenceNode {
			a.errorAt(s.contract, n.Line, "PREDICATE_SOURCE_INCOMPLETE", "contract "+group+" must be a sequence of Predicate references")
			continue
		}
		for _, item := range sequence(n) {
			if unalias(item).Kind != yaml.MappingNode {
				a.errorAt(s.contract, item.Line, "PREDICATE_SOURCE_INCOMPLETE", "contract "+group+" entry must be a mapping with a Predicate ID")
				continue
			}
			for _, field := range mapKeys(item) {
				if field != "predicate" && field != "expect" {
					a.errorAt(s.contract, child(item, field).Line, "PREDICATE_SOURCE_INCOMPLETE", "unsupported contract "+group+" field: "+field)
				}
			}
			predicate := child(item, "predicate")
			id := scalar(predicate)
			if predicate == nil || predicate.Kind != yaml.ScalarNode || predicate.Tag != "!!str" || id == "" {
				a.errorAt(s.contract, item.Line, "PREDICATE_SOURCE_INCOMPLETE", "contract "+group+" entry requires a nonempty string Predicate ID")
				continue
			}
			if expected := child(item, "expect"); expected != nil && (expected.Kind != yaml.ScalarNode || expected.Tag != "!!str" || expected.Value == "") {
				a.errorAt(s.contract, expected.Line, "PREDICATE_SOURCE_INCOMPLETE", "contract Predicate expectation must be a nonempty string")
			}
			if s.preds[id] == 0 {
				a.errorAt(s.contract, item.Line, "PREDICATE_NOT_FOUND", "unknown contract Predicate: "+id)
			}
		}
	}
}

func (a *analysis) parseContract(s *skillEntry) {
	root := a.parseYAML(s.contract)
	if root == nil {
		return
	}
	s.contractAST = root
	metadata := child(root, "metadata")
	if got := scalar(child(metadata, "name")); got != "" && got != s.name {
		a.errorAt(s.contract, metadata.Line, "SKILL_NAME_MISMATCH", "metadata.name must match skills/ directory")
	}
	spec := child(root, "spec")
	if spec == nil {
		a.errorAt(s.contract, root.Line, "CONTRACT_SPEC", "missing contract spec")
		return
	}
	entry := scalar(child(spec, "entryProcedure"))
	entryNode := child(spec, "entryProcedure")
	if entryNode == nil || entryNode.Kind != yaml.ScalarNode || entryNode.Tag != "!!str" || entry == "" {
		a.errorAt(s.contract, spec.Line, "SKILL_ENTRY_MISSING", "missing spec.entryProcedure")
	}
	s.entry = entry
	for _, capability := range sequence(child(child(spec, "allowedResources"), "capabilities")) {
		if v := scalar(capability); v != "" {
			s.allowed[v] = true
		}
	}
	for _, dep := range sequence(child(spec, "dependencies")) {
		id := scalar(child(dep, "skill"))
		if id == "" {
			id = scalar(child(dep, "name"))
		}
		if id != "" {
			s.deps[id] = dep.Line
		}
	}
	if schema := scalar(child(child(spec, "inputs"), "requestSchema")); schema != "" {
		s.requestSchema = schema
		filename := schema
		if strings.HasPrefix(schema, "lib/") {
			if !validLibraryPath(schema) {
				a.errorAt(s.contract, spec.Line, "LIB_SCHEMA_INVALID", "requestSchema has an unsafe Library path")
			} else {
				s.libraryFiles[filename] = true
			}
		} else {
			if !strings.HasPrefix(schema, "schemas/") || !validPath("skills/"+s.name+"/"+schema) {
				a.errorAt(s.contract, spec.Line, "EXTERNAL_SCHEMA_MISSING", "requestSchema must be a Skill schemas/ path or imported lib/ path")
			} else {
				filename = "skills/" + s.name + "/" + schema
			}
		}
		if filename != "" && a.files[filename] == nil {
			a.errorAt(s.contract, spec.Line, "EXTERNAL_SCHEMA_MISSING", "requestSchema not found: "+schema)
		} else if filename != "" {
			s.schemaRoots[filename] = true
		}
	}
	s.contextSchemas = map[string]string{}
	for _, key := range []string{"subject", "facts", "observations"} {
		if p := scalar(child(child(child(spec, "context"), "schemas"), key)); p != "" {
			if !strings.HasPrefix(p, "lib/host/") && !strings.HasPrefix(p, "lib/observations/") {
				a.errorAt(s.contract, spec.Line, "EXTERNAL_SCHEMA_MISSING", "context schema must be imported from lib/host/ or lib/observations/: "+p)
				continue
			}
			if !validLibraryPath(p) {
				a.errorAt(s.contract, spec.Line, "LIB_SCHEMA_INVALID", "context schema has an unsafe Library path: "+p)
				continue
			}
			s.contextSchemas[key] = p
			s.libraryFiles[p] = true
			s.schemaRoots[p] = true
		}
	}
}

func (a *analysis) parseProcedure(s *skillEntry) {
	root := a.parseYAML(s.procedure)
	if root == nil {
		return
	}
	s.procedureAST = root
	for _, field := range mapKeys(root) {
		if field != "procedures" {
			a.errorAt(s.procedure, child(root, field).Line, "PROCEDURE_UNSUPPORTED", "unknown Procedure document field: "+field)
		}
	}
	procedures := child(root, "procedures")
	if procedures == nil || unalias(procedures).Kind != yaml.MappingNode || len(mapKeys(procedures)) == 0 {
		line := root.Line
		if procedures != nil {
			line = procedures.Line
		}
		a.errorAt(s.procedure, line, "PROCEDURES_MISSING", "procedures must be a nonempty mapping")
		return
	}
	a.validateStringMappingKeys(s.procedure, procedures, "Procedure names", "PROCEDURE_UNSUPPORTED")
	for _, name := range mapKeys(procedures) {
		s.procedures[name] = true
	}
	if s.entry != "" && !s.procedures[s.entry] {
		a.errorAt(s.contract, 0, "ENTRY_NOT_FOUND", "entryProcedure does not exist: "+s.entry)
	}
	for _, pname := range mapKeys(procedures) {
		proc := child(procedures, pname)
		if proc == nil || unalias(proc).Kind != yaml.MappingNode {
			line := procedures.Line
			if proc != nil {
				line = proc.Line
			}
			a.errorAt(s.procedure, line, "PROCEDURE_UNSUPPORTED", "Procedure definition must be a mapping: "+pname)
			continue
		}
		for _, field := range mapKeys(proc) {
			switch field {
			case "inputs", "start", "nodes", "applicability":
			default:
				a.errorAt(s.procedure, child(proc, field).Line, "PROCEDURE_UNSUPPORTED", "unknown Procedure field: "+field)
			}
		}
		if inputs := child(proc, "inputs"); inputs != nil {
			if unalias(inputs).Kind != yaml.MappingNode {
				a.errorAt(s.procedure, inputs.Line, "PROCEDURE_UNSUPPORTED", "Procedure inputs must be a mapping")
			} else {
				a.validateStringMappingKeys(s.procedure, inputs, "Procedure input names", "PROCEDURE_UNSUPPORTED")
				for _, inputName := range mapKeys(inputs) {
					input := child(inputs, inputName)
					if unalias(input).Kind != yaml.MappingNode {
						a.errorAt(s.procedure, input.Line, "PROCEDURE_UNSUPPORTED", "Procedure input must declare type and from: "+inputName)
						continue
					}
					for _, field := range mapKeys(input) {
						if field != "type" && field != "from" {
							a.errorAt(s.procedure, child(input, field).Line, "PROCEDURE_UNSUPPORTED", "unknown Procedure input field: "+field)
						}
					}
					typ, from := scalar(child(input, "type")), scalar(child(input, "from"))
					if !validProcedureInputType(typ) || from == "" {
						a.errorAt(s.procedure, input.Line, "PROCEDURE_UNSUPPORTED", "Procedure input requires a supported type and from source: "+inputName)
					}
				}
			}
		}
		if applicability := child(proc, "applicability"); applicability != nil {
			if unalias(applicability).Kind != yaml.MappingNode {
				a.errorAt(s.procedure, applicability.Line, "PROCEDURE_UNSUPPORTED", "Procedure applicability must be a mapping")
			}
			for _, field := range mapKeys(applicability) {
				if field != "predicates" {
					a.errorAt(s.procedure, child(applicability, field).Line, "PROCEDURE_UNSUPPORTED", "unknown applicability field: "+field)
				}
			}
			predicates := child(applicability, "predicates")
			if predicates != nil && unalias(predicates).Kind != yaml.MappingNode {
				a.errorAt(s.procedure, predicates.Line, "PROCEDURE_UNSUPPORTED", "applicability.predicates must be a mapping")
			}
			if predicates != nil && unalias(predicates).Kind == yaml.MappingNode {
				a.validateStringMappingKeys(s.procedure, predicates, "Predicate references", "PROCEDURE_UNSUPPORTED")
			}
			for _, id := range mapKeys(predicates) {
				if s.preds[id] == 0 {
					a.errorAt(s.procedure, child(predicates, id).Line, "PREDICATE_NOT_FOUND", "unknown applicability Predicate: "+id)
				}
			}
		}
		start := scalar(child(proc, "start"))
		startNode := child(proc, "start")
		nodes := child(proc, "nodes")
		if nodes == nil || unalias(nodes).Kind != yaml.MappingNode || len(mapKeys(nodes)) == 0 {
			line := proc.Line
			if nodes != nil {
				line = nodes.Line
			}
			a.errorAt(s.procedure, line, "PROCEDURE_UNSUPPORTED", "Procedure nodes must be a nonempty mapping")
			continue
		}
		a.validateStringMappingKeys(s.procedure, nodes, "Node names", "PROCEDURE_UNSUPPORTED")
		if startNode == nil || startNode.Kind != yaml.ScalarNode || startNode.Tag != "!!str" || start == "" || child(nodes, start) == nil {
			a.errorAt(s.procedure, proc.Line, "NODE_TARGET_UNKNOWN", "procedure start must name a node: "+start)
		}
		names := mapKeys(nodes)
		known := make(map[string]bool)
		for _, n := range names {
			known[n] = true
		}
		prior := make(map[string]bool)
		for _, n := range names {
			node := child(nodes, n)
			if n == "" {
				a.errorAt(s.procedure, node.Line, "PROCEDURE_UNSUPPORTED", "node names must be nonempty strings")
				continue
			}
			actions := 0
			for _, action := range []string{"call", "script", "procedure", "verify", "reconcile", "terminal"} {
				if child(node, action) != nil {
					actions++
				}
			}
			if actions != 1 {
				a.errorAt(s.procedure, node.Line, "PROCEDURE_UNSUPPORTED", "each node must have exactly one supported action")
			}
			for _, action := range []string{"call", "script", "verify", "reconcile", "terminal"} {
				if value := child(node, action); value != nil && (value.Kind != yaml.ScalarNode || value.Tag != "!!str" || value.Value == "") {
					a.errorAt(s.procedure, value.Line, "PROCEDURE_UNSUPPORTED", action+" action must be a nonempty string")
				}
			}
			if ref := child(node, "procedure"); ref != nil {
				if ref.Kind == yaml.MappingNode {
					skillRef, procedureRef := child(ref, "skill"), child(ref, "procedure")
					if skillRef == nil || skillRef.Kind != yaml.ScalarNode || skillRef.Tag != "!!str" || skillRef.Value == "" || procedureRef == nil || procedureRef.Kind != yaml.ScalarNode || procedureRef.Tag != "!!str" || procedureRef.Value == "" {
						a.errorAt(s.procedure, ref.Line, "PROCEDURE_UNSUPPORTED", "explicit Procedure reference requires nonempty string skill and procedure fields")
					}
				} else if ref.Kind != yaml.ScalarNode || ref.Tag != "!!str" || ref.Value == "" {
					a.errorAt(s.procedure, ref.Line, "PROCEDURE_UNSUPPORTED", "procedure action must be a local name or {skill, procedure} reference")
				}
			}
			if child(node, "with") != nil && child(node, "call") == nil && child(node, "script") == nil && child(node, "procedure") == nil {
				a.errorAt(s.procedure, child(node, "with").Line, "PROCEDURE_UNSUPPORTED", "with is only valid for call, script, or procedure actions")
			}
			if child(node, "on") != nil && child(node, "terminal") != nil {
				a.errorAt(s.procedure, child(node, "on").Line, "PROCEDURE_UNSUPPORTED", "terminal nodes cannot declare outcome branches")
			}
			for _, key := range mapKeys(node) {
				switch key {
				case "call", "script", "procedure", "verify", "reconcile", "terminal", "with", "on", "preconditions", "postconditions":
				default:
					a.errorAt(s.procedure, child(node, key).Line, "PROCEDURE_UNSUPPORTED", "unknown node field: "+key)
				}
			}
			if op := scalar(child(node, "call")); op != "" {
				s.calls[op] = child(node, "call").Line
			}
			if pred := scalar(child(node, "verify")); pred != "" && s.preds[pred] == 0 {
				a.errorAt(s.procedure, child(node, "verify").Line, "PREDICATE_NOT_FOUND", "unknown Predicate: "+pred)
			}
			if reconcile := scalar(child(node, "reconcile")); reconcile != "" {
				r := child(node, "reconcile")
				if !known[reconcile] {
					a.errorAt(s.procedure, r.Line, "NODE_TARGET_UNKNOWN", "reconcile target is not a node: "+reconcile)
				} else if !prior[reconcile] {
					a.errorAt(s.procedure, r.Line, "NODE_TARGET_UNKNOWN", "reconcile target must be declared before the reconcile node: "+reconcile)
				} else if operation := scalar(child(child(nodes, reconcile), "call")); operation == "" {
					a.errorAt(s.procedure, r.Line, "PROCEDURE_UNSUPPORTED", "reconcile target must be a Tool call node: "+reconcile)
				} else if tool, ok := a.tools[operation]; !ok || tool.tool.ReconcileOutputSchema == nil {
					a.errorAt(s.procedure, r.Line, "PROCEDURE_UNSUPPORTED", "reconcile target Tool has no declared reconcileOutputSchema: "+operation)
				}
			}
			if script := scalar(child(node, "script")); script != "" {
				if !validSkillScriptPath(script) {
					a.errorAt(s.procedure, child(node, "script").Line, "PROCEDURE_UNSUPPORTED", "Script reference must be a normalized scripts/*.py path: "+script)
				}
				s.scripts[script] = child(node, "script").Line
			}
			if ref := child(node, "procedure"); ref != nil {
				skill, target := scalar(child(ref, "skill")), scalar(child(ref, "procedure"))
				if unalias(ref).Kind == yaml.MappingNode {
					for _, field := range mapKeys(ref) {
						if field != "skill" && field != "procedure" {
							a.errorAt(s.procedure, child(ref, field).Line, "PROCEDURE_UNSUPPORTED", "unknown Procedure reference field: "+field)
						}
					}
				}
				if skill == "" || skill == s.name {
					target = scalar(ref)
					if skill != "" {
						target = scalar(child(ref, "procedure"))
					}
					if target == "" || !s.procedures[target] {
						a.errorAt(s.procedure, ref.Line, "PROCEDURE_NOT_FOUND", "missing local Procedure: "+target)
					} else {
						if s.procedureCalls[pname] == nil {
							s.procedureCalls[pname] = map[string]int{}
						}
						s.procedureCalls[pname][target] = ref.Line
					}
				} else if skill != s.name {
					s.deps[skill] = ref.Line
					if s.procedureRefs[skill] == nil {
						s.procedureRefs[skill] = map[string]int{}
					}
					s.procedureRefs[skill][target] = ref.Line
					if target == "" {
						a.errorAt(s.procedure, ref.Line, "PROCEDURE_NOT_FOUND", "cross-Skill procedure reference requires procedure")
					}
				}
			}
			if branch := child(node, "on"); branch != nil {
				a.validateOutcomeBranches(s.procedure, branch, known, "on")
			}
			for _, scope := range []string{"preconditions", "postconditions"} {
				scopeNode := child(node, scope)
				if scopeNode != nil && unalias(scopeNode).Kind != yaml.MappingNode {
					a.errorAt(s.procedure, scopeNode.Line, "PROCEDURE_UNSUPPORTED", scope+" must be a mapping")
				}
				if scopeNode != nil {
					for _, field := range mapKeys(scopeNode) {
						if field != "steps" && field != "on" {
							a.errorAt(s.procedure, child(scopeNode, field).Line, "PROCEDURE_UNSUPPORTED", "unknown "+scope+" field: "+field)
						}
					}
					if branch := child(scopeNode, "on"); branch != nil {
						a.validateOutcomeBranches(s.procedure, branch, known, scope+".on")
					}
				}
				if steps := child(scopeNode, "steps"); steps != nil {
					if unalias(steps).Kind != yaml.SequenceNode {
						a.errorAt(s.procedure, steps.Line, "PROCEDURE_UNSUPPORTED", scope+".steps must be a sequence")
					}
					seenSteps := map[string]bool{}
					for _, step := range sequence(steps) {
						stepName := child(step, "name")
						id := scalar(stepName)
						script := scalar(child(step, "script"))
						call := scalar(child(step, "call"))
						verify := scalar(child(step, "verify"))
						count := 0
						for _, action := range []string{"script", "call", "verify"} {
							value := child(step, action)
							if value == nil {
								continue
							}
							count++
							if value.Kind != yaml.ScalarNode || value.Tag != "!!str" || value.Value == "" {
								a.errorAt(s.procedure, value.Line, "PROCEDURE_UNSUPPORTED", "nested "+action+" action must be a nonempty string")
							}
						}
						if stepName == nil || stepName.Kind != yaml.ScalarNode || stepName.Tag != "!!str" || id == "" || count != 1 {
							a.errorAt(s.procedure, step.Line, "PROCEDURE_UNSUPPORTED", "step requires name and exactly one script or call action")
							continue
						}
						if seenSteps[id] {
							a.errorAt(s.procedure, child(step, "name").Line, "PROCEDURE_UNSUPPORTED", "duplicate nested step name: "+id)
						}
						seenSteps[id] = true
						if script != "" && !validSkillScriptPath(script) {
							a.errorAt(s.procedure, child(step, "script").Line, "PROCEDURE_UNSUPPORTED", "Script reference must be a normalized scripts/*.py path: "+script)
						}
						if verify != "" && child(step, "with") != nil {
							a.errorAt(s.procedure, child(step, "with").Line, "PROCEDURE_UNSUPPORTED", "with is only valid for nested call or script steps")
						}
						for _, field := range mapKeys(step) {
							switch field {
							case "name", "script", "call", "verify", "with":
							default:
								a.errorAt(s.procedure, child(step, field).Line, "PROCEDURE_UNSUPPORTED", "unknown nested step field: "+field)
							}
						}
						if verify != "" && s.preds[verify] == 0 {
							a.errorAt(s.procedure, child(step, "verify").Line, "PREDICATE_NOT_FOUND", "unknown Predicate: "+verify)
						}
						if script != "" {
							s.scripts[script] = child(step, "script").Line
						}
						if call != "" {
							s.calls[call] = child(step, "call").Line
						}
					}
				}
			}
			prior[n] = true
		}
	}
	a.validateLocalProcedureCycles(s)
}

func (a *analysis) validateOutcomeBranches(filename string, branch *yaml.Node, known map[string]bool, label string) {
	branch = unalias(branch)
	if branch == nil || branch.Kind != yaml.MappingNode {
		line := 0
		if branch != nil {
			line = branch.Line
		}
		a.errorAt(filename, line, "PROCEDURE_UNSUPPORTED", label+" must map nonempty outcome names to node names")
		return
	}
	if len(branch.Content) == 0 {
		a.errorAt(filename, branch.Line, "PROCEDURE_UNSUPPORTED", label+" must contain at least one outcome branch")
		return
	}
	for i := 0; i+1 < len(branch.Content); i += 2 {
		outcome, targetNode := unalias(branch.Content[i]), unalias(branch.Content[i+1])
		if outcome.Kind != yaml.ScalarNode || outcome.Tag != "!!str" || outcome.Value == "" {
			a.errorAt(filename, outcome.Line, "PROCEDURE_UNSUPPORTED", label+" outcome names must be nonempty strings")
			continue
		}
		if targetNode.Kind != yaml.ScalarNode || targetNode.Tag != "!!str" || targetNode.Value == "" || !known[targetNode.Value] {
			target := ""
			if targetNode.Kind == yaml.ScalarNode {
				target = targetNode.Value
			}
			a.errorAt(filename, targetNode.Line, "NODE_TARGET_UNKNOWN", "unknown "+label+" target: "+target)
		}
	}
}

func validProcedureInputType(typ string) bool {
	switch typ {
	case "string", "integer", "number", "boolean", "object", "array":
		return true
	default:
		return false
	}
}

func (a *analysis) validateStringMappingKeys(filename string, mapping *yaml.Node, label, code string) {
	mapping = unalias(mapping)
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := unalias(mapping.Content[i])
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || key.Value == "" {
			a.errorAt(filename, key.Line, code, label+" must be nonempty strings")
		}
	}
}

func validSkillScriptPath(rel string) bool {
	return strings.HasPrefix(rel, "scripts/") && strings.HasSuffix(rel, ".py") && validPath("skills/_/"+rel)
}

func (a *analysis) validateLocalProcedureCycles(s *skillEntry) {
	state := map[string]int{}
	var visit func(string)
	visit = func(name string) {
		if state[name] == 1 {
			a.errorAt(s.procedure, s.procedureCalls[name][name], "PROCEDURE_CYCLE", "local Procedure invocation cycle includes "+name)
			return
		}
		if state[name] == 2 {
			return
		}
		state[name] = 1
		for _, target := range sortedStringMapKeys(s.procedureCalls[name]) {
			line := s.procedureCalls[name][target]
			if state[target] == 1 {
				a.errorAt(s.procedure, line, "PROCEDURE_CYCLE", "local Procedure invocation cycle: "+name+" -> "+target)
			} else {
				visit(target)
			}
		}
		state[name] = 2
	}
	for _, name := range sortedStringMapKeys(s.procedures) {
		visit(name)
	}
}

func (a *analysis) selectSkills(targets []string) map[string]bool {
	selected := make(map[string]bool)
	var visit func(name string, dependency bool)
	visit = func(name string, dependency bool) {
		if selected[name] {
			return
		}
		s, ok := a.skills[name]
		if !ok {
			code := "DEPENDENCY_NOT_FOUND"
			if !dependency {
				code = "TARGET_NOT_FOUND"
			}
			a.errorAt("skills/"+name, 0, code, "Skill is missing: "+name)
			return
		}
		selected[name] = true
		for _, dep := range sortedStringMapKeys(s.deps) {
			visit(dep, true)
		}
	}
	for _, name := range targets {
		visit(name, false)
	}
	return selected
}

func (a *analysis) validateSkills(selected map[string]bool) {
	names := make([]string, 0, len(selected))
	for name := range selected {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		s := a.skills[name]
		if s == nil {
			continue
		}
		for _, dep := range sortedStringMapKeys(s.deps) {
			line := s.deps[dep]
			if a.skills[dep] == nil {
				a.errorAt(s.procedure, line, "DEPENDENCY_NOT_FOUND", "referenced Skill is missing: "+dep)
			} else {
				for _, procedure := range sortedStringMapKeys(s.procedureRefs[dep]) {
					refLine := s.procedureRefs[dep][procedure]
					if !a.skills[dep].procedures[procedure] {
						a.errorAt(s.procedure, refLine, "PROCEDURE_NOT_FOUND", "referenced cross-Skill Procedure is missing: "+dep+"/"+procedure)
					}
				}
			}
		}
		for _, script := range sortedStringMapKeys(s.scripts) {
			line := s.scripts[script]
			if a.files["skills/"+name+"/"+script] == nil {
				a.errorAt(s.procedure, line, "SCRIPT_NOT_FOUND", "Script missing: "+script)
			}
		}
		for _, op := range sortedStringMapKeys(s.calls) {
			line := s.calls[op]
			if _, ok := a.tools[op]; !ok {
				a.errorAt(s.procedure, line, "LIB_OPERATION_UNKNOWN", "Tool operation not imported: "+op)
			}
			if len(s.allowed) > 0 && !s.allowed[op] {
				a.errorAt(s.procedure, line, "TOOL_NOT_DECLARED", "Tool operation is not in allowedResources: "+op)
			}
		}
	}
	state := map[string]int{}
	var visit func(string)
	visit = func(name string) {
		if state[name] == 1 {
			a.errorAt("skills/"+name+"/assurance/contract.yaml", 0, "DEPENDENCY_CYCLE", "Skill dependency cycle includes "+name)
			return
		}
		if state[name] == 2 {
			return
		}
		state[name] = 1
		if s := a.skills[name]; s != nil {
			for _, dep := range sortedStringMapKeys(s.deps) {
				if selected[dep] {
					visit(dep)
				}
			}
		}
		state[name] = 2
	}
	for _, name := range names {
		visit(name)
	}
}
