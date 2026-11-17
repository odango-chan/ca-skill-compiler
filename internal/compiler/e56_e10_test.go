package compiler

// E56 and E10b conformance subset: diagnostics whose exact trigger is decided by
// the static graph/availability pipeline and therefore need a focused assertion.

import (
	"testing"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func TestE56TerminalAsProcedureStart(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "start: inspect-registration", "start: SUCCEEDED")
	if got := Check(fixture.request(t)); !v4HasCode(got, "PROCEDURE_START_MISSING") {
		t.Fatalf("terminal spelling as procedure start not rejected: %+v", got)
	}
}

func TestE56TerminalSpellingAsNodeID(t *testing.T) {
	fixture := wp3aBaseFixture(t)
	fixture.replaceFile(wp3aProcedurePath, "start: inspect-registration", "start: SUCCEEDED")
	fixture.replaceFile(wp3aProcedurePath, "  inspect-registration:", "  SUCCEEDED:")
	if got := Check(fixture.request(t)); !v4HasCode(got, "NODE_ID_RESERVED") {
		t.Fatalf("terminal spelling as Node ID not rejected: %+v", got)
	}
}

// TestE10bReconcileOutputUnavailable exercises the E10b/E63 branch directly:
// a Consumer reads an output that is only reachable through the Producer's
// COMMITTED edge, and the Producer is a MUTATION whose Owner declares no
// recoverable success output.
func TestE10bReconcileOutputUnavailable(t *testing.T) {
	value := &yaml.Node{
		Kind:   yaml.ScalarNode,
		Tag:    "!!str",
		Value:  "mutate.operation.output.subscriptionRef",
		Line:   3,
		Column: 9,
	}
	outputs := map[string]producerOutput{
		"mutate": {
			node:        "operation",
			schema:      &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"},
			effect:      "MUTATION",
			recoverable: false,
		},
	}
	reachability := map[string]map[string]bool{
		"COMMITTED": {"consume": true},
	}
	diagnostics := checkProducerPath(value, "consume", "observation", schemaRegistry{}, outputs,
		map[string]map[string]bool{}, reachability, "skills/demo/assurance/procedure.yaml")
	if len(diagnostics) != 1 || diagnostics[0].Code != "RECONCILE_OUTPUT_UNAVAILABLE" {
		t.Fatalf("COMMITTED-only mutation output diagnostics = %+v, want one RECONCILE_OUTPUT_UNAVAILABLE", diagnostics)
	}
	if diagnostics[0].Line != 3 || diagnostics[0].Column != 9 {
		t.Fatalf("diagnostic lacks the SourcePath position: %+v", diagnostics[0])
	}
}
