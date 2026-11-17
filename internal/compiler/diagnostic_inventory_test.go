package compiler

// Frozen diagnostic inventory guard: every code the Compiler emits in these
// fixtures must exist in the pinned Appendix H diagnostic index (except the two
// Compiler API envelope codes).

import (
	"encoding/json"
	"os"
	"testing"
)

func TestEmittedDiagnosticCodesAreFrozen(t *testing.T) {
	raw, err := os.ReadFile("../../tests/conformance/v1alpha4/diagnostic-index.json")
	if err != nil {
		t.Fatalf("read diagnostic index: %v", err)
	}
	var index struct {
		Diagnostics []struct {
			Code  string `json:"code"`
			Phase string `json:"phase"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatalf("decode diagnostic index: %v", err)
	}
	frozen := map[string]bool{}
	for _, entry := range index.Diagnostics {
		frozen[entry.Code] = true
	}
	// Compiler API envelope diagnostics are not CA-SL Appendix H identities.
	envelope := map[string]bool{"REQUEST_INVALID": true, "INPUT_LIMIT": true}

	requests := []BuildRequest{
		wp3aBaseFixture(t).request(t),
		v4TestRequest(v4TestContract, v4TestProcedure, v4TestPredicates, v4TestSchema),
	}
	for _, request := range requests {
		for _, diagnostic := range Check(request) {
			if diagnostic.Phase != "C_STATIC" && diagnostic.Phase != "C_WARNING" {
				t.Fatalf("non-static phase emitted: %+v", diagnostic)
			}
			if !frozen[diagnostic.Code] && !envelope[diagnostic.Code] {
				t.Fatalf("diagnostic code is not in the frozen Appendix H index: %+v", diagnostic)
			}
		}
	}
}
