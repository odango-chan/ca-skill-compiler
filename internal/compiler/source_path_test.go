package compiler

// WP6 slice 1 conformance: SourcePath grammar and Predicate arity.

import (
	"strings"
	"testing"
)

func TestWP6SourcePathGrammarPositive(t *testing.T) {
	request := testRequest(testContract, testProcedure, testPredicates, testSchema)
	diagnostics := Check(request)
	if hasCode(diagnostics, "SOURCE_PATH_INVALID") || hasCode(diagnostics, "PREDICATE_ARITY_MISMATCH") {
		t.Fatalf("valid SourcePaths rejected: %+v", diagnostics)
	}
}

func TestWP6InvalidSourcePaths(t *testing.T) {
	cases := map[string]string{
		"bare string":       strings.Replace(testProcedure, "ueRef: context.ueRef", "ueRef: ue-1", 1),
		"unknown node root": strings.Replace(testProcedure, "arguments: [read.observation.output]", "arguments: [ghost.observation.output]", 1),
		"stage mismatch":    strings.Replace(testProcedure, "arguments: [read.observation.output]", "arguments: [read.operation.output]", 1),
		"invalid segment":   strings.Replace(testProcedure, "arguments: [read.observation.output]", "arguments: [read.observation.output.0bad]", 1),
		"bare context":      strings.Replace(testProcedure, "arguments: [read.observation.output]", "arguments: [context]", 1),
	}
	for name, procedure := range cases {
		request := testRequest(testContract, procedure, testPredicates, testSchema)
		diagnostics := Check(request)
		found := false
		for _, diagnostic := range diagnostics {
			if diagnostic.Code == "SOURCE_PATH_INVALID" && diagnostic.Path != "" && diagnostic.Line > 0 && diagnostic.Column > 0 {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s: expected SOURCE_PATH_INVALID with source location: %+v", name, diagnostics)
		}
	}
}

func TestWP6QuotedStringLiteralIsNotSourcePath(t *testing.T) {
	procedure := strings.Replace(testProcedure, "ueRef: context.ueRef", "ueRef: 'ue-1'", 1)
	request := testRequest(testContract, procedure, testPredicates, testSchema)
	if hasCode(Check(request), "SOURCE_PATH_INVALID") {
		t.Fatalf("quoted string literal was parsed as a SourcePath: %+v", Check(request))
	}
}

func TestWP6PredicateArityMismatch(t *testing.T) {
	procedure := strings.Replace(testProcedure, "arguments: [context.ueRef]", "arguments: [context.ueRef, context.ueRef]", 1)
	request := testRequest(testContract, procedure, testPredicates, testSchema)
	if !hasCode(Check(request), "PREDICATE_ARITY_MISMATCH") {
		t.Fatalf("Predicate arity mismatch not detected: %+v", Check(request))
	}
}
