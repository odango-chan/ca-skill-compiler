package compiler

// WP9 pipeline wiring: every referenced Procedure / Predicate Script must
// statically satisfy the frozen Script ABI. Parse or ABI failures are reported
// as SCRIPT_CONTRACT_INVALID at the source Script reference. Author Python is
// never executed.

import (
	"sort"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

type scriptSchemaSet struct {
	Inputs          map[string]map[string]any
	Outputs         map[string]map[string]any
	PredicateInputs map[string]map[string]any
}

func validateScriptABI(req BuildRequest, identity dependencyIdentityResult, skill *SkillAST, files map[string][]byte, procedurePath, predicatesPath string, add func(string, *yaml.Node, string, string)) scriptSchemaSet {
	schemas := scriptSchemaSet{Inputs: map[string]map[string]any{}, Outputs: map[string]map[string]any{}, PredicateInputs: map[string]map[string]any{}}
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
			optional := optionalInputKeys(abi.InputSchema)
			if err := validateScriptOptionalKeyAccess(source, optional); err != nil {
				add(procedurePath, scriptNode, "SCRIPT_CONTRACT_INVALID", "Procedure Script optional input access: "+err.Error())
				continue
			}
			schemas.Inputs[base+script] = abi.InputSchema
			schemas.Outputs[base+script] = abi.OutputSchema
		}
	}
	if skill.Predicates != nil {
		registry := buildSchemaRegistry(req, skill, identity, req.development)
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
			predicateProperties := map[string]any{}
			for _, parameter := range sequence(predicate.Parameters) {
				parameter = unalias(parameter)
				if parameter == nil || parameter.Kind != yaml.MappingNode {
					continue
				}
				for i := 0; i+1 < len(parameter.Content); i += 2 {
					if name := scalar(unalias(parameter.Content[i])); name != "" {
						allowed[name] = true
						if schema := predicateParameterSchema(parameter.Content[i+1], registry); schema != nil {
							predicateProperties[name] = nodeValue(schema)
						}
					}
				}
			}
			if err := validateScriptInputKeyAccess(source, allowed, false); err != nil {
				add(predicatesPath, scriptNode, "SCRIPT_CONTRACT_INVALID", "Predicate Script input access: "+err.Error())
				continue
			}
			if len(predicateProperties) == len(allowed) {
				schemas.PredicateInputs[base+script] = map[string]any{
					"type": "object", "additionalProperties": false,
					"properties": predicateProperties, "required": sortedAnyMapKeys(predicateProperties),
				}
			}
		}
	}
	return schemas
}

func sortedAnyMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func optionalInputKeys(schema map[string]any) map[string]bool {
	optional := map[string]bool{}
	properties, _ := schema["properties"].(map[string]any)
	required := map[string]bool{}
	if list, ok := schema["required"].([]string); ok {
		for _, key := range list {
			required[key] = true
		}
	}
	for key := range properties {
		if !required[key] {
			optional[key] = true
		}
	}
	return optional
}
