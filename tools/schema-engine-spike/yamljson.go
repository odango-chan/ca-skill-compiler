package spike

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var jsonNumberPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)

// yamlJSON converts a parsed YAML document to JSON data without routing numbers
// through float64. The original yaml.Node remains available to callers for
// source positions; this spike only measures semantic conversion and engines.
func YAMLJSON(root *yaml.Node) (any, error) {
	if root == nil {
		return nil, fmt.Errorf("nil YAML node")
	}
	if root.Kind == yaml.DocumentNode {
		if len(root.Content) != 1 {
			return nil, fmt.Errorf("expected one YAML document")
		}
		root = root.Content[0]
	}
	return convertYAMLNode(root, make(map[*yaml.Node]bool), 0, new(int))
}

func convertYAMLNode(n *yaml.Node, active map[*yaml.Node]bool, depth int, count *int) (any, error) {
	if depth > 256 {
		return nil, fmt.Errorf("YAML nesting exceeds 256")
	}
	*count++
	if *count > 1_000_000 {
		return nil, fmt.Errorf("YAML node count exceeds 1000000")
	}
	if n.Kind == yaml.AliasNode {
		if n.Alias == nil || active[n.Alias] {
			return nil, fmt.Errorf("cyclic or unresolved YAML alias")
		}
		return convertYAMLNode(n.Alias, active, depth+1, count)
	}
	if active[n] {
		return nil, fmt.Errorf("cyclic YAML node")
	}
	active[n] = true
	defer delete(active, n)

	switch n.Kind {
	case yaml.ScalarNode:
		return convertScalar(n)
	case yaml.SequenceNode:
		out := make([]any, 0, len(n.Content))
		for _, child := range n.Content {
			v, err := convertYAMLNode(child, active, depth+1, count)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case yaml.MappingNode:
		if len(n.Content)%2 != 0 {
			return nil, fmt.Errorf("malformed YAML mapping")
		}
		out := make(map[string]any, len(n.Content)/2)
		for i := 0; i < len(n.Content); i += 2 {
			key := n.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
				return nil, fmt.Errorf("JSON object key must be a YAML string")
			}
			if _, exists := out[key.Value]; exists {
				return nil, fmt.Errorf("duplicate YAML mapping key %q", key.Value)
			}
			v, err := convertYAMLNode(n.Content[i+1], active, depth+1, count)
			if err != nil {
				return nil, err
			}
			out[key.Value] = v
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported YAML node kind %d", n.Kind)
	}
}

func convertScalar(n *yaml.Node) (any, error) {
	s := strings.TrimSpace(n.Value)
	switch n.Tag {
	case "!!str":
		return n.Value, nil
	case "!!null":
		return nil, nil
	case "!!bool":
		return strings.EqualFold(s, "true"), nil
	case "!!int":
		return yamlInteger(s)
	case "!!float":
		return yamlDecimal(s)
	default:
		return nil, fmt.Errorf("YAML tag %q is not representable in JSON", n.Tag)
	}
}

func yamlInteger(s string) (json.Number, error) {
	s = strings.ReplaceAll(s, "_", "")
	sign := ""
	if strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") {
		if s[0] == '-' {
			sign = "-"
		}
		s = s[1:]
	}
	base := 10
	switch {
	case strings.HasPrefix(strings.ToLower(s), "0x"):
		base, s = 16, s[2:]
	case strings.HasPrefix(strings.ToLower(s), "0o"):
		base, s = 8, s[2:]
	case strings.HasPrefix(strings.ToLower(s), "0b"):
		base, s = 2, s[2:]
	}
	n, ok := new(big.Int).SetString(s, base)
	if !ok {
		return "", fmt.Errorf("invalid YAML integer %q", s)
	}
	if sign == "-" {
		n.Neg(n)
	}
	return json.Number(n.String()), nil
}

func yamlDecimal(s string) (json.Number, error) {
	s = strings.ReplaceAll(s, "_", "")
	s = strings.TrimPrefix(s, "+")
	lower := strings.ToLower(s)
	if lower == ".inf" || lower == "-.inf" || lower == ".nan" || lower == "-.nan" || lower == "+.inf" || lower == "+.nan" {
		return "", fmt.Errorf("non-finite YAML number %q is not JSON", s)
	}
	if strings.HasPrefix(s, ".") {
		s = "0" + s
	} else if strings.HasPrefix(s, "-.") {
		s = "-0" + s[1:]
	}
	if strings.Contains(s, ".") {
		mantissa := s
		if i := strings.IndexAny(s, "eE"); i >= 0 {
			mantissa = s[:i]
		}
		if strings.HasSuffix(mantissa, ".") {
			if strings.ContainsAny(s, "eE") {
				i := strings.IndexAny(s, "eE")
				s = s[:i] + "0" + s[i:]
			} else {
				s += "0"
			}
		}
	}
	if !jsonNumberPattern.MatchString(s) || !json.Valid([]byte(s)) {
		return "", fmt.Errorf("YAML number %q cannot be represented as a JSON number", s)
	}
	return json.Number(s), nil
}
