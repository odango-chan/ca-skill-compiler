package ingresscatalog

import (
	"strconv"
	"strings"
)

// jsonPointer is a decoded RFC 6901 JSON Pointer. The empty slice is the whole
// document ("#"); every element is one unescaped reference token.
type jsonPointer []string

// parsePointerFragment parses a fragment that has already been percent-decoded
// exactly once. The fragment must be empty (whole document) or begin with "/".
// RFC 6901 escaping is validated: "~" may only introduce "~0" or "~1".
func parsePointerFragment(fragment string) (jsonPointer, error) {
	if fragment == "" {
		return nil, nil
	}
	if !strings.HasPrefix(fragment, "/") {
		return nil, newError(ErrInvalidPointer, "JSON Pointer fragment must be empty or begin with '/'")
	}
	rawTokens := strings.Split(fragment[1:], "/")
	pointer := make(jsonPointer, 0, len(rawTokens))
	for _, raw := range rawTokens {
		token, err := unescapePointerToken(raw)
		if err != nil {
			return nil, err
		}
		pointer = append(pointer, token)
	}
	return pointer, nil
}

// parseCanonicalPointer parses a pointer stored in the canonical "#/..." wire
// form used by manifests and diagnostics.
func parseCanonicalPointer(pointer string) (jsonPointer, error) {
	if pointer == "" || pointer == "#" {
		return nil, nil
	}
	if !strings.HasPrefix(pointer, "#") {
		return nil, newError(ErrInvalidPointer, "pointer must begin with '#'")
	}
	fragment := pointer[1:]
	if fragment == "" {
		return nil, nil
	}
	if !strings.HasPrefix(fragment, "/") {
		return nil, newError(ErrInvalidPointer, "pointer fragment must begin with '/'")
	}
	return parsePointerFragment(fragment)
}

func unescapePointerToken(raw string) (string, error) {
	if !strings.ContainsRune(raw, '~') {
		return raw, nil
	}
	var builder strings.Builder
	builder.Grow(len(raw))
	for i := 0; i < len(raw); i++ {
		if raw[i] != '~' {
			builder.WriteByte(raw[i])
			continue
		}
		if i+1 >= len(raw) {
			return "", newError(ErrInvalidPointer, "dangling '~' in JSON Pointer token")
		}
		switch raw[i+1] {
		case '0':
			builder.WriteByte('~')
		case '1':
			builder.WriteByte('/')
		default:
			return "", newError(ErrInvalidPointer, "invalid '~' escape in JSON Pointer token")
		}
		i++
	}
	return builder.String(), nil
}

func escapePointerToken(token string) string {
	token = strings.ReplaceAll(token, "~", "~0")
	return strings.ReplaceAll(token, "/", "~1")
}

// canonicalPointer renders a decoded pointer as "#" or "#/token/...".
func canonicalPointer(pointer jsonPointer) string {
	if len(pointer) == 0 {
		return "#"
	}
	var builder strings.Builder
	builder.WriteByte('#')
	for _, token := range pointer {
		builder.WriteByte('/')
		builder.WriteString(escapePointerToken(token))
	}
	return builder.String()
}

// appendPointer returns a new pointer with suffix appended to base.
func appendPointer(base, suffix jsonPointer) jsonPointer {
	result := make(jsonPointer, 0, len(base)+len(suffix))
	result = append(result, base...)
	result = append(result, suffix...)
	return result
}

// resolvePointer walks a JSON-compatible value. Missing locations and invalid
// array indices are rejected; RFC 6901 "-" is never a valid resolution target.
func resolvePointer(document any, pointer jsonPointer) (any, error) {
	current := document
	for _, token := range pointer {
		switch node := current.(type) {
		case map[string]any:
			next, ok := node[token]
			if !ok {
				return nil, newError(ErrMissingPointer, "JSON Pointer token '"+token+"' does not exist")
			}
			current = next
		case []any:
			index, ok := parseArrayIndex(token)
			if !ok || index >= len(node) {
				return nil, newError(ErrMissingPointer, "JSON Pointer array index '"+token+"' does not exist")
			}
			current = node[index]
		default:
			return nil, newError(ErrMissingPointer, "JSON Pointer traverses a scalar value")
		}
	}
	return current, nil
}

func parseArrayIndex(token string) (int, bool) {
	if token == "" || token == "-" {
		return 0, false
	}
	if len(token) > 1 && token[0] == '0' {
		return 0, false
	}
	for i := 0; i < len(token); i++ {
		if token[i] < '0' || token[i] > '9' {
			return 0, false
		}
	}
	index, err := strconv.Atoi(token)
	if err != nil || index < 0 {
		return 0, false
	}
	return index, true
}
