package compiler

// Inherited Package / Contract / PredicateSet / Procedure structural checks.
//
// CA-SL v1alpha5 inherits the released v1alpha4 language semantics. These are
// the version-neutral structural obligations of that inherited language: field
// allowlists, identifier grammar, stage shape, route shape, DAG and final
// Postcondition rules. They are part of the single v1alpha5 pipeline and are
// not a separate language target.
//
// Package layout: this file owns the version-neutral shape checks that used to
// were formerly defined beside the retired target entry. No target dispatch remains.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
	"regexp"
	"strings"
)

var (
	nodeIDPattern       = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
	contractGatePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*[.]precondition$`)
)

var (
	reservedTerminals = map[string]bool{"SUCCEEDED": true, "FAILED": true, "SAFE_STOPPED": true, "UNRESOLVED": true}
	businessStages    = map[string]bool{"precondition": true, "operation": true, "observation": true, "postcondition": true}
	contractFields    = map[string]bool{"goal": true, "precondition": true, "requestSchema": true, "contextSchema": true, "dependencies": true, "capabilities": true, "limits": true}
	predicateFields   = map[string]bool{"parameters": true, "evidence": true, "evaluator": true}
)

// checkAllowedFields rejects every author field outside the inherited
// allowlist. Recognized legacy Wait / v0.2 syntax keeps its stable diagnostic
// identity instead of being silently translated.
func checkAllowedFields(node *yaml.Node, file string, allowed map[string]bool, add func(string, *yaml.Node, string, string)) {
	if node == nil || node.Kind != yaml.MappingNode {
		add(file, node, "SOURCE_ROOT_INVALID", "Expected a mapping")
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		name := node.Content[i].Value
		if allowed[name] {
			continue
		}
		switch legacyFieldClass(name) {
		case "wait":
			add(file, node.Content[i], "UNSUPPORTED_WAIT", "Wait/Await control structures are not part of the inherited CA-SL procedure language")
		case "v02":
			add(file, node.Content[i], "UNSUPPORTED_V02_SYNTAX", "Recognized legacy v0.2 syntax is not part of CA-SL v1alpha5")
		default:
			add(file, node.Content[i], "UNKNOWN_FIELD", "Unsupported CA-SL author field "+name)
		}
	}
}

func checkPredicates(file string, source *PredicateSetAST, add func(string, *yaml.Node, string, string)) {
	if source == nil || source.Raw == nil {
		return
	}
	root := source.Raw
	checkAllowedFields(root, file, map[string]bool{"predicates": true}, add)
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
		checkAllowedFields(def, file, predicateFields, add)
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

// checkProcedure validates the inherited Procedure Node/stage/route shape. The
// optional node-entry Wait declaration is part of the single v1alpha5 Node
// shape; its own structural rules are enforced by the v1alpha5 Wait and graph
// passes on the same Procedure AST.
func checkProcedure(file string, source *ProcedureAST, contract *ContractAST, add func(string, *yaml.Node, string, string)) {
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
	checkAllowedFields(root, file, procedureFields, add)
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
	nodeFields := map[string]bool{
		"precondition": true, "operation": true,
		"observation": true, "postcondition": true, "wait": true,
	}
	for i := 0; i+1 < len(nodes.Content); i += 2 {
		id, n := nodes.Content[i].Value, nodes.Content[i+1]
		if !nodeIDPattern.MatchString(id) || reservedTerminals[id] || id == "context" {
			add(file, nodes.Content[i], "NODE_ID_RESERVED", "Invalid or reserved Node ID")
		}
		checkAllowedFields(n, file, nodeFields, add)
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
			checkAllowedFields(body, file, allowed, add)
			if stage == "precondition" || stage == "postcondition" {
				calls := child(body, "predicates")
				if calls == nil || calls.Kind != yaml.SequenceNode || len(calls.Content) == 0 {
					add(file, calls, "PREDICATE_GROUP_INVALID", "Stage predicates must be a nonempty ordered list")
				} else {
					for _, call := range calls.Content {
						checkAllowedFields(call, file, map[string]bool{"predicate": true, "arguments": true}, add)
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
				if target == "" || (!ids[target] && !reservedTerminals[target]) {
					add(file, child(routes, outcome), "CONTROL_FLOW_DANGLING", "Route value is neither a terminal nor an existing Node")
				} else if ids[target] {
					adjacency[id] = append(adjacency[id], target)
				}
				if outcome == "OUTCOME_UNKNOWN" && target != "UNRESOLVED" && target != "SAFE_STOPPED" {
					add(file, child(routes, outcome), "ROUTE_OUTCOME_UNKNOWN", "OUTCOME_UNKNOWN may only safely stop")
				}
			}
			// Owner Effect and SourcePath assignability are proven by the
			// dedicated linking and typing passes, not by guessed classification.
		}
		// Appendix L.1: the Wait TIMEOUT route is a control-flow edge of the
		// expanded graph. Its target is validated by the v1alpha5 Wait graph
		// pass; here it must only participate in the DAG/cycle proof.
		waitRoutes := child(child(n, "wait"), "routes")
		for _, outcome := range mapKeys(waitRoutes) {
			target := scalar(child(waitRoutes, outcome))
			if target != "" && ids[target] {
				adjacency[id] = append(adjacency[id], target)
			}
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
