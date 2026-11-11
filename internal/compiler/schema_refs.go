package compiler

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// expandSchemaRefs resolves only explicit project-local JSON Pointer references.
// It never performs network or filesystem access; every document comes from the
// current in-memory project file map.
func (a *analysis) expandSchemaRefs(filename string, document, schema *yaml.Node) (*yaml.Node, error) {
	return a.expandSchemaNode(filename, document, schema, map[string]bool{})
}

func (a *analysis) expandSchemaNode(filename string, document, schema *yaml.Node, active map[string]bool) (*yaml.Node, error) {
	schema = unalias(schema)
	if schema == nil {
		return nil, fmt.Errorf("schema node is empty")
	}
	if schema.Kind == yaml.MappingNode {
		if ref := child(schema, "$ref"); ref != nil {
			if len(mapKeys(schema)) != 1 {
				return nil, fmt.Errorf("$ref siblings are unsupported")
			}
			refText := scalar(ref)
			if refText == "" || strings.Contains(refText, "%") || strings.Contains(refText, "?") {
				return nil, fmt.Errorf("invalid project-local $ref: %q", refText)
			}
			refPath, fragment, _ := strings.Cut(refText, "#")
			if strings.Contains(refPath, "://") || strings.HasPrefix(refPath, "/") || strings.Contains(refPath, "\\") {
				return nil, fmt.Errorf("remote or absolute $ref is forbidden: %q", refText)
			}
			targetFile := filename
			if refPath != "" {
				if strings.HasPrefix(refPath, "skills/") || strings.HasPrefix(refPath, "lib/") {
					targetFile = path.Clean(refPath)
				} else {
					targetFile = path.Clean(path.Join(path.Dir(filename), refPath))
				}
			}
			if !validPath(targetFile) || a.files[targetFile] == nil {
				return nil, fmt.Errorf("$ref target is not an imported project file: %s", targetFile)
			}
			if targetFile != filename {
				if strings.HasPrefix(filename, "lib/") && !strings.HasPrefix(targetFile, "lib/") {
					return nil, fmt.Errorf("Library Schema references must remain in lib/: %s", targetFile)
				}
				if strings.HasPrefix(filename, "skills/") && strings.HasPrefix(targetFile, "skills/") {
					fromSkill, toSkill := strings.SplitN(filename, "/", 3)[1], strings.SplitN(targetFile, "/", 3)[1]
					if fromSkill != toSkill {
						return nil, fmt.Errorf("Schema references cannot cross Skill ownership: %s", targetFile)
					}
				}
				if a.schemaRefs[filename] == nil {
					a.schemaRefs[filename] = map[string]bool{}
				}
				a.schemaRefs[filename][targetFile] = true
			}
			targetDocument := document
			if targetFile != filename {
				targetDocument = a.parseYAML(targetFile)
				if targetDocument == nil {
					return nil, fmt.Errorf("$ref target cannot be parsed: %s", targetFile)
				}
			}
			target, err := jsonPointer(targetDocument, fragment)
			if err != nil {
				return nil, fmt.Errorf("$ref %q: %w", refText, err)
			}
			identity := targetFile + "#" + fragment
			if active[identity] {
				return nil, fmt.Errorf("cyclic $ref: %s", identity)
			}
			active[identity] = true
			expanded, err := a.expandSchemaNode(targetFile, targetDocument, target, active)
			delete(active, identity)
			if err == nil && unalias(expanded).Kind != yaml.MappingNode {
				err = fmt.Errorf("$ref target must resolve to a Schema object")
			}
			return expanded, err
		}
		clone := *schema
		clone.Content = nil
		for i := 0; i+1 < len(schema.Content); i += 2 {
			key := schema.Content[i]
			rawValue := schema.Content[i+1]
			var value *yaml.Node
			var err error
			switch key.Value {
			case "$defs", "definitions":
				if _, err = a.expandSchemaMap(filename, document, rawValue, active); err != nil {
					return nil, err
				}
				continue
			case "properties", "patternProperties", "dependentSchemas":
				value, err = a.expandSchemaMap(filename, document, rawValue, active)
			case "items", "additionalProperties", "unevaluatedProperties", "unevaluatedItems", "propertyNames", "contains", "not", "if", "then", "else", "contentSchema":
				value, err = a.expandSchemaValue(filename, document, rawValue, active)
			case "allOf", "anyOf", "oneOf":
				value, err = a.expandSchemaSequence(filename, document, rawValue, active)
			default:
				value = cloneYAMLNode(rawValue)
			}
			if err != nil {
				return nil, err
			}
			clone.Content = append(clone.Content, cloneYAMLNode(key), value)
		}
		return &clone, nil
	}
	return cloneYAMLNode(schema), nil
}

func (a *analysis) expandSchemaMap(filename string, document, mapping *yaml.Node, active map[string]bool) (*yaml.Node, error) {
	mapping = unalias(mapping)
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return cloneYAMLNode(mapping), nil
	}
	clone := *mapping
	clone.Content = nil
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		value, err := a.expandSchemaNode(filename, document, mapping.Content[i+1], active)
		if err != nil {
			return nil, err
		}
		clone.Content = append(clone.Content, cloneYAMLNode(mapping.Content[i]), value)
	}
	return &clone, nil
}

func (a *analysis) expandSchemaValue(filename string, document, node *yaml.Node, active map[string]bool) (*yaml.Node, error) {
	node = unalias(node)
	if node != nil && node.Kind == yaml.SequenceNode {
		return a.expandSchemaSequence(filename, document, node, active)
	}
	if node != nil && node.Kind == yaml.MappingNode {
		return a.expandSchemaNode(filename, document, node, active)
	}
	return cloneYAMLNode(node), nil
}

func (a *analysis) expandSchemaSequence(filename string, document, node *yaml.Node, active map[string]bool) (*yaml.Node, error) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.SequenceNode {
		return cloneYAMLNode(node), nil
	}
	clone := *node
	clone.Content = make([]*yaml.Node, 0, len(node.Content))
	for _, item := range node.Content {
		value, err := a.expandSchemaNode(filename, document, item, active)
		if err != nil {
			return nil, err
		}
		clone.Content = append(clone.Content, value)
	}
	return &clone, nil
}

func jsonPointer(document *yaml.Node, fragment string) (*yaml.Node, error) {
	if fragment == "" {
		return unalias(document), nil
	}
	if !strings.HasPrefix(fragment, "/") {
		return nil, fmt.Errorf("fragment must be an empty pointer or JSON Pointer")
	}
	current := unalias(document)
	for _, part := range strings.Split(fragment[1:], "/") {
		var decoded strings.Builder
		for i := 0; i < len(part); i++ {
			if part[i] != '~' {
				decoded.WriteByte(part[i])
				continue
			}
			if i+1 >= len(part) || (part[i+1] != '0' && part[i+1] != '1') {
				return nil, fmt.Errorf("invalid JSON Pointer escape in %q", part)
			}
			i++
			if part[i] == '0' {
				decoded.WriteByte('~')
			} else {
				decoded.WriteByte('/')
			}
		}
		part = decoded.String()
		switch current.Kind {
		case yaml.MappingNode:
			current = child(current, part)
		case yaml.SequenceNode:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(current.Content) {
				return nil, fmt.Errorf("pointer component %q is not a sequence index", part)
			}
			current = current.Content[index]
		default:
			return nil, fmt.Errorf("pointer component %q cannot be resolved", part)
		}
		if current == nil {
			return nil, fmt.Errorf("pointer component %q does not exist", part)
		}
		current = unalias(current)
	}
	return current, nil
}

func cloneYAMLNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	copy := *node
	copy.Content = make([]*yaml.Node, len(node.Content))
	for i, child := range node.Content {
		copy.Content[i] = cloneYAMLNode(child)
	}
	return &copy
}
