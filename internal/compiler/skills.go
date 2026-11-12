package compiler

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type skillEntry struct {
	name           string
	ast            *SkillAST
	contract       string
	contractAST    *yaml.Node
	procedure      string
	procedureAST   *yaml.Node
	calls          map[string]int
	scripts        map[string]int
	procedures     map[string]bool
	allowed        map[string]bool
	scriptRefPaths map[string]string
	nodes          map[string]map[string]*yaml.Node
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
		s := &skillEntry{name: name, ast: &SkillAST{Name: name, Span: SourceSpan{Path: base + "SKILL.md"}, Predicates: map[string]PredicateAST{}}, contract: base + "assurance/contract.yaml", procedure: base + "assurance/procedure.yaml",
			calls: make(map[string]int), scripts: make(map[string]int), deps: make(map[string]int), procedureCalls: make(map[string]map[string]int),
			procedures: make(map[string]bool), allowed: make(map[string]bool), scriptRefPaths: make(map[string]string), nodes: make(map[string]map[string]*yaml.Node), procedureRefs: make(map[string]map[string]int), libraryFiles: make(map[string]bool), schemaRoots: make(map[string]bool)}
		a.skills[name] = s
		if _, ok := a.files[base+"SKILL.md"]; !ok {
			a.errorAt(base+"SKILL.md", 0, "SKILL_ENTRY_MISSING", "SKILL.md is required")
		} else {
			a.parseSkillFrontmatter(s)
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
		a.validatePredicateScriptReferences(s)
		a.parseCases(s)
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
			a.prepareProcedureInputSchemas(s, proc)
		}
	}
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

func (a *analysis) validatePredicateScriptReferences(s *skillEntry) {
	needsContext := false
	for _, id := range sortedStringMapKeys(s.ast.Predicates) {
		predicate := s.ast.Predicates[id]
		if predicate.EvaluatorType != "script" || predicate.EvaluatorRef == "" {
			continue
		}
		if !validSkillScriptPath(predicate.EvaluatorRef) {
			a.errorAt(predicate.Span.Path, predicate.Span.Line, "PREDICATE_SOURCE_INCOMPLETE", "Predicate Script ref must be a package-relative scripts/*.py path: "+predicate.EvaluatorRef)
			continue
		}
		s.scripts[predicate.EvaluatorRef] = predicate.Span.Line
		s.scriptRefPaths[predicate.EvaluatorRef] = predicate.Span.Path
		needsContext = true
	}
	if needsContext {
		contract := s.ast.Contract
		if contract == nil || contract.RequestSchema == "" {
			a.errorAt(s.contract, 0, "EXTERNAL_SCHEMA_MISSING", "Predicate Evaluation Context requires the Skill Request Schema")
		}
		for _, key := range []string{"subject", "facts", "observations", "evidence"} {
			path := ""
			if contract != nil {
				path = contract.ContextSchemas[key]
			}
			if path == "" {
				a.errorAt(s.contract, 0, "EXTERNAL_SCHEMA_MISSING", "Predicate Evaluation Context requires context.schemas."+key)
				continue
			}
			if a.files[path] == nil {
				a.errorAt(s.contract, 0, "EXTERNAL_SCHEMA_MISSING", "Predicate Evaluation Context Owner Schema is not imported: "+path)
			}
		}
	}
}

func (a *analysis) parseCases(s *skillEntry) {
	prefix := "skills/" + s.name + "/cases/"
	for _, filename := range sortedStringMapKeys(a.files) {
		if !strings.HasPrefix(filename, prefix) || !strings.HasSuffix(filename, "/case.yaml") {
			continue
		}
		relative := strings.TrimPrefix(filename, prefix)
		parts := strings.Split(relative, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] != "case.yaml" {
			a.errorAt(filename, 1, "CASE_INVALID", "Case source must be cases/<case-name>/case.yaml")
			continue
		}
		root := a.parseYAML(filename)
		if root == nil {
			continue
		}
		for _, field := range mapKeys(root) {
			switch field {
			case "apiVersion", "kind", "metadata", "spec":
			default:
				a.errorAt(filename, child(root, field).Line, "CASE_INVALID", "unknown SkillTestCase field: "+field)
			}
		}
		if scalar(child(root, "apiVersion")) != "connectionagent.io/v1alpha1" || scalar(child(root, "kind")) != "SkillTestCase" {
			a.errorAt(filename, root.Line, "CASE_INVALID", "Case must be a connectionagent.io/v1alpha1 SkillTestCase")
		}
		metadata := child(root, "metadata")
		name := scalar(child(metadata, "name"))
		owner := scalar(child(metadata, "skill"))
		if name == "" || owner != s.name || name != parts[0] {
			line := root.Line
			if metadata != nil {
				line = metadata.Line
			}
			a.errorAt(filename, line, "CASE_INVALID", "Case metadata.skill and metadata.name must match its owning package and directory")
		}
		if metadata == nil || unalias(metadata).Kind != yaml.MappingNode {
			a.errorAt(filename, root.Line, "CASE_INVALID", "Case metadata must be a mapping")
		} else {
			for _, field := range mapKeys(metadata) {
				if field != "skill" && field != "name" {
					a.errorAt(filename, child(metadata, field).Line, "CASE_INVALID", "unknown Case metadata field: "+field)
				}
			}
		}
		spec := child(root, "spec")
		if spec == nil || unalias(spec).Kind != yaml.MappingNode {
			a.errorAt(filename, root.Line, "CASE_INVALID", "Case spec must be a mapping")
		} else {
			for _, field := range mapKeys(spec) {
				switch field {
				case "classes", "purpose", "context", "expect", "fixtures":
				default:
					a.errorAt(filename, child(spec, field).Line, "CASE_INVALID", "unknown Case spec field: "+field)
				}
			}
			if classes := child(spec, "classes"); classes != nil {
				if unalias(classes).Kind != yaml.SequenceNode || len(sequence(classes)) == 0 {
					a.errorAt(filename, classes.Line, "CASE_INVALID", "Case classes must be a nonempty string sequence")
				} else {
					for _, class := range sequence(classes) {
						if unalias(class).Kind != yaml.ScalarNode || unalias(class).Tag != "!!str" || scalar(class) == "" {
							a.errorAt(filename, class.Line, "CASE_INVALID", "Case class values must be nonempty strings")
						}
					}
				}
			}
			if purpose := child(spec, "purpose"); purpose != nil && (unalias(purpose).Kind != yaml.ScalarNode || unalias(purpose).Tag != "!!str" || strings.TrimSpace(scalar(purpose)) == "") {
				a.errorAt(filename, purpose.Line, "CASE_INVALID", "Case purpose must be a nonempty string")
			}
		}
		s.ast.Cases = append(s.ast.Cases, CaseAST{Name: name, Path: filename, Span: sourceSpan(filename, root)})
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
		switch field {
		case "apiVersion", "kind", "metadata", "predicates":
		default:
			a.errorAt(filename, child(root, field).Line, "PREDICATE_SOURCE_INCOMPLETE", "unknown PredicateSet field: "+field)
		}
	}
	if scalar(child(root, "apiVersion")) != "connectionagent.io/v1alpha1" || scalar(child(root, "kind")) != "PredicateSet" {
		a.errorAt(filename, root.Line, "PREDICATE_SOURCE_INCOMPLETE", "document must be a connectionagent.io/v1alpha1 PredicateSet")
	}
	metadata := child(root, "metadata")
	if metadata == nil || unalias(metadata).Kind != yaml.MappingNode || scalar(child(metadata, "skill")) != s.name {
		a.errorAtNode(filename, metadata, "PREDICATE_SOURCE_INCOMPLETE", "PredicateSet metadata.skill must match its Skill package")
	} else {
		a.validateMappingFields(filename, metadata, "PredicateSet metadata", "PREDICATE_SOURCE_INCOMPLETE", "skill")
	}
	predicates := child(root, "predicates")
	if predicates == nil || unalias(predicates).Kind != yaml.MappingNode {
		a.errorAt(filename, root.Line, "PREDICATE_SOURCE_INCOMPLETE", "PredicateSet requires a predicates mapping")
		return
	}
	a.validateStringMappingKeys(filename, predicates, "Predicate IDs", "PREDICATE_SOURCE_INCOMPLETE")
	for _, id := range mapKeys(predicates) {
		predicate := child(predicates, id)
		parsedPredicate := PredicateAST{Name: id, Span: sourceSpan(filename, predicate), Raw: predicate, Observations: map[string]ObservationAST{}}
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
		parsedPredicate.EvaluatorType = evaluatorType
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
				parsedPredicate.EvaluatorRef = scalar(ref)
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
				parsedPredicate.Observations[name] = ObservationAST{Name: name, Ref: scalar(ref), Scope: scalar(child(observation, "scope")), Freshness: scalar(child(observation, "freshness")), Span: sourceSpan(filename, observation)}
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
		s.ast.Predicates[id] = parsedPredicate
	}
}

func (a *analysis) validateContractPredicates(s *skillEntry) {
	spec := child(s.contractAST, "spec")
	if spec == nil {
		return
	}
	for _, group := range []string{"preconditions", "requirements", "postconditions"} {
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
			if !s.hasPredicate(id) {
				a.errorAt(s.contract, item.Line, "PREDICATE_NOT_FOUND", "unknown contract Predicate: "+id)
			}
		}
	}
}

func (s *skillEntry) hasPredicate(id string) bool {
	if s == nil || s.ast == nil {
		return false
	}
	_, exists := s.ast.Predicates[id]
	return exists
}

func (a *analysis) parseContract(s *skillEntry) {
	root := a.parseYAML(s.contract)
	if root == nil {
		return
	}
	s.contractAST = root
	s.ast.Contract = &SkillContractAST{Span: sourceSpan(s.contract, root), Raw: root, Name: scalar(child(child(root, "metadata"), "name")),
		SkillType: scalar(child(child(root, "spec"), "skillType")), Goal: scalar(child(child(root, "spec"), "goal")),
		EntryProcedure: scalar(child(child(root, "spec"), "entryProcedure")), ContextSchemas: map[string]string{}}
	if scalar(child(root, "apiVersion")) != "connectionagent.io/v1alpha1" || scalar(child(root, "kind")) != "SkillContract" {
		a.errorAt(s.contract, root.Line, "CONTRACT_SPEC", "document must be a connectionagent.io/v1alpha1 SkillContract")
	}
	for _, field := range mapKeys(root) {
		if field != "apiVersion" && field != "kind" && field != "metadata" && field != "spec" {
			a.errorAt(s.contract, child(root, field).Line, "CONTRACT_SPEC", "unknown SkillContract field: "+field)
		}
	}
	metadata := child(root, "metadata")
	if got := scalar(child(metadata, "name")); got == "" || got != s.name {
		line := root.Line
		if metadata != nil {
			line = metadata.Line
		}
		a.errorAt(s.contract, line, "SKILL_NAME_MISMATCH", "metadata.name must match skills/ directory")
	}
	if metadata != nil && unalias(metadata).Kind == yaml.MappingNode {
		for _, field := range mapKeys(metadata) {
			if field != "name" {
				a.errorAt(s.contract, child(metadata, field).Line, "CONTRACT_SPEC", "unknown SkillContract metadata field: "+field)
			}
		}
	} else {
		a.errorAt(s.contract, root.Line, "CONTRACT_SPEC", "SkillContract metadata must be a mapping")
	}
	spec := child(root, "spec")
	if spec == nil {
		a.errorAt(s.contract, root.Line, "CONTRACT_SPEC", "missing contract spec")
		return
	}
	if unalias(spec).Kind != yaml.MappingNode {
		a.errorAt(s.contract, spec.Line, "CONTRACT_SPEC", "SkillContract spec must be a mapping")
		return
	}
	for _, field := range mapKeys(spec) {
		switch field {
		case "skillType", "goal", "entryProcedure", "inputs", "context", "allowedResources", "preconditions", "requirements", "postconditions", "planning", "failureSemantics":
		default:
			a.errorAt(s.contract, child(spec, field).Line, "CONTRACT_SPEC", "unknown SkillContract spec field: "+field)
		}
	}
	for _, field := range []string{"inputs", "context", "allowedResources", "planning", "failureSemantics"} {
		container := child(spec, field)
		if container == nil {
			continue
		}
		if unalias(container).Kind != yaml.MappingNode {
			a.errorAtNode(s.contract, container, "CONTRACT_SPEC", "SkillContract "+field+" must be an object")
			continue
		}
		switch field {
		case "inputs":
			a.validateMappingFields(s.contract, container, "SkillContract inputs", "CONTRACT_SPEC", "requestSchema")
		case "context":
			a.validateMappingFields(s.contract, container, "SkillContract context", "CONTRACT_SPEC", "requiredFacts", "schemas")
			if schemas := child(container, "schemas"); schemas != nil {
				if unalias(schemas).Kind != yaml.MappingNode {
					a.errorAtNode(s.contract, schemas, "CONTRACT_SPEC", "SkillContract context.schemas must be an object")
				} else {
					a.validateMappingFields(s.contract, schemas, "SkillContract context.schemas", "CONTRACT_SPEC", "subject", "facts", "observations", "evidence")
				}
			}
		case "allowedResources":
			a.validateMappingFields(s.contract, container, "SkillContract allowedResources", "CONTRACT_SPEC", "capabilities")
			if values := child(container, "capabilities"); values != nil {
				a.validateStringSequence(s.contract, values, "SkillContract allowedResources.capabilities", "CONTRACT_SPEC")
			}
		case "planning":
			a.validateMappingFields(s.contract, container, "SkillContract planning", "CONTRACT_SPEC", "maxActions", "allowDynamicProcedure")
		case "failureSemantics":
			a.validateMappingFields(s.contract, container, "SkillContract failureSemantics", "CONTRACT_SPEC", "outcomeUnknown")
			if outcome := child(container, "outcomeUnknown"); outcome != nil {
				if unalias(outcome).Kind != yaml.MappingNode {
					a.errorAtNode(s.contract, outcome, "CONTRACT_SPEC", "failureSemantics.outcomeUnknown must be an object")
				} else {
					a.validateMappingFields(s.contract, outcome, "failureSemantics.outcomeUnknown", "CONTRACT_SPEC", "requireReconcile")
				}
			}
		}
	}
	if goal := child(spec, "goal"); goal != nil && !isNonemptyString(goal) {
		a.errorAtNode(s.contract, goal, "CONTRACT_SPEC", "SkillContract goal must be a nonempty string")
	}
	if skillType := child(spec, "skillType"); skillType != nil && !isNonemptyString(skillType) {
		a.errorAtNode(s.contract, skillType, "CONTRACT_SPEC", "SkillContract skillType must be a nonempty string")
	}
	if context := child(spec, "context"); context != nil {
		if facts := child(context, "requiredFacts"); facts != nil {
			a.validateStringSequence(s.contract, facts, "SkillContract context.requiredFacts", "CONTRACT_SPEC")
		}
	}
	entry := s.ast.Contract.EntryProcedure
	entryNode := child(spec, "entryProcedure")
	if entryNode == nil || entryNode.Kind != yaml.ScalarNode || entryNode.Tag != "!!str" || entry == "" {
		a.errorAt(s.contract, spec.Line, "SKILL_ENTRY_MISSING", "missing spec.entryProcedure")
	}
	for _, capability := range sequence(child(child(spec, "allowedResources"), "capabilities")) {
		if v := scalar(capability); v != "" {
			s.ast.Contract.AllowedCapabilities = append(s.ast.Contract.AllowedCapabilities, v)
			s.allowed[v] = true
		}
	}
	if required := child(child(spec, "context"), "requiredFacts"); required != nil {
		for _, fact := range sequence(required) {
			if scalar(fact) != "" {
				s.ast.Contract.RequiredFacts = append(s.ast.Contract.RequiredFacts, scalar(fact))
			}
		}
	}
	if schema := scalar(child(child(spec, "inputs"), "requestSchema")); schema != "" {
		s.ast.Contract.RequestSchema = schema
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
	for _, key := range []string{"subject", "facts", "observations", "evidence"} {
		if p := scalar(child(child(child(spec, "context"), "schemas"), key)); p != "" {
			if !validLibraryPath(p) {
				a.errorAt(s.contract, spec.Line, "LIB_SCHEMA_INVALID", "context schema has an unsafe Library path: "+p)
				continue
			}
			s.ast.Contract.ContextSchemas[key] = p
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
	s.ast.Procedure = parseProcedureAST(s.procedure, root)
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
	entry := ""
	if s.ast.Contract != nil {
		entry = s.ast.Contract.EntryProcedure
	}
	if entry != "" && !s.procedures[entry] {
		a.errorAt(s.contract, 0, "ENTRY_NOT_FOUND", "entryProcedure does not exist: "+entry)
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
						a.errorAt(s.procedure, input.Line, "PROCEDURE_UNSUPPORTED", "Procedure input must declare a from source: "+inputName)
						continue
					}
					for _, field := range mapKeys(input) {
						if field != "type" && field != "from" {
							a.errorAt(s.procedure, child(input, field).Line, "PROCEDURE_UNSUPPORTED", "unknown Procedure input field: "+field)
						}
					}
					typNode, fromNode := child(input, "type"), child(input, "from")
					typ := scalar(typNode)
					if typNode != nil && (!validProcedureInputType(typ) || unalias(typNode).Kind != yaml.ScalarNode || unalias(typNode).Tag != "!!str") {
						a.errorAt(s.procedure, typNode.Line, "PROCEDURE_UNSUPPORTED", "Procedure input type assertion must be a supported string: "+inputName)
					}
					if fromNode == nil || unalias(fromNode).Kind != yaml.ScalarNode || unalias(fromNode).Tag != "!!str" || scalar(fromNode) == "" {
						a.errorAt(s.procedure, input.Line, "PROCEDURE_UNSUPPORTED", "Procedure input requires a nonempty from source: "+inputName)
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
				if _, exists := s.ast.Predicates[id]; !exists {
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
			if child(node, "with") != nil && child(node, "call") == nil && child(node, "script") == nil && child(node, "procedure") == nil && child(node, "verify") == nil {
				a.errorAt(s.procedure, child(node, "with").Line, "PROCEDURE_UNSUPPORTED", "with is only valid for call, script, procedure, or verify actions")
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
			if pred := scalar(child(node, "verify")); pred != "" && !s.hasPredicate(pred) {
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
				} else if tool, ok := a.tools[operation]; !ok {
					a.errorAt(s.procedure, r.Line, "PROCEDURE_UNSUPPORTED", "reconcile target Operation is not imported: "+operation)
				} else {
					mutation := child(nodes, reconcile)
					if tool.effect != "MUTATION" {
						a.errorAt(s.procedure, r.Line, "PROCEDURE_RECONCILE_TARGET_INVALID", "reconcile target must be a MUTATION Operation: "+operation)
					}
					if scalar(child(child(mutation, "on"), "OUTCOME_UNKNOWN")) != n {
						a.errorAt(s.procedure, r.Line, "PROCEDURE_RECONCILE_TARGET_INVALID", "reconcile node must be the exact mutation's OUTCOME_UNKNOWN successor: "+reconcile)
					}
					if tool.tool.ReconcileOutputSchema == nil {
						a.errorAt(s.procedure, r.Line, "PROCEDURE_UNSUPPORTED", "reconcile target Tool has no declared reconcileOutputSchema: "+operation)
					}
				}
			}
			if script := scalar(child(node, "script")); script != "" {
				if !validSkillScriptPath(script) {
					a.errorAt(s.procedure, child(node, "script").Line, "PROCEDURE_UNSUPPORTED", "Script reference must be a normalized scripts/*.py path: "+script)
				}
				s.scripts[script] = child(node, "script").Line
				s.scriptRefPaths[script] = s.procedure
			}
			parsedNode := s.ast.Procedure.Procedures[pname].Nodes[n]
			if parsedNode.Action == ActionProcedure {
				ref := parsedNode.ProcedureRef
				if !ref.Explicit || ref.Skill == s.name {
					if ref.Procedure == "" || !s.procedures[ref.Procedure] {
						a.errorAt(s.procedure, ref.Span.Line, "PROCEDURE_NOT_FOUND", "missing local Procedure: "+ref.Procedure)
					} else {
						if s.procedureCalls[pname] == nil {
							s.procedureCalls[pname] = map[string]int{}
						}
						s.procedureCalls[pname][ref.Procedure] = ref.Span.Line
					}
				} else {
					s.deps[ref.Skill] = ref.Span.Line
					if s.procedureRefs[ref.Skill] == nil {
						s.procedureRefs[ref.Skill] = map[string]int{}
					}
					s.procedureRefs[ref.Skill][ref.Procedure] = ref.Span.Line
					if ref.Procedure == "" {
						a.errorAt(s.procedure, ref.Span.Line, "PROCEDURE_NOT_FOUND", "cross-Skill procedure reference requires procedure")
					}
				}
			}
			a.validateNodeOutcomeDomain(s.procedure, node, child(node, "on"))
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
						a.validatePhaseOutcomeDomain(s.procedure, scope, branch)
					} else {
						a.errorAtNode(s.procedure, scopeNode, "PROCEDURE_OUTCOME_MISSING", scope+" requires an explicit on route")
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
						if count != 1 || (id == "" && verify == "") || (stepName != nil && (stepName.Kind != yaml.ScalarNode || stepName.Tag != "!!str" || id == "")) {
							a.errorAt(s.procedure, step.Line, "PROCEDURE_UNSUPPORTED", "step requires exactly one script, call, or verify action; script and call steps require a name")
							continue
						}
						if id != "" && seenSteps[id] {
							a.errorAt(s.procedure, child(step, "name").Line, "PROCEDURE_UNSUPPORTED", "duplicate nested step name: "+id)
						}
						if id != "" {
							seenSteps[id] = true
						}
						if script != "" && !validSkillScriptPath(script) {
							a.errorAt(s.procedure, child(step, "script").Line, "PROCEDURE_UNSUPPORTED", "Script reference must be a normalized scripts/*.py path: "+script)
						}
						for _, field := range mapKeys(step) {
							switch field {
							case "name", "script", "call", "verify", "with":
							default:
								a.errorAt(s.procedure, child(step, field).Line, "PROCEDURE_UNSUPPORTED", "unknown nested step field: "+field)
							}
						}
						if verify != "" && !s.hasPredicate(verify) {
							a.errorAt(s.procedure, child(step, "verify").Line, "PREDICATE_NOT_FOUND", "unknown Predicate: "+verify)
						}
						if script != "" {
							s.scripts[script] = child(step, "script").Line
							s.scriptRefPaths[script] = s.procedure
						}
						if call != "" {
							s.calls[call] = child(step, "call").Line
						}
					}
				}
			}
			prior[n] = true
		}
		a.validateProcedureFlow(s.procedure, proc, start, names)
	}
	a.validateLocalProcedureCycles(s)
}

func (a *analysis) validateNodeOutcomeDomain(filename string, node, branches *yaml.Node) {
	action := ""
	for _, candidate := range []string{"call", "script", "procedure", "verify", "reconcile", "terminal"} {
		if child(node, candidate) != nil {
			action = candidate
			break
		}
	}
	allowed := map[string]bool{}
	switch action {
	case "verify":
		allowed = outcomeSet("PASS", "FAIL", "UNKNOWN")
	case "procedure":
		allowed = outcomeSet("SUCCEEDED", "FAILED", "UNRESOLVED", "SAFE_STOPPED")
	case "reconcile":
		allowed = outcomeSet("COMMITTED", "NOT_COMMITTED", "UNRESOLVED")
	case "script":
		allowed = outcomeSet("SUCCEEDED", "FAILED", "DENIED", "UNKNOWN", "UNRESOLVED")
	case "call":
		allowed = outcomeSet("COMMITTED", "NOT_COMMITTED", "OUTCOME_UNKNOWN", "FAILED", "DENIED", "SUCCEEDED", "UNKNOWN", "UNRESOLVED")
	}
	if action == "terminal" {
		value := scalar(child(node, "terminal"))
		if !outcomeSet("SUCCEEDED", "FAILED", "UNRESOLVED", "SAFE_STOPPED")[value] {
			a.errorAtNode(filename, child(node, "terminal"), "PROCEDURE_OUTCOME_INVALID", "terminal result must be SUCCEEDED, FAILED, UNRESOLVED, or SAFE_STOPPED")
		}
	}
	if len(allowed) == 0 {
		return
	}
	for _, key := range mapKeys(branches) {
		if !allowed[key] {
			a.errorAtNode(filename, child(branches, key), "PROCEDURE_OUTCOME_INVALID", fmt.Sprintf("outcome %q is not valid for %s action", key, action))
		}
	}
}

func (a *analysis) validatePhaseOutcomeDomain(filename, phase string, branches *yaml.Node) {
	var allowed map[string]bool
	if phase == "preconditions" {
		allowed = outcomeSet("FAILED", "UNRESOLVED")
	} else {
		allowed = outcomeSet("PASS", "FAIL", "UNKNOWN")
	}
	for _, key := range mapKeys(branches) {
		if !allowed[key] {
			a.errorAtNode(filename, child(branches, key), "PROCEDURE_OUTCOME_INVALID", fmt.Sprintf("outcome %q is not valid for %s", key, phase))
		}
	}
	for key := range allowed {
		if child(branches, key) == nil {
			a.errorAtNode(filename, branches, "PROCEDURE_OUTCOME_MISSING", fmt.Sprintf("%s route must declare %s", phase, key))
		}
	}
}

func outcomeSet(values ...string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func (a *analysis) validateProcedureFlow(filename string, proc *yaml.Node, start string, names []string) {
	nodes := child(proc, "nodes")
	known := make(map[string]bool, len(names))
	for _, name := range names {
		known[name] = true
	}
	if !known[start] {
		return
	}
	adj := make(map[string][]string, len(names))
	for _, name := range names {
		node := child(nodes, name)
		for _, scope := range []string{"on", "preconditions", "postconditions"} {
			branches := child(node, scope)
			if scope != "on" {
				branches = child(branches, "on")
			}
			for _, target := range mapValues(branches) {
				if known[target] {
					adj[name] = append(adj[name], target)
				}
			}
		}
	}
	reachable := map[string]bool{}
	var walk func(string)
	walk = func(current string) {
		if reachable[current] {
			return
		}
		reachable[current] = true
		for _, next := range adj[current] {
			walk(next)
		}
	}
	walk(start)
	for _, name := range names {
		if !reachable[name] {
			a.errorAtNode(filename, child(nodes, name), "PROCEDURE_NODE_UNREACHABLE", "node is unreachable from procedure start: "+name)
		}
	}
}

func mapValues(mapping *yaml.Node) []string {
	mapping = unalias(mapping)
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	values := make([]string, 0, len(mapping.Content)/2)
	for i := 1; i < len(mapping.Content); i += 2 {
		value := unalias(mapping.Content[i])
		if value.Kind == yaml.ScalarNode && value.Tag == "!!str" {
			values = append(values, value.Value)
		}
	}
	return values
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

func (a *analysis) validateStringSequence(filename string, list *yaml.Node, label, code string) {
	list = unalias(list)
	if list == nil || list.Kind != yaml.SequenceNode {
		a.errorAtNode(filename, list, code, label+" must be a sequence of nonempty strings")
		return
	}
	seen := map[string]bool{}
	for _, item := range list.Content {
		item = unalias(item)
		if item.Kind != yaml.ScalarNode || item.Tag != "!!str" || item.Value == "" {
			a.errorAtNode(filename, item, code, label+" entries must be nonempty strings")
			continue
		}
		if seen[item.Value] {
			a.errorAtNode(filename, item, code, label+" entries must be unique")
		}
		seen[item.Value] = true
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
				refPath := s.scriptRefPaths[script]
				if refPath == "" {
					refPath = s.procedure
				}
				a.errorAt(refPath, line, "SCRIPT_NOT_FOUND", "Script missing: "+script)
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
