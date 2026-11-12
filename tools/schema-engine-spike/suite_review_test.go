package spike

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCompatibilityMatchesPinnedSuiteRules(t *testing.T) {
	for value, want := range map[string]bool{
		"":                 true,
		"7":                true,
		"2019":             true,
		"<=2019":           false,
		"2020":             true,
		"=2020":            true,
		"6,<=2019":         false,
		"9999":             false,
		"unknown":          false,
		"2021":             false,
		"2027":             false,
		"2028":             false,
		"=3,7,<=2020":      true,
		"<=3,7":            true,
		"=2019,<=2020":     true,
		"6,<=2019,unknown": false,
	} {
		if got := compatible2020(value); got != want {
			t.Errorf("compatible2020(%q) = %v, want %v", value, got, want)
		}
	}
}

func TestCompatibilityRangesAndDisjunctionBoundaries(t *testing.T) {
	for _, tc := range []struct {
		compatibility string
		version       int
		want          bool
	}{
		{"6,<=2019", compatibilityRank("6"), true},
		{"6,<=2019", compatibilityRank("7"), true},
		{"6,<=2019", compatibilityRank("2019"), true},
		{"6,<=2019", compatibilityRank("2020"), false},
		{"=3,7,<=2020", compatibilityRank("3"), true},
		{"=3,7,<=2020", compatibilityRank("4"), false},
		{"=3,7,<=2020", compatibilityRank("7"), true},
		{"=3,7,<=2020", compatibilityRank("2020"), true},
		{"=3,7,<=2020", compatibilityRank("2027"), false},
		{"<=3,7", compatibilityRank("3"), true},
		{"<=3,7", compatibilityRank("4"), false},
		{"<=3,7", compatibilityRank("7"), true},
		{"<=3,7", compatibilityRank("2020"), true},
	} {
		if got := compatibleRelease(tc.compatibility, tc.version); got != tc.want {
			t.Errorf("compatibleRelease(%q, %d) = %v, want %v", tc.compatibility, tc.version, got, tc.want)
		}
	}
}

type suiteAccountingCandidate struct{ compileCalls int }

func (c *suiteAccountingCandidate) Name() string { return "suite-accounting-test" }

func (c *suiteAccountingCandidate) Compile(documents map[string]any, root string, _ bool, _ string) (Compiled, error) {
	c.compileCalls++
	schema, _ := documents[root].(map[string]any)
	if fail, _ := schema["compile-error"].(bool); fail {
		return Compiled{}, errors.New("intentional fixture compile failure")
	}
	reject, _ := schema["reject"].(bool)
	return Compiled{Validator: suiteAccountingValidator{reject: reject}}, nil
}

type suiteAccountingValidator struct{ reject bool }

func (v suiteAccountingValidator) Validate(value any) error {
	if v.reject && value == "reject" {
		return errors.New("intentional fixture validation failure")
	}
	return nil
}

func TestOfficialSuiteCountsEachCaseExactlyOnce(t *testing.T) {
	root := t.TempDir()
	writeSuiteGroups(t, filepath.Join(root, "validation", "tests", "required.json"), []suiteGroup{
		{Description: "required compatible", Tests: []suiteCase{{Description: "one", Data: "ok", Valid: true}, {Description: "two", Data: "ok", Valid: true}}},
		{Description: "required skipped", Compatibility: "=2019", Tests: []suiteCase{{Description: "one", Data: "ok", Valid: true}, {Description: "two", Data: "ok", Valid: true}}},
		{Description: "required compile failure", Schema: map[string]any{"compile-error": true}, Tests: []suiteCase{{Description: "compile", Data: "ok", Valid: true}}},
		{Description: "required validation failure", Schema: map[string]any{"reject": true}, Tests: []suiteCase{{Description: "validate", Data: "reject", Valid: true}}},
	})
	writeSuiteGroups(t, filepath.Join(root, "validation", "tests", "optional", "optional.json"), []suiteGroup{
		{Description: "optional compatible", Tests: []suiteCase{{Description: "pass", Data: "ok", Valid: true}}},
		{Description: "optional skipped", Compatibility: "=2019", Tests: []suiteCase{{Description: "one", Data: "ok", Valid: true}, {Description: "two", Data: "ok", Valid: true}}},
	})
	writeSuiteGroups(t, filepath.Join(root, "validation", "tests", "tagged.json"), []suiteGroup{
		{Description: "optional compile failure", Optional: true, Schema: map[string]any{"compile-error": true}, Tests: []suiteCase{{Description: "compile", Data: "ok", Valid: true}}},
		{Description: "format assertion validation failure", Tags: []string{"format-assertion"}, Schema: map[string]any{"reject": true}, Tests: []suiteCase{{Description: "validate", Data: "reject", Valid: true}}},
	})

	candidate := &suiteAccountingCandidate{}
	result, err := RunOfficialSuite(candidate, root, "")
	if err != nil {
		t.Fatal(err)
	}
	wantRequired := SuiteCounts{Total: 6, Passed: 2, Failed: 2, Skipped: 2, CompileFailures: 1, ValidationFailures: 1}
	if result.Required != wantRequired {
		t.Errorf("Required counts = %+v, want %+v", result.Required, wantRequired)
	}
	wantOptional := SuiteCounts{Total: 5, Passed: 1, Failed: 2, Skipped: 2, CompileFailures: 1, ValidationFailures: 1}
	if result.Optional != wantOptional {
		t.Errorf("Optional counts = %+v, want %+v", result.Optional, wantOptional)
	}
	if result.Required.Total != result.Required.Passed+result.Required.Failed+result.Required.Skipped {
		t.Errorf("Required cases were duplicated or omitted: %+v", result.Required)
	}
	if result.Required.Failed != result.Required.CompileFailures+result.Required.ValidationFailures {
		t.Errorf("Required failure types do not sum to failed cases: %+v", result.Required)
	}
	if result.Optional.Total != result.Optional.Passed+result.Optional.Failed+result.Optional.Skipped {
		t.Errorf("Optional cases were duplicated or omitted: %+v", result.Optional)
	}
	if result.Optional.Failed != result.Optional.CompileFailures+result.Optional.ValidationFailures {
		t.Errorf("Optional failure types do not sum to failed cases: %+v", result.Optional)
	}
	if result.CompatibleGroups != 6 || result.SkippedGroups != 2 {
		t.Errorf("group counts compatible=%d skipped=%d", result.CompatibleGroups, result.SkippedGroups)
	}
	if candidate.compileCalls != 6 {
		t.Errorf("compiled %d groups, want only 6 compatible groups", candidate.compileCalls)
	}
	if len(result.Failures) != 4 {
		t.Fatalf("retained failures = %d, want Required and Optional compile and validation failures", len(result.Failures))
	}
	if result.Failures[0].FailureKind != "compile" || result.Failures[1].FailureKind != "validation" || result.Failures[2].FailureKind != "compile" || result.Failures[3].FailureKind != "validation" {
		t.Errorf("failure kinds = %q, %q, %q, %q", result.Failures[0].FailureKind, result.Failures[1].FailureKind, result.Failures[2].FailureKind, result.Failures[3].FailureKind)
	}
}

func writeSuiteGroups(t *testing.T, path string, groups []suiteGroup) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(suiteFile{Tests: groups})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
}
