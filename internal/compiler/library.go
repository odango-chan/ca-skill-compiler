package compiler

import (
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type toolEntry struct {
	source              string
	line                int
	effect              string
	tool                LibraryTool
	inputNode           *yaml.Node
	outputNode          *yaml.Node
	reconcileOutputNode *yaml.Node
}

func nodeValue(n *yaml.Node) any {
	n = unalias(n)
	if n == nil {
		return nil
	}
	var value any
	if n.Decode(&value) != nil {
		return nil
	}
	return value
}

func (a *analysis) importLibraries() {
	statuses := map[string]string{}
	if a.files["lib/manifest.yaml"] != nil {
		manifest := a.parseYAML("lib/manifest.yaml")
		if manifest != nil {
			a.validateMappingFields("lib/manifest.yaml", manifest, "library manifest", "LIB_MANIFEST_PATH", "libraries")
		}
		libraries := child(manifest, "libraries")
		if libraries != nil && unalias(libraries).Kind != yaml.SequenceNode {
			a.errorAtNode("lib/manifest.yaml", libraries, "LIB_MANIFEST_PATH", "libraries must be an array")
		}
		for _, item := range sequence(libraries) {
			if unalias(item).Kind != yaml.MappingNode {
				a.errorAtNode("lib/manifest.yaml", item, "LIB_MANIFEST_PATH", "manifest library entry must be an object")
				continue
			}
			a.validateMappingFields("lib/manifest.yaml", item, "manifest library entry", "LIB_MANIFEST_PATH", "path", "status")
			p, status := scalar(child(item, "path")), scalar(child(item, "status"))
			if !isNonemptyString(child(item, "path")) || !validLibraryPath(p) {
				a.errorAt("lib/manifest.yaml", item.Line, "LIB_MANIFEST_PATH", "manifest library path must be a normalized lib/ path")
				continue
			}
			if status == "" {
				status = "unspecified"
			}
			if status != "declared" && status != "proposed" && status != "unspecified" {
				a.errorAt("lib/manifest.yaml", item.Line, "LIB_MANIFEST_STATUS", "status must be declared, proposed, or unspecified")
				continue
			}
			if _, ok := statuses[p]; ok {
				a.errorAt("lib/manifest.yaml", item.Line, "LIB_MANIFEST_DUPLICATE", "library path declared more than once: "+p)
				continue
			}
			statuses[p] = status
		}
	}
	names := make([]string, 0, len(a.files))
	for filename := range a.files {
		if strings.HasPrefix(filename, "lib/tools/") && (strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")) {
			names = append(names, filename)
		}
	}
	sort.Strings(names)
	for _, filename := range names {
		root := a.parseYAML(filename)
		if root == nil {
			continue
		}
		if scalar(child(root, "apiVersion")) != "ca-agent-demo.tool/v1" || scalar(child(root, "kind")) != "ToolCatalog" {
			a.errorAt(filename, root.Line, "LIB_SCHEMA_INVALID", "expected ca-agent-demo.tool/v1 ToolCatalog")
		}
		a.validateMappingFields(filename, root, "ToolCatalog", "LIB_SCHEMA_INVALID", "apiVersion", "kind", "metadata", "x-common", "spec")
		metadata := child(root, "metadata")
		if metadata == nil || unalias(metadata).Kind != yaml.MappingNode {
			a.errorAt(filename, root.Line, "LIB_SCHEMA_INVALID", "ToolCatalog metadata must be an object")
		} else {
			a.validateMappingFields(filename, metadata, "ToolCatalog metadata", "LIB_SCHEMA_INVALID", "name", "owner")
			for _, field := range []string{"name", "owner"} {
				if !isNonemptyString(child(metadata, field)) {
					a.errorAt(filename, metadata.Line, "LIB_SCHEMA_INVALID", "ToolCatalog metadata requires nonempty string "+field)
				}
			}
		}
		if common := child(root, "x-common"); common != nil && unalias(common).Kind != yaml.MappingNode {
			a.errorAt(filename, common.Line, "LIB_SCHEMA_INVALID", "x-common must be a mapping of local Schema fragments")
		}
		spec := child(root, "spec")
		if spec == nil || unalias(spec).Kind != yaml.MappingNode {
			a.errorAt(filename, root.Line, "LIB_SCHEMA_INVALID", "ToolCatalog spec must be an object")
		} else {
			a.validateMappingFields(filename, spec, "ToolCatalog spec", "LIB_SCHEMA_INVALID", "contracts")
		}
		contractsNode := child(spec, "contracts")
		if contractsNode != nil && unalias(contractsNode).Kind != yaml.SequenceNode {
			a.errorAt(filename, contractsNode.Line, "LIB_SCHEMA_INVALID", "ToolCatalog spec.contracts must be an array")
			continue
		}
		contracts := sequence(contractsNode)
		if len(contracts) == 0 {
			a.errorAt(filename, root.Line, "LIB_SCHEMA_INVALID", "ToolCatalog has no spec.contracts")
			continue
		}
		for _, item := range contracts {
			if unalias(item).Kind != yaml.MappingNode {
				a.errorAt(filename, item.Line, "LIB_SCHEMA_INVALID", "Tool operation entry must be an object")
				continue
			}
			a.validateMappingFields(filename, item, "Tool operation", "LIB_SCHEMA_INVALID", "operation", "adapter", "description", "effect", "inputSchema", "outputSchema", "reconcileOutputSchema")
			idNode := child(item, "operation")
			id := scalar(idNode)
			if !isNonemptyString(idNode) {
				a.errorAt(filename, item.Line, "LIB_SCHEMA_INVALID", "Tool contract has no operation ID")
				continue
			}
			for _, field := range []string{"adapter", "description"} {
				if value := child(item, field); value != nil && unalias(value).Tag != "!!str" {
					a.errorAt(filename, value.Line, "LIB_SCHEMA_INVALID", "Tool "+field+" must be a string: "+id)
				}
			}
			if a.ambiguousOperations[id] {
				continue
			}
			if prev, exists := a.tools[id]; exists {
				a.errorAt(filename, item.Line, "LIB_OPERATION_DUPLICATE", "operation "+id+" already declared in "+prev.source)
				delete(a.tools, id)
				a.ambiguousOperations[id] = true
				continue
			}
			input := child(item, "inputSchema")
			if input == nil {
				a.errorAt(filename, item.Line, "LIB_SCHEMA_INVALID", "operation "+id+" has no inputSchema")
			}
			effect := scalar(child(item, "effect"))
			if effect != "READ" && effect != "MUTATION" {
				a.errorAt(filename, item.Line, "LIB_SCHEMA_INVALID", "operation "+id+" has unsupported/missing Effect")
			}
			if input != nil && unalias(input).Kind != yaml.MappingNode {
				a.errorAt(filename, input.Line, "LIB_SCHEMA_INVALID", "inputSchema must be an object")
			}
			output := child(item, "outputSchema")
			if output != nil && unalias(output).Kind != yaml.MappingNode {
				a.errorAt(filename, output.Line, "LIB_SCHEMA_INVALID", "outputSchema must be an object")
			}
			reconcile := child(item, "reconcileOutputSchema")
			if reconcile != nil && unalias(reconcile).Kind != yaml.MappingNode {
				a.errorAt(filename, reconcile.Line, "LIB_SCHEMA_INVALID", "reconcileOutputSchema must be an object")
			}
			status := statuses[filename]
			if status == "" {
				status = "unspecified"
			}
			inputNode, outputNode := unalias(input), unalias(output)
			if inputNode != nil {
				if expanded, err := a.expandSchemaRefs(filename, root, inputNode); err != nil {
					a.errorAt(filename, inputNode.Line, "LIB_SCHEMA_INVALID", "operation "+id+" inputSchema: "+err.Error())
				} else {
					inputNode = expanded
					if err := validateSchemaShape(inputNode); err != nil {
						a.errorAt(filename, inputNode.Line, "LIB_SCHEMA_INVALID", "operation "+id+" inputSchema: "+err.Error())
					}
				}
			}
			if outputNode != nil {
				if expanded, err := a.expandSchemaRefs(filename, root, outputNode); err != nil {
					a.errorAt(filename, outputNode.Line, "LIB_SCHEMA_INVALID", "operation "+id+" outputSchema: "+err.Error())
				} else {
					outputNode = expanded
					if err := validateSchemaShape(outputNode); err != nil {
						a.errorAt(filename, outputNode.Line, "LIB_SCHEMA_INVALID", "operation "+id+" outputSchema: "+err.Error())
					}
				}
			}
			reconcileNode := (*yaml.Node)(nil)
			if reconcile != nil {
				if expanded, err := a.expandSchemaRefs(filename, root, unalias(reconcile)); err != nil {
					a.errorAt(filename, reconcile.Line, "LIB_SCHEMA_INVALID", "operation "+id+" reconcileOutputSchema: "+err.Error())
				} else if err := validateSchemaShape(expanded); err != nil {
					a.errorAt(filename, reconcile.Line, "LIB_SCHEMA_INVALID", "operation "+id+" reconcileOutputSchema: "+err.Error())
				} else {
					reconcileNode = expanded
				}
			}
			tool := LibraryTool{OperationID: id, SourcePath: filename, SourceStatus: status, DeclaredEffect: effect,
				Description: scalar(child(item, "description")), InputSchema: nodeValue(input), OutputSchema: nodeValue(output), ReconcileOutputSchema: nodeValue(reconcile)}
			a.tools[id] = toolEntry{source: filename, line: item.Line, effect: effect, tool: tool, inputNode: inputNode, outputNode: outputNode, reconcileOutputNode: reconcileNode}
		}
	}
	for p := range statuses {
		if a.files[p] == nil {
			a.errorAt("lib/manifest.yaml", 0, "LIB_MANIFEST_PATH", "manifest references a library file that was not imported: "+p)
		}
	}
}

func (a *analysis) validateMappingFields(filename string, mapping *yaml.Node, label, code string, allowed ...string) {
	permitted := make(map[string]bool, len(allowed))
	for _, field := range allowed {
		permitted[field] = true
	}
	for _, field := range mapKeys(mapping) {
		if !permitted[field] {
			value := child(mapping, field)
			line := mapping.Line
			if value != nil {
				line = value.Line
			}
			a.errorAt(filename, line, code, "unknown "+label+" field: "+field)
		}
	}
}

func isNonemptyString(node *yaml.Node) bool {
	node = unalias(node)
	return node != nil && node.Kind == yaml.ScalarNode && node.Tag == "!!str" && node.Value != ""
}

func validLibraryPath(p string) bool { return strings.HasPrefix(p, "lib/") && validPath(p) }

func (a *analysis) libraryTools() []LibraryTool {
	ids := make([]string, 0, len(a.tools))
	for id := range a.tools {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]LibraryTool, 0, len(ids))
	for _, id := range ids {
		result = append(result, a.tools[id].tool)
	}
	return result
}
