package compiler

// Shared Python TypedDict-to-Schema derivation used by the active v1alpha4 Script
// ABI. The regex-era legacy Script parser is build-tagged out.

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func typedDictSchema(name string, classes map[string]pyClass, active map[string]bool) (map[string]any, error) {
	c, ok := classes[name]
	if !ok {
		return nil, fmt.Errorf("unknown TypedDict %s", name)
	}
	if active[name] {
		return nil, fmt.Errorf("recursive TypedDict %s is unsupported", name)
	}
	if len(c.fields) == 0 {
		return nil, fmt.Errorf("output TypedDict %s must not be empty", name)
	}
	active[name] = true
	defer delete(active, name)
	props := map[string]any{}
	required := []string{}
	for _, f := range c.fields {
		typ := strings.TrimSpace(f.typ)
		optional := false
		if strings.HasPrefix(typ, "NotRequired[") && strings.HasSuffix(typ, "]") {
			optional = true
			typ = typ[len("NotRequired[") : len(typ)-1]
		}
		schema, err := typeSchema(typ, classes, active)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", f.name, err)
		}
		props[f.name] = schema
		if !optional {
			required = append(required, f.name)
		}
	}
	result := map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		result["required"] = required
	}
	return result, nil
}

func typeSchema(expr string, classes map[string]pyClass, active map[string]bool) (map[string]any, error) {
	expr = strings.TrimSpace(expr)
	if inner, ok := generic(expr, "Annotated"); ok {
		parts := splitTop(inner)
		if len(parts) != 2 {
			return nil, fmt.Errorf("Annotated requires a type and literal constraints")
		}
		base, err := typeSchema(parts[0], classes, active)
		if err != nil {
			return nil, err
		}
		constraints, err := parseLiteralMap(parts[1])
		if err != nil {
			return nil, err
		}
		for k, v := range constraints {
			if !allowedConstraint(k, base["type"]) {
				return nil, fmt.Errorf("unsupported constraint %s for %v", k, base["type"])
			}
			if !validConstraint(k, v) {
				return nil, fmt.Errorf("invalid value for constraint %s", k)
			}
			base[k] = v
		}
		if low, ok := numberConstraint(constraints, "minLength", "minimum", "minItems"); ok {
			if high, ok := numberConstraint(constraints, "maxLength", "maximum", "maxItems"); ok && low.Cmp(high) > 0 {
				return nil, fmt.Errorf("minimum constraint exceeds maximum")
			}
		}
		return base, nil
	}
	if inner, ok := generic(expr, "Literal"); ok {
		vals := splitTop(inner)
		if len(vals) == 0 {
			return nil, fmt.Errorf("Literal must not be empty")
		}
		parsed := []any{}
		var typ string
		for _, v := range vals {
			x, e := literal(v)
			if e != nil {
				return nil, e
			}
			t := jsonType(x)
			if t == "" {
				return nil, fmt.Errorf("Literal constant %s is unsupported", v)
			}
			if typ != "" && typ != t {
				return nil, fmt.Errorf("mixed Literal types are unsupported")
			}
			typ = t
			duplicate := false
			for _, existing := range parsed {
				if reflect.DeepEqual(existing, x) {
					duplicate = true
					break
				}
			}
			if !duplicate {
				parsed = append(parsed, x)
			}
		}
		if len(parsed) == 1 {
			return map[string]any{"type": typ, "const": parsed[0]}, nil
		}
		return map[string]any{"type": typ, "enum": parsed}, nil
	}
	if inner, ok := generic(expr, "list"); ok {
		item, err := typeSchema(inner, classes, active)
		if err != nil {
			return nil, err
		}
		return map[string]any{"type": "array", "items": item}, nil
	}
	if strings.HasPrefix(expr, "dict") || expr == "Any" || expr == "object" {
		return nil, fmt.Errorf("broad output type %s is forbidden", expr)
	}
	switch expr {
	case "str":
		return map[string]any{"type": "string"}, nil
	case "int":
		return map[string]any{"type": "integer"}, nil
	case "float":
		return map[string]any{"type": "number"}, nil
	case "bool":
		return map[string]any{"type": "boolean"}, nil
	}
	if _, ok := classes[expr]; ok {
		return typedDictSchema(expr, classes, active)
	}
	return nil, fmt.Errorf("unsupported output type %s", expr)
}

func generic(s, name string) (string, bool) {
	prefix := name + "["
	if strings.HasPrefix(s, prefix) && strings.HasSuffix(s, "]") {
		return s[len(prefix) : len(s)-1], true
	}
	return "", false
}

func splitTop(s string) []string {
	out := []string{}
	depth, start := 0, 0
	var quote byte
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		switch c {
		case '[', '{', '(':
			depth++
		case ']', '}', ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	out = append(out, strings.TrimSpace(s[start:]))
	return out
}

func parseLiteralMap(s string) (map[string]any, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return nil, fmt.Errorf("Annotated constraints must be a literal mapping")
	}
	m := map[string]any{}
	body := strings.TrimSpace(s[1 : len(s)-1])
	if body == "" {
		return m, nil
	}
	for _, entry := range splitTop(body) {
		p := strings.SplitN(entry, ":", 2)
		if len(p) != 2 {
			return nil, fmt.Errorf("invalid constraint %s", entry)
		}
		keyValue, err := literal(strings.TrimSpace(p[0]))
		k, ok := keyValue.(string)
		if err != nil || !ok || k == "" {
			return nil, fmt.Errorf("constraint keys must be nonempty string literals")
		}
		if _, exists := m[k]; exists {
			return nil, fmt.Errorf("duplicate Annotated constraint %s", k)
		}
		v, err := literal(p[1])
		if err != nil {
			return nil, err
		}
		m[k] = v
	}
	return m, nil
}

func literal(s string) (any, error) {
	s = strings.TrimSpace(s)
	if len(s) >= 3 && (s[0] == 'r' || s[0] == 'R') && (s[1] == '\'' || s[1] == '"') {
		quote := s[1]
		if s[len(s)-1] != quote {
			return nil, fmt.Errorf("unterminated raw string literal")
		}
		value := s[2 : len(s)-1]
		trailingSlashes := 0
		for i := len(value) - 1; i >= 0 && value[i] == '\\'; i-- {
			trailingSlashes++
		}
		if trailingSlashes%2 != 0 {
			return nil, fmt.Errorf("raw string literal cannot end with an odd number of backslashes")
		}
		return value, nil
	}
	if s == "True" {
		return true, nil
	}
	if s == "False" {
		return false, nil
	}
	if s == "None" {
		return nil, nil
	}
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		value := s[1 : len(s)-1]
		if strings.ContainsAny(value, "'\\") {
			return nil, fmt.Errorf("unsupported escape or quote in string literal")
		}
		return value, nil
	}
	decoder := json.NewDecoder(strings.NewReader(s))
	decoder.UseNumber()
	var v any
	if err := decoder.Decode(&v); err == nil {
		var extra any
		if decoder.Decode(&extra) == io.EOF {
			return v, nil
		}
	}
	if rat, ok := exactRational(s); ok {
		normalized := strings.ReplaceAll(strings.TrimSpace(s), "_", "")
		normalized = strings.TrimPrefix(normalized, "+")
		if strings.HasPrefix(normalized, ".") {
			normalized = "0" + normalized
		} else if strings.HasPrefix(normalized, "-.") {
			normalized = "-0" + normalized[1:]
		}
		if strings.HasSuffix(normalized, ".") {
			normalized += "0"
		}
		if rat.Sign() == 0 && strings.HasPrefix(normalized, "-") {
			normalized = strings.TrimPrefix(normalized, "-")
		}
		if _, err := json.Marshal(json.Number(normalized)); err == nil {
			return json.Number(normalized), nil
		}
	}
	return nil, fmt.Errorf("non-literal type expression %s", s)
}

func jsonType(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case nil:
		return "null"
	case json.Number:
		rat, ok := exactRational(string(v.(json.Number)))
		if !ok {
			return ""
		}
		if rat.IsInt() {
			return "integer"
		}
		return "number"
	case float64:
		if float64(int64(v.(float64))) == v.(float64) {
			return "integer"
		}
		return "number"
	default:
		return ""
	}
}

func allowedConstraint(key string, typ any) bool {
	switch typ {
	case "string":
		return key == "minLength" || key == "maxLength" || key == "pattern"
	case "integer", "number":
		return key == "minimum" || key == "maximum"
	case "array":
		return key == "minItems" || key == "maxItems" || key == "uniqueItems"
	}
	return false
}

func validConstraint(key string, v any) bool {
	switch key {
	case "pattern":
		pattern, ok := v.(string)
		if !ok {
			return false
		}
		_, err := regexp.Compile(pattern)
		return err == nil
	case "uniqueItems":
		_, ok := v.(bool)
		return ok
	case "minLength", "maxLength", "minItems", "maxItems":
		n, ok := literalRational(v)
		return ok && n.Sign() >= 0 && n.IsInt()
	case "minimum", "maximum":
		_, ok := literalRational(v)
		return ok
	}
	return false
}

func numberConstraint(m map[string]any, keys ...string) (*big.Rat, bool) {
	for _, k := range keys {
		if v, ok := literalRational(m[k]); ok {
			return v, true
		}
	}
	return nil, false
}

func literalRational(value any) (*big.Rat, bool) {
	switch number := value.(type) {
	case json.Number:
		return exactRational(string(number))
	case float64:
		return exactRational(strconv.FormatFloat(number, 'g', -1, 64))
	default:
		return nil, false
	}
}
