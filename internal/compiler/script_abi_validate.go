package compiler

// WP9 pipeline wiring: every referenced Procedure / Predicate Script must
// statically satisfy the frozen Script ABI. Parse or ABI failures are reported
// as SCRIPT_CONTRACT_INVALID at the source Script reference. Author Python is
// never executed.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validateV1Alpha4ScriptABI(skill *V1Alpha4SkillAST, files map[string][]byte, procedurePath, predicatesPath string, add func(string, *yaml.Node, string, string)) map[string]map[string]any {
	schemas := map[string]map[string]any{}
	if skill == nil {
		return schemas
	}
	base := "skills/" + skill.Name + "/"
	if skill.Procedure != nil {
		for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
			node := skill.Procedure.ByID[nodeID]
			if node == nil || node.Operation == nil || node.Operation.Script == nil {
				continue
			}
			scriptNode := unalias(node.Operation.Script)
			script := scalar(scriptNode)
			if !validLocalScriptPath(script) {
				continue // path grammar is diagnosed by the Script-path slice
			}
			source, ok := files[base+script]
			if !ok {
				continue // missing file is diagnosed by the Script-path slice
			}
			abi, err := deriveProcedureScriptABI(source)
			if err != nil {
				add(procedurePath, scriptNode, "SCRIPT_CONTRACT_INVALID", "Procedure Script ABI: "+err.Error())
				continue
			}
			allowed := map[string]bool{}
			if properties, ok := abi.InputSchema["properties"].(map[string]any); ok {
				for key := range properties {
					allowed[key] = true
				}
			}
			if err := validateScriptInputKeyAccess(source, allowed, false); err != nil {
				add(procedurePath, scriptNode, "SCRIPT_CONTRACT_INVALID", "Procedure Script input access: "+err.Error())
				continue
			}
			schemas[base+script] = abi.InputSchema
		}
	}
	if skill.Predicates != nil {
		for _, predicateID := range sortedStringMapKeys(skill.Predicates.Definitions) {
			predicate := skill.Predicates.Definitions[predicateID]
			if predicate == nil || predicate.Evaluator == nil {
				continue
			}
			evaluator := unalias(predicate.Evaluator)
			if evaluator == nil || evaluator.Kind != yaml.MappingNode || scalar(child(evaluator, "type")) != "script" {
				continue
			}
			scriptNode := unalias(child(evaluator, "script"))
			script := scalar(scriptNode)
			if !validLocalScriptPath(script) {
				continue
			}
			source, ok := files[base+script]
			if !ok {
				continue
			}
			if _, err := derivePredicateScriptABI(source); err != nil {
				add(predicatesPath, scriptNode, "SCRIPT_CONTRACT_INVALID", "Predicate Script ABI: "+err.Error())
				continue
			}
			allowed := map[string]bool{}
			for _, parameter := range sequence(predicate.Parameters) {
				parameter = unalias(parameter)
				if parameter == nil || parameter.Kind != yaml.MappingNode {
					continue
				}
				for i := 0; i+1 < len(parameter.Content); i += 2 {
					if name := scalar(unalias(parameter.Content[i])); name != "" {
						allowed[name] = true
					}
				}
			}
			if err := validateScriptInputKeyAccess(source, allowed, false); err != nil {
				add(predicatesPath, scriptNode, "SCRIPT_CONTRACT_INVALID", "Predicate Script input access: "+err.Error())
			}
		}
	}
	return schemas
}
