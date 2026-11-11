package compiler

import (
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type skillEntry struct {
	name           string
	contract       string
	procedure      string
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
	deps           map[string]int
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
			calls: make(map[string]int), scripts: make(map[string]int), deps: make(map[string]int), preds: make(map[string]int),
			procedures: make(map[string]bool), allowed: make(map[string]bool), nodes: make(map[string]map[string]*yaml.Node), procedureRefs: make(map[string]map[string]int), libraryFiles: make(map[string]bool)}
		a.skills[name] = s
		if _, ok := a.files[base+"SKILL.md"]; !ok {
			a.errorAt(base+"SKILL.md", 0, "SKILL_MISSING", "SKILL.md is required")
		}
		for _, filename := range []string{s.contract, s.procedure} {
			if _, ok := a.files[filename]; !ok {
				a.errorAt(filename, 0, "SOURCE_MISSING", "required Skill source missing")
			}
		}
		if _, ok := a.files[s.contract]; ok {
			a.parseContract(s)
		}
		if _, ok := a.files[s.procedure]; ok {
			a.parseProcedure(s)
		}
		if filename := base + "assurance/predicates.yaml"; a.files[filename] != nil {
			root := a.parseYAML(filename)
			for _, pred := range mapKeys(child(root, "predicates")) {
				s.preds[pred] = 1
			}
		}
	}
	if len(a.skills) == 0 {
		a.errorAt("skills/", 0, "NO_SKILLS", "project contains no Skill")
	}
}

func (a *analysis) parseContract(s *skillEntry) {
	root := a.parseYAML(s.contract)
	if root == nil {
		return
	}
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
	if entry == "" {
		a.errorAt(s.contract, spec.Line, "ENTRY_MISSING", "missing spec.entryProcedure")
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
		if !strings.HasPrefix(schema, "lib/") {
			filename = "skills/" + s.name + "/" + schema
		} else {
			s.libraryFiles[filename] = true
		}
		if a.files[filename] == nil {
			a.errorAt(s.contract, spec.Line, "REQUEST_SCHEMA_MISSING", "requestSchema not found: "+schema)
		}
	}
	s.contextSchemas = map[string]string{}
	for _, key := range []string{"subject", "facts", "observations"} {
		if p := scalar(child(child(child(spec, "context"), "schemas"), key)); p != "" {
			s.contextSchemas[key] = p
			if strings.HasPrefix(p, "lib/") {
				s.libraryFiles[p] = true
			}
		}
	}
}

func (a *analysis) parseProcedure(s *skillEntry) {
	root := a.parseYAML(s.procedure)
	if root == nil {
		return
	}
	procedures := child(root, "procedures")
	if procedures == nil {
		a.errorAt(s.procedure, root.Line, "PROCEDURES_MISSING", "missing procedures map")
		return
	}
	for _, name := range mapKeys(procedures) {
		s.procedures[name] = true
	}
	if s.entry != "" && !s.procedures[s.entry] {
		a.errorAt(s.contract, 0, "ENTRY_NOT_FOUND", "entryProcedure does not exist: "+s.entry)
	}
	for _, pname := range mapKeys(procedures) {
		proc := child(procedures, pname)
		for _, field := range mapKeys(proc) {
			switch field {
			case "inputs", "start", "nodes":
			default:
				a.errorAt(s.procedure, child(proc, field).Line, "PROCEDURE_UNSUPPORTED", "unknown Procedure field: "+field)
			}
		}
		start := scalar(child(proc, "start"))
		nodes := child(proc, "nodes")
		if start == "" || child(nodes, start) == nil {
			a.errorAt(s.procedure, proc.Line, "NODE_TARGET_UNKNOWN", "procedure start must name a node: "+start)
		}
		names := mapKeys(nodes)
		known := make(map[string]bool)
		for _, n := range names {
			known[n] = true
		}
		for _, n := range names {
			node := child(nodes, n)
			actions := 0
			for _, action := range []string{"call", "script", "procedure", "verify", "reconcile", "terminal"} {
				if child(node, action) != nil {
					actions++
				}
			}
			if actions != 1 {
				a.errorAt(s.procedure, node.Line, "PROCEDURE_UNSUPPORTED", "each node must have exactly one supported action")
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
			if reconcile := scalar(child(node, "reconcile")); reconcile != "" && !known[reconcile] {
				a.errorAt(s.procedure, child(node, "reconcile").Line, "NODE_TARGET_UNKNOWN", "reconcile target is not a node: "+reconcile)
			}
			if script := scalar(child(node, "script")); script != "" {
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
				if skill == "" {
					target = scalar(ref)
					if target == "" || !s.procedures[target] {
						a.errorAt(s.procedure, ref.Line, "PROCEDURE_NOT_FOUND", "missing local Procedure: "+target)
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
				if unalias(branch).Kind != yaml.MappingNode {
					a.errorAt(s.procedure, branch.Line, "PROCEDURE_UNSUPPORTED", "on must map outcomes to node names")
				}
				for _, outcome := range mapKeys(branch) {
					target := scalar(child(branch, outcome))
					if !known[target] {
						a.errorAt(s.procedure, child(branch, outcome).Line, "NODE_TARGET_UNKNOWN", "unknown outcome target: "+target)
					}
				}
			}
			for _, scope := range []string{"preconditions", "postconditions"} {
				if steps := child(child(node, scope), "steps"); steps != nil {
					for _, step := range sequence(steps) {
						id := scalar(child(step, "name"))
						script := scalar(child(step, "script"))
						call := scalar(child(step, "call"))
						verify := scalar(child(step, "verify"))
						count := 0
						if script != "" {
							count++
						}
						if call != "" {
							count++
						}
						if verify != "" {
							count++
						}
						if id == "" || count != 1 {
							a.errorAt(s.procedure, step.Line, "PROCEDURE_UNSUPPORTED", "step requires name and exactly one script or call action")
							continue
						}
						for _, field := range mapKeys(step) {
							switch field {
							case "name", "script", "call", "verify", "with", "on":
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
		}
		a.validateProcedureBindings(s, proc, nodes, s.procedure)
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
		for dep := range s.deps {
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
		for dep, line := range s.deps {
			if a.skills[dep] == nil {
				a.errorAt(s.procedure, line, "DEPENDENCY_NOT_FOUND", "referenced Skill is missing: "+dep)
			} else {
				for procedure, refLine := range s.procedureRefs[dep] {
					if !a.skills[dep].procedures[procedure] {
						a.errorAt(s.procedure, refLine, "PROCEDURE_NOT_FOUND", "referenced cross-Skill Procedure is missing: "+dep+"/"+procedure)
					}
				}
			}
		}
		for script, line := range s.scripts {
			if a.files["skills/"+name+"/"+script] == nil {
				a.errorAt(s.procedure, line, "SCRIPT_NOT_FOUND", "Script missing: "+script)
			}
		}
		for op, line := range s.calls {
			if _, ok := a.tools[op]; !ok {
				a.errorAt(s.procedure, line, "TOOL_NOT_FOUND", "Tool operation not imported: "+op)
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
			for dep := range s.deps {
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
