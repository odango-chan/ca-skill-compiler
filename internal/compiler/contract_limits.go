package compiler

// Contract limits validation: §11.3/§11.6 limits.maxActions is an optional
// positive integer action-budget upper bound. ACTION_LIMIT_INVALID is the frozen
// identity for an invalid value; undefined keys inside limits are UNKNOWN_FIELD.

import (
	"strconv"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validateContractLimits(contractPath string, contract *ContractAST, add func(string, *yaml.Node, string, string)) {
	if contract == nil || contract.Limits == nil {
		return
	}
	limits := unalias(contract.Limits)
	if limits.Kind != yaml.MappingNode {
		add(contractPath, limits, "ACTION_LIMIT_INVALID", "Contract.limits must be a mapping containing maxActions")
		return
	}
	for i := 0; i+1 < len(limits.Content); i += 2 {
		keyNode := unalias(limits.Content[i])
		value := unalias(limits.Content[i+1])
		key := scalar(keyNode)
		if key != "maxActions" {
			add(contractPath, keyNode, "UNKNOWN_FIELD", "Unsupported Contract limits field: "+key)
			continue
		}
		if value == nil || value.Kind != yaml.ScalarNode || value.Tag != "!!int" {
			add(contractPath, value, "ACTION_LIMIT_INVALID", "limits.maxActions must be a positive integer")
			continue
		}
		budget, err := strconv.Atoi(value.Value)
		if err != nil || budget <= 0 {
			add(contractPath, value, "ACTION_LIMIT_INVALID", "limits.maxActions must be a positive integer")
		}
	}
}
