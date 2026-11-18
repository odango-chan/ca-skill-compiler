package compiler

// Shared Schema value helpers used by the v1alpha5 G.4/F.5 validation.

import (
	"encoding/json"
	"math/big"
	"strconv"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validJSONLiteral(node *yaml.Node) bool {
	node = unalias(node)
	if node == nil {
		return false
	}
	switch node.Kind {
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!str", "!!bool", "!!int", "!!float", "!!null":
			_, err := json.Marshal(nodeValue(node))
			return err == nil
		default:
			return false
		}
	case yaml.SequenceNode:
		for _, item := range node.Content {
			if !validJSONLiteral(item) {
				return false
			}
		}
		return true
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			if unalias(node.Content[i]).Tag != "!!str" || !validJSONLiteral(node.Content[i+1]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func schemaNumber(schema *yaml.Node, key string) *big.Rat {
	n := child(schema, key)
	if n == nil {
		return nil
	}
	v, ok := exactRational(scalar(n))
	if !ok {
		return nil
	}
	return v
}

func jsonValuesEqual(left, right *yaml.Node) bool {
	left, right = unalias(left), unalias(right)
	if left == nil || right == nil || left.Kind != right.Kind {
		return left == nil && right == nil
	}
	if left.Kind == yaml.ScalarNode {
		leftNumber := left.Tag == "!!int" || left.Tag == "!!float"
		rightNumber := right.Tag == "!!int" || right.Tag == "!!float"
		if leftNumber || rightNumber {
			if !leftNumber || !rightNumber {
				return false
			}
			l, lok := exactRational(left.Value)
			r, rok := exactRational(right.Value)
			return lok && rok && l.Cmp(r) == 0
		}
		if left.Tag == "!!null" || right.Tag == "!!null" {
			return left.Tag == "!!null" && right.Tag == "!!null"
		}
		if left.Tag == "!!bool" && right.Tag == "!!bool" {
			l, lok := strconv.ParseBool(left.Value)
			r, rok := strconv.ParseBool(right.Value)
			return lok == nil && rok == nil && l == r
		}
		return left.Tag == right.Tag && left.Value == right.Value
	}
	if len(left.Content) != len(right.Content) {
		return false
	}
	if left.Kind == yaml.SequenceNode {
		for i := range left.Content {
			if !jsonValuesEqual(left.Content[i], right.Content[i]) {
				return false
			}
		}
		return true
	}
	if left.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(left.Content); i += 2 {
			key := unalias(left.Content[i])
			other := child(right, key.Value)
			if other == nil || !jsonValuesEqual(left.Content[i+1], other) {
				return false
			}
		}
		return true
	}
	return false
}

func exactRational(raw string) (*big.Rat, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false
	}
	exponent := 0
	if i := strings.IndexAny(raw, "eE"); i >= 0 {
		parsed, err := strconv.Atoi(raw[i+1:])
		if err != nil || parsed < -10000 || parsed > 10000 {
			return nil, false
		}
		exponent, raw = parsed, raw[:i]
	}
	sign := ""
	if raw[0] == '-' || raw[0] == '+' {
		sign, raw = raw[:1], raw[1:]
	}
	parts := strings.Split(raw, ".")
	if len(parts) > 2 {
		return nil, false
	}
	whole, fraction := parts[0], ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	digits := whole + fraction
	scale := len(fraction) - exponent
	if digits == "" || len(digits) > 1024 || scale < -1024 || scale > 1024 {
		return nil, false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return nil, false
		}
	}
	numerator, ok := new(big.Int).SetString(digits, 10)
	if !ok {
		return nil, false
	}
	if sign == "-" {
		numerator.Neg(numerator)
	}
	denominator := big.NewInt(1)
	ten := big.NewInt(10)
	if scale > 0 {
		denominator.Exp(ten, big.NewInt(int64(scale)), nil)
	} else if scale < 0 {
		numerator.Mul(numerator, new(big.Int).Exp(ten, big.NewInt(int64(-scale)), nil))
	}
	return new(big.Rat).SetFrac(numerator, denominator), true
}

// nodeValue converts a normalized YAML node to a JSON-compatible Go value.
func nodeValue(n *yaml.Node) any {
	n = unalias(n)
	if n == nil {
		return nil
	}
	value, err := yaml.ToValue(n)
	if err != nil {
		return nil
	}
	return value
}

// schemaNode converts a generated Schema value to a normalized YAML node via a
// JSON round-trip, so json.Number values stay JSON numeric instances.
func schemaNode(v any) *yaml.Node {
	encoded, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	node, err := yaml.ParseValue(encoded)
	if err != nil {
		return nil
	}
	return node
}
