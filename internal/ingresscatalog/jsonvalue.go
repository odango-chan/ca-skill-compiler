package ingresscatalog

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

// maxSafeInteger is 2^53-1. RFC 8785/I-JSON requires numbers that are exactly
// representable as IEEE-754 doubles; integers outside this range are rejected
// before closure construction.
const maxSafeInteger int64 = 1<<53 - 1

// parseStrictJSONValue parses raw bytes through the single normalized YAML AST
// (which already rejects duplicate mapping keys, non-string keys and
// unsupported tags) and converts the result to JSON-compatible Go values.
func parseStrictJSONValue(raw []byte) (any, error) {
	root, err := yaml.ParseValue(raw)
	if err != nil {
		return nil, newError(ErrArtifactParse, err.Error())
	}
	return jsonValueFromYAML(root)
}

func jsonValueFromYAML(node *yaml.Node) (any, error) {
	if node == nil {
		return nil, newError(ErrUnsupportedValue, "nil YAML node")
	}
	switch node.Kind {
	case yaml.AliasNode:
		return jsonValueFromYAML(node.Alias)
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!str":
			return node.Value, nil
		case "!!null":
			return nil, nil
		case "!!bool":
			switch node.Value {
			case "true":
				return true, nil
			case "false":
				return false, nil
			default:
				return nil, newError(ErrUnsupportedValue, "unsupported Boolean lexeme: "+node.Value)
			}
		case "!!int":
			value, err := yaml.ParseIntegerLexeme(node.Value)
			if err != nil {
				return nil, newError(ErrUnsupportedValue, "unsupported integer lexeme: "+node.Value)
			}
			return normalizeInteger(value)
		case "!!float":
			value, err := yaml.ParseFiniteFloatLexeme(node.Value)
			if err != nil {
				return nil, newError(ErrUnsupportedValue, "unsupported floating-point lexeme: "+node.Value)
			}
			return normalizeFloat(value)
		default:
			return nil, newError(ErrUnsupportedValue, "unsupported YAML scalar tag: "+node.Tag)
		}
	case yaml.SequenceNode:
		values := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			value, err := jsonValueFromYAML(child)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, nil
	case yaml.MappingNode:
		values := make(map[string]any, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			if keyNode.Kind == yaml.AliasNode {
				keyNode = keyNode.Alias
			}
			if keyNode == nil || keyNode.Kind != yaml.ScalarNode || keyNode.Tag != "!!str" {
				return nil, newError(ErrUnsupportedValue, "JSON object keys must be strings")
			}
			value, err := jsonValueFromYAML(node.Content[i+1])
			if err != nil {
				return nil, err
			}
			values[keyNode.Value] = value
		}
		return values, nil
	default:
		return nil, newError(ErrUnsupportedValue, "unsupported YAML AST node kind")
	}
}

// normalizeJSONCompatibleValue converts a caller-supplied parsed document into
// the exact JSON-compatible representation used for equality checks against the
// parse of the raw artifact bytes. It rejects non-finite numbers and unsafe
// integers; integral values are represented as int64 and all other finite
// numbers as float64.
func normalizeJSONCompatibleValue(value any) (any, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case bool, string:
		return typed, nil
	case int:
		return normalizeInteger(int64(typed))
	case int8:
		return normalizeInteger(int64(typed))
	case int16:
		return normalizeInteger(int64(typed))
	case int32:
		return normalizeInteger(int64(typed))
	case int64:
		return normalizeInteger(typed)
	case uint:
		return normalizeUnsignedInteger(uint64(typed))
	case uint8:
		return normalizeUnsignedInteger(uint64(typed))
	case uint16:
		return normalizeUnsignedInteger(uint64(typed))
	case uint32:
		return normalizeUnsignedInteger(uint64(typed))
	case uint64:
		return normalizeUnsignedInteger(typed)
	case float32:
		return normalizeFloat(float64(typed))
	case float64:
		return normalizeFloat(typed)
	case json.Number:
		return normalizeJSONNumber(typed)
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, child := range typed {
			normalized, err := normalizeJSONCompatibleValue(child)
			if err != nil {
				return nil, err
			}
			result[key] = normalized
		}
		return result, nil
	case []any:
		result := make([]any, 0, len(typed))
		for _, child := range typed {
			normalized, err := normalizeJSONCompatibleValue(child)
			if err != nil {
				return nil, err
			}
			result = append(result, normalized)
		}
		return result, nil
	case map[string]string:
		result := make(map[string]any, len(typed))
		for key, child := range typed {
			result[key] = child
		}
		return result, nil
	default:
		return nil, newError(ErrUnsupportedValue, fmt.Sprintf("unsupported parsed document value %T", value))
	}
}

func normalizeUnsignedInteger(value uint64) (any, error) {
	if value > uint64(maxSafeInteger) {
		return nil, newError(ErrUnsupportedValue, "integer exceeds the exact safe-integer range")
	}
	return int64(value), nil
}

func normalizeInteger(value int64) (any, error) {
	if value > maxSafeInteger || value < -maxSafeInteger {
		return nil, newError(ErrUnsupportedValue, "integer exceeds the exact safe-integer range")
	}
	return value, nil
}

func normalizeFloat(value float64) (any, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, newError(ErrUnsupportedValue, "non-finite numbers cannot be represented as JSON")
	}
	// A finite float that is integral and inside the safe-integer range has one
	// exact JSON representation; normalize it so that 1 and 1.0 compare equal.
	if value == math.Trunc(value) && value <= float64(maxSafeInteger) && value >= -float64(maxSafeInteger) {
		return int64(value), nil
	}
	return value, nil
}

func normalizeJSONNumber(value json.Number) (any, error) {
	text := string(value)
	if integer, err := strconv.ParseInt(text, 10, 64); err == nil {
		return normalizeInteger(integer)
	}
	unsigned, unsignedErr := strconv.ParseUint(text, 10, 64)
	if unsignedErr == nil {
		return normalizeUnsignedInteger(unsigned)
	}
	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil, newError(ErrUnsupportedValue, "unsupported JSON number: "+text)
	}
	return normalizeFloat(parsed)
}

// jsonValuesEqual reports deep equality over normalized JSON-compatible values.
func jsonValuesEqual(left, right any) (bool, error) {
	normalizedLeft, err := normalizeJSONCompatibleValue(left)
	if err != nil {
		return false, err
	}
	normalizedRight, err := normalizeJSONCompatibleValue(right)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(normalizedLeft, normalizedRight), nil
}
