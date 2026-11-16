package compiler

// WP9 restricted Python key-access analysis for the Script input mapping.
//
// This is a deliberately small static subset: it inspects the parsed CST only;
// it never executes the function body. Literal subscript/attribute reads must
// name a declared TypedDict key; dynamic or computed keys are rejected, as
// CA-SL §32.1 requires. NotRequired guard proofs and full data-flow analysis are
// later WP9 work.

import (
	"fmt"

	"github.com/odango-chan/ca-skill-compiler/internal/pythoncst"
)

// validateScriptInputKeyAccess checks literal reads of the single `input`
// mapping in a Script body against the declared input keys.
func validateScriptInputKeyAccess(source []byte, allowed map[string]bool, allowEvidence bool) error {
	module, err := parseScriptModule(source)
	if err != nil {
		return err
	}
	body := module.Main.Field("body")
	if body == nil {
		return nil
	}
	var walk func(node *pythoncst.Node) error
	walk = func(node *pythoncst.Node) error {
		if node == nil {
			return nil
		}
		switch node.Kind {
		case "subscript":
			value := node.Field("value")
			if value != nil && value.Kind == "identifier" && value.Text(source) == "input" {
				keyNode := node.Field("subscript")
				key, literal := literalSubscriptKey(keyNode, source)
				if !literal {
					return fmt.Errorf("dynamic or non-literal Script input key access is not statically analyzable")
				}
				if key == "evidence" && !allowEvidence {
					return fmt.Errorf("Script input `evidence` projection is unavailable until a frozen Evidence Descriptor publication exists")
				}
				if !allowed[key] {
					return fmt.Errorf("Script reads an undeclared input key %q", key)
				}
			}
		case "attribute":
			object := node.Field("object")
			attribute := node.Field("attribute")
			if object != nil && object.Kind == "identifier" && object.Text(source) == "input" && attribute != nil {
				key := attribute.Text(source)
				if !allowed[key] {
					return fmt.Errorf("Script reads an undeclared input key %q", key)
				}
			}
		}
		for _, child := range node.Children {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(body)
}

func literalSubscriptKey(node *pythoncst.Node, source []byte) (string, bool) {
	if node == nil || node.Kind != "string" {
		return "", false
	}
	text := node.Text(source)
	if len(text) < 2 {
		return "", false
	}
	quote := text[0]
	if quote != '\'' && quote != '"' {
		return "", false
	}
	if text[len(text)-1] != quote {
		return "", false
	}
	inner := text[1 : len(text)-1]
	if inner == "" {
		return "", false
	}
	for i := 0; i < len(inner); i++ {
		if inner[i] == '\\' || inner[i] == quote {
			return "", false // escape sequences are not statically attributed here
		}
	}
	return inner, true
}
