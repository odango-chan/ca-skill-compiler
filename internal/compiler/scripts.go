package compiler

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

var (
	classRE           = regexp.MustCompile(`^class ([A-Za-z_][A-Za-z0-9_]*)\(TypedDict\):\s*$`)
	fieldRE           = regexp.MustCompile(`^\s+([A-Za-z_][A-Za-z0-9_]*):\s*(.+?)\s*(#.*)?$`)
	functionalStartRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*\s*=\s*TypedDict\s*\(`)
	functionalNameRE  = regexp.MustCompile(`^\s*TypedDict\s*\(\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]\s*,\s*\{`)
	functionalFieldRE = regexp.MustCompile(`^\s*['"]([^'"]+)['"]\s*:\s*(.*)\s*$`)
	mainRE            = regexp.MustCompile(`(?m)^def main\(input:\s*dict\)\s*->\s*([A-Za-z_][A-Za-z0-9_]*):`)
	mainDeclRE        = regexp.MustCompile(`(?m)^(?:async\s+)?def main\b`)
)

type pyField struct{ name, typ string }
type pyClass struct{ fields []pyField }

func (a *analysis) compileScripts(selected map[string]bool) {
	for _, skill := range sortedStringMapKeys(selected) {
		s := a.skills[skill]
		if s == nil {
			continue
		}
		for _, rel := range sortedStringMapKeys(s.scripts) {
			line := s.scripts[rel]
			path := "skills/" + skill + "/" + rel
			data, ok := a.files[path]
			if !ok {
				continue
			}
			schema, err := parseScriptOutput(string(data))
			if err != nil {
				a.errorAt(path, line, "SCRIPT_TYPE_UNSUPPORTED", err.Error())
				continue
			}
			b, err := json.MarshalIndent(schema, "", "  ")
			if err != nil {
				a.errorAt(path, line, "SCRIPT_TYPE_UNSUPPORTED", err.Error())
				continue
			}
			b = append(b, '\n')
			name := strings.TrimSuffix(strings.TrimPrefix(rel, "scripts/"), ".py")
			out := "skills/" + skill + "/schemas/generated/" + name + ".output.schema.json"
			if !validPath(out) {
				a.errorAt(path, line, "OUTPUT_LIMIT", "generated Script Schema path exceeds the BuildResult contract: "+out)
				continue
			}
			if sourcePath, collides := collidingPath(a.files, out); collides {
				a.errorAt(path, line, "PROJECT_DUPLICATE_FILE", "generated Script schema path conflicts with source "+sourcePath+": "+out)
				continue
			}
			for other, files := range a.generated {
				for _, f := range files {
					if pathsCollide(f.Path, out) {
						a.errorAt(path, line, "PROJECT_DUPLICATE_FILE", "generated Script schema collision with "+other)
						out = ""
					}
				}
			}
			if out != "" {
				a.generated[skill] = append(a.generated[skill], File{Path: out, Bytes: b})
			}
		}
	}
}

func collidingPath(files map[string][]byte, candidate string) (string, bool) {
	for filename := range files {
		if pathsCollide(filename, candidate) {
			return filename, true
		}
	}
	return "", false
}

func pathsCollide(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}

func parseScriptOutput(src string) (map[string]any, error) {
	code := maskPythonStringsAndComments(src)
	matches := mainRE.FindAllStringSubmatch(code, -1)
	if len(matches) != 1 || len(mainDeclRE.FindAllString(code, -1)) != 1 {
		return nil, fmt.Errorf("expected main(input: dict) -> nonempty TypedDict output; legacy metadata is not an interface")
	}
	match := matches[0]
	imports := map[string]bool{}
	sourceLines, codeLines := strings.Split(src, "\n"), strings.Split(code, "\n")
	for _, line := range codeLines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "from typing import ") {
			for _, name := range strings.Split(strings.TrimPrefix(line, "from typing import "), ",") {
				name = strings.TrimSpace(strings.Split(name, " as ")[0])
				if name != "" {
					imports[name] = true
				}
			}
		}
	}
	if !imports["TypedDict"] {
		return nil, fmt.Errorf("Output declarations must import TypedDict from typing")
	}
	classes := map[string]pyClass{}
	classSeen := map[string]bool{}
	fieldSeen := map[string]map[string]bool{}
	functionalLines, functionalClasses, err := parseFunctionalTypedDicts(src, code)
	if err != nil {
		return nil, err
	}
	for name, class := range functionalClasses {
		if classSeen[name] {
			return nil, fmt.Errorf("duplicate TypedDict declaration %s", name)
		}
		classSeen[name] = true
		classes[name] = class
	}
	current := ""
	for i := 0; i < len(sourceLines); i++ {
		line := sourceLines[i]
		trim := strings.TrimSpace(line)
		codeLine := codeLines[i]
		codeTrim := strings.TrimSpace(codeLine)
		if functionalLines[i] {
			current = ""
			continue
		}
		if m := classRE.FindStringSubmatch(codeLine); m != nil {
			current = m[1]
			if classSeen[current] {
				return nil, fmt.Errorf("duplicate TypedDict declaration %s", current)
			}
			classSeen[current] = true
			fieldSeen[current] = map[string]bool{}
			classes[current] = pyClass{}
			continue
		}
		if strings.HasPrefix(codeTrim, "class ") {
			current = ""
			continue
		}
		if codeTrim == "" {
			if current != "" && trim != "" && !strings.HasPrefix(trim, "#") {
				return nil, fmt.Errorf("unsupported TypedDict body statement in %s: %s", current, trim)
			}
			continue
		}
		if current != "" {
			if codeLine[0] != ' ' && codeLine[0] != '\t' {
				current = ""
			} else {
				f := fieldRE.FindStringSubmatch(line)
				if f == nil {
					return nil, fmt.Errorf("unsupported TypedDict body statement in %s: %s", current, trim)
				}
				typ := f[2]
				balance := bracketBalance(codeLine)
				for balance > 0 {
					i++
					if i >= len(sourceLines) {
						return nil, fmt.Errorf("unterminated annotation in %s.%s", current, f[1])
					}
					typ += " " + strings.TrimSpace(sourceLines[i])
					balance += bracketBalance(codeLines[i])
				}
				if fieldSeen[current][f[1]] {
					return nil, fmt.Errorf("duplicate TypedDict field %s.%s", current, f[1])
				}
				fieldSeen[current][f[1]] = true
				classes[current] = pyClass{fields: append(classes[current].fields, pyField{f[1], typ})}
			}
		}
	}
	for className, class := range classes {
		for _, field := range class.fields {
			for _, name := range []string{"Literal", "Annotated", "NotRequired"} {
				if strings.Contains(field.typ, name+"[") && !imports[name] {
					return nil, fmt.Errorf("%s must be imported from typing (used by %s.%s)", name, className, field.name)
				}
			}
		}
	}
	name := match[1]
	if _, ok := classes[name]; !ok {
		return nil, fmt.Errorf("main return type %s is not a local TypedDict", name)
	}
	return typedDictSchema(name, classes, map[string]bool{})
}

func parseFunctionalTypedDicts(src, code string) (map[int]bool, map[string]pyClass, error) {
	lines := map[int]bool{}
	classes := map[string]pyClass{}
	codeLines := strings.Split(code, "\n")
	offset := 0
	for lineIndex, codeLine := range codeLines {
		trimmed := strings.TrimSpace(codeLine)
		if !functionalStartRE.MatchString(trimmed) {
			offset += len(codeLine) + 1
			continue
		}
		lineStart := offset
		openParen := strings.Index(codeLine, "(")
		openParen += lineStart
		callEnd, ok := matchingDelimiter(code, openParen, '(', ')')
		if !ok {
			return nil, nil, fmt.Errorf("unterminated functional TypedDict declaration")
		}
		callStart := lineStart + strings.Index(codeLine, "TypedDict")
		callSource := src[callStart : callEnd+1]
		nameMatch := functionalNameRE.FindStringSubmatch(callSource)
		if nameMatch == nil {
			return nil, nil, fmt.Errorf("functional TypedDict requires a literal name and field mapping")
		}
		name := nameMatch[1]
		openBraceRel := strings.Index(code[callStart:callEnd+1], "{")
		if openBraceRel < 0 {
			return nil, nil, fmt.Errorf("functional TypedDict %s requires a field mapping", name)
		}
		openBrace := callStart + openBraceRel
		closeBrace, ok := matchingDelimiter(code, openBrace, '{', '}')
		if !ok || closeBrace > callEnd {
			return nil, nil, fmt.Errorf("unterminated field mapping in functional TypedDict %s", name)
		}
		body := src[openBrace+1 : closeBrace]
		class := pyClass{}
		seen := map[string]bool{}
		for _, item := range splitTop(body) {
			if strings.TrimSpace(item) == "" {
				continue
			}
			field := functionalFieldRE.FindStringSubmatch(item)
			if field == nil {
				return nil, nil, fmt.Errorf("unsupported functional TypedDict field in %s: %s", name, strings.TrimSpace(item))
			}
			key := field[1]
			if key == "" {
				return nil, nil, fmt.Errorf("functional TypedDict %s has an invalid field name", name)
			}
			if seen[key] {
				return nil, nil, fmt.Errorf("duplicate TypedDict field %s.%s", name, key)
			}
			seen[key] = true
			typ := strings.TrimSpace(field[2])
			if typ == "" {
				return nil, nil, fmt.Errorf("functional TypedDict field %s.%s has no type", name, key)
			}
			class.fields = append(class.fields, pyField{name: key, typ: typ})
		}
		classes[name] = class
		for i := lineIndex; i <= lineIndex+strings.Count(code[lineStart:callEnd+1], "\n"); i++ {
			lines[i] = true
		}
		consumed := strings.Count(code[lineStart:callEnd+1], "\n")
		for i := 0; i <= consumed && lineIndex+i < len(codeLines); i++ {
			offset += len(codeLines[lineIndex+i]) + 1
		}
		lineIndex += consumed
	}
	return lines, classes, nil
}

func matchingDelimiter(src string, open int, left, right byte) (int, bool) {
	if open < 0 || open >= len(src) || src[open] != left {
		return 0, false
	}
	depth := 0
	for i := open; i < len(src); i++ {
		if src[i] == left {
			depth++
		} else if src[i] == right {
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func bracketBalance(code string) int {
	balance := 0
	for i := 0; i < len(code); i++ {
		switch code[i] {
		case '(', '[', '{':
			balance++
		case ')', ']', '}':
			balance--
		}
	}
	return balance
}

func maskPythonStringsAndComments(src string) string {
	masked := []byte(src)
	var quote byte
	triple, escaped, comment := false, false, false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if comment {
			if c == '\n' {
				comment = false
			} else {
				masked[i] = ' '
			}
			continue
		}
		if quote != 0 {
			if c == '\n' {
				if !triple && !escaped {
					quote = 0
				}
				escaped = false
				continue
			}
			if escaped {
				masked[i] = ' '
				escaped = false
				continue
			}
			if c == '\\' {
				masked[i] = ' '
				escaped = true
				continue
			}
			if c == quote && triple && i+2 < len(src) && src[i+1] == quote && src[i+2] == quote {
				masked[i], masked[i+1], masked[i+2] = ' ', ' ', ' '
				i += 2
				quote, triple = 0, false
				continue
			}
			masked[i] = ' '
			if c == quote && !triple {
				quote = 0
			}
			continue
		}
		if c == '#' {
			comment = true
			masked[i] = ' '
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			if i+2 < len(src) && src[i+1] == c && src[i+2] == c {
				triple = true
				masked[i], masked[i+1], masked[i+2] = ' ', ' ', ' '
				i += 2
			} else {
				triple = false
				masked[i] = ' '
			}
		}
	}
	return string(masked)
}

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
