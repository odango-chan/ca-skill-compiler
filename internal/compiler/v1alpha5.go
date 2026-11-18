package compiler

// CA-SL v1alpha5 static validation pipeline.
//
// v1alpha5 is one language target built from one pipeline. For every selected
// Skill the v1alpha5 Procedure/Wait AST is parsed, the WP5 Procedure
// control-flow graph is built and validated (Wait entry vertices,
// EVENT_RECEIVED/TIMEOUT control edges composed with the inherited graph, DAG,
// exact route targets and the final-Postcondition-PASS goal rule), the WP3
// trigger/correlation binding pass links every Wait selector to exactly one
// trusted Ingress Operation export and proves the triggering
// Operation/correlation contract on every reachable incoming path, the WP4
// wait.output pass binds that typed Producer to the exact normalized request
// Schema, proves event-accepted-path availability and applies conservative
// argument typing, and the WP6 response-boundary pass proves the paired
// response boundary, post-dominance, typed response-result Producer Output
// Schema compatibility and the Ingress-owned response contract/mapping
// completeness.
//
// CA-SL v1alpha5 inherits the released v1alpha4 normative semantics, so the same
// Skill run also executes validateInheritedSkillPackage (package, Contract,
// PredicateSet, Script, Dependency identity, Owner/Tool/Capability linking,
// Schema/CEL, Procedure/assurance, Evidence/Mutation/Reconcile/retry static
// checks) on the same normalized parse result. The Wait passes compose with the
// inherited graph instead of forming a second language pipeline.
//
// Build then assembles the candidate deterministic artifact tree through
// v1alpha5_artifact.go. Build success is static-compilation evidence only: it is
// not Runtime authorization, deployment qualification or product/5GC
// qualification. Remaining unvalidated candidate obligations are recorded in
// docs/known-limitations-v1alpha5.md.

import (
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

// validateV1Alpha5 is the v1alpha5 Build path. It re-runs the strict request
// admission and the full static pipeline, then assembles the deterministic
// artifact tree only when no error diagnostic was produced. Assembly failures
// fail the whole Build atomically with a clearly marked internal diagnostic;
// real source diagnostics are never masked.
func validateV1Alpha5(req BuildRequest) BuildResult {
	if diagnostics := validateBuildRequestV1Alpha5(req); len(diagnostics) > 0 {
		return failedV1Alpha5BuildResult(diagnostics)
	}
	diagnostics := checkV1Alpha5Sources(req)
	if hasErrorDiagnostic(diagnostics) {
		return failedV1Alpha5BuildResult(diagnostics)
	}
	result, err := assembleV1Alpha5SuccessResult(req)
	if err != nil {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Phase:    "C_STATIC",
			Code:     "CAPABILITY_INPUT_UNTRUSTED",
			Path:     "",
			Message:  limitDiagnosticMessage("fail-closed: v1alpha5 artifact assembly failed: " + err.Error()),
		})
		return failedV1Alpha5BuildResult(diagnostics)
	}
	return result
}

// checkV1Alpha5Sources parses every selected Skill package exactly once and
// returns the v1alpha5 Procedure/Wait diagnostics. Only the Procedure author
// file is consumed by WP2; the other YAML files are still parsed once so
// malformed YAML and duplicate keys surface as YAML_PARSE_ERROR.
func checkV1Alpha5Sources(req BuildRequest) []Diagnostic {
	var diagnostics []Diagnostic
	add := func(file string, n *yaml.Node, code, message string) {
		line, column := 0, 0
		if n != nil {
			line, column = n.Line, n.Column
		}
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Phase:    "C_STATIC",
			Code:     code,
			Path:     limitDiagnosticPath(file),
			Line:     line,
			Column:   column,
			Message:  limitDiagnosticMessage(message),
		})
	}

	files := make(map[string][]byte, len(req.Files))
	skillNames := make(map[string]bool)
	for _, file := range req.Files {
		if _, exists := files[file.Path]; exists {
			continue
		}
		files[file.Path] = file.Bytes
		parts := strings.Split(file.Path, "/")
		if len(parts) < 2 {
			continue
		}
		skillNames[parts[1]] = true
	}
	if len(skillNames) == 0 {
		add("", nil, "SOURCE_ROOT_INVALID", "No logical Skill package rooted at skills/<name>")
		return sortedV1Alpha5Diagnostics(diagnostics)
	}

	selected := make(map[string]bool)
	if len(req.Targets) == 0 {
		for name := range skillNames {
			selected[name] = true
		}
	} else {
		for _, target := range req.Targets {
			if !skillNames[target] {
				add("", nil, "SOURCE_ROOT_INVALID", "Selected Skill target does not exist in BuildRequest.files: "+target)
				continue
			}
			selected[target] = true
		}
	}

	for _, name := range sortedStringMapKeys(selected) {
		base := "skills/" + name + "/"
		procedurePath := base + "assurance/procedure.yaml"

		// Parse every YAML author file exactly once; the Procedure AST and the
		// Wait declaration share these normalized nodes.
		parsedFiles := make(map[string]*yaml.Node)
		for _, filename := range sortedStringMapKeys(files) {
			if strings.HasPrefix(filename, base) &&
				(strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")) {
				parsedFiles[filename] = v1alpha5ParseYAML(filename, files[filename], add)
			}
		}
		if _, exists := parsedFiles[procedurePath]; !exists {
			parsedFiles[procedurePath] = v1alpha5ParseYAML(procedurePath, nil, add)
		}

		skill, skillDiagnostics := parseV1Alpha5SkillAST(name, parsedFiles)
		diagnostics = append(diagnostics, skillDiagnostics...)
		// WP5: validate the v1alpha5 Procedure control-flow graph after the AST
		// parse, in the same static pipeline as the WP2 Wait rules.
		diagnostics = append(diagnostics, validateV1Alpha5ProcedureGraph(procedurePath, skill)...)
		// WP3: after graph validation, exactly link every Wait selector to one
		// trusted Ingress Operation export and validate the trigger/correlation
		// binding on every reachable incoming path.
		bindingDiagnostics, waitBindingSource := validateV1Alpha5WaitBindings(req, skill, procedurePath, parsedFiles)
		diagnostics = append(diagnostics, bindingDiagnostics...)
		// WP4: after WP3 binding, bind every wait.output Producer SourcePath to the
		// exact linked normalized request Schema, reject unavailable paths using
		// WP5 event-accepted-edge dominance and apply conservative argument typing.
		// Build stays fail-closed below for the later work packages.
		diagnostics = append(diagnostics, validateV1Alpha5WaitOutputSourcePaths(skill, procedurePath, waitBindingSource)...)
		// WP9: run the complete inherited v1alpha5 static semantic pipeline on the
		// same normalized parse result. CA-SL v1alpha5 inherits the released
		// v1alpha4 package, dependency, Owner, Tool, Capability, Schema, CEL,
		// Predicate, Script, Procedure and assurance obligations; Appendix L only
		// adds the node-entry Wait declaration, which the inherited procedure pass
		// admits through the v1alpha5 allowance.
		inheritedDiagnostics, _ := validateInheritedSkillPackage(
			req, name, files, parsedFiles,
			inheritedValidationOptions{
				allowWait:             true,
				missingSourceMessage:  "Required v1alpha5 author source is missing",
				additionalUsedAliases: v1alpha5IngressDependencyAliases(req, skill, parsedFiles),
			},
			add,
		)
		diagnostics = append(diagnostics, inheritedDiagnostics...)
	}

	return sortedV1Alpha5Diagnostics(diagnostics)
}

// v1alpha5IngressDependencyAliases returns the Contract dependency aliases that
// supply an IngressCatalog consumed by at least one Node-entry Wait of this
// Skill. The inherited dependency-identity pass otherwise reports such an alias
// as unused because a Wait selector is not a qualified author reference, even
// though the released Ingress contract is genuinely linked through it.
func v1alpha5IngressDependencyAliases(req BuildRequest, skill *V1Alpha5SkillAST, parsedFiles map[string]*yaml.Node) map[string]bool {
	used := map[string]bool{}
	if skill == nil || skill.Procedure == nil {
		return used
	}
	hasWait := false
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		if node := skill.Procedure.ByID[nodeID]; node != nil && node.Wait != nil {
			hasWait = true
			break
		}
	}
	if !hasWait {
		return used
	}
	contractNode := parsedFiles["skills/"+skill.Name+"/assurance/contract.yaml"]
	dependencies := child(contractNode, "dependencies")
	if dependencies == nil || dependencies.Kind != yaml.MappingNode {
		return used
	}
	snapshots := make(map[string]ResolvedDependency, len(req.ResolvedDependencies))
	for _, dependency := range req.ResolvedDependencies {
		if _, exists := snapshots[dependency.DependencyID]; !exists {
			snapshots[dependency.DependencyID] = dependency
		}
	}
	for i := 0; i+1 < len(dependencies.Content); i += 2 {
		alias := scalar(dependencies.Content[i])
		snapshot, exists := snapshots[alias]
		if !exists {
			continue
		}
		for _, file := range snapshot.Files {
			if !strings.HasSuffix(file.Path, ".yaml") && !strings.HasSuffix(file.Path, ".yml") {
				continue
			}
			root, err := yaml.Parse(file.Bytes)
			if err != nil {
				continue
			}
			if scalar(child(root, "apiVersion")) == "connectionagent.io/ingresscatalog/v1alpha1" &&
				scalar(child(root, "kind")) == "IngressCatalog" {
				used[alias] = true
				break
			}
		}
	}
	return used
}

func v1alpha5ParseYAML(file string, data []byte, add func(string, *yaml.Node, string, string)) *yaml.Node {
	if data == nil {
		add(file, nil, "SOURCE_ROOT_INVALID", "Required v1alpha5 author source is missing")
		return nil
	}
	root, err := yaml.Parse(data)
	if err != nil {
		line, column := yaml.ParseErrorLocation(err)
		node := &yaml.Node{Line: line, Column: column}
		code := "YAML_PARSE_ERROR"
		if yaml.IsRootError(err) {
			code = "SOURCE_ROOT_INVALID"
		}
		add(file, node, code, err.Error())
		return nil
	}
	return root
}

func sortedV1Alpha5Diagnostics(diagnostics []Diagnostic) []Diagnostic {
	diagnostics = dedupeV1Alpha5Diagnostics(diagnostics)
	sortDiagnostics(diagnostics)
	if diagnostics == nil {
		return []Diagnostic{}
	}
	return diagnostics
}

// dedupeV1Alpha5Diagnostics drops duplicates produced when the inherited
// procedure pass and the v1alpha5 Wait AST pass both validate the same shared
// Procedure structure. Two diagnostics are duplicates only when every wire
// identity field (severity, phase, code, path, line, column and message) is
// identical, so distinct findings at the same location are always preserved.
func dedupeV1Alpha5Diagnostics(diagnostics []Diagnostic) []Diagnostic {
	if len(diagnostics) < 2 {
		return diagnostics
	}
	type key struct {
		severity, phase, code, path, message string
		line, column                         int
	}
	seen := make(map[key]bool, len(diagnostics))
	kept := make([]Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		k := key{
			severity: diagnostic.Severity, phase: diagnostic.Phase,
			code: diagnostic.Code, path: diagnostic.Path,
			line: diagnostic.Line, column: diagnostic.Column,
			message: diagnostic.Message,
		}
		if seen[k] {
			continue
		}
		seen[k] = true
		kept = append(kept, diagnostic)
	}
	return kept
}
