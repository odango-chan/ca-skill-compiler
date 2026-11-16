package yamlast

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	goccyast "github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/lexer"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/token"
)

type Kind uint8

const (
	ScalarNode Kind = iota + 1
	SequenceNode
	MappingNode
	AliasNode
	UnsupportedNode
)

type ScalarStyle uint8

const (
	PlainStyle ScalarStyle = iota
	SingleQuotedStyle
	DoubleQuotedStyle
	LiteralStyle
	FoldedStyle
	TaggedStyle
	AliasStyle
)

const (
	maxDocumentBytes = 2 << 20
	maxTokenCount    = 80000
	maxNodeCount     = 20000
	maxDepth         = 64
	maxAliasCount    = 4096
)

type Node struct {
	Kind    Kind
	Tag     string
	Value   string
	Raw     string
	Style   ScalarStyle
	Line    int
	Column  int
	Content []*Node
	Alias   *Node
}

type ParseErrorKind string

const (
	SyntaxError ParseErrorKind = "syntax"
	RootError   ParseErrorKind = "root"
	LimitError  ParseErrorKind = "limit"
)

type ParseError struct {
	Kind    ParseErrorKind
	Message string
	Line    int
	Column  int
}

func (e *ParseError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

var sourcePositionPattern = regexp.MustCompile("\\[(\\d+):(\\d+)\\]")
var finiteDecimalPattern = regexp.MustCompile("^-?(?:0|[1-9][0-9]*)(?:\\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$")
var integerPattern = regexp.MustCompile("^-?(?:0|[1-9][0-9]*)$")

func Parse(data []byte) (*Node, error) {
	return parse(data, true)
}

// ParseValue parses one YAML value of any root kind. It is used only for
// already-serialized JSON values that must enter the same normalized AST.
func ParseValue(data []byte) (*Node, error) {
	return parse(data, false)
}

func parse(data []byte, requireMappingRoot bool) (*Node, error) {
	if len(data) == 0 {
		return nil, &ParseError{Kind: RootError, Message: "YAML document is empty"}
	}
	if len(data) > maxDocumentBytes {
		return nil, &ParseError{Kind: LimitError, Message: "YAML document exceeds the 2 MiB byte limit"}
	}
	if !utf8.Valid(data) {
		return nil, &ParseError{Kind: SyntaxError, Message: "YAML document is not valid UTF-8"}
	}

	tokens := lexer.Tokenize(string(data))
	if err := preflight(tokens); err != nil {
		return nil, err
	}
	file, err := parser.Parse(tokens, 0)
	if err != nil {
		return nil, parseLibraryError(err)
	}
	if file == nil || len(file.Docs) != 1 || file.Docs[0] == nil {
		return nil, &ParseError{Kind: SyntaxError, Message: "exactly one YAML document is required"}
	}
	if file.Docs[0].Body == nil {
		return nil, &ParseError{Kind: RootError, Message: "YAML document must not be empty"}
	}

	converter := converter{anchors: make(map[string]*Node)}
	root, err := converter.convert(file.Docs[0].Body, 0)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return nil, &ParseError{Kind: RootError, Message: "YAML document must not be empty"}
	}
	if requireMappingRoot && root.Kind != MappingNode {
		line, column := rootPosition(file.Docs[0].Body)
		return nil, &ParseError{Kind: RootError, Message: "YAML document must have a mapping root", Line: line, Column: column}
	}
	if err := validateBounds(root); err != nil {
		return nil, err
	}
	return root, nil
}

func preflight(tokens token.Tokens) error {
	if len(tokens) > maxTokenCount {
		return &ParseError{Kind: LimitError, Message: "YAML token count exceeds the 80000-token limit"}
	}
	flowDepth := 0
	aliases := 0
	for _, item := range tokens {
		if item == nil {
			continue
		}
		line, column := tokenPosition(item)
		if item.Position != nil && item.Position.IndentLevel > maxDepth {
			return &ParseError{Kind: LimitError, Message: "YAML indentation depth exceeds 64", Line: line, Column: column}
		}
		switch item.Type {
		case token.SequenceStartType, token.MappingStartType:
			flowDepth++
			if flowDepth > maxDepth {
				return &ParseError{Kind: LimitError, Message: "YAML flow nesting exceeds 64", Line: line, Column: column}
			}
		case token.SequenceEndType, token.MappingEndType:
			flowDepth--
		case token.AliasType:
			aliases++
			if aliases > maxAliasCount {
				return &ParseError{Kind: LimitError, Message: "YAML alias count exceeds 4096", Line: line, Column: column}
			}
		}
	}
	return nil
}

func parseLibraryError(err error) error {
	message := err.Error()
	result := &ParseError{Kind: SyntaxError, Message: message}
	match := sourcePositionPattern.FindStringSubmatch(message)
	if len(match) == 3 {
		result.Line, _ = strconv.Atoi(match[1])
		result.Column, _ = strconv.Atoi(match[2])
	}
	return result
}

func tokenPosition(tk *token.Token) (int, int) {
	if tk == nil || tk.Position == nil {
		return 0, 0
	}
	return tk.Position.Line, tk.Position.Column
}

func rootPosition(node goccyast.Node) (int, int) {
	if node == nil {
		return 0, 0
	}
	return tokenPosition(node.GetToken())
}

type converter struct {
	anchors map[string]*Node
}

func (c *converter) convert(source goccyast.Node, depth int) (*Node, error) {
	if source == nil {
		return nil, &ParseError{Kind: SyntaxError, Message: "YAML contains an empty node"}
	}
	if depth > maxDepth {
		line, column := rootPosition(source)
		return nil, &ParseError{Kind: LimitError, Message: "YAML AST nesting exceeds 64", Line: line, Column: column}
	}

	switch node := source.(type) {
	case *goccyast.MappingNode:
		line, column := rootPosition(node)
		result := &Node{Kind: MappingNode, Tag: "!!map", Line: line, Column: column, Content: make([]*Node, 0, len(node.Values)*2)}
		seenKeys := make(map[string]bool, len(node.Values))
		for _, entry := range node.Values {
			if entry == nil || entry.Key == nil || entry.Value == nil {
				return nil, &ParseError{Kind: SyntaxError, Message: "YAML mapping contains an incomplete entry", Line: line, Column: column}
			}
			key, err := c.convert(entry.Key, depth+1)
			if err != nil {
				return nil, err
			}
			resolvedKey := unalias(key)
			if resolvedKey == nil || resolvedKey.Kind != ScalarNode || resolvedKey.Tag != "!!str" {
				return nil, &ParseError{Kind: SyntaxError, Message: "YAML mapping keys must be strings", Line: key.Line, Column: key.Column}
			}
			if seenKeys[resolvedKey.Value] {
				return nil, &ParseError{Kind: SyntaxError, Message: "duplicate YAML mapping key: " + resolvedKey.Value, Line: key.Line, Column: key.Column}
			}
			seenKeys[resolvedKey.Value] = true
			value, err := c.convert(entry.Value, depth+1)
			if err != nil {
				return nil, err
			}
			result.Content = append(result.Content, key, value)
		}
		return result, nil

	case *goccyast.SequenceNode:
		line, column := rootPosition(node)
		result := &Node{Kind: SequenceNode, Tag: "!!seq", Line: line, Column: column, Content: make([]*Node, 0, len(node.Values))}
		for _, item := range node.Values {
			if entry, ok := item.(*goccyast.SequenceEntryNode); ok {
				item = entry.Value
			}
			value, err := c.convert(item, depth+1)
			if err != nil {
				return nil, err
			}
			result.Content = append(result.Content, value)
		}
		return result, nil

	case *goccyast.AnchorNode:
		name := scalarText(node.Name)
		if name == "" {
			line, column := rootPosition(node)
			return nil, &ParseError{Kind: SyntaxError, Message: "YAML anchor name must be a nonempty scalar", Line: line, Column: column}
		}
		// YAML aliases bind to the most recent preceding anchor with this name.
		placeholder := &Node{Kind: UnsupportedNode, Value: name, Style: AliasStyle}
		c.anchors[name] = placeholder
		value, err := c.convert(node.Value, depth+1)
		if err != nil {
			return nil, err
		}
		if value == nil {
			line, column := rootPosition(node)
			return nil, &ParseError{Kind: SyntaxError, Message: "YAML anchor has no value", Line: line, Column: column}
		}
		*placeholder = *value
		return placeholder, nil

	case *goccyast.AliasNode:
		name := scalarText(node.Value)
		line, column := rootPosition(node)
		target := c.anchors[name]
		if name == "" || target == nil {
			return nil, &ParseError{Kind: SyntaxError, Message: "YAML alias references an undefined anchor: " + name, Line: line, Column: column}
		}
		return &Node{Kind: AliasNode, Value: target.Value, Raw: node.GetToken().Origin, Style: AliasStyle, Line: line, Column: column, Alias: target}, nil

	case *goccyast.TagNode:
		line, column := rootPosition(node)
		tag, ok := normalizeCoreTag(node.Start.Value)
		if !ok {
			return nil, &ParseError{Kind: SyntaxError, Message: "custom or unsupported YAML tag is not allowed: " + node.Start.Value, Line: line, Column: column}
		}
		value, err := c.convert(node.Value, depth+1)
		if err != nil {
			return nil, err
		}
		if !tagMatchesNode(tag, value) {
			return nil, &ParseError{Kind: SyntaxError, Message: "explicit YAML tag does not match its value kind: " + node.Start.Value, Line: line, Column: column}
		}
		value.Tag = tag
		value.Style = TaggedStyle
		value.Raw = node.Start.Origin + value.Raw
		value.Line, value.Column = line, column
		return value, nil

	case *goccyast.DirectiveNode:
		line, column := rootPosition(node)
		return nil, &ParseError{Kind: SyntaxError, Message: "YAML directives are not supported in CA-SL source", Line: line, Column: column}

	case *goccyast.MergeKeyNode:
		line, column := rootPosition(node)
		return nil, &ParseError{Kind: SyntaxError, Message: "YAML merge keys are not supported in CA-SL source", Line: line, Column: column}

	case *goccyast.MappingKeyNode:
		line, column := rootPosition(node)
		return nil, &ParseError{Kind: SyntaxError, Message: "complex YAML mapping keys are not supported", Line: line, Column: column}

	case *goccyast.LiteralNode:
		line, column := rootPosition(node)
		raw := ""
		if node.Start != nil {
			raw += node.Start.Origin
		}
		if node.Value != nil && node.Value.GetToken() != nil {
			raw += node.Value.GetToken().Origin
		}
		style := LiteralStyle
		if node.Start != nil && node.Start.Type == token.FoldedType {
			style = FoldedStyle
		}
		value := ""
		if node.Value != nil {
			value = node.Value.Value
		}
		return &Node{Kind: ScalarNode, Tag: "!!str", Value: value, Raw: raw, Style: style, Line: line, Column: column}, nil

	case *goccyast.StringNode:
		line, column := rootPosition(node)
		style := PlainStyle
		tag := "!!str"
		switch node.GetToken().Type {
		case token.SingleQuoteType:
			style = SingleQuotedStyle
		case token.DoubleQuoteType:
			style = DoubleQuotedStyle
		default:
			// YAML 1.2 scientific notation is a float even when the upstream
			// token classifier leaves exponent-only forms as StringType.
			if finiteDecimalPattern.MatchString(node.Value) && strings.ContainsAny(node.Value, "eE") {
				tag = "!!float"
			}
		}
		return &Node{Kind: ScalarNode, Tag: tag, Value: node.Value, Raw: node.GetToken().Origin, Style: style, Line: line, Column: column}, nil

	case *goccyast.IntegerNode:
		return scalarNode(node, "!!int")
	case *goccyast.FloatNode, *goccyast.InfinityNode, *goccyast.NanNode:
		return scalarNode(node, "!!float")
	case *goccyast.BoolNode:
		return scalarNode(node, "!!bool")
	case *goccyast.NullNode:
		return scalarNode(node, "!!null")
	default:
		line, column := rootPosition(source)
		return nil, &ParseError{Kind: SyntaxError, Message: fmt.Sprintf("unsupported YAML AST node type %T", source), Line: line, Column: column}
	}
}

func scalarNode(source goccyast.Node, tag string) (*Node, error) {
	line, column := rootPosition(source)
	tk := source.GetToken()
	value := ""
	raw := ""
	if tk != nil {
		value = tk.Value
		raw = tk.Origin
	}
	return &Node{Kind: ScalarNode, Tag: tag, Value: value, Raw: raw, Style: PlainStyle, Line: line, Column: column}, nil
}

func scalarText(source goccyast.Node) string {
	if source == nil {
		return ""
	}
	switch node := source.(type) {
	case *goccyast.StringNode:
		return node.Value
	case *goccyast.IntegerNode, *goccyast.FloatNode, *goccyast.BoolNode, *goccyast.NullNode:
		if node.GetToken() != nil {
			return node.GetToken().Value
		}
	}
	if scalar, ok := source.(goccyast.ScalarNode); ok {
		if value, ok := scalar.GetValue().(string); ok {
			return value
		}
	}
	return ""
}

func normalizeCoreTag(tag string) (string, bool) {
	switch tag {
	case "!!str", "tag:yaml.org,2002:str":
		return "!!str", true
	case "!!int", "tag:yaml.org,2002:int":
		return "!!int", true
	case "!!float", "tag:yaml.org,2002:float":
		return "!!float", true
	case "!!bool", "tag:yaml.org,2002:bool":
		return "!!bool", true
	case "!!null", "tag:yaml.org,2002:null":
		return "!!null", true
	case "!!map", "tag:yaml.org,2002:map":
		return "!!map", true
	case "!!seq", "tag:yaml.org,2002:seq":
		return "!!seq", true
	default:
		return "", false
	}
}

func tagMatchesNode(tag string, node *Node) bool {
	if node == nil {
		return false
	}
	switch tag {
	case "!!map":
		return node.Kind == MappingNode
	case "!!seq":
		return node.Kind == SequenceNode
	case "!!str", "!!int", "!!float", "!!bool", "!!null":
		return node.Kind == ScalarNode
	default:
		return false
	}
}

func unalias(node *Node) *Node {
	for hops := 0; node != nil && node.Kind == AliasNode && hops < maxDepth; hops++ {
		node = node.Alias
	}
	return node
}

func validateBounds(root *Node) error {
	active := make(map[*Node]bool)
	count := 0
	var walk func(*Node, int) error
	walk = func(node *Node, depth int) error {
		if node == nil {
			return nil
		}
		if depth > maxDepth {
			return &ParseError{Kind: LimitError, Message: "YAML alias expansion or nesting exceeds 64", Line: node.Line, Column: node.Column}
		}
		if active[node] {
			return &ParseError{Kind: LimitError, Message: "cyclic YAML alias is not allowed", Line: node.Line, Column: node.Column}
		}
		count++
		if count > maxNodeCount {
			return &ParseError{Kind: LimitError, Message: "YAML expanded AST exceeds 20000 nodes", Line: node.Line, Column: node.Column}
		}
		active[node] = true
		defer delete(active, node)
		if node.Kind == AliasNode {
			if node.Alias == nil {
				return &ParseError{Kind: SyntaxError, Message: "YAML alias has no resolved anchor", Line: node.Line, Column: node.Column}
			}
			if active[node.Alias] {
				return &ParseError{Kind: LimitError, Message: "cyclic YAML alias is not allowed", Line: node.Line, Column: node.Column}
			}
			if err := walk(node.Alias, depth+1); err != nil {
				return err
			}
		}
		for _, child := range node.Content {
			if err := walk(child, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(root, 0)
}

func ParseErrorLocation(err error) (int, int) {
	if parsed, ok := err.(*ParseError); ok {
		return parsed.Line, parsed.Column
	}
	return 0, 0
}

func IsRootError(err error) bool {
	parsed, ok := err.(*ParseError)
	return ok && parsed.Kind == RootError
}

func IsLimitError(err error) bool {
	parsed, ok := err.(*ParseError)
	return ok && parsed.Kind == LimitError
}

func IsSyntaxError(err error) bool {
	parsed, ok := err.(*ParseError)
	return ok && parsed.Kind == SyntaxError
}

func StringValue(node *Node) string {
	node = unalias(node)
	if node == nil || node.Kind != ScalarNode {
		return ""
	}
	return node.Value
}

func IsQuoted(node *Node) bool {
	node = unalias(node)
	return node != nil && (node.Style == SingleQuotedStyle || node.Style == DoubleQuotedStyle)
}

func IsPlain(node *Node) bool {
	node = unalias(node)
	return node != nil && node.Style == PlainStyle
}

func IsScalarLiteralStyle(node *Node) bool {
	node = unalias(node)
	return node != nil && (node.Style == SingleQuotedStyle || node.Style == DoubleQuotedStyle)
}

func ParseIntegerLexeme(value string) (int64, error) {
	if !integerPattern.MatchString(value) {
		return 0, fmt.Errorf("unsupported integer lexeme")
	}
	return strconv.ParseInt(value, 10, 64)
}

func ParseFiniteFloatLexeme(value string) (float64, error) {
	if !finiteDecimalPattern.MatchString(value) || !strings.ContainsAny(value, ".eE") {
		return 0, fmt.Errorf("unsupported floating-point lexeme")
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	mantissa := strings.SplitN(strings.ReplaceAll(value, "E", "e"), "e", 2)[0]
	mantissa = strings.NewReplacer("-", "", ".", "").Replace(mantissa)
	nonZero := strings.TrimLeft(mantissa, "0") != ""
	if parsed == 0 && nonZero {
		return 0, fmt.Errorf("nonzero float underflows to zero")
	}
	if parsed != parsed || parsed > 1.7976931348623157e308 || parsed < -1.7976931348623157e308 {
		return 0, fmt.Errorf("float must be finite")
	}
	return parsed, nil
}

func ValidateScalarLiteral(node *Node) error {
	if node != nil && node.Kind == AliasNode {
		return fmt.Errorf("aliases are not ScalarLiteral values")
	}
	node = unalias(node)
	if node == nil || node.Kind != ScalarNode {
		return fmt.Errorf("ScalarLiteral must be a YAML scalar")
	}
	switch node.Style {
	case SingleQuotedStyle, DoubleQuotedStyle:
		if strings.ContainsAny(node.Raw, "\r\n") {
			return fmt.Errorf("quoted ScalarLiteral must be one line")
		}
		return nil
	case PlainStyle:
		switch node.Tag {
		case "!!null":
			if node.Value != "null" {
				return fmt.Errorf("bare null must be lowercase null")
			}
		case "!!bool":
			if node.Value != "true" && node.Value != "false" {
				return fmt.Errorf("bare Boolean must be lowercase true or false")
			}
		case "!!int":
			_, err := ParseIntegerLexeme(node.Value)
			return err
		case "!!float":
			_, err := ParseFiniteFloatLexeme(node.Value)
			return err
		default:
			return fmt.Errorf("bare strings are valid only as SourcePath expressions")
		}
		return nil
	default:
		return fmt.Errorf("ScalarLiteral does not allow block scalars, aliases or explicit tags")
	}
}

func ToValue(node *Node) (any, error) {
	if node == nil {
		return nil, nil
	}
	if node.Kind == AliasNode {
		if node.Alias == nil {
			return nil, fmt.Errorf("unresolved YAML alias")
		}
		return ToValue(node.Alias)
	}
	switch node.Kind {
	case ScalarNode:
		switch node.Tag {
		case "!!str":
			return node.Value, nil
		case "!!bool":
			return strconv.ParseBool(strings.ToLower(node.Value))
		case "!!int":
			if value, err := strconv.ParseInt(node.Value, 10, 64); err == nil {
				return value, nil
			}
			return strconv.ParseUint(node.Value, 10, 64)
		case "!!float":
			return strconv.ParseFloat(node.Value, 64)
		case "!!null":
			return nil, nil
		default:
			return nil, fmt.Errorf("unsupported YAML scalar tag %q", node.Tag)
		}
	case SequenceNode:
		values := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			value, err := ToValue(child)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, nil
	case MappingNode:
		values := make(map[string]any, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := unalias(node.Content[i])
			if key == nil || key.Kind != ScalarNode || key.Tag != "!!str" {
				return nil, fmt.Errorf("YAML mapping key must be a string")
			}
			value, err := ToValue(node.Content[i+1])
			if err != nil {
				return nil, err
			}
			values[key.Value] = value
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported YAML AST node kind %d", node.Kind)
	}
}
