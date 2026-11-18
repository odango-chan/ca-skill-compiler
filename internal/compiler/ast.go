package compiler

import yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"

// SkillAST is the typed, source-positioned view of one Skill package.
// Its raw nodes all point into the one normalized parse result per YAML file.
type SkillAST struct {
	Name       string
	Files      map[string]*yaml.Node
	Contract   *ContractAST
	Procedure  *ProcedureAST
	Predicates *PredicateSetAST
}

type ContractAST struct {
	Raw           *yaml.Node
	Span          SourceSpan
	Goal          *yaml.Node
	Precondition  *yaml.Node
	RequestSchema *yaml.Node
	ContextSchema *yaml.Node
	Dependencies  *yaml.Node
	Capabilities  *yaml.Node
	Limits        *yaml.Node
}

type PredicateSetAST struct {
	Raw         *yaml.Node
	Span        SourceSpan
	Definitions map[string]*PredicateAST
}

type PredicateAST struct {
	ID         string
	Raw        *yaml.Node
	Span       SourceSpan
	Parameters *yaml.Node
	Evidence   *yaml.Node
	Evaluator  *yaml.Node
}

type ProcedureAST struct {
	Raw   *yaml.Node
	Span  SourceSpan
	Start *yaml.Node
	Nodes *yaml.Node
	ByID  map[string]*ProcedureNodeAST
}

type ProcedureNodeAST struct {
	ID            string
	Raw           *yaml.Node
	Span          SourceSpan
	Precondition  *PredicateStageAST
	Operation     *OperationStageAST
	Observation   *ObservationStageAST
	Postcondition *PredicateStageAST
}

type PredicateStageAST struct {
	Raw        *yaml.Node
	Span       SourceSpan
	Predicates []*PredicateCallAST
	Routes     *yaml.Node
}

type PredicateCallAST struct {
	Raw       *yaml.Node
	Span      SourceSpan
	Predicate *yaml.Node
	Arguments *yaml.Node
}

type OperationStageAST struct {
	Raw       *yaml.Node
	Span      SourceSpan
	Tool      *yaml.Node
	Script    *yaml.Node
	Arguments *yaml.Node
	Routes    *yaml.Node
}

type ObservationStageAST struct {
	Raw       *yaml.Node
	Span      SourceSpan
	Tool      *yaml.Node
	Arguments *yaml.Node
	Routes    *yaml.Node
}

func parseSkillAST(name string, files map[string]*yaml.Node) *SkillAST {
	base := "skills/" + name + "/"
	ast := &SkillAST{Name: name, Files: files}

	if raw := files[base+"assurance/contract.yaml"]; raw != nil {
		ast.Contract = &ContractAST{
			Raw: raw, Span: sourceSpan(base+"assurance/contract.yaml", raw),
			Goal: child(raw, "goal"), Precondition: child(raw, "precondition"),
			RequestSchema: child(raw, "requestSchema"), ContextSchema: child(raw, "contextSchema"),
			Dependencies: child(raw, "dependencies"), Capabilities: child(raw, "capabilities"),
			Limits: child(raw, "limits"),
		}
	}
	if raw := files[base+"assurance/predicates.yaml"]; raw != nil {
		set := child(raw, "predicates")
		predicateSet := &PredicateSetAST{
			Raw: raw, Span: sourceSpan(base+"assurance/predicates.yaml", raw),
			Definitions: make(map[string]*PredicateAST),
		}
		for _, id := range mapKeys(set) {
			definition := child(set, id)
			predicateSet.Definitions[id] = &PredicateAST{
				ID: id, Raw: definition, Span: sourceSpan(base+"assurance/predicates.yaml", definition),
				Parameters: child(definition, "parameters"), Evidence: child(definition, "evidence"),
				Evaluator: child(definition, "evaluator"),
			}
		}
		ast.Predicates = predicateSet
	}
	if raw := files[base+"assurance/procedure.yaml"]; raw != nil {
		nodes := child(raw, "nodes")
		procedure := &ProcedureAST{
			Raw: raw, Span: sourceSpan(base+"assurance/procedure.yaml", raw),
			Start: child(raw, "start"), Nodes: nodes,
			ByID: make(map[string]*ProcedureNodeAST),
		}
		for _, id := range mapKeys(nodes) {
			node := child(nodes, id)
			parsed := &ProcedureNodeAST{ID: id, Raw: node, Span: sourceSpan(base+"assurance/procedure.yaml", node)}
			if stage := child(node, "precondition"); stage != nil {
				parsed.Precondition = parsePredicateStage(base+"assurance/procedure.yaml", stage)
			}
			if stage := child(node, "operation"); stage != nil {
				parsed.Operation = &OperationStageAST{
					Raw: stage, Span: sourceSpan(base+"assurance/procedure.yaml", stage),
					Tool: child(stage, "tool"), Script: child(stage, "script"),
					Arguments: child(stage, "arguments"), Routes: child(stage, "routes"),
				}
			}
			if stage := child(node, "observation"); stage != nil {
				parsed.Observation = &ObservationStageAST{
					Raw: stage, Span: sourceSpan(base+"assurance/procedure.yaml", stage),
					Tool: child(stage, "tool"), Arguments: child(stage, "arguments"),
					Routes: child(stage, "routes"),
				}
			}
			if stage := child(node, "postcondition"); stage != nil {
				parsed.Postcondition = parsePredicateStage(base+"assurance/procedure.yaml", stage)
			}
			procedure.ByID[id] = parsed
		}
		ast.Procedure = procedure
	}
	return ast
}

func parsePredicateStage(path string, raw *yaml.Node) *PredicateStageAST {
	stage := &PredicateStageAST{
		Raw: raw, Span: sourceSpan(path, raw), Routes: child(raw, "routes"),
		Predicates: make([]*PredicateCallAST, 0),
	}
	for _, call := range sequence(child(raw, "predicates")) {
		stage.Predicates = append(stage.Predicates, &PredicateCallAST{
			Raw: call, Span: sourceSpan(path, call),
			Predicate: child(call, "predicate"), Arguments: child(call, "arguments"),
		})
	}
	return stage
}
