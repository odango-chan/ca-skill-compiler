package compiler

// Determinism and diagnostic-contract gates: identical semantic output under input
// order permutations, and the frozen diagnostic ordering/phase shape.

import (
	"reflect"
	"sort"
	"testing"
)

func permuteRequest(request BuildRequest) BuildRequest {
	permuted := request
	permuted.Files = reverseFiles(request.Files)
	permuted.ResolvedDependencies = make([]ResolvedDependency, len(request.ResolvedDependencies))
	for index, dependency := range request.ResolvedDependencies {
		permuted.ResolvedDependencies[len(request.ResolvedDependencies)-1-index] = ResolvedDependency{
			DependencyID: dependency.DependencyID,
			Digest:       dependency.Digest,
			Files:        reverseFiles(dependency.Files),
		}
	}
	if request.TrustedBuildContext != nil {
		bindings := append([]TrustedDependencyBinding(nil), request.TrustedBuildContext.DependencyBindings...)
		sort.Slice(bindings, func(i, j int) bool { return bindings[i].DependencyID > bindings[j].DependencyID })
		context := *request.TrustedBuildContext
		context.DependencyBindings = bindings
		permuted.TrustedBuildContext = &context
	}
	return permuted
}

func TestDeterminismCheckBuildUnderPermutation(t *testing.T) {
	requests := []BuildRequest{
		wp3aBaseFixture(t).request(t),
		testRequest(testContract, testProcedure, testPredicates, testSchema),
	}
	for index, request := range requests {
		permuted := permuteRequest(request)
		if !reflect.DeepEqual(Check(request), Check(permuted)) {
			t.Fatalf("Check diagnostics depend on input order (case %d)", index)
		}
		if !reflect.DeepEqual(Build(request), Build(permuted)) {
			t.Fatalf("Build result depends on input order (case %d)", index)
		}
		if !reflect.DeepEqual(InspectDependencies(request), InspectDependencies(permuted)) {
			t.Fatalf("DependencyInspection depends on input order (case %d)", index)
		}
	}
}

func TestDiagnosticContractShapeAndOrder(t *testing.T) {
	requests := []BuildRequest{
		wp3aBaseFixture(t).request(t),
		testRequest(testContract, testProcedure, testPredicates, testSchema),
	}
	for _, request := range requests {
		diagnostics := Check(request)
		for _, diagnostic := range diagnostics {
			if diagnostic.Phase != "C_STATIC" && diagnostic.Phase != "C_WARNING" {
				t.Fatalf("unexpected phase: %+v", diagnostic)
			}
			if diagnostic.Phase == "C_STATIC" && diagnostic.Severity != "error" {
				t.Fatalf("C_STATIC must be error: %+v", diagnostic)
			}
			if diagnostic.Phase == "C_WARNING" && diagnostic.Severity != "warning" {
				t.Fatalf("C_WARNING must be warning: %+v", diagnostic)
			}
			if diagnostic.Code == "" || diagnostic.Message == "" {
				t.Fatalf("diagnostic identity/message must be nonempty: %+v", diagnostic)
			}
		}
		if !sort.SliceIsSorted(diagnostics, func(i, j int) bool {
			return diagnosticLess(diagnostics[i], diagnostics[j])
		}) {
			t.Fatalf("diagnostics are not in the frozen presentation order: %+v", diagnostics)
		}
	}
}

func diagnosticLess(a, b Diagnostic) bool {
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	if a.Column != b.Column {
		return a.Column < b.Column
	}
	rank := func(phase string) int {
		if phase == "C_STATIC" {
			return 0
		}
		return 1
	}
	if rank(a.Phase) != rank(b.Phase) {
		return rank(a.Phase) < rank(b.Phase)
	}
	if a.Code != b.Code {
		return a.Code < b.Code
	}
	return a.Message < b.Message
}
