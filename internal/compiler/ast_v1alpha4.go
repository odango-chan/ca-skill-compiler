package compiler

import yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"

// V1Alpha4SkillAST is the typed, source-positioned view of one Skill package.
// Its raw nodes all point into the one normalized parse result per YAML file.
type V1Alpha4SkillAST struct {
	Name       string
	Files      map[string]*yaml.Node
	Contract   *V1Alpha4ContractAST
	Procedure  *V1Alpha4ProcedureAST
	Predicates *V1Alpha4PredicateSetAST
}

type V1Alpha4ContractAST struct {
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

type V1Alpha4PredicateSetAST struct {
	Raw         *yaml.Node
	Span        SourceSpan
	Definitions map[string]*V1Alpha4PredicateAST
}

type V1Alpha4PredicateAST struct {
	ID         string
	Raw        *yaml.Node
	Span       SourceSpan
	Parameters *yaml.Node
	Evidence   *yaml.Node
	Evaluator  *yaml.Node
}

type V1Alpha4ProcedureAST struct {
	Raw   *yaml.Node
	Span  SourceSpan
	Start *yaml.Node
	Nodes *yaml.Node
	ByID  map[string]*V1Alpha4ProcedureNodeAST
}

type V1Alpha4ProcedureNodeAST struct {
	ID            string
	Raw           *yaml.Node
	Span          SourceSpan
	Precondition  *V1Alpha4PredicateStageAST
	Operation     *V1Alpha4OperationStageAST
	Observation   *V1Alpha4ObservationStageAST
	Postcondition *V1Alpha4PredicateStageAST
}

type V1Alpha4PredicateStageAST struct {
	Raw        *yaml.Node
	Span       SourceSpan
	Predicates []*V1Alpha4PredicateCallAST
	Routes     *yaml.Node
}

type V1Alpha4PredicateCallAST struct {
	Raw       *yaml.Node
	Span      SourceSpan
	Predicate *yaml.Node
	Arguments *yaml.Node
}

type V1Alpha4OperationStageAST struct {
	Raw       *yaml.Node
	Span      SourceSpan
	Tool      *yaml.Node
	Script    *yaml.Node
	Arguments *yaml.Node
	Routes    *yaml.Node
}

type V1Alpha4ObservationStageAST struct {
	Raw       *yaml.Node
	Span      SourceSpan
	Tool      *yaml.Node
	Arguments *yaml.Node
	Routes    *yaml.Node
}

func parseV1Alpha4SkillAST(name string, files map[string]*yaml.Node) *V1Alpha4SkillAST {
	base := "skills/" + name + "/"
	ast := &V1Alpha4SkillAST{Name: name, Files: files}

	if raw := files[base+"assurance/contract.yaml"]; raw != nil {
		ast.Contract = &V1Alpha4ContractAST{
			Raw: raw, Span: sourceSpan(base+"assurance/contract.yaml", raw),
			Goal: child(raw, "goal"), Precondition: child(raw, "precondition"),
			RequestSchema: child(raw, "requestSchema"), ContextSchema: child(raw, "contextSchema"),
			Dependencies: child(raw, "dependencies"), Capabilities: child(raw, "capabilities"),
			Limits: child(raw, "limits"),
		}
	}
	if raw := files[base+"assurance/predicates.yaml"]; raw != nil {
		set := child(raw, "predicates")
		predicateSet := &V1Alpha4PredicateSetAST{
			Raw: raw, Span: sourceSpan(base+"assurance/predicates.yaml", raw),
			Definitions: make(map[string]*V1Alpha4PredicateAST),
		}
		for _, id := range mapKeys(set) {
			definition := child(set, id)
			predicateSet.Definitions[id] = &V1Alpha4PredicateAST{
				ID: id, Raw: definition, Span: sourceSpan(base+"assurance/predicates.yaml", definition),
				Parameters: child(definition, "parameters"), Evidence: child(definition, "evidence"),
				Evaluator: child(definition, "evaluator"),
			}
		}
		ast.Predicates = predicateSet
	}
	if raw := files[base+"assurance/procedure.yaml"]; raw != nil {
		nodes := child(raw, "nodes")
		procedure := &V1Alpha4ProcedureAST{
			Raw: raw, Span: sourceSpan(base+"assurance/procedure.yaml", raw),
			Start: child(raw, "start"), Nodes: nodes,
			ByID: make(map[string]*V1Alpha4ProcedureNodeAST),
		}
		for _, id := range mapKeys(nodes) {
			node := child(nodes, id)
			parsed := &V1Alpha4ProcedureNodeAST{ID: id, Raw: node, Span: sourceSpan(base+"assurance/procedure.yaml", node)}
			if stage := child(node, "precondition"); stage != nil {
				parsed.Precondition = parseV1Alpha4PredicateStage(base+"assurance/procedure.yaml", stage)
			}
			if stage := child(node, "operation"); stage != nil {
				parsed.Operation = &V1Alpha4OperationStageAST{
					Raw: stage, Span: sourceSpan(base+"assurance/procedure.yaml", stage),
					Tool: child(stage, "tool"), Script: child(stage, "script"),
					Arguments: child(stage, "arguments"), Routes: child(stage, "routes"),
				}
			}
			if stage := child(node, "observation"); stage != nil {
				parsed.Observation = &V1Alpha4ObservationStageAST{
					Raw: stage, Span: sourceSpan(base+"assurance/procedure.yaml", stage),
					Tool: child(stage, "tool"), Arguments: child(stage, "arguments"),
					Routes: child(stage, "routes"),
				}
			}
			if stage := child(node, "postcondition"); stage != nil {
				parsed.Postcondition = parseV1Alpha4PredicateStage(base+"assurance/procedure.yaml", stage)
			}
			procedure.ByID[id] = parsed
		}
		ast.Procedure = procedure
	}
	return ast
}

func parseV1Alpha4PredicateStage(path string, raw *yaml.Node) *V1Alpha4PredicateStageAST {
	stage := &V1Alpha4PredicateStageAST{
		Raw: raw, Span: sourceSpan(path, raw), Routes: child(raw, "routes"),
		Predicates: make([]*V1Alpha4PredicateCallAST, 0),
	}
	for _, call := range sequence(child(raw, "predicates")) {
		stage.Predicates = append(stage.Predicates, &V1Alpha4PredicateCallAST{
			Raw: call, Span: sourceSpan(path, call),
			Predicate: child(call, "predicate"), Arguments: child(call, "arguments"),
		})
	}
	return stage
}
