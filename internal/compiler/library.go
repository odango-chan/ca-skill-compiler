package compiler

import (
	"sort"
	"strings"
)

type toolEntry struct {
	source string
	line   int
	effect string
}

func (a *analysis) importLibraries() {
	names := make([]string, 0, len(a.files))
	for filename := range a.files {
		if strings.HasPrefix(filename, "lib/tools/") && (strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")) {
			names = append(names, filename)
		}
	}
	sort.Strings(names)
	for _, filename := range names {
		root := a.parseYAML(filename)
		if root == nil { continue }
		if scalar(child(root, "kind")) != "ToolCatalog" {
			a.errorAt(filename, root.Line, "LIB_KIND", "expected ca-agent-demo ToolCatalog")
			continue
		}
		contracts := sequence(child(child(root, "spec"), "contracts"))
		if len(contracts) == 0 {
			a.errorAt(filename, root.Line, "LIB_EMPTY", "ToolCatalog has no spec.contracts")
			continue
		}
		for _, item := range contracts {
			id := scalar(child(item, "operation"))
			if id == "" {
				a.errorAt(filename, item.Line, "LIB_OPERATION", "Tool contract has no operation ID")
				continue
			}
			if prev, exists := a.tools[id]; exists {
				a.errorAt(filename, item.Line, "LIB_DUPLICATE_OPERATION", "operation "+id+" already declared in "+prev.source)
				continue
			}
			if child(item, "inputSchema") == nil {
				a.errorAt(filename, item.Line, "LIB_INPUT_SCHEMA", "operation "+id+" has no inputSchema")
			}
			effect := scalar(child(item, "effect"))
			if effect != "READ" && effect != "MUTATION" {
				a.errorAt(filename, item.Line, "LIB_EFFECT", "operation "+id+" has unsupported/missing Effect")
			}
			a.tools[id] = toolEntry{source: filename, line: item.Line, effect: effect}
		}
	}
}
