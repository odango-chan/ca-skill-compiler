package compiler

import (
	"gopkg.in/yaml.v3"
)

// parseYAML parses a full YAML document (including anchors and aliases), but never
// performs application-level evaluation, file reads, or remote $ref resolution.
func (a *analysis) parseYAML(filename string) *yaml.Node {
	var doc yaml.Node
	if err := yaml.Unmarshal(a.files[filename], &doc); err != nil {
		a.errorAt(filename, 1, "YAML_PARSE", err.Error())
		return nil
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		a.errorAt(filename, 1, "YAML_ROOT", "expected one mapping document")
		return nil
	}
	if !boundedYAML(&doc, make(map[*yaml.Node]bool), new(int), 0) {
		a.errorAt(filename, 1, "YAML_LIMIT", "YAML aliases, nesting, or node count exceed safe bounds")
		return nil
	}
	if line, key := duplicateYAMLKey(&doc); key != "" {
		a.errorAt(filename, line, "YAML_DUPLICATE_KEY", "duplicate YAML mapping key: "+key)
		return nil
	}
	return doc.Content[0]
}

func boundedYAML(n *yaml.Node, seen map[*yaml.Node]bool, count *int, depth int) bool {
	if n == nil { return true }
	if depth > 64 || *count > 20000 || seen[n] { return false }
	seen[n] = true
	*count++
	if n.Kind == yaml.AliasNode && n.Alias != nil {
		if !boundedYAML(n.Alias, seen, count, depth + 1) { return false }
	}
	for _, node := range n.Content {
		if !boundedYAML(node, seen, count, depth + 1) { return false }
	}
	delete(seen, n)
	return *count <= 20000
}

// Duplicate YAML keys are rejected even when parsing into yaml.Node (which
// otherwise preserves the duplicate rather than validating it).
func duplicateYAMLKey(root *yaml.Node) (int, string) {
	var inspect func(*yaml.Node, map[*yaml.Node]bool) (int, string)
	inspect = func(n *yaml.Node, active map[*yaml.Node]bool) (int, string) {
		if n == nil || active[n] { return 0, "" }
		active[n] = true
		defer delete(active, n)
		if n.Kind == yaml.AliasNode { return inspect(n.Alias, active) }
		if n.Kind == yaml.MappingNode {
			keys := make(map[string]bool)
			for i := 0; i+1 < len(n.Content); i += 2 {
				key := n.Content[i].Value
				if keys[key] { return n.Content[i].Line, key }
				keys[key] = true
				if line, dup := inspect(n.Content[i+1], active); dup != "" { return line, dup }
			}
			return 0, ""
		}
		for _, child := range n.Content {
			if line, dup := inspect(child, active); dup != "" { return line, dup }
		}
		return 0, ""
	}
	return inspect(root, make(map[*yaml.Node]bool))
}

func unalias(n *yaml.Node) *yaml.Node {
	for i := 0; n != nil && n.Kind == yaml.AliasNode && i < 64; i++ {
		n = n.Alias
	}
	return n
}

func child(n *yaml.Node, key string) *yaml.Node {
	n = unalias(n)
	if n == nil || n.Kind != yaml.MappingNode { return nil }
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key { return unalias(n.Content[i+1]) }
	}
	return nil
}

func scalar(n *yaml.Node) string {
	n = unalias(n)
	if n == nil || n.Kind != yaml.ScalarNode { return "" }
	return n.Value
}

func sequence(n *yaml.Node) []*yaml.Node {
	n = unalias(n)
	if n == nil || n.Kind != yaml.SequenceNode { return nil }
	return n.Content
}

func mapKeys(n *yaml.Node) []string {
	n = unalias(n)
	if n == nil || n.Kind != yaml.MappingNode { return nil }
	keys := make([]string, 0, len(n.Content)/2)
	for i := 0; i+1 < len(n.Content); i += 2 { keys = append(keys, n.Content[i].Value) }
	return keys
}

func inspectMapping(n *yaml.Node, visit func(key string, value *yaml.Node)) {
	n = unalias(n)
	if n == nil { return }
	switch n.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			key, value := n.Content[i].Value, unalias(n.Content[i+1])
			visit(key, value)
			inspectMapping(value, visit)
		}
	case yaml.SequenceNode:
		for _, item := range n.Content { inspectMapping(item, visit) }
	}
}
