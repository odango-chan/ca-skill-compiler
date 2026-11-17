package compiler

// WP9 NotRequired guard proof (CA-SL §32.1): an optional input key may only be
// read after proving presence (`"key" in input`) or through a presence-tolerant
// read (`input.get("key")`). Direct access to an optional key is rejected.
//
// This is a deliberately small static analysis over the CST; it never executes
// the function body and does not attempt full data-flow.

import (
	"fmt"

	"github.com/odango-chan/ca-skill-compiler/internal/pythoncst"
)

type optionalKeyAccess struct {
	optional map[string]bool
}

func validateScriptOptionalKeyAccess(source []byte, optional map[string]bool) error {
	if len(optional) == 0 {
		return nil
	}
	module, err := parseScriptModule(source)
	if err != nil {
		return err
	}
	body := module.Main.Field("body")
	if body == nil {
		return nil
	}
	guarded := map[string]bool{}
	safeGet := map[string]bool{}
	var scanGuards func(node *pythoncst.Node)
	scanGuards = func(node *pythoncst.Node) {
		if node == nil {
			return
		}
		if node.Kind == "comparison_operator" {
			text := node.Text(source)
			for key := range optional {
				if stringContains(text, "in input") && stringContains(text, fmt.Sprintf("%q", key)) {
					guarded[key] = true
				}
			}
		}
		if node.Kind == "call" {
			function := node.Field("function")
			if function != nil && function.Kind == "attribute" {
				object := function.Field("object")
				attribute := function.Field("attribute")
				if object != nil && object.Text(source) == "input" && attribute != nil && attribute.Text(source) == "get" {
					if arguments := node.Field("arguments"); arguments != nil {
						for _, argument := range arguments.Children {
							if key, ok := literalSubscriptKey(argument, source); ok {
								safeGet[key] = true
							}
						}
					}
				}
			}
		}
		for _, child := range node.Children {
			scanGuards(child)
		}
	}
	scanGuards(body)

	var walk func(node *pythoncst.Node) error
	walk = func(node *pythoncst.Node) error {
		if node == nil {
			return nil
		}
		switch node.Kind {
		case "subscript":
			value := node.Field("value")
			if value != nil && value.Kind == "identifier" && value.Text(source) == "input" {
				if key, ok := literalSubscriptKey(node.Field("subscript"), source); ok {
					if optional[key] && !guarded[key] {
						return fmt.Errorf("optional input key %q is read without proving presence", key)
					}
				}
			}
		case "attribute":
			object := node.Field("object")
			attribute := node.Field("attribute")
			if object != nil && object.Kind == "identifier" && object.Text(source) == "input" && attribute != nil {
				key := attribute.Text(source)
				if optional[key] && !guarded[key] {
					return fmt.Errorf("optional input key %q is read without proving presence", key)
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
	_ = safeGet
	return walk(body)
}

func stringContains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
