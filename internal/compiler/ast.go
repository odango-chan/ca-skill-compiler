//go:build legacy_v01

package compiler

import yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"

// SourceSpan is the source location retained by the parsed Skill AST.
type SourceSpan struct {
	Path   string
	Line   int
	Column int
}

// SkillAST is the structural result for one Skill package. YAML nodes are kept
// only at value boundaries where JSON Schema and binding validation need their
// original scalar/collection types; graph structure is represented explicitly.
type SkillAST struct {
	Name       string
	Span       SourceSpan
	Metadata   SkillMetadataAST
	Contract   *SkillContractAST
	Procedure  *ProcedureDocumentAST
	Predicates map[string]PredicateAST
	Cases      []CaseAST
}

type SkillMetadataAST struct {
	Name        string
	Description string
	Span        SourceSpan
}

type SkillContractAST struct {
	Span                SourceSpan
	Raw                 *yaml.Node
	Name                string
	SkillType           string
	Goal                string
	EntryProcedure      string
	RequestSchema       string
	ContextSchemas      map[string]string
	RequiredFacts       []string
	AllowedCapabilities []string
}

type PredicateAST struct {
	Name          string
	Span          SourceSpan
	EvaluatorType string
	EvaluatorRef  string
	Observations  map[string]ObservationAST
	Raw           *yaml.Node
}

type ObservationAST struct {
	Name      string
	Ref       string
	Scope     string
	Freshness string
	Span      SourceSpan
}

type ProcedureDocumentAST struct {
	Span       SourceSpan
	Procedures map[string]*ProcedureAST
}

type ProcedureAST struct {
	Name   string
	Span   SourceSpan
	Raw    *yaml.Node
	Start  string
	Inputs map[string]ProcedureInputAST
	Nodes  map[string]ProcedureNodeAST
}

type ProcedureInputAST struct {
	Name string
	Type string
	From string
	Span SourceSpan
	Raw  *yaml.Node
}

type NodeAction string

const (
	ActionCall      NodeAction = "call"
	ActionScript    NodeAction = "script"
	ActionProcedure NodeAction = "procedure"
	ActionVerify    NodeAction = "verify"
	ActionReconcile NodeAction = "reconcile"
	ActionTerminal  NodeAction = "terminal"
)

type ProcedureNodeAST struct {
	Name           string
	Action         NodeAction
	Reference      string
	ProcedureRef   ProcedureReferenceAST
	Span           SourceSpan
	Raw            *yaml.Node
	With           *yaml.Node
	Branches       *yaml.Node
	Preconditions  []ProcedureStepAST
	Postconditions []ProcedureStepAST
}

type ProcedureReferenceAST struct {
	Skill     string
	Procedure string
	Explicit  bool
	Span      SourceSpan
}

type ProcedureStepAST struct {
	Name      string
	Action    NodeAction
	Reference string
	Span      SourceSpan
	Raw       *yaml.Node
	With      *yaml.Node
}

type CaseAST struct {
	Name string
	Path string
	Span SourceSpan
}

func sourceSpan(path string, node *yaml.Node) SourceSpan {
	span := SourceSpan{Path: path}
	if node != nil {
		node = unalias(node)
		span.Line, span.Column = node.Line, node.Column
	}
	return span
}

func parseProcedureAST(filename string, root *yaml.Node) *ProcedureDocumentAST {
	document := &ProcedureDocumentAST{Span: sourceSpan(filename, root), Procedures: map[string]*ProcedureAST{}}
	procedures := child(root, "procedures")
	for _, name := range mapKeys(procedures) {
		raw := child(procedures, name)
		procedure := &ProcedureAST{Name: name, Span: sourceSpan(filename, raw), Raw: raw,
			Start: scalar(child(raw, "start")), Inputs: map[string]ProcedureInputAST{}, Nodes: map[string]ProcedureNodeAST{}}
		for _, inputName := range mapKeys(child(raw, "inputs")) {
			input := child(child(raw, "inputs"), inputName)
			procedure.Inputs[inputName] = ProcedureInputAST{Name: inputName, Type: scalar(child(input, "type")),
				From: scalar(child(input, "from")), Span: sourceSpan(filename, input), Raw: input}
		}
		nodes := child(raw, "nodes")
		for _, nodeName := range mapKeys(nodes) {
			node := child(nodes, nodeName)
			parsed := ProcedureNodeAST{Name: nodeName, Span: sourceSpan(filename, node), Raw: node,
				With: child(node, "with"), Branches: child(node, "on")}
			for _, action := range []NodeAction{ActionCall, ActionScript, ActionProcedure, ActionVerify, ActionReconcile, ActionTerminal} {
				if value := child(node, string(action)); value != nil {
					parsed.Action = action
					parsed.Reference = scalar(value)
					if action == ActionProcedure {
						parsed.ProcedureRef = ProcedureReferenceAST{Span: sourceSpan(filename, value)}
						if unalias(value).Kind == yaml.MappingNode {
							parsed.ProcedureRef.Explicit = true
							parsed.ProcedureRef.Skill = scalar(child(value, "skill"))
							parsed.ProcedureRef.Procedure = scalar(child(value, "procedure"))
						} else {
							parsed.ProcedureRef.Procedure = scalar(value)
						}
					}
					break
				}
			}
			for _, scope := range []string{"preconditions", "postconditions"} {
				var steps []ProcedureStepAST
				for _, step := range sequence(child(child(node, scope), "steps")) {
					parsedStep := ProcedureStepAST{Name: scalar(child(step, "name")), Span: sourceSpan(filename, step), Raw: step, With: child(step, "with")}
					for _, action := range []NodeAction{ActionCall, ActionScript, ActionVerify} {
						if value := child(step, string(action)); value != nil {
							parsedStep.Action, parsedStep.Reference = action, scalar(value)
							break
						}
					}
					steps = append(steps, parsedStep)
				}
				if scope == "preconditions" {
					parsed.Preconditions = steps
				} else {
					parsed.Postconditions = steps
				}
			}
			procedure.Nodes[nodeName] = parsed
		}
		document.Procedures[name] = procedure
	}
	return document
}

func (s *skillEntry) procedureASTFor(raw *yaml.Node) *ProcedureAST {
	if s == nil || s.ast == nil || s.ast.Procedure == nil {
		return nil
	}
	for _, procedure := range s.ast.Procedure.Procedures {
		if procedure.Raw == raw {
			return procedure
		}
	}
	return nil
}
