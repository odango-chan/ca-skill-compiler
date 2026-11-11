package compiler

import (
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type skillEntry struct {
	name      string
	contract  string
	procedure string
	calls     map[string]int
	scripts   map[string]int
	deps      map[string]int
	preds     map[string]int
	procedures map[string]bool
	allowed    map[string]bool
}

func (a *analysis) parseSkills() {
	names := make(map[string]bool)
	for filename := range a.files {
		if !strings.HasPrefix(filename, "skills/") { continue }
		p := strings.Split(filename, "/")
		if len(p) >= 3 { names[p[1]] = true }
	}
	sorted := make([]string, 0, len(names))
	for name := range names { sorted = append(sorted, name) }
	sort.Strings(sorted)
	for _, name := range sorted {
		base := "skills/"+name+"/"
		s := &skillEntry{name: name, contract: base+"assurance/contract.yaml", procedure: base+"assurance/procedure.yaml",
			calls: make(map[string]int), scripts: make(map[string]int), deps: make(map[string]int), preds: make(map[string]int),
			procedures: make(map[string]bool), allowed: make(map[string]bool)}
		a.skills[name] = s
		if _, ok := a.files[base+"SKILL.md"]; !ok {
			a.errorAt(base+"SKILL.md", 0, "SKILL_MISSING", "SKILL.md is required")
		}
		for _, filename := range []string{s.contract, s.procedure} {
			if _, ok := a.files[filename]; !ok { a.errorAt(filename, 0, "SOURCE_MISSING", "required Skill source missing") }
		}
		if _, ok := a.files[s.contract]; ok { a.parseContract(s) }
		if _, ok := a.files[s.procedure]; ok { a.parseProcedure(s) }
		if filename := base+"assurance/predicates.yaml"; a.files[filename] != nil {
			root := a.parseYAML(filename)
			for _, pred := range mapKeys(child(root, "predicates")) { s.preds[pred] = 1 }
		}
	}
	if len(a.skills) == 0 { a.errorAt("skills/", 0, "NO_SKILLS", "project contains no Skill") }
}

func (a *analysis) parseContract(s *skillEntry) {
	root := a.parseYAML(s.contract)
	if root == nil { return }
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
	if entry == "" { a.errorAt(s.contract, spec.Line, "ENTRY_MISSING", "missing spec.entryProcedure") }
	s.procedures["__entry__"+entry] = true
	for _, capability := range sequence(child(child(spec, "allowedResources"), "capabilities")) {
		if v := scalar(capability); v != "" { s.allowed[v] = true }
	}
	for _, dep := range sequence(child(spec, "dependencies")) {
		id := scalar(child(dep, "skill"))
		if id == "" { id = scalar(child(dep, "name")) }
		if id != "" { s.deps[id] = dep.Line }
	}
	if schema := scalar(child(child(spec, "inputs"), "requestSchema")); schema != "" {
		if !strings.HasPrefix(schema, "schemas/") || a.files["skills/"+s.name+"/"+schema] == nil {
			a.errorAt(s.contract, spec.Line, "REQUEST_SCHEMA_MISSING", "requestSchema not found inside Skill: "+schema)
		}
	}
}

func (a *analysis) parseProcedure(s *skillEntry) {
	root := a.parseYAML(s.procedure)
	if root == nil { return }
	procedures := child(root, "procedures")
	if procedures == nil {
		a.errorAt(s.procedure, root.Line, "PROCEDURES_MISSING", "missing procedures map")
		return
	}
	for _, name := range mapKeys(procedures) { s.procedures[name] = true }
	for key := range s.procedures {
		if strings.HasPrefix(key, "__entry__") && !s.procedures[strings.TrimPrefix(key, "__entry__")] {
			a.errorAt(s.contract, 0, "ENTRY_NOT_FOUND", "entryProcedure does not exist: "+strings.TrimPrefix(key, "__entry__"))
		}
	}
	inspectMapping(procedures, func(key string, value *yaml.Node) {
		switch key {
		case "call":
			if id := scalar(value); id != "" { s.calls[id] = value.Line }
		case "script":
			if id := scalar(value); strings.HasPrefix(id, "scripts/") { s.scripts[id] = value.Line }
		case "procedure":
			if name := scalar(child(value, "skill")); name != "" && name != s.name {
				s.deps[name] = value.Line
			} else if name := scalar(value); name != "" && !s.procedures[name] {
				a.errorAt(s.procedure, value.Line, "PROCEDURE_NOT_FOUND", "missing local Procedure: "+name)
			}
		}
	})
}

func (a *analysis) selectSkills(targets []string) map[string]bool {
	selected := make(map[string]bool)
	var visit func(name string)
	visit = func(name string) {
		if selected[name] { return }
		s, ok := a.skills[name]
		if !ok {
			a.errorAt("skills/"+name, 0, "DEPENDENCY_NOT_FOUND", "Skill dependency is missing: "+name)
			return
		}
		selected[name] = true
		for dep := range s.deps { visit(dep) }
	}
	for _, name := range targets { visit(name) }
	return selected
}

func (a *analysis) validateSkills(selected map[string]bool) {
	names := make([]string, 0, len(selected))
	for name := range selected { names = append(names, name) }
	sort.Strings(names)
	for _, name := range names {
		s := a.skills[name]
		if s == nil { continue }
		for dep, line := range s.deps {
			if a.skills[dep] == nil {
				a.errorAt(s.procedure, line, "DEPENDENCY_NOT_FOUND", "referenced Skill is missing: "+dep)
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
}
