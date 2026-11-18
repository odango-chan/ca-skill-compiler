package compiler

// Contract limits conformance: §11.3/§11.6 maxActions.

import (
	"strings"
	"testing"
)

func TestContractLimitsPositive(t *testing.T) {
	contract := testContract + "limits:\n  maxActions: 3\n"
	request := testRequest(contract, testProcedure, testPredicates, testSchema)
	if hasCode(Check(request), "ACTION_LIMIT_INVALID") || hasCode(Check(request), "UNKNOWN_FIELD") {
		t.Fatalf("valid limits rejected: %+v", Check(request))
	}
}

func TestContractLimitsInvalid(t *testing.T) {
	cases := map[string]string{
		"zero":          "limits:\n  maxActions: 0\n",
		"negative":      "limits:\n  maxActions: -1\n",
		"string":        "limits:\n  maxActions: \"3\"\n",
		"null":          "limits:\n  maxActions: null\n",
		"unknown field": "limits:\n  maxActions: 3\n  other: 1\n",
	}
	for name, limits := range cases {
		contract := strings.Replace(testContract, "capabilities: [fivegc-session-read]\n", "capabilities: [fivegc-session-read]\n"+limits, 1)
		request := testRequest(contract, testProcedure, testPredicates, testSchema)
		diagnostics := Check(request)
		if !hasCode(diagnostics, "ACTION_LIMIT_INVALID") && !hasCode(diagnostics, "UNKNOWN_FIELD") {
			t.Fatalf("%s: invalid limits accepted: %+v", name, diagnostics)
		}
	}
}
