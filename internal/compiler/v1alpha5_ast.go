package compiler

// CA-SL v1alpha5 typed Procedure AST.
//
// v1alpha5 is an independently versioned language target. This AST parses the
// inherited Procedure shape (start/nodes) plus the Appendix L.1 Node-entry Wait
// amendment. Every Raw node points into the single normalized YAML parse result
// per source file; the Wait declaration is parsed in the same Procedure pass,
// not by a detached helper.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

// V1Alpha5Target is the independent CA-SL v1alpha5 language target identifier.
const V1Alpha5Target = "connectionagent.io/v1alpha5"

var v1alpha5ProcedureFields = map[string]bool{"start": true, "nodes": true}

// V1Alpha5SkillAST is the typed, source-positioned view of one v1alpha5 Skill
// package. Files holds the one normalized parse result per YAML author file.
type V1Alpha5SkillAST struct {
	Name      string
	Files     map[string]*yaml.Node
	Procedure *V1Alpha5ProcedureAST
}

// V1Alpha5ProcedureAST is the parsed v1alpha5 Procedure document.
type V1Alpha5ProcedureAST struct {
	Raw   *yaml.Node
	Span  SourceSpan
	Start *yaml.Node
	Nodes *yaml.Node
	ByID  map[string]*V1Alpha5ProcedureNodeAST
}

// V1Alpha5ProcedureNodeAST is one declared Procedure Node: its Wait entry
// control (nil when absent) and the raw nodes of the four inherited business
// stages. Later work packages consume the raw stage nodes for trigger linking,
// SourcePath, graph and response-boundary analysis.
type V1Alpha5ProcedureNodeAST struct {
	ID            string
	Raw           *yaml.Node
	Span          SourceSpan
	Wait          *V1Alpha5WaitAST
	Precondition  *yaml.Node
	Operation     *yaml.Node
	Observation   *yaml.Node
	Postcondition *yaml.Node
}

// HasBusinessStage reports whether the Node declares at least one of the four
// inherited business stages. wait does not count as a business stage.
func (node *V1Alpha5ProcedureNodeAST) HasBusinessStage() bool {
	if node == nil {
		return false
	}
	return node.Precondition != nil || node.Operation != nil ||
		node.Observation != nil || node.Postcondition != nil
}

// parseV1Alpha5SkillAST parses the v1alpha5 Procedure author file out of the
// single normalized parse result and returns every source-positioned diagnostic
// from the Procedure/Wait structural and lexical rules.
func parseV1Alpha5SkillAST(name string, files map[string]*yaml.Node) (*V1Alpha5SkillAST, []Diagnostic) {
	base := "skills/" + name + "/"
	ast := &V1Alpha5SkillAST{Name: name, Files: files}
	raw := files[base+"assurance/procedure.yaml"]
	if raw == nil {
		return ast, nil
	}
	procedure, diagnostics := parseV1Alpha5ProcedureAST(base+"assurance/procedure.yaml", raw)
	ast.Procedure = procedure
	return ast, diagnostics
}

func parseV1Alpha5ProcedureAST(path string, raw *yaml.Node) (*V1Alpha5ProcedureAST, []Diagnostic) {
	var diagnostics []Diagnostic
	add := func(at *yaml.Node, code, message string) {
		line, column := 0, 0
		if at != nil {
			line, column = at.Line, at.Column
		}
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Phase:    "C_STATIC",
			Code:     code,
			Path:     limitDiagnosticPath(path),
			Line:     line,
			Column:   column,
			Message:  limitDiagnosticMessage(message),
		})
	}

	raw = unalias(raw)
	if raw == nil || raw.Kind != yaml.MappingNode {
		if raw != nil {
			add(raw, "SOURCE_ROOT_INVALID", "Procedure must be a mapping")
		}
		return nil, diagnostics
	}

	v1alpha5CheckUnknownFields(raw, v1alpha5ProcedureFields, add)

	procedure := &V1Alpha5ProcedureAST{
		Raw:   raw,
		Span:  sourceSpan(path, raw),
		Start: child(raw, "start"),
		Nodes: child(raw, "nodes"),
		ByID:  make(map[string]*V1Alpha5ProcedureNodeAST),
	}

	nodes := unalias(procedure.Nodes)
	if nodes == nil || nodes.Kind != yaml.MappingNode {
		add(v1alpha5FallbackNode(nodes, raw), "SOURCE_ROOT_INVALID", "Procedure.nodes must be a mapping")
		return procedure, diagnostics
	}

	declared := make(map[string]bool, len(nodes.Content)/2)
	for i := 0; i+1 < len(nodes.Content); i += 2 {
		key := unalias(nodes.Content[i])
		if key == nil {
			continue
		}
		nodeID := key.Value
		declared[nodeID] = true
		if !nodeIDPattern.MatchString(nodeID) || reservedTerminals[nodeID] || nodeID == "context" {
			add(key, "NODE_ID_RESERVED", "Invalid or reserved Node ID")
		}
		nodeRaw := unalias(nodes.Content[i+1])
		parsed := &V1Alpha5ProcedureNodeAST{
			ID:   nodeID,
			Raw:  nodeRaw,
			Span: sourceSpan(path, nodeRaw),
		}
		if nodeRaw == nil || nodeRaw.Kind != yaml.MappingNode {
			add(v1alpha5FallbackNode(nodeRaw, key), "SOURCE_ROOT_INVALID", "Procedure Node must be a mapping")
			procedure.ByID[nodeID] = parsed
			continue
		}
		parsed.Precondition = child(nodeRaw, "precondition")
		parsed.Operation = child(nodeRaw, "operation")
		parsed.Observation = child(nodeRaw, "observation")
		parsed.Postcondition = child(nodeRaw, "postcondition")
		wait, waitDiagnostics := parseV1Alpha5WaitNode(path, nodeID, nodeRaw)
		parsed.Wait = wait
		diagnostics = append(diagnostics, waitDiagnostics...)
		procedure.ByID[nodeID] = parsed
	}

	// start must be one exact NodeId declared by the same Procedure. An absent
	// or invalid start is anchored at the start node when present, else at the
	// nearest existing parent object (the Procedure root).
	start, startIsString := v1alpha5NonEmptyString(procedure.Start)
	if !startIsString || !nodeIDPattern.MatchString(start) || reservedTerminals[start] || start == "context" || !declared[start] {
		add(v1alpha5FallbackNode(procedure.Start, raw), "PROCEDURE_START_MISSING", "start must resolve to a declared Node")
	}

	return procedure, diagnostics
}
