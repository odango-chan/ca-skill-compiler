package ingresscatalog

import (
	"strings"
	"unicode/utf8"
)

// uriReference is the subset of RFC 3986 URI-reference structure needed by
// Schema Closure Digest v1. Query strings are retained only because JSON Schema
// 2020-12 $id allows them; canonical document URIs reject them.
type uriReference struct {
	hasScheme    bool
	scheme       string
	hasAuthority bool
	host         string
	port         string
	path         string
	hasQuery     bool
	query        string
	hasFragment  bool
	fragment     string
}

const uriUnreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
const uriSubDelims = "!$&'()*+,;="

func isURISchemeChar(c byte, first bool) bool {
	if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
		return true
	}
	if first {
		return false
	}
	return c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.'
}

func isHexDigit(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'A' && c <= 'F' || c >= 'a' && c <= 'f'
}

func hexValue(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return c - 'a' + 10
	}
}

func isUnreservedByte(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' ||
		c == '-' || c == '.' || c == '_' || c == '~'
}

func isSubDelim(c byte) bool {
	return strings.IndexByte(uriSubDelims, c) >= 0
}

func validatePercentEscapes(raw string) error {
	for i := 0; i < len(raw); i++ {
		if raw[i] != '%' {
			continue
		}
		if i+2 >= len(raw) || !isHexDigit(raw[i+1]) || !isHexDigit(raw[i+2]) {
			return newError(ErrInvalidURI, "malformed percent escape")
		}
		i += 2
	}
	return nil
}

// parseURIReference parses raw without normalizing. Callers must canonicalize
// before using the result as an identity key.
func parseURIReference(raw string) (*uriReference, error) {
	if raw == "" {
		return &uriReference{}, nil
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] <= 0x20 || raw[i] >= 0x7f || raw[i] == '\\' {
			return nil, newError(ErrInvalidURI, "URI reference contains whitespace, control bytes or a backslash")
		}
	}
	if err := validatePercentEscapes(raw); err != nil {
		return nil, err
	}
	ref := &uriReference{}
	remaining := raw
	if hash := strings.IndexByte(remaining, '#'); hash >= 0 {
		ref.hasFragment = true
		ref.fragment = remaining[hash+1:]
		remaining = remaining[:hash]
		if strings.ContainsRune(ref.fragment, '#') {
			return nil, newError(ErrInvalidURI, "URI reference contains more than one fragment separator")
		}
	}
	if question := strings.IndexByte(remaining, '?'); question >= 0 {
		ref.hasQuery = true
		ref.query = remaining[question+1:]
		remaining = remaining[:question]
	}
	if colon := schemeBoundary(remaining); colon > 0 {
		for i := 0; i < colon; i++ {
			if !isURISchemeChar(remaining[i], i == 0) {
				return nil, newError(ErrInvalidURI, "invalid URI scheme")
			}
		}
		ref.hasScheme = true
		ref.scheme = remaining[:colon]
		remaining = remaining[colon+1:]
	}
	if strings.HasPrefix(remaining, "//") {
		ref.hasAuthority = true
		rest := remaining[2:]
		authority := rest
		if slash := strings.IndexByte(rest, '/'); slash >= 0 {
			authority = rest[:slash]
			ref.path = rest[slash:]
		}
		host, port, err := parseAuthority(authority)
		if err != nil {
			return nil, err
		}
		ref.host, ref.port = host, port
	} else {
		ref.path = remaining
	}
	return ref, nil
}

func schemeBoundary(value string) int {
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case ':':
			if i == 0 {
				return -1
			}
			return i
		case '/', '?', '#':
			return -1
		}
	}
	return -1
}

func parseAuthority(authority string) (string, string, error) {
	if strings.ContainsRune(authority, '@') {
		return "", "", newError(ErrInvalidURI, "URI user-info is not allowed")
	}
	host := authority
	port := ""
	if strings.HasPrefix(authority, "[") {
		end := strings.IndexByte(authority, ']')
		if end < 0 {
			return "", "", newError(ErrInvalidURI, "unterminated IPv6 literal host")
		}
		host = authority[:end+1]
		rest := authority[end+1:]
		if rest != "" {
			if !strings.HasPrefix(rest, ":") {
				return "", "", newError(ErrInvalidURI, "invalid characters after IPv6 literal host")
			}
			port = rest[1:]
		}
	} else if colon := strings.LastIndexByte(authority, ':'); colon >= 0 {
		candidate := authority[colon+1:]
		if candidate != "" && isAllDigits(candidate) {
			host = authority[:colon]
			port = candidate
		}
	}
	if host == "" {
		return "", "", newError(ErrInvalidURI, "URI host must not be empty")
	}
	if !validHost(host) {
		return "", "", newError(ErrInvalidURI, "URI host contains unsupported characters")
	}
	if port != "" && !isAllDigits(port) {
		return "", "", newError(ErrInvalidURI, "URI port must be numeric")
	}
	return host, port, nil
}

func isAllDigits(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func validHost(host string) bool {
	for i := 0; i < len(host); i++ {
		c := host[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			continue
		}
		switch c {
		case '-', '.', '_', '~', '[', ']', ':':
			continue
		}
		return false
	}
	return true
}

// percentDecodeOnce decodes the fragment exactly once and requires the result
// to be strict UTF-8.
func percentDecodeOnce(value string) (string, error) {
	decoded := make([]byte, 0, len(value))
	for i := 0; i < len(value); i++ {
		if value[i] != '%' {
			decoded = append(decoded, value[i])
			continue
		}
		if i+2 >= len(value) || !isHexDigit(value[i+1]) || !isHexDigit(value[i+2]) {
			return "", newError(ErrInvalidURI, "malformed percent escape in fragment")
		}
		decoded = append(decoded, hexValue(value[i+1])<<4|hexValue(value[i+2]))
		i += 2
	}
	if !utf8.Valid(decoded) {
		return "", newError(ErrInvalidURI, "percent-decoded fragment is not valid UTF-8")
	}
	return string(decoded), nil
}

// normalizePercent decodes percent-encoded unreserved characters and uppercases
// every remaining percent-escape.
func normalizePercent(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for i := 0; i < len(value); i++ {
		if value[i] != '%' || i+2 >= len(value) {
			builder.WriteByte(value[i])
			continue
		}
		decoded := hexValue(value[i+1])<<4 | hexValue(value[i+2])
		if isUnreservedByte(decoded) {
			builder.WriteByte(decoded)
		} else {
			builder.WriteByte('%')
			builder.WriteByte(upperHex(value[i+1]))
			builder.WriteByte(upperHex(value[i+2]))
		}
		i += 2
	}
	return builder.String()
}

func upperHex(c byte) byte {
	if c >= 'a' && c <= 'f' {
		return c - 'a' + 'A'
	}
	return c
}

// removeDotSegments implements RFC 3986 section 5.2.4.
func removeDotSegments(path string) string {
	if path == "" {
		return ""
	}
	input := path
	output := make([]string, 0, 8)
	for input != "" {
		switch {
		case strings.HasPrefix(input, "../"):
			input = input[3:]
		case strings.HasPrefix(input, "./"):
			input = input[2:]
		case strings.HasPrefix(input, "/./"):
			input = "/" + input[3:]
		case input == "/.":
			input = "/"
		case strings.HasPrefix(input, "/../"):
			input = "/" + input[4:]
			if len(output) > 0 {
				output = output[:len(output)-1]
			}
		case input == "/..":
			input = "/"
			if len(output) > 0 {
				output = output[:len(output)-1]
			}
		case input == "." || input == "..":
			input = ""
		default:
			next := input
			if strings.HasPrefix(next, "/") {
				next = next[1:]
			}
			index := strings.IndexByte(next, '/')
			if index < 0 {
				output = append(output, input)
				input = ""
			} else {
				output = append(output, input[:index+1])
				input = input[index+1:]
			}
		}
	}
	return strings.Join(output, "")
}

// canonicalString renders an absolute URI as its canonical identity string.
func (ref *uriReference) canonicalString(requireHierarchical, allowQuery bool) (string, error) {
	if !ref.hasScheme {
		return "", newError(ErrInvalidURI, "URI must be absolute")
	}
	if ref.hasFragment {
		return "", newError(ErrInvalidURI, "URI used as a resource identity must not contain a fragment")
	}
	if ref.hasQuery && !allowQuery {
		return "", newError(ErrInvalidURI, "URI must not contain a query string")
	}
	var builder strings.Builder
	builder.WriteString(strings.ToLower(ref.scheme))
	builder.WriteByte(':')
	if ref.hasAuthority {
		host := strings.ToLower(ref.host)
		port := ref.port
		switch strings.ToLower(ref.scheme) {
		case "http":
			if port == "80" {
				port = ""
			}
		case "https":
			if port == "443" {
				port = ""
			}
		}
		path := removeDotSegments(ref.path)
		if path == "" {
			path = "/"
		}
		if !strings.HasPrefix(path, "/") {
			return "", newError(ErrInvalidURI, "hierarchical URI path must begin with '/'")
		}
		if strings.Contains(path, "//") {
			return "", newError(ErrInvalidURI, "URI path contains repeated separators")
		}
		path = normalizePercent(path)
		builder.WriteString("//")
		builder.WriteString(host)
		if port != "" {
			builder.WriteByte(':')
			builder.WriteString(port)
		}
		builder.WriteString(path)
	} else {
		if requireHierarchical {
			return "", newError(ErrInvalidURI, "URI must be hierarchical with an authority")
		}
		path := ref.path
		if path == "" {
			return "", newError(ErrInvalidURI, "opaque URI must not have an empty path")
		}
		builder.WriteString(normalizePercent(path))
	}
	if ref.hasQuery {
		if strings.Contains(ref.query, "#") {
			return "", newError(ErrInvalidURI, "invalid query string")
		}
		builder.WriteByte('?')
		builder.WriteString(normalizePercent(ref.query))
	}
	return builder.String(), nil
}

// canonicalizeDocumentURI validates and canonicalizes an absolute hierarchical
// document URI: lowercase scheme/host, no default HTTP(S) port, no user-info,
// query or fragment, dot segments and repeated separators removed, percent
// escapes normalized.
func canonicalizeDocumentURI(raw string) (string, error) {
	ref, err := parseURIReference(raw)
	if err != nil {
		return "", err
	}
	if !ref.hasScheme || !ref.hasAuthority {
		return "", newError(ErrInvalidURI, "document URI must be absolute and hierarchical")
	}
	return ref.canonicalString(true, false)
}

// encodeSnapshotPath percent-encodes a raw relative snapshot path for use as a
// URI-reference path. Every byte outside the RFC 3986 pchar set is encoded.
func encodeSnapshotPath(snapshotPath string) (string, error) {
	if snapshotPath == "" {
		return "", newError(ErrSnapshotPath, "snapshot path must not be empty")
	}
	if strings.HasPrefix(snapshotPath, "/") {
		return "", newError(ErrSnapshotPath, "snapshot path must be relative")
	}
	if strings.ContainsRune(snapshotPath, '\\') {
		return "", newError(ErrSnapshotPath, "snapshot path must not contain backslashes")
	}
	for i := 0; i < len(snapshotPath); i++ {
		if snapshotPath[i] < 0x20 || snapshotPath[i] == 0x7f {
			return "", newError(ErrSnapshotPath, "snapshot path contains control characters")
		}
	}
	segments := strings.Split(snapshotPath, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return "", newError(ErrSnapshotPath, "snapshot path must be normalized: empty, '.' and '..' segments are rejected")
		}
	}
	var builder strings.Builder
	builder.Grow(len(snapshotPath))
	for i := 0; i < len(snapshotPath); i++ {
		c := snapshotPath[i]
		if isUnreservedByte(c) || isSubDelim(c) || c == ':' || c == '@' || c == '/' {
			builder.WriteByte(c)
			continue
		}
		builder.WriteByte('%')
		builder.WriteByte("0123456789ABCDEF"[c>>4])
		builder.WriteByte("0123456789ABCDEF"[c&0x0f])
	}
	return builder.String(), nil
}

// resolveReferenceURI resolves ref against an absolute hierarchical base using
// RFC 3986 section 5.2 (without query-specific policy).
func resolveReferenceURI(base *uriReference, ref *uriReference) (*uriReference, error) {
	if base == nil || !base.hasScheme || !base.hasAuthority {
		return nil, newError(ErrInvalidURI, "base URI must be absolute and hierarchical")
	}
	target := &uriReference{}
	if ref.hasScheme {
		target.hasScheme = true
		target.scheme = ref.scheme
		target.hasAuthority = ref.hasAuthority
		target.host, target.port = ref.host, ref.port
		target.path = ref.path
		target.hasQuery, target.query = ref.hasQuery, ref.query
	} else {
		target.hasScheme = true
		target.scheme = base.scheme
		if ref.hasAuthority {
			target.hasAuthority = true
			target.host, target.port = ref.host, ref.port
			target.path = ref.path
			target.hasQuery, target.query = ref.hasQuery, ref.query
		} else {
			target.hasAuthority = base.hasAuthority
			target.host, target.port = base.host, base.port
			if ref.path == "" {
				target.path = base.path
				if ref.hasQuery {
					target.hasQuery, target.query = true, ref.query
				} else {
					target.hasQuery, target.query = base.hasQuery, base.query
				}
			} else {
				if strings.HasPrefix(ref.path, "/") {
					target.path = ref.path
				} else {
					target.path = mergeReferencePaths(base, ref.path)
				}
				target.hasQuery, target.query = ref.hasQuery, ref.query
			}
		}
	}
	target.hasFragment, target.fragment = ref.hasFragment, ref.fragment
	return target, nil
}

func mergeReferencePaths(base *uriReference, refPath string) string {
	if base.hasAuthority && base.path == "" {
		return "/" + refPath
	}
	if index := strings.LastIndexByte(base.path, '/'); index >= 0 {
		return base.path[:index+1] + refPath
	}
	return refPath
}
