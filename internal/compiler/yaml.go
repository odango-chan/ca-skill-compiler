package compiler

import yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"

// parseYAML is the single source parser used by the legacy analysis helpers
// and the v1alpha4 Core. Both consume the same normalized AST adapter.
func (a *analysis) parseYAML(filename string) *yaml.Node {
	root, err := yaml.Parse(a.files[filename])
	if err != nil {
		line, _ := yaml.ParseErrorLocation(err)
		if line == 0 {
			line = 1
		}
		code := "YAML_PARSE"
		if yaml.IsRootError(err) {
			code = "YAML_ROOT"
		}
		if yaml.IsLimitError(err) {
			code = "YAML_LIMIT"
		}
		a.errorAt(filename, line, code, err.Error())
		return nil
	}
	return root
}

func unalias(node *yaml.Node) *yaml.Node {
	for hops := 0; node != nil && node.Kind == yaml.AliasNode && hops < 64; hops++ {
		node = node.Alias
	}
	return node
}

func child(node *yaml.Node, key string) *yaml.Node {
	node = unalias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		mapKey := unalias(node.Content[i])
		if mapKey != nil && mapKey.Value == key {
			return unalias(node.Content[i+1])
		}
	}
	return nil
}

func scalar(node *yaml.Node) string {
	node = unalias(node)
	if node == nil || node.Kind != yaml.ScalarNode {
		return ""
	}
	return node.Value
}

func sequence(node *yaml.Node) []*yaml.Node {
	node = unalias(node)
	if node == nil || node.Kind != yaml.SequenceNode {
		return nil
	}
	return node.Content
}

func mapKeys(node *yaml.Node) []string {
	node = unalias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	keys := make([]string, 0, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := unalias(node.Content[i])
		if key != nil {
			keys = append(keys, key.Value)
		}
	}
	return keys
}

func inspectMapping(node *yaml.Node, visit func(key string, value *yaml.Node)) {
	node = unalias(node)
	if node == nil {
		return
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, value := unalias(node.Content[i]), unalias(node.Content[i+1])
			if key == nil {
				continue
			}
			visit(key.Value, value)
			inspectMapping(value, visit)
		}
	case yaml.SequenceNode:
		for _, item := range node.Content {
			inspectMapping(item, visit)
		}
	}
}
