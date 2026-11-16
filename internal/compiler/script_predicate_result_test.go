package compiler

// WP9 Predicate Evaluation Result shape conformance (CA-SL §32.2).

import (
	"strings"
	"testing"
)

func TestPredicateResultShapeRejected(t *testing.T) {
	cases := map[string]struct{ old, new string }{
		"missing result":      {`    result: Literal["PASS", "FAIL", "UNKNOWN"]` + "\n", ""},
		"wrong result enum":   {`    result: Literal["PASS", "FAIL", "UNKNOWN"]`, `    result: Literal["PASS", "FAIL"]`},
		"reasonCode type":     {"    reasonCode: str", "    reasonCode: int"},
		"evidenceUsed type":   {"    evidenceUsed: list[str]", "    evidenceUsed: list[int]"},
		"evidenceUsed absent": {"    evidenceUsed: list[str]\n", ""},
	}
	for name, change := range cases {
		source := strings.Replace(predicateScriptABI, change.old, change.new, 1)
		if source == predicateScriptABI {
			t.Fatalf("%s: test fixture replacement did not apply", name)
		}
		if _, err := derivePredicateScriptABI([]byte(source)); err == nil {
			t.Fatalf("%s: nonconforming Predicate result accepted", name)
		}
	}
}

func TestPredicateResultShapeAccepted(t *testing.T) {
	if _, err := derivePredicateScriptABI([]byte(predicateScriptABI)); err != nil {
		t.Fatalf("canonical Predicate result rejected: %v", err)
	}
}
