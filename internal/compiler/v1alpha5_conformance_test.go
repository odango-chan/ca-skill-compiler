package compiler

// WP7: v1alpha5 EW/IN conformance-register integrity test.
//
// The register is machine-readable traceability, not an executed conformance
// suite. This test enforces that the register is complete, deterministic and in
// sync with the Markdown matrix, that every implemented row names a test that
// actually exists, and that every referenced diagnostic identity is a real
// L.11 / Ingress-registry / inherited identity. It never asserts that a
// deferred, external or specification-only row passed anything.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

const (
	wp7Implemented   = "COMPILER_STATIC_IMPLEMENTED_TESTED"
	wp7Deferred      = "COMPILER_STATIC_DEFERRED"
	wp7External      = "HOST_RUNTIME_INTEGRATION_OWNED"
	wp7Specification = "SPECIFICATION_ONLY_NOT_EXECUTED"
)

var wp7PassClaimPattern = regexp.MustCompile(`\bPASS(ED)?\b`)

var wp7Classifications = map[string]bool{
	wp7Implemented:   true,
	wp7Deferred:      true,
	wp7External:      true,
	wp7Specification: true,
}

// wp7EWCount and wp7INCount are the Appendix L.12 / #62 section 14 case counts.
const (
	wp7EWCount = 36
	wp7INCount = 63
)

// Normative code sets for the register integrity check. These are hard-coded
// here from Appendix L.11/L.11.2 and the Ingress diagnostic registry v1alpha1;
// the register may not reference anything else.
var (
	wp7NormativeV1Alpha5Codes = map[string]bool{
		"WAIT_DECLARATION_INVALID":         true,
		"WAIT_TIMEOUT_INVALID":             true,
		"WAIT_INGRESS_EXPORT_NOT_FOUND":    true,
		"WAIT_INGRESS_EXPORT_AMBIGUOUS":    true,
		"WAIT_CORRELATION_BINDING_INVALID": true,
		"WAIT_TIMEOUT_ROUTE_INVALID":       true,
		"WAIT_NODE_STAGE_REQUIRED":         true,
		"WAIT_RESPONSE_BOUNDARY_INVALID":   true,
		"WAIT_RESPONSE_MAPPING_INVALID":    true,
	}
	wp7HostAdmissionCodes = map[string]bool{
		"WAIT_DURATION_LIMIT_EXCEEDED": true,
	}
	wp7IngressRegistryCodes = map[string]bool{
		"INGRESS_CATALOG_INVALID":                  true,
		"INGRESS_EXPORT_ID_DUPLICATE":              true,
		"INGRESS_EVENT_SELECTOR_DUPLICATE":         true,
		"INGRESS_PROTOCOL_ARTIFACT_INVALID":        true,
		"INGRESS_PEER_PROFILE_REFERENCE_INVALID":   true,
		"INGRESS_NORMALIZATION_INVALID":            true,
		"INGRESS_BINARY_CONTENT_LIFECYCLE_INVALID": true,
		"INGRESS_CORRELATION_INVALID":              true,
		"INGRESS_RESPONSE_CONTRACT_INVALID":        true,
		"INGRESS_RESPONSE_MAPPING_INVALID":         true,
	}
	// Inherited frozen v1alpha4 Appendix H identities explicitly reused by
	// Appendix L.11; they are not new v1alpha5 codes and have no messageKey in
	// the L.11.1 registry.
	wp7InheritedCodes = map[string]bool{
		"YAML_PARSE_ERROR":                   true,
		"UNKNOWN_FIELD":                      true,
		"CONTROL_FLOW_DANGLING":              true,
		"CONTROL_FLOW_CYCLE":                 true,
		"SOURCE_PATH_INVALID":                true,
		"SOURCE_NOT_AVAILABLE":               true,
		"INPUT_TYPE_MISMATCH":                true,
		"SCHEMA_MISSING":                     true,
		"SCHEMA_INVALID":                     true,
		"DEPENDENCY_EXPORT_OWNER_UNVERIFIED": true,
		"DEPENDENCY_OWNER_MISMATCH":          true,
		"DEPENDENCY_IDENTITY_UNVERIFIED":     true,
		"EXPORT_ID_MISSING":                  true,
		"EXPORT_ID_AMBIGUOUS":                true,
		"ENTRY_PRECONDITION_BYPASSED":        true,
		// The v1alpha5 Procedure source admission also reuses these frozen
		// source/procedure identities.
		"PROCEDURE_START_MISSING":     true,
		"NODE_ID_RESERVED":            true,
		"SOURCE_ROOT_INVALID":         true,
		"REQUEST_INVALID":             true,
		"INPUT_LIMIT":                 true,
		"PACKAGE_PATH_INVALID":        true,
		"LANGUAGE_TARGET_UNSUPPORTED": true,
	}
	wp7KnownDeferredWPs = map[string]bool{
		"WP1": true, "WP3": true, "WP4": true, "WP5": true, "WP6": true,
	}
)

// wp7TestTable is the explicit Go map/test-name table. Referencing the real
// functions makes a renamed or removed test a compile failure, so a stale
// register entry cannot survive.
var wp7TestTable = map[string]func(*testing.T){
	"TestV1Alpha5DiagnosticContractEWAnchors":              TestV1Alpha5DiagnosticContractEWAnchors,
	"TestV1Alpha5DiagnosticContractMessageKeyRegistry":     TestV1Alpha5DiagnosticContractMessageKeyRegistry,
	"TestV1Alpha5DiagnosticContractResponseBoundaryScope":  TestV1Alpha5DiagnosticContractResponseBoundaryScope,
	"TestV1Alpha5DiagnosticContractDeterminismAndOrdering": TestV1Alpha5DiagnosticContractDeterminismAndOrdering,
	"TestV1Alpha5DiagnosticContractSourceLocations":        TestV1Alpha5DiagnosticContractSourceLocations,
	"TestV1Alpha5DiagnosticContractNoInventedWaitCodes":    TestV1Alpha5DiagnosticContractNoInventedWaitCodes,
	"TestV1Alpha5DiagnosticEnvelopeUnchanged":              TestV1Alpha5DiagnosticEnvelopeUnchanged,
	"TestV1Alpha5WaitDiagnosticAnchors":                    TestV1Alpha5WaitDiagnosticAnchors,
}

type wp7Diagnostic struct {
	Code           string `json:"code"`
	Phase          string `json:"phase"`
	Severity       string `json:"severity"`
	MessageKey     string `json:"messageKey,omitempty"`
	Template       string `json:"template,omitempty"`
	SourceLocation string `json:"sourceLocation,omitempty"`
}

type wp7Case struct {
	ID             string         `json:"id"`
	Owner          string         `json:"owner"`
	Classification string         `json:"classification"`
	Expected       string         `json:"expected"`
	Diagnostic     *wp7Diagnostic `json:"diagnostic,omitempty"`
	RelatedCodes   []string       `json:"relatedCodes,omitempty"`
	DeferredTo     string         `json:"deferredTo,omitempty"`
	DeferredNote   string         `json:"deferredNote,omitempty"`
	Test           string         `json:"test,omitempty"`
}

type wp7Counts struct {
	Implemented int            `json:"COMPILER_STATIC_IMPLEMENTED_TESTED"`
	Deferred    int            `json:"COMPILER_STATIC_DEFERRED"`
	External    int            `json:"HOST_RUNTIME_INTEGRATION_OWNED"`
	SpecOnly    int            `json:"SPECIFICATION_ONLY_NOT_EXECUTED"`
	EW          map[string]int `json:"ew"`
	IN          map[string]int `json:"in"`
}

type wp7Register struct {
	SchemaVersion  string    `json:"schemaVersion"`
	LanguageTarget string    `json:"languageTarget"`
	Status         string    `json:"status"`
	Scope          string    `json:"scope"`
	Counts         wp7Counts `json:"counts"`
	Cases          []wp7Case `json:"cases"`
}

func wp7RepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve the WP7 test source location")
	}
	// internal/compiler/<this file> -> repository root.
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func loadWP7Register(t *testing.T) wp7Register {
	t.Helper()
	path := filepath.Join(wp7RepoRoot(t), "docs", "workstreams", "wp7-conformance-v1alpha5.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read WP7 register %s: %v", path, err)
	}
	var register wp7Register
	if err := json.Unmarshal(data, &register); err != nil {
		t.Fatalf("parse WP7 register %s: %v", path, err)
	}
	return register
}

func wp7ExpectedIDs() []string {
	ids := make([]string, 0, wp7EWCount+wp7INCount)
	for i := 1; i <= wp7EWCount; i++ {
		ids = append(ids, "EW-"+wp7ThreeDigits(i))
	}
	for i := 1; i <= wp7INCount; i++ {
		ids = append(ids, "IN-"+wp7ThreeDigits(i))
	}
	return ids
}

func wp7ThreeDigits(value int) string {
	switch {
	case value < 10:
		return "00" + string(rune('0'+value))
	case value < 100:
		return "0" + string(rune('0'+value/10)) + string(rune('0'+value%10))
	default:
		return string(rune('0'+value/100)) + string(rune('0'+(value/10)%10)) + string(rune('0'+value%10))
	}
}

// TestV1Alpha5ConformanceRegisterIntegrity is the register integrity gate.
func TestV1Alpha5ConformanceRegisterIntegrity(t *testing.T) {
	register := loadWP7Register(t)

	if register.SchemaVersion == "" {
		t.Fatal("register.schemaVersion must be set")
	}
	if register.LanguageTarget != V1Alpha5Target {
		t.Fatalf("register.languageTarget = %q, want %q", register.LanguageTarget, V1Alpha5Target)
	}
	if register.Status != "NORMATIVE_OBLIGATIONS_NOT_QUALIFIED" {
		t.Fatalf("register.status = %q; WP7 must not claim qualification or PASS", register.Status)
	}
	if strings.Contains(strings.ToUpper(register.Status), "PASS") {
		t.Fatalf("register.status must not claim PASS: %q", register.Status)
	}
	if register.Scope == "" {
		t.Fatal("register.scope must state the traceability-only limits")
	}

	expected := wp7ExpectedIDs()
	if len(register.Cases) != len(expected) {
		t.Fatalf("register has %d cases, want exactly %d", len(register.Cases), len(expected))
	}

	seen := make(map[string]bool, len(register.Cases))
	idPattern := regexp.MustCompile(`^(EW|IN)-[0-9]{3}$`)
	counts := map[string]int{}
	ewCounts := map[string]int{}
	inCounts := map[string]int{}

	for index, testCase := range register.Cases {
		if !idPattern.MatchString(testCase.ID) {
			t.Fatalf("case %d has unknown id %q", index, testCase.ID)
		}
		if seen[testCase.ID] {
			t.Fatalf("duplicate case id %q", testCase.ID)
		}
		seen[testCase.ID] = true
		// Deterministic ordering: exactly EW-001..EW-036 then IN-001..IN-063.
		if testCase.ID != expected[index] {
			t.Fatalf("case %d id = %q, want deterministic order position %q",
				index, testCase.ID, expected[index])
		}

		if testCase.Owner == "" {
			t.Fatalf("%s owner must be set", testCase.ID)
		}
		if testCase.Expected == "" {
			t.Fatalf("%s expected must be set", testCase.ID)
		}
		if !wp7Classifications[testCase.Classification] {
			t.Fatalf("%s has invalid classification %q", testCase.ID, testCase.Classification)
		}
		counts[testCase.Classification]++
		if strings.HasPrefix(testCase.ID, "EW-") {
			ewCounts[testCase.Classification]++
		} else {
			inCounts[testCase.Classification]++
		}

		if testCase.Diagnostic != nil {
			if testCase.Diagnostic.Code == "" {
				t.Fatalf("%s diagnostic has no code", testCase.ID)
			}
			if !wp7KnownCode(testCase.Diagnostic.Code) {
				t.Fatalf("%s references unknown diagnostic code %q", testCase.ID, testCase.Diagnostic.Code)
			}
			if strings.HasPrefix(testCase.Diagnostic.Code, "WAIT_") &&
				!wp7NormativeV1Alpha5Codes[testCase.Diagnostic.Code] &&
				!wp7HostAdmissionCodes[testCase.Diagnostic.Code] {
				t.Fatalf("%s references invented WAIT_* code %q", testCase.ID, testCase.Diagnostic.Code)
			}
			if strings.HasPrefix(testCase.Diagnostic.Code, "INGRESS_") &&
				!wp7IngressRegistryCodes[testCase.Diagnostic.Code] {
				t.Fatalf("%s references unknown Ingress registry code %q", testCase.ID, testCase.Diagnostic.Code)
			}
			if testCase.Classification == wp7Implemented || testCase.Classification == wp7Deferred {
				if testCase.Diagnostic.Phase != "C_STATIC" {
					t.Fatalf("%s compiler diagnostic phase = %q, want C_STATIC",
						testCase.ID, testCase.Diagnostic.Phase)
				}
				if testCase.Diagnostic.Severity != "error" {
					t.Fatalf("%s compiler diagnostic severity = %q, want error",
						testCase.ID, testCase.Diagnostic.Severity)
				}
			}
		}
		for _, related := range testCase.RelatedCodes {
			if !wp7KnownCode(related) {
				t.Fatalf("%s references unknown related diagnostic code %q", testCase.ID, related)
			}
		}

		switch testCase.Classification {
		case wp7Implemented:
			if testCase.Test == "" {
				t.Fatalf("%s is implemented+tested but names no executable test", testCase.ID)
			}
			baseTest := strings.SplitN(testCase.Test, "/", 2)[0]
			if _, ok := wp7TestTable[baseTest]; !ok {
				t.Fatalf("%s names test %q that is not in the checked-in Go test table",
					testCase.ID, testCase.Test)
			}
			if testCase.Diagnostic == nil {
				t.Fatalf("%s is implemented+tested but has no diagnostic identity", testCase.ID)
			}
			contract, ok := v1Alpha5NormativeDiagnostics[testCase.Diagnostic.Code]
			if !ok {
				t.Fatalf("%s implemented diagnostic %q has no L.11.1 registry entry",
					testCase.ID, testCase.Diagnostic.Code)
			}
			if testCase.Diagnostic.MessageKey != contract.MessageKey {
				t.Fatalf("%s messageKey = %q, want %q", testCase.ID,
					testCase.Diagnostic.MessageKey, contract.MessageKey)
			}
			if testCase.Diagnostic.Template != contract.Template {
				t.Fatalf("%s template = %q, want %q", testCase.ID,
					testCase.Diagnostic.Template, contract.Template)
			}
			if testCase.Diagnostic.SourceLocation == "" {
				t.Fatalf("%s implemented diagnostic must document its sourceLocation rule", testCase.ID)
			}
			if !v1Alpha5WaitImplementedDiagnosticCodes[testCase.Diagnostic.Code] {
				t.Fatalf("%s claims a code the WP2 static path does not implement: %q",
					testCase.ID, testCase.Diagnostic.Code)
			}
		case wp7Deferred:
			if testCase.Test != "" {
				t.Fatalf("%s is deferred but claims executable test %q", testCase.ID, testCase.Test)
			}
			if !wp7KnownDeferredWPs[testCase.DeferredTo] {
				t.Fatalf("%s deferredTo = %q, want one of WP1/WP3/WP4/WP5/WP6",
					testCase.ID, testCase.DeferredTo)
			}
		case wp7External:
			if testCase.Test != "" {
				t.Fatalf("%s is external but claims executable test %q", testCase.ID, testCase.Test)
			}
		case wp7Specification:
			if testCase.Test != "" {
				t.Fatalf("%s is specification-only but claims executable test %q", testCase.ID, testCase.Test)
			}
		}
	}

	for _, id := range expected {
		if !seen[id] {
			t.Fatalf("missing required case id %q", id)
		}
	}

	// Exactly the WP2-implemented EW set may be COMPILER_STATIC_IMPLEMENTED_TESTED.
	wantImplemented := []string{
		"EW-013", "EW-014", "EW-027", "EW-033", "EW-034", "EW-035", "EW-036",
	}
	var gotImplemented []string
	for _, testCase := range register.Cases {
		if testCase.Classification == wp7Implemented {
			gotImplemented = append(gotImplemented, testCase.ID)
		}
	}
	if strings.Join(gotImplemented, ",") != strings.Join(wantImplemented, ",") {
		t.Fatalf("implemented+tested set = %v, want exactly %v", gotImplemented, wantImplemented)
	}

	if counts[wp7Implemented] != 7 ||
		counts[wp7Deferred] != 50 ||
		counts[wp7External] != 41 ||
		counts[wp7Specification] != 1 {
		t.Fatalf("unexpected classification totals: %#v", counts)
	}
	if ewCounts[wp7Implemented] != 7 || ewCounts[wp7Deferred] != 10 ||
		ewCounts[wp7External] != 18 || ewCounts[wp7Specification] != 1 {
		t.Fatalf("unexpected EW classification totals: %#v", ewCounts)
	}
	if inCounts[wp7Implemented] != 0 || inCounts[wp7Deferred] != 40 ||
		inCounts[wp7External] != 23 || inCounts[wp7Specification] != 0 {
		t.Fatalf("unexpected IN classification totals: %#v", inCounts)
	}

	// The register's own summary counts must agree with the rows.
	if register.Counts.Implemented != counts[wp7Implemented] ||
		register.Counts.Deferred != counts[wp7Deferred] ||
		register.Counts.External != counts[wp7External] ||
		register.Counts.SpecOnly != counts[wp7Specification] {
		t.Fatalf("register summary counts disagree with rows: summary=%#v rows=%#v",
			register.Counts, counts)
	}
	if register.Counts.EW[wp7Implemented] != ewCounts[wp7Implemented] ||
		register.Counts.EW[wp7Deferred] != ewCounts[wp7Deferred] ||
		register.Counts.EW[wp7External] != ewCounts[wp7External] ||
		register.Counts.EW[wp7Specification] != ewCounts[wp7Specification] ||
		register.Counts.IN[wp7Deferred] != inCounts[wp7Deferred] ||
		register.Counts.IN[wp7External] != inCounts[wp7External] {
		t.Fatalf("register per-set counts disagree with rows: summary=%#v ew=%#v in=%#v",
			register.Counts, ewCounts, inCounts)
	}
}

func wp7KnownCode(code string) bool {
	return wp7NormativeV1Alpha5Codes[code] ||
		wp7HostAdmissionCodes[code] ||
		wp7IngressRegistryCodes[code] ||
		wp7InheritedCodes[code] ||
		code == "WAIT_DURATION_LIMIT_EXCEEDED"
}

// TestV1Alpha5ConformanceMarkdownSync asserts that the Markdown matrix and the
// JSON register describe exactly the same case-id -> classification mapping.
func TestV1Alpha5ConformanceMarkdownSync(t *testing.T) {
	register := loadWP7Register(t)
	path := filepath.Join(wp7RepoRoot(t), "docs", "workstreams", "wp7-conformance-v1alpha5.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read WP7 Markdown matrix %s: %v", path, err)
	}

	markdown := make(map[string]string)
	linePattern := regexp.MustCompile(`^\|\s*((?:EW|IN)-[0-9]{3})\s*\|`)
	for lineNumber, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		match := linePattern.FindStringSubmatch(trimmed)
		if match == nil {
			continue
		}
		cells := strings.Split(trimmed, "|")
		if len(cells) < 6 {
			t.Fatalf("markdown matrix line %d is malformed: %q", lineNumber+1, line)
		}
		classification := strings.Trim(strings.TrimSpace(cells[3]), "`")
		if !wp7Classifications[classification] {
			// Non-matrix tables in the document (for example the diagnostic
			// identity table) also start with a case id; they are not the
			// classification matrix, so they are skipped rather than parsed.
			continue
		}
		id := match[1]
		if _, exists := markdown[id]; exists {
			t.Fatalf("markdown matrix has duplicate row for %s", id)
		}
		markdown[id] = classification
	}

	if len(markdown) != len(register.Cases) {
		t.Fatalf("markdown matrix has %d classification rows, JSON register has %d cases",
			len(markdown), len(register.Cases))
	}
	for _, testCase := range register.Cases {
		classification, ok := markdown[testCase.ID]
		if !ok {
			t.Fatalf("markdown matrix is missing case %s", testCase.ID)
		}
		if classification != testCase.Classification {
			t.Fatalf("%s markdown classification = %q, JSON classification = %q",
				testCase.ID, classification, testCase.Classification)
		}
	}
	for id := range markdown {
		if !regexp.MustCompile(`^(EW|IN)-[0-9]{3}$`).MatchString(id) {
			t.Fatalf("markdown matrix contains unknown case id %q", id)
		}
	}
}

// TestV1Alpha5ConformanceNoPassClaims asserts the register never represents a
// deferred/external/specification-only row as an executed result. This guards
// against traceability inflation in the machine-readable register.
func TestV1Alpha5ConformanceNoPassClaims(t *testing.T) {
	register := loadWP7Register(t)
	for _, testCase := range register.Cases {
		switch testCase.Classification {
		case wp7Implemented:
			if _, ok := wp7TestTable[strings.SplitN(testCase.Test, "/", 2)[0]]; !ok {
				t.Fatalf("%s claims a non-existent executable test %q", testCase.ID, testCase.Test)
			}
		case wp7Deferred, wp7External, wp7Specification:
			if wp7PassClaimPattern.MatchString(strings.ToUpper(testCase.Expected)) {
				t.Fatalf("%s (%s) must not claim PASS: %q",
					testCase.ID, testCase.Classification, testCase.Expected)
			}
		}
	}
}

// TestV1Alpha5ConformanceDeterministicIDs is a compact independent check that
// the register case order is exactly EW-001..EW-036, IN-001..IN-063.
func TestV1Alpha5ConformanceDeterministicIDs(t *testing.T) {
	register := loadWP7Register(t)
	got := make([]string, 0, len(register.Cases))
	for _, testCase := range register.Cases {
		got = append(got, testCase.ID)
	}
	expected := wp7ExpectedIDs()
	if strings.Join(got, ",") != strings.Join(expected, ",") {
		gotSorted := append([]string(nil), got...)
		expectedSorted := append([]string(nil), expected...)
		sort.Strings(gotSorted)
		sort.Strings(expectedSorted)
		if strings.Join(gotSorted, ",") != strings.Join(expectedSorted, ",") {
			t.Fatalf("register case-id set mismatch: got %v, want %v", got, expected)
		}
		t.Fatalf("register case order is not deterministic EW then IN order")
	}
}
