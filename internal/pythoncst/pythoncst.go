// Package pythoncst is the WP9 Python concrete-syntax-tree adapter.
//
// It isolates the gositter parser (ADR: gositter v0.2.0, Python grammar package
// only) from the Compiler Script ABI. It parses author Python source statically;
// it never executes, imports or evaluates Python. gositter node types are not
// exposed: callers receive the Compiler-owned projection below.
package pythoncst

import (
	"fmt"
	"strings"

	"github.com/dobbo-ca/gositter"
	"github.com/dobbo-ca/gositter/grammars/python"
)

const (
	// MaxSourceBytes bounds one Script file before it is handed to the parser.
	MaxSourceBytes = 1 << 20
	maxNodes       = 200000
	// maxStall bounds parser progress stalls on pathological input.
	maxStall = 1000
)

// Position is a 0-based source position; Column counts bytes.
type Position struct {
	Row    int
	Column int
}

// Kind is the Tree-sitter grammar node kind name.
type Kind = string

// Node is the Compiler-owned CST projection. It exposes only structure, text,
// positions and named field access.
type Node struct {
	Kind     Kind
	Start    int
	End      int
	Position Position
	Fields   map[string]*Node
	Children []*Node
}

// Text returns the node's source bytes as a string.
func (n *Node) Text(source []byte) string {
	if n == nil || n.Start < 0 || n.End > len(source) || n.Start > n.End {
		return ""
	}
	return string(source[n.Start:n.End])
}

// Field returns the named child, or nil.
func (n *Node) Field(name string) *Node {
	if n == nil {
		return nil
	}
	return n.Fields[name]
}

// Error is a deterministic parse failure with a source position.
type Error struct {
	Offset  int
	Row     int
	Column  int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s at %d:%d", e.Message, e.Row+1, e.Column+1)
}

// Parse parses Python source into the Compiler-owned CST. A syntax error or a
// missing-token tree is reported as *Error; resource bounds are enforced.
func Parse(source []byte) (root *Node, err error) {
	if len(source) == 0 {
		return nil, &Error{Message: "empty Python Script source"}
	}
	if len(source) > MaxSourceBytes {
		return nil, &Error{Message: fmt.Sprintf("Python Script source exceeds the %d byte limit", MaxSourceBytes)}
	}
	parser := gositter.NewParser()
	defer parser.Reset()
	if err := parser.SetLanguage(python.Language()); err != nil {
		return nil, &Error{Message: "Python grammar is unavailable in this build: " + err.Error()}
	}
	tree := parser.ParseBudget(source, maxStall)
	if tree == nil {
		return nil, &Error{Message: "Python parser resource budget exceeded"}
	}
	rootNode := tree.RootNode()
	if rootNode.IsNull() {
		return nil, &Error{Message: "Python parser returned no syntax tree"}
	}
	if rootNode.HasError() {
		if bad, ok := firstErrorNode(rootNode); ok {
			pos := bad.StartPosition()
			return nil, &Error{Offset: int(bad.StartByte()), Row: int(pos.Row), Column: int(pos.Column), Message: "Python syntax error or missing token near " + truncate(bad.Kind())}
		}
		return nil, &Error{Message: "Python syntax error"}
	}
	count := 0
	root, err = convert(rootNode, source, &count)
	if err != nil {
		return nil, err
	}
	return root, nil
}

func firstErrorNode(n gositter.Node) (gositter.Node, bool) {
	if n.IsError() || n.IsMissing() {
		return n, true
	}
	for i := uint32(0); i < n.NamedChildCount(); i++ {
		child := n.NamedChild(i)
		if child.IsError() || child.IsMissing() || child.HasError() {
			return firstErrorNode(child)
		}
	}
	return gositter.Node{}, false
}

func convert(n gositter.Node, source []byte, count *int) (*Node, error) {
	*count++
	if *count > maxNodes {
		return nil, &Error{Message: "Python syntax tree exceeds the node budget"}
	}
	startPos := n.StartPosition()
	out := &Node{
		Kind:     n.Kind(),
		Start:    int(n.StartByte()),
		End:      int(n.EndByte()),
		Position: Position{Row: int(startPos.Row), Column: int(startPos.Column)},
		Fields:   map[string]*Node{},
	}
	named := n.NamedChildCount()
	if named == 0 {
		return out, nil
	}
	out.Children = make([]*Node, 0, named)
	for i := uint32(0); i < named; i++ {
		child := n.NamedChild(i)
		converted, err := convert(child, source, count)
		if err != nil {
			return nil, err
		}
		out.Children = append(out.Children, converted)
		// Field names are looked up on the original node; one field name may
		// repeat (for example import names), in which case the first is kept.
		for j := uint32(0); j < n.ChildCount(); j++ {
			if !n.Child(j).Equals(child) {
				continue
			}
			if name := n.FieldNameForChild(j); name != "" {
				if _, exists := out.Fields[name]; !exists {
					out.Fields[name] = converted
				}
			}
			break
		}
	}
	return out, nil
}

func truncate(value string) string {
	if len(value) > 64 {
		return value[:64] + "..."
	}
	return strings.TrimSpace(value)
}
