package yamlast

import (
	"fmt"
	"strings"
	"testing"
)

func field(root *Node, name string) *Node {
	root = unalias(root)
	if root == nil || root.Kind != MappingNode {
		return nil
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := unalias(root.Content[i])
		if key != nil && key.Value == name {
			return root.Content[i+1]
		}
	}
	return nil
}

func TestParsePreservesYAML12ScalarKindsStylesAndPositions(t *testing.T) {
	root, err := Parse([]byte("name: demo\nplainBool: true\nupperBool: TRUE\nnullValue: null\nscientific: 1e2\nquotedBool: \"true\"\nsingleQuoted: 'x'\nnumber: 1.20\n"))
	if err != nil {
		t.Fatal(err)
	}
	if root.Kind != MappingNode || root.Tag != "!!map" {
		t.Fatalf("root = %#v", root)
	}
	name := field(root, "name")
	if name == nil || name.Tag != "!!str" || name.Value != "demo" || name.Style != PlainStyle {
		t.Fatalf("plain string not preserved: %#v", name)
	}
	plainBool := field(root, "plainBool")
	if plainBool == nil || plainBool.Tag != "!!bool" || plainBool.Value != "true" || plainBool.Style != PlainStyle {
		t.Fatalf("plain Boolean not preserved: %#v", plainBool)
	}
	upperBool := field(root, "upperBool")
	if upperBool == nil || upperBool.Tag != "!!bool" || upperBool.Value != "TRUE" {
		t.Fatalf("YAML core token was not preserved for later CA-SL rejection: %#v", upperBool)
	}
	nullValue := field(root, "nullValue")
	if nullValue == nil || nullValue.Tag != "!!null" || nullValue.Value != "null" {
		t.Fatalf("null token not preserved: %#v", nullValue)
	}
	scientific := field(root, "scientific")
	if scientific == nil || scientific.Tag != "!!float" || scientific.Value != "1e2" {
		t.Fatalf("exponent-only decimal must be classified as float: %#v", scientific)
	}
	quoted := field(root, "quotedBool")
	if quoted == nil || quoted.Tag != "!!str" || quoted.Value != "true" || quoted.Style != DoubleQuotedStyle || !strings.Contains(quoted.Raw, "\"") {
		t.Fatalf("double-quoted scalar not preserved: %#v", quoted)
	}
	single := field(root, "singleQuoted")
	if single == nil || single.Value != "x" || single.Style != SingleQuotedStyle {
		t.Fatalf("single-quoted scalar not preserved: %#v", single)
	}
	number := field(root, "number")
	if number == nil || number.Tag != "!!float" || number.Value != "1.20" {
		t.Fatalf("numeric lexical form not preserved: %#v", number)
	}
	if name.Line != 1 || name.Column == 0 || quoted.Line != 6 {
		t.Fatalf("source positions not retained: name=%+v quoted=%+v", name, quoted)
	}
}

func TestParseResolvesAcyclicAliasesAndRejectsCycles(t *testing.T) {
	root, err := Parse([]byte("base: &base\n  name: demo\ncopy: *base\n"))
	if err != nil {
		t.Fatal(err)
	}
	copyNode := field(root, "copy")
	if copyNode == nil || copyNode.Kind != AliasNode || unalias(copyNode) == nil {
		t.Fatalf("alias was not retained and resolved: %#v", copyNode)
	}
	if scalar := field(unalias(copyNode), "name"); scalar == nil || scalar.Value != "demo" {
		t.Fatalf("alias target contents missing: %#v", scalar)
	}

	_, err = Parse([]byte("self: &self { next: *self }\n"))
	if err == nil || !strings.Contains(err.Error(), "cyclic YAML alias") {
		t.Fatalf("cyclic alias was not rejected: %v", err)
	}
}

func TestParseRejectsDuplicateUndefinedAndDuplicateAnchorNames(t *testing.T) {
	for name, source := range map[string]string{
		"duplicate-key": "value: 1\nvalue: 2\n",
		"undefined-alias": "value: *missing\n",
		"duplicate-anchor": "one: &same value\ntwo: &same other\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(source)); err == nil {
				t.Fatalf("invalid YAML accepted: %s", source)
			}
		})
	}
}

func TestParseRejectsUnsupportedYAMLForms(t *testing.T) {
	cases := map[string]string{
		"multiple-documents": "---\na: 1\n---\nb: 2\n",
		"custom-tag":         "value: !custom hello\n",
		"merge-key":          "base: &base {name: demo}\nvalue: {<<: *base}\n",
		"complex-key":        "? [one, two]\n: value\n",
		"non-string-key":     "1: value\n",
		"directive":          "%YAML 1.2\n---\na: 1\n",
		"non-mapping-root":   "- one\n- two\n",
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(source)); err == nil {
				t.Fatalf("unsupported YAML form accepted: %s", source)
			}
		})
	}
}

func TestParseEnforcesDepthAliasAndExpandedNodeBounds(t *testing.T) {
	flow := strings.Repeat("[", maxDepth+1) + "0" + strings.Repeat("]", maxDepth+1)
	if _, err := Parse([]byte(flow)); err == nil || !IsLimitError(err) {
		t.Fatalf("flow depth limit not enforced: %v", err)
	}

	var aliases strings.Builder
	aliases.WriteString("base: &base {name: demo}\nvalues:\n")
	for i := 0; i <= maxAliasCount; i++ {
		aliases.WriteString("  - *base\n")
	}
	if _, err := Parse([]byte(aliases.String())); err == nil || !IsLimitError(err) {
		t.Fatalf("alias count limit not enforced: %v", err)
	}

	var expanded strings.Builder
	expanded.WriteString("base: &base {")
	for i := 0; i < 1100; i++ {
		if i > 0 {
			expanded.WriteString(", ")
		}
		expanded.WriteString(fmt.Sprintf("k%d: v", i))
	}
	expanded.WriteString("}\nvalues:\n")
	for i := 0; i < 20; i++ {
		expanded.WriteString("  - *base\n")
	}
	if _, err := Parse([]byte(expanded.String())); err == nil || !IsLimitError(err) {
		t.Fatalf("alias expansion node limit not enforced: %v", err)
	}
}

func TestValidateScalarLiteralUsesFrozenCAStyleSubset(t *testing.T) {
	accepted := []string{
		"value: true\n",
		"value: false\n",
		"value: null\n",
		"value: 0\n",
		"value: -0\n",
		"value: -9223372036854775808\n",
		"value: 1.25\n",
		"value: 1e2\n",
		"value: -1.25E-2\n",
		"value: \"true\"\n",
		"value: 'null'\n",
	}
	for _, source := range accepted {
		t.Run(source, func(t *testing.T) {
			root, err := Parse([]byte(source))
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateScalarLiteral(field(root, "value")); err != nil {
				t.Fatalf("valid ScalarLiteral rejected: %v", err)
			}
		})
	}

	rejected := []string{
		"value: TRUE\n",
		"value: NULL\n",
		"value: ~\n",
		"value: +1\n",
		"value: 01\n",
		"value: 0x10\n",
		"value: 1.\n",
		"value: .5\n",
		"value: 1e9999\n",
		"value: 1e-9999\n",
		"value: .nan\n",
		"value: .inf\n",
		"value: |\n  text\n",
	}
	for _, source := range rejected {
		t.Run(source, func(t *testing.T) {
			root, err := Parse([]byte(source))
			if err != nil {
				// A YAML syntax rejection is also an acceptable fail-closed result.
				return
			}
			if err := ValidateScalarLiteral(field(root, "value")); err == nil {
				t.Fatalf("invalid ScalarLiteral accepted: %s", source)
			}
		})
	}
}

func TestValidateScalarLiteralRejectsAliasesAndMultilineQuotes(t *testing.T) {
	root, err := Parse([]byte("base: &base \"value\"\ncopy: *base\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateScalarLiteral(field(root, "copy")); err == nil {
		t.Fatal("alias accepted as ScalarLiteral")
	}

	root, err = Parse([]byte("value: \"first\n  second\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateScalarLiteral(field(root, "value")); err == nil {
		t.Fatal("multiline quoted scalar accepted")
	}
}
