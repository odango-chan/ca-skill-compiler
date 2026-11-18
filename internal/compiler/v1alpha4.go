package compiler

// CA-SL v1alpha4 source/graph admission only, not product qualification.
// Frozen normative authority: ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
	"regexp"
	"sort"
	"strings"
)

const V1Alpha4Target = "connectionagent.io/v1alpha4"

// v1alpha4ResidualFailClosedMessage marks the placeholder error that keeps Build
// fail-closed while later work packages are unimplemented. It is not a language
// diagnostic identity and must disappear as the pipeline is completed.
const v1alpha4ResidualFailClosedMessage = "v1alpha4 semantic linking and Owner qualification are not yet implemented; compilation fails closed"

var v4NodeID = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
var v4ContractGate = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*[.]precondition$`)
var v4Terminals = map[string]bool{"SUCCEEDED": true, "FAILED": true, "SAFE_STOPPED": true, "UNRESOLVED": true}
var v4Stages = map[string]bool{"precondition": true, "operation": true, "observation": true, "postcondition": true}
var v4ContractFields = map[string]bool{"goal": true, "precondition": true, "requestSchema": true, "contextSchema": true, "dependencies": true, "capabilities": true, "limits": true}
var v4PredicateFields = map[string]bool{"parameters": true, "evidence": true, "evaluator": true}

// CheckV1Alpha4 is a convenience wrapper around the only accepted v1alpha4 API.
func CheckV1Alpha4(req BuildRequest) []Diagnostic {
	return Check(req)
}

func validateV1Alpha4(req BuildRequest) BuildResult {
	diagnostics := make([]Diagnostic, 0)
	errorAt := func(file string, n *yaml.Node, code, message string) {
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
	if req.LanguageTarget != V1Alpha4Target {
		errorAt("", nil, "LANGUAGE_TARGET_UNSUPPORTED", "Unsupported CA-SL target: "+req.LanguageTarget)
		return failedBuildResult(diagnostics)
	}

	files := make(map[string][]byte, len(req.Files))
	skillNames := make(map[string]bool)
	for _, file := range req.Files {
		if !validSourcePath(file.Path) {
			errorAt(file.Path, nil, "PACKAGE_PATH_INVALID", "Input must use a normalized skills/<skill-name>/... source path")
			continue
		}
		if _, exists := files[file.Path]; exists {
			errorAt(file.Path, nil, "PACKAGE_PATH_INVALID", "Duplicate source file path")
			continue
		}
		if len(file.Bytes) > maxFileSize {
			errorAt(file.Path, nil, "INPUT_LIMIT", "File exceeds the 2 MiB raw byte limit")
			continue
		}
		files[file.Path] = file.Bytes
		parts := strings.Split(file.Path, "/")
		skillNames[parts[1]] = true
	}
	if len(skillNames) == 0 {
		errorAt("", nil, "SOURCE_ROOT_INVALID", "No logical Skill package rooted at skills/<name>")
	}

	selected := make(map[string]bool)
	if len(req.Targets) == 0 {
		for name := range skillNames {
			selected[name] = true
		}
	} else {
		for _, target := range req.Targets {
			if !skillNames[target] {
				errorAt("", nil, "SOURCE_ROOT_INVALID", "Selected Skill target does not exist in BuildRequest.files: "+target)
				continue
			}
			selected[target] = true
		}
	}
	names := make([]string, 0, len(selected))
	for name := range selected {
		names = append(names, name)
	}
	sort.Strings(names)

	artifactInputs := map[string]artifactInput{}
	for _, name := range names {
		inheritedDiagnostics, input := validateInheritedSkillPackage(
			req, name, files, nil,
			inheritedValidationOptions{missingSourceMessage: "Required v1alpha4 author source is missing"},
			errorAt,
		)
		diagnostics = append(diagnostics, inheritedDiagnostics...)
		if input.skill != nil && input.skill.Contract != nil && input.skill.Contract.Raw != nil {
			artifactInputs[name] = input
		}
	}

	if !hasErrorDiagnostic(diagnostics) {
		if reason := unsupportedSemanticsReason(artifactInputs); reason != "" {
			errorAt("", nil, "CAPABILITY_INPUT_UNTRUSTED", "fail-closed: "+reason)
		}
	}
	if hasErrorDiagnostic(diagnostics) {
		return failedBuildResult(diagnostics)
	}
	if result, ok := assembleV1Alpha4SuccessResult(req, names, artifactInputs, diagnostics, errorAt); ok {
		return result
	}
	return failedBuildResult(diagnostics)
}

func v4ParseYAML(file string, data []byte, add func(string, *yaml.Node, string, string)) *yaml.Node {
	if data == nil {
		add(file, nil, "SOURCE_ROOT_INVALID", "Required v1alpha4 author source is missing")
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

func v4Allowed(node *yaml.Node, file string, allowed map[string]bool, add func(string, *yaml.Node, string, string)) {
	if node == nil || node.Kind != yaml.MappingNode {
		add(file, node, "SOURCE_ROOT_INVALID", "Expected a mapping")
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		name := node.Content[i].Value
		if allowed[name] {
			continue
		}
		switch v4LegacyFieldClass(name) {
		case "wait":
			add(file, node.Content[i], "UNSUPPORTED_WAIT", "Wait/Await control structures are not part of CA-SL v1alpha4")
		case "v02":
			add(file, node.Content[i], "UNSUPPORTED_V02_SYNTAX", "Recognized legacy v0.2 syntax is not part of CA-SL v1alpha4")
		default:
			add(file, node.Content[i], "UNKNOWN_FIELD", "Unsupported v1alpha4 author field "+name)
		}
	}
}
func v4CheckPredicates(file string, source *V1Alpha4PredicateSetAST, add func(string, *yaml.Node, string, string)) {
	if source == nil || source.Raw == nil {
		return
	}
	root := source.Raw
	v4Allowed(root, file, map[string]bool{"predicates": true}, add)
	set := child(root, "predicates")
	if set == nil || set.Kind != yaml.MappingNode {
		add(file, set, "SOURCE_ROOT_INVALID", "PredicateSet must have predicates mapping")
		return
	}
	for i := 0; i+1 < len(set.Content); i += 2 {
		if key := unalias(set.Content[i]); key == nil || !validExportID(scalar(key)) {
			add(file, key, "IDENTIFIER_INVALID", "PredicateId must satisfy the Dotted Identifier grammar")
		}
		def := set.Content[i+1]
		v4Allowed(def, file, v4PredicateFields, add)
		params := child(def, "parameters")
		if params == nil || params.Kind != yaml.SequenceNode {
			add(file, params, "PREDICATE_GROUP_INVALID", "parameters must be an ordered parameter list")
		} else {
			for _, item := range params.Content {
				if item.Kind != yaml.MappingNode || len(item.Content) != 2 {
					add(file, item, "PREDICATE_GROUP_INVALID", "Each parameter is a one-key declaration")
					continue
				}
				if keyNode := unalias(item.Content[0]); keyNode == nil || !validDependencyID(scalar(keyNode)) {
					add(file, keyNode, "IDENTIFIER_INVALID", "Predicate parameter name must satisfy the Identifier grammar")
				}
			}
		}
	}
}

func v4CheckProcedure(file string, source *V1Alpha4ProcedureAST, contract *V1Alpha4ContractAST, add func(string, *yaml.Node, string, string), allowWait bool) {
	if source == nil || source.Raw == nil {
		return
	}
	root := source.Raw
	procedureFields := map[string]bool{"start": true, "nodes": true}
	if goalNode := child(root, "goal"); goalNode != nil {
		add(file, goalNode, "SOURCE_ROOT_INVALID", "Procedure must not declare a goal; the goal belongs to the Contract")
		procedureFields["goal"] = true
	}
	if inputsNode := child(root, "inputs"); inputsNode != nil {
		add(file, inputsNode, "SOURCE_ROOT_INVALID", "Procedure root must not declare the removed inputs field")
		procedureFields["inputs"] = true
	}
	v4Allowed(root, file, procedureFields, add)
	start := scalar(source.Start)
	nodes := source.Nodes
	if nodes == nil || nodes.Kind != yaml.MappingNode {
		add(file, nodes, "SOURCE_ROOT_INVALID", "Procedure.nodes must be a mapping")
		return
	}
	ids := map[string]bool{}
	for _, id := range mapKeys(nodes) {
		ids[id] = true
	}
	if !ids[start] {
		add(file, child(root, "start"), "PROCEDURE_START_MISSING", "start must resolve to a declared Node")
	}
	adjacency := map[string][]string{}
	for i := 0; i+1 < len(nodes.Content); i += 2 {
		id, n := nodes.Content[i].Value, nodes.Content[i+1]
		if !v4NodeID.MatchString(id) || v4Terminals[id] || id == "context" {
			add(file, nodes.Content[i], "NODE_ID_RESERVED", "Invalid or reserved Node ID")
		}
		nodeFields := v4Stages
		if allowWait {
			// CA-SL v1alpha5 Appendix L.1 adds the optional node-entry wait
			// declaration to the inherited Node shape. It is not a fifth business
			// stage: the Wait-only structural rules are owned by the v1alpha5
			// Wait AST/graph passes, while the inherited business-stage rules
			// below still apply to a Wait-bearing Node.
			nodeFields = map[string]bool{
				"precondition": true, "operation": true,
				"observation": true, "postcondition": true, "wait": true,
			}
		}
		v4Allowed(n, file, nodeFields, add)
		stageOrder := []string{"precondition", "operation", "observation", "postcondition"}
		nStage := 0
		for _, stage := range stageOrder {
			if child(n, stage) != nil {
				nStage++
			}
		}
		if nStage == 0 {
			add(file, n, "SOURCE_ROOT_INVALID", "Node must contain a stage")
		}
		for _, stage := range stageOrder {
			body := child(n, stage)
			if body == nil {
				continue
			}
			allowed := map[string]bool{"routes": true}
			if stage == "precondition" || stage == "postcondition" {
				allowed["predicates"] = true
			} else {
				allowed["tool"] = true
				allowed["arguments"] = true
				if stage == "operation" {
					allowed["script"] = true
					allowed["retryOf"] = true
				}
			}
			if stage == "postcondition" {
				if goalNode := child(body, "goal"); goalNode != nil {
					add(file, goalNode, "PREDICATE_GROUP_INVALID", "Postcondition must not declare the removed goal field")
					allowed["goal"] = true
				}
			}
			v4Allowed(body, file, allowed, add)
			if stage == "precondition" || stage == "postcondition" {
				calls := child(body, "predicates")
				if calls == nil || calls.Kind != yaml.SequenceNode || len(calls.Content) == 0 {
					add(file, calls, "PREDICATE_GROUP_INVALID", "Stage predicates must be a nonempty ordered list")
				} else {
					for _, call := range calls.Content {
						v4Allowed(call, file, map[string]bool{"predicate": true, "arguments": true}, add)
						args := child(call, "arguments")
						predicateNode := child(call, "predicate")
						if scalar(predicateNode) == "" || args == nil || args.Kind != yaml.SequenceNode {
							add(file, call, "PREDICATE_GROUP_INVALID", "PredicateCall requires predicate and positional arguments")
							continue
						}
						if !validExportID(scalar(predicateNode)) {
							add(file, predicateNode, "IDENTIFIER_INVALID", "PredicateCall predicate must satisfy the frozen PredicateId (DottedId) grammar")
						}
					}
				}
			} else {
				tool, script := child(body, "tool"), child(body, "script")
				if (tool == nil) == (script == nil) || (stage == "observation" && script != nil) {
					add(file, body, "CONTROL_FLOW_AMBIGUOUS", "Operation needs exactly one tool/script; Observation needs tool")
				}
				if args := child(body, "arguments"); args == nil || args.Kind != yaml.MappingNode {
					add(file, body, "SOURCE_ROOT_INVALID", "Tool or Script arguments must be a mapping")
				} else {
					for j := 1; j < len(args.Content); j += 2 {
						v := args.Content[j]
						if v.Kind != yaml.ScalarNode {
							add(file, v, "SOURCE_PATH_INVALID", "Arguments must be scalar SourcePaths or supported literals")
						}
					}
				}
			}
			routes := child(body, "routes")
			if routes != nil && routes.Kind != yaml.MappingNode {
				add(file, routes, "SOURCE_ROOT_INVALID", "routes must be a mapping")
				continue
			}
			if stage == "precondition" || stage == "postcondition" {
				for _, required := range []string{"FAIL", "UNKNOWN"} {
					if child(routes, required) == nil {
						add(file, body, "ROUTE_OUTCOME_MISSING", "Predicate Stage missing "+required)
					}
				}
			}
			for _, outcome := range mapKeys(routes) {
				target := scalar(child(routes, outcome))
				if target == "" || (!ids[target] && !v4Terminals[target]) {
					add(file, child(routes, outcome), "CONTROL_FLOW_DANGLING", "Route value is neither a terminal nor an existing Node")
				} else if ids[target] {
					adjacency[id] = append(adjacency[id], target)
				}
				if outcome == "OUTCOME_UNKNOWN" && target != "UNRESOLVED" && target != "SAFE_STOPPED" {
					add(file, child(routes, outcome), "ROUTE_OUTCOME_UNKNOWN", "OUTCOME_UNKNOWN may only safely stop")
				}
			}
			// Owner Effect and SourcePath assignability remain unverified until exact
			// Tool/Schema publication. No guessed Owner result classification.
		}
	}
	// Final-Postcondition precision: only a reachable final Postcondition PASS
	// edge proves the Goal (E42/E117/E120). An unreachable legal-looking edge is
	// reported as UNREACHABLE_NODE by the graph slice and must not mask a missing
	// reachable goal proof or legitimize a bypass.
	reachable := map[string]bool{}
	if _, declared := ids[start]; declared {
		queue := []string{start}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			if reachable[current] {
				continue
			}
			reachable[current] = true
			queue = append(queue, adjacency[current]...)
		}
	}
	anySuccess, legalSuccess := false, false
	for id := range reachable {
		n := child(nodes, id)
		for _, stage := range []string{"precondition", "operation", "observation", "postcondition"} {
			routes := child(child(n, stage), "routes")
			for _, outcome := range mapKeys(routes) {
				if scalar(child(routes, outcome)) != "SUCCEEDED" {
					continue
				}
				anySuccess = true
				if stage == "postcondition" && outcome == "PASS" {
					legalSuccess = true
				}
			}
		}
	}
	if anySuccess && !legalSuccess {
		add(file, root, "FINAL_POSTCONDITION_MISSING", "Only a final Postcondition PASS may request SUCCEEDED")
	} else if legalSuccess {
		for id := range reachable {
			n := child(nodes, id)
			for _, stage := range []string{"precondition", "operation", "observation", "postcondition"} {
				for _, outcome := range mapKeys(child(child(n, stage), "routes")) {
					if scalar(child(child(child(n, stage), "routes"), outcome)) == "SUCCEEDED" &&
						(stage != "postcondition" || outcome != "PASS") {
						add(file, child(child(n, stage), "routes"), "SUCCESS_BYPASSES_FINAL_POSTCONDITION", "Non-final stage requests SUCCEEDED")
					}
				}
			}
		}
	}
	visited := map[string]bool{}
	active := map[string]bool{}
	cycle := false
	var visit func(string)
	visit = func(id string) {
		if active[id] {
			cycle = true
			return
		}
		if visited[id] {
			return
		}
		visited[id] = true
		active[id] = true
		for _, child := range adjacency[id] {
			visit(child)
		}
		active[id] = false
	}
	if ids[start] {
		visit(start)
	}
	if cycle {
		add(file, root, "CONTROL_FLOW_CYCLE", "Procedure must be a DAG")
	}
	if contract != nil && contract.Precondition != nil {
		gate := scalar(contract.Precondition)
		if gate != "" {
			parts := strings.Split(gate, ".")
			if len(parts) != 2 || !ids[parts[0]] || child(child(nodes, parts[0]), "precondition") == nil {
				add(file, contract.Precondition, "ENTRY_PRECONDITION_REF_INVALID", "Contract precondition must resolve to a declared Precondition stage")
			} else if ids[start] && !reachable[parts[0]] {
				// E52 sixth variant: a syntactically valid Precondition stage that is
				// not in the start-reachable DAG is an invalid global reference and
				// takes precedence over the entry-bypass diagnosis.
				add(file, contract.Precondition, "ENTRY_PRECONDITION_REF_INVALID", "Contract precondition must resolve to a start-reachable Precondition stage")
			}
		}
	}
}
