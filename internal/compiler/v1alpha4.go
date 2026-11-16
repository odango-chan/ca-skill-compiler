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

	for _, name := range names {
		base := "skills/" + name + "/"
		contractPath := base + "assurance/contract.yaml"
		procedurePath := base + "assurance/procedure.yaml"
		predicatesPath := base + "assurance/predicates.yaml"
		if files[base+"SKILL.md"] == nil {
			errorAt(base+"SKILL.md", nil, "SOURCE_ROOT_INVALID", "SKILL.md missing")
		}

		// Parse every YAML author file exactly once. Contract, Procedure,
		// PredicateSet and local Schema checks share these normalized nodes.
		parsedFiles := make(map[string]*yaml.Node)
		for _, filename := range sortedStringMapKeys(files) {
			if strings.HasPrefix(filename, base) && (strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")) {
				parsedFiles[filename] = v4ParseYAML(filename, files[filename], errorAt)
			}
		}
		for _, required := range []string{contractPath, procedurePath, predicatesPath} {
			if _, exists := parsedFiles[required]; !exists {
				parsedFiles[required] = v4ParseYAML(required, nil, errorAt)
			}
		}

		sourceAST := parseV1Alpha4SkillAST(name, parsedFiles)
		contractAST := sourceAST.Contract
		if contractAST != nil && contractAST.Raw != nil {
			contract := contractAST.Raw
			v4Allowed(contract, contractPath, v4ContractFields, errorAt)
			if scalar(contractAST.Goal) == "" {
				errorAt(contractPath, contract, "SOURCE_ROOT_INVALID", "Contract requires goal")
			}
			if gate := contractAST.Precondition; gate != nil {
				if gate.Kind != yaml.ScalarNode || gate.Tag != "!!str" || !v4ContractGate.MatchString(gate.Value) {
					errorAt(contractPath, gate, "ENTRY_PRECONDITION_REF_INVALID", "precondition is one scalar node.precondition, not a list")
				}
			}
			// WP4A: every Skill-owned Schema root participates in the frozen G.4
			// allowlist before any generic Schema engine may consume it.
			for _, filename := range sortedStringMapKeys(parsedFiles) {
				if !strings.HasPrefix(filename, base+"schemas/") || !(strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")) {
					continue
				}
				if schemaRoot := parsedFiles[filename]; schemaRoot != nil {
					for _, issue := range validateG4Schema(schemaRoot) {
						errorAt(filename, issue.node, "SCHEMA_INVALID", issue.message)
					}
				}
			}
			if request := contractAST.RequestSchema; request != nil {
				ref := scalar(request)
				schemaPath := base + ref
				if !strings.HasPrefix(ref, "schemas/") || files[schemaPath] == nil || parsedFiles[schemaPath] == nil {
					errorAt(contractPath, request, "REQUEST_SCHEMA_MISSING", "Skill-owned request Schema is missing")
				} else {
					validateV1Alpha4RequestSchemaRoot(schemaPath, parsedFiles[schemaPath], errorAt)
				}
			}
			// WP3A: prove Alias -> snapshot digest -> trusted binding -> Stable
			// Package ID for the selected Skill only. Later stages stay fail-closed.
			identity := validateV1Alpha4DependencyIdentity(req, sourceAST)
			diagnostics = append(diagnostics, identity.Diagnostics...)
			// WP3B: index the verified snapshots' exact callable/Schema exports and
			// prove trusted Owner attribution for the references the Skill uses.
			ownerLinking := validateV1Alpha4OwnerLinking(req, sourceAST, identity)
			diagnostics = append(diagnostics, ownerLinking.Diagnostics...)
			// WP3C: bind exact callable membership and Effect against the trusted
			// deployment CapabilityContracts input. A Skill Grant is not proof of
			// authorization; missing trusted deployment input fails closed.
			capabilityLinking := validateV1Alpha4CapabilityLinking(req, sourceAST, ownerLinking)
			diagnostics = append(diagnostics, capabilityLinking.Diagnostics...)
			// WP4B: resolve the exact Schema reference closure and enforce the
			// Request/Context root shape. Unresolvable references never guess.
			schemaDiagnostics := validateV1Alpha4SchemaClosure(req, sourceAST, identity, ownerLinking)
			diagnostics = append(diagnostics, schemaDiagnostics...)
			// WP5: reachability and global Precondition dominance.
			procedureDiagnostics := validateV1Alpha4ProcedureGraph(contractAST, sourceAST.Procedure, ownerLinking, contractPath, procedurePath)
			diagnostics = append(diagnostics, procedureDiagnostics...)
			sourcePathDiagnostics := validateV1Alpha4SourcePathSchemas(req, sourceAST, identity, contractPath, procedurePath)
			diagnostics = append(diagnostics, sourcePathDiagnostics...)
			producerSourcePathDiagnostics := validateV1Alpha4ProducerSourcePaths(req, sourceAST, identity, ownerLinking, procedurePath)
			diagnostics = append(diagnostics, producerSourcePathDiagnostics...)
			argumentKeyDiagnostics := validateV1Alpha4ArgumentKeys(req, sourceAST, identity, ownerLinking, procedurePath)
			diagnostics = append(diagnostics, argumentKeyDiagnostics...)
			argumentValueDiagnostics := validateV1Alpha4ArgumentValueTypes(req, sourceAST, identity, ownerLinking, procedurePath)
			diagnostics = append(diagnostics, argumentValueDiagnostics...)
		}
		if sourceAST.Predicates != nil && sourceAST.Predicates.Raw != nil {
			v4CheckPredicates(predicatesPath, sourceAST.Predicates, errorAt)
			v4CheckPredicateCEL(predicatesPath, sourceAST.Predicates, errorAt)
		}
		if sourceAST.Procedure != nil && sourceAST.Procedure.Raw != nil {
			v4CheckProcedure(procedurePath, sourceAST.Procedure, contractAST, errorAt)
			validateV1Alpha4ProcedureStageRoutes(procedurePath, sourceAST.Procedure, errorAt)
			validateV1Alpha4SourcePaths(procedurePath, sourceAST, errorAt)
		}
	}

	// A warning alone must not hide the later-stage fail-closed boundary: no
	// successful Build may be returned while WP3B-WP12 semantics are unimplemented.
	if !hasErrorDiagnostic(diagnostics) {
		errorAt("", nil, "CAPABILITY_INPUT_UNTRUSTED",
			"v1alpha4 semantic linking and Owner qualification are not yet implemented; compilation fails closed")
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
		if !allowed[node.Content[i].Value] {
			add(file, node.Content[i], "UNKNOWN_FIELD", "Unsupported v1alpha4 author field "+node.Content[i].Value)
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
		def := set.Content[i+1]
		v4Allowed(def, file, v4PredicateFields, add)
		params := child(def, "parameters")
		if params == nil || params.Kind != yaml.SequenceNode {
			add(file, params, "PREDICATE_GROUP_INVALID", "parameters must be an ordered parameter list")
		} else {
			for _, item := range params.Content {
				if item.Kind != yaml.MappingNode || len(item.Content) != 2 {
					add(file, item, "PREDICATE_GROUP_INVALID", "Each parameter is a one-key declaration")
				}
			}
		}
	}
}

func v4CheckProcedure(file string, source *V1Alpha4ProcedureAST, contract *V1Alpha4ContractAST, add func(string, *yaml.Node, string, string)) {
	if source == nil || source.Raw == nil {
		return
	}
	root := source.Raw
	v4Allowed(root, file, map[string]bool{"start": true, "nodes": true}, add)
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
	legalSuccess, anySuccess := false, false
	for i := 0; i+1 < len(nodes.Content); i += 2 {
		id, n := nodes.Content[i].Value, nodes.Content[i+1]
		if !v4NodeID.MatchString(id) || v4Terminals[id] || id == "context" {
			add(file, nodes.Content[i], "NODE_ID_RESERVED", "Invalid or reserved Node ID")
		}
		v4Allowed(n, file, v4Stages, add)
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
			v4Allowed(body, file, allowed, add)
			if stage == "precondition" || stage == "postcondition" {
				calls := child(body, "predicates")
				if calls == nil || calls.Kind != yaml.SequenceNode || len(calls.Content) == 0 {
					add(file, calls, "PREDICATE_GROUP_INVALID", "Stage predicates must be a nonempty ordered list")
				} else {
					for _, call := range calls.Content {
						v4Allowed(call, file, map[string]bool{"predicate": true, "arguments": true}, add)
						args := child(call, "arguments")
						if scalar(child(call, "predicate")) == "" || args == nil || args.Kind != yaml.SequenceNode {
							add(file, call, "PREDICATE_GROUP_INVALID", "PredicateCall requires predicate and positional arguments")
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
					add(file, child(routes, outcome), "ROUTE_TARGET_INVALID", "Unknown route target")
				} else if ids[target] {
					adjacency[id] = append(adjacency[id], target)
				}
				if target == "SUCCEEDED" {
					anySuccess = true
					if stage == "postcondition" && outcome == "PASS" {
						legalSuccess = true
					}
				}
				if outcome == "OUTCOME_UNKNOWN" && target != "UNRESOLVED" && target != "SAFE_STOPPED" {
					add(file, child(routes, outcome), "ROUTE_OUTCOME_UNKNOWN", "OUTCOME_UNKNOWN may only safely stop")
				}
			}
			// Owner Effect and SourcePath assignability remain unverified until exact
			// Tool/Schema publication. No guessed Owner result classification.
		}
	}
	if anySuccess && !legalSuccess {
		add(file, root, "FINAL_POSTCONDITION_MISSING", "Only a final Postcondition PASS may request SUCCEEDED")
	} else if legalSuccess {
		for _, id := range mapKeys(nodes) {
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
			}
		}
	}
}
