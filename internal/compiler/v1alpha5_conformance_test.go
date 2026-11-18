package compiler

// WP7b: post-integration v1alpha5 EW/IN conformance-register integrity test.
//
// The register is machine-readable traceability, not an executed conformance
// suite. This test enforces that the register is complete, deterministic and in
// sync with the Markdown matrix, that every implemented row names tests that
// actually exist, and that every referenced diagnostic identity is a real
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
	wp7INCount = 66
)

// Normative code sets for the register integrity check (hard-coded from
// Appendix L.11/L.11.2 and the Ingress diagnostic registry v1alpha1).
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
	wp7InheritedCodes = map[string]bool{
		"YAML_PARSE_ERROR":                   true,
		"UNKNOWN_FIELD":                      true,
		"CONTROL_FLOW_DANGLING":              true,
		"CONTROL_FLOW_CYCLE":                 true,
		"SOURCE_PATH_INVALID":                true,
		"SOURCE_NOT_AVAILABLE":               true,
		"INPUT_TYPE_MISMATCH":                true,
		"INPUT_NULLABILITY_MISMATCH":         true,
		"SCHEMA_MISSING":                     true,
		"SCHEMA_INVALID":                     true,
		"DEPENDENCY_EXPORT_OWNER_UNVERIFIED": true,
		"DEPENDENCY_OWNER_MISMATCH":          true,
		"DEPENDENCY_IDENTITY_UNVERIFIED":     true,
		"DEPENDENCY_DUPLICATE":               true,
		"DEPENDENCY_ID_MISMATCH":             true,
		"DEPENDENCY_SNAPSHOT_MISSING":        true,
		"DEPENDENCY_REQUIREMENT_INVALID":     true,
		"EXPORT_ID_MISSING":                  true,
		"EXPORT_ID_AMBIGUOUS":                true,
		"ENTRY_PRECONDITION_BYPASSED":        true,
		"PROCEDURE_START_MISSING":            true,
		"NODE_ID_RESERVED":                   true,
		"SOURCE_ROOT_INVALID":                true,
		"REQUEST_INVALID":                    true,
		"INPUT_LIMIT":                        true,
		"PACKAGE_PATH_INVALID":               true,
		"LANGUAGE_TARGET_UNSUPPORTED":        true,
	}
	wp7KnownDeferredWPs = map[string]bool{
		"WP1": true, "WP3": true, "WP4": true, "WP5": true, "WP6": true,
	}
)

// wp7TestTable is the explicit same-package Go map/test-name table. Referencing
// the real functions makes a renamed or removed test a compile failure.
var wp7TestTable = map[string]func(*testing.T){
	"TestV1Alpha5DiagnosticContractEWAnchors":          TestV1Alpha5DiagnosticContractEWAnchors,
	"TestV1Alpha5GraphUnknownResponseAfterTarget":      TestV1Alpha5GraphUnknownResponseAfterTarget,
	"TestV1Alpha5WaitWithoutIngressExportIsNotFound":   TestV1Alpha5WaitWithoutIngressExportIsNotFound,
	"TestV1Alpha5WP3PositiveTrustedIngressLink":        TestV1Alpha5WP3PositiveTrustedIngressLink,
	"TestV1Alpha5WP3RetentionWindowInvalid":            TestV1Alpha5WP3RetentionWindowInvalid,
	"TestV1Alpha5WP3SelectorAmbiguous":                 TestV1Alpha5WP3SelectorAmbiguous,
	"TestV1Alpha5WP3SelectorNotFound":                  TestV1Alpha5WP3SelectorNotFound,
	"TestV1Alpha5WP3StartWaitWithoutArmingStart":       TestV1Alpha5WP3StartWaitWithoutArmingStart,
	"TestV1Alpha5WP3TriggerNotDominatingOnOnePath":     TestV1Alpha5WP3TriggerNotDominatingOnOnePath,
	"TestV1Alpha5WP3TriggerOperationAmbiguous":         TestV1Alpha5WP3TriggerOperationAmbiguous,
	"TestV1Alpha5WP3TriggerOperationMissing":           TestV1Alpha5WP3TriggerOperationMissing,
	"TestV1Alpha5WP3IngressExportOwnerMismatch":       TestV1Alpha5WP3IngressExportOwnerMismatch,
	"TestV1Alpha5WP3IngressExportOwnerRecordMissing":  TestV1Alpha5WP3IngressExportOwnerRecordMissing,
	"TestV1Alpha5EW031EntryPreconditionBypass":        TestV1Alpha5EW031EntryPreconditionBypass,
	"TestV1Alpha5WP3TriggerSelfOutputUnavailable":     TestV1Alpha5WP3TriggerSelfOutputUnavailable,
	"TestV1Alpha5WP3WireDiscriminatorDuplicateIdentityInvalid": TestV1Alpha5WP3WireDiscriminatorDuplicateIdentityInvalid,
	"TestV1Alpha5WP6DuplicateDispositionID":           TestV1Alpha5WP6DuplicateDispositionID,
	"TestV1Alpha5WP6DuplicateDispositionStatus":       TestV1Alpha5WP6DuplicateDispositionStatus,
	"TestV1Alpha5WP4TimeoutHandlerReadIsUnavailable":   TestV1Alpha5WP4TimeoutHandlerReadIsUnavailable,
	"TestV1Alpha5WP6BoundaryNodeWithoutProducerStage":  TestV1Alpha5WP6BoundaryNodeWithoutProducerStage,
	"TestV1Alpha5WP6EarlyFailureMappingMissing":        TestV1Alpha5WP6EarlyFailureMappingMissing,
	"TestV1Alpha5WP6MappingCoverageIncomplete":         TestV1Alpha5WP6MappingCoverageIncomplete,
	"TestV1Alpha5WP6MultipartBinaryPartUnmapped":       TestV1Alpha5WP6MultipartBinaryPartUnmapped,
	"TestV1Alpha5WP6PositiveSingleBodylessDisposition": TestV1Alpha5WP6PositiveSingleBodylessDisposition,
	"TestV1Alpha5WP6ProducerOutputSchemaIncompatible":  TestV1Alpha5WP6ProducerOutputSchemaIncompatible,
	"TestV1Alpha5WP6ProducerOutputSchemaUnavailable":   TestV1Alpha5WP6ProducerOutputSchemaUnavailable,
	"TestV1Alpha5WP6SuccessBeforeBoundary":             TestV1Alpha5WP6SuccessBeforeBoundary,
	"TestV1Alpha5WP6UnreachableBoundary":               TestV1Alpha5WP6UnreachableBoundary,
}

// wp7CrossPackageTestTable names implemented tests that live in another package
// of this repository (the WP1 IngressCatalog admission tests). They cannot be
// referenced as Go function values from package compiler, so their existence is
// verified by scanning that package's checked-in test sources.
var wp7CrossPackageTestTable = map[string]string{
	"TestCatalogDuplicateEventSelector":                "internal/ingresscatalog",
	"TestCatalogDuplicateExportID":                     "internal/ingresscatalog",
	"TestCatalogInvalidAPIVersionAndKind":              "internal/ingresscatalog",
	"TestCatalogMissingSchemaClosure":                  "internal/ingresscatalog",
	"TestCatalogProtocolArtifactIdentityUnverified":    "internal/ingresscatalog",
	"TestCatalogProtocolMetadataMismatch":              "internal/ingresscatalog",
	"TestNormalizationMediaTypeCorrespondence":         "internal/ingresscatalog",
	"TestNormalizationDuplicateSourceMediaType":        "internal/ingresscatalog",
	"TestNormalizationRequiredNormalizedPropertyUncovered": "internal/ingresscatalog",
	"TestNormalizationDanglingTarget":                  "internal/ingresscatalog",
	"TestNormalizationTransformTargetTypeMismatch":     "internal/ingresscatalog",
	"TestNormalizationDuplicateTarget":                 "internal/ingresscatalog",
	"TestNormalizationMultipartBoundsInconsistent":     "internal/ingresscatalog",
	"TestBinaryContentReferenceSchemaUnresolved":       "internal/ingresscatalog",
	"TestBinaryContentRetentionBelowRecoveryWindow":    "internal/ingresscatalog",
	"TestPeerProfileReferenceInvalid":                  "internal/ingresscatalog",
	"TestPinnedOpenAPIResponseMultipartRequiresMultipartMapping": "internal/ingresscatalog",
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
	Tests          []string       `json:"tests,omitempty"`
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

// wp7CaseTests returns every executable test name named by one register row.
func wp7CaseTests(c wp7Case) []string {
	tests := make([]string, 0, 1+len(c.Tests))
	if c.Test != "" {
		tests = append(tests, c.Test)
	}
	tests = append(tests, c.Tests...)
	return tests
}

func wp7TestExists(t *testing.T, name string) bool {
	t.Helper()
	if _, ok := wp7TestTable[name]; ok {
		return true
	}
	packageDir, ok := wp7CrossPackageTestTable[name]
	if !ok {
		return false
	}
	matches, err := filepath.Glob(filepath.Join(wp7RepoRoot(t), packageDir, "*_test.go"))
	if err != nil || len(matches) == 0 {
		return false
	}
	pattern := regexp.MustCompile(`^func ` + regexp.QuoteMeta(name) + `\(`)
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read cross-package test source %s: %v", path, err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			if pattern.MatchString(strings.TrimSpace(line)) {
				return true
			}
		}
	}
	return false
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
	if register.Status != "NORMATIVE_OBLIGATIONS_LOCALLY_TESTED_NOT_QUALIFIED" {
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
	var implementedIDs []string

	for index, testCase := range register.Cases {
		if !idPattern.MatchString(testCase.ID) {
			t.Fatalf("case %d has unknown id %q", index, testCase.ID)
		}
		if seen[testCase.ID] {
			t.Fatalf("duplicate case id %q", testCase.ID)
		}
		seen[testCase.ID] = true
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

		if testCase.Diagnostic != nil {
			if testCase.Diagnostic.Code == "" {
				t.Fatalf("%s diagnostic has no code", testCase.ID)
			}
			if !wp7KnownCode(testCase.Diagnostic.Code) {
				t.Fatalf("%s references unknown diagnostic code %q", testCase.ID, testCase.Diagnostic.Code)
			}
			switch testCase.Classification {
			case wp7Implemented, wp7Deferred:
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
			names := wp7CaseTests(testCase)
			if len(names) == 0 {
				t.Fatalf("%s is implemented+tested but names no executable test", testCase.ID)
			}
			for _, name := range names {
				if !wp7TestExists(t, name) {
					t.Fatalf("%s names test %q that does not exist in its checked-in package",
						testCase.ID, name)
				}
			}
			implementedIDs = append(implementedIDs, testCase.ID)
			if testCase.Diagnostic != nil {
				if contract, ok := v1alpha5NormativeDiagnosticContract(testCase.Diagnostic.Code); ok {
					if testCase.Diagnostic.MessageKey != contract.MessageKey {
						t.Fatalf("%s messageKey = %q, want %q", testCase.ID,
							testCase.Diagnostic.MessageKey, contract.MessageKey)
					}
					if testCase.Diagnostic.Template != contract.Template {
						t.Fatalf("%s template = %q, want %q", testCase.ID,
							testCase.Diagnostic.Template, contract.Template)
					}
				} else if !wp7InheritedCodes[testCase.Diagnostic.Code] {
					t.Fatalf("%s implemented diagnostic %q has no normative registry entry",
						testCase.ID, testCase.Diagnostic.Code)
				}
				if testCase.Diagnostic.SourceLocation == "" {
					t.Fatalf("%s implemented diagnostic must document its sourceLocation rule", testCase.ID)
				}
			}
		case wp7Deferred:
			if len(wp7CaseTests(testCase)) != 0 {
				t.Fatalf("%s is deferred but claims executable tests %v",
					testCase.ID, wp7CaseTests(testCase))
			}
			if !wp7KnownDeferredWPs[testCase.DeferredTo] {
				t.Fatalf("%s deferredTo = %q, want one of WP1/WP3/WP4/WP5/WP6",
					testCase.ID, testCase.DeferredTo)
			}
		case wp7External:
			if len(wp7CaseTests(testCase)) != 0 {
				t.Fatalf("%s is external but claims executable tests %v",
					testCase.ID, wp7CaseTests(testCase))
			}
		case wp7Specification:
			if len(wp7CaseTests(testCase)) != 0 {
				t.Fatalf("%s is specification-only but claims executable tests %v",
					testCase.ID, wp7CaseTests(testCase))
			}
		}
	}

	for _, id := range expected {
		if !seen[id] {
			t.Fatalf("missing required case id %q", id)
		}
	}

	// The post-integration implemented+tested set is pinned so this register
	// cannot silently inflate or shrink.
	wantImplemented := []string{
		"EW-012", "EW-013", "EW-014", "EW-015", "EW-016", "EW-018", "EW-019",
		"EW-021", "EW-022", "EW-027", "EW-030", "EW-031", "EW-033", "EW-034", "EW-035", "EW-036",
		"IN-001", "IN-002", "IN-003", "IN-004", "IN-005", "IN-006", "IN-007", "IN-013",
		"IN-014", "IN-015", "IN-016", "IN-018", "IN-019", "IN-020", "IN-021", "IN-023",
		"IN-028", "IN-029", "IN-030", "IN-031", "IN-032", "IN-033", "IN-039", "IN-040",
		"IN-042", "IN-044", "IN-046", "IN-047", "IN-055", "IN-056", "IN-057", "IN-059",
		"IN-060", "IN-062", "IN-063",
	}
	if strings.Join(implementedIDs, ",") != strings.Join(wantImplemented, ",") {
		t.Fatalf("implemented+tested set = %v, want exactly %v", implementedIDs, wantImplemented)
	}

	if counts[wp7Implemented] != 51 ||
		counts[wp7Deferred] != 5 ||
		counts[wp7External] != 45 ||
		counts[wp7Specification] != 1 {
		t.Fatalf("unexpected classification totals: %#v", counts)
	}
	if register.Counts.Implemented != counts[wp7Implemented] ||
		register.Counts.Deferred != counts[wp7Deferred] ||
		register.Counts.External != counts[wp7External] ||
		register.Counts.SpecOnly != counts[wp7Specification] {
		t.Fatalf("register summary counts disagree with rows: summary=%#v rows=%#v",
			register.Counts, counts)
	}
	if register.Counts.EW[wp7Implemented] != 16 || register.Counts.EW[wp7Deferred] != 1 ||
		register.Counts.EW[wp7External] != 18 || register.Counts.EW[wp7Specification] != 1 {
		t.Fatalf("unexpected EW summary counts: %#v", register.Counts.EW)
	}
	if register.Counts.IN[wp7Implemented] != 35 || register.Counts.IN[wp7Deferred] != 4 ||
		register.Counts.IN[wp7External] != 27 || register.Counts.IN[wp7Specification] != 0 {
		t.Fatalf("unexpected IN summary counts: %#v", register.Counts.IN)
	}
}

func wp7KnownCode(code string) bool {
	if wp7NormativeV1Alpha5Codes[code] ||
		wp7HostAdmissionCodes[code] ||
		wp7IngressRegistryCodes[code] ||
		wp7InheritedCodes[code] {
		return true
	}
	return false
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
			// Non-matrix tables (for example the diagnostic identity table)
			// also start with a case id; they are not the classification matrix.
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
}

// TestV1Alpha5ConformanceNoPassClaims asserts the register never represents a
// deferred/external/specification-only row as an executed result.
func TestV1Alpha5ConformanceNoPassClaims(t *testing.T) {
	register := loadWP7Register(t)
	for _, testCase := range register.Cases {
		switch testCase.Classification {
		case wp7Implemented:
			names := wp7CaseTests(testCase)
			if len(names) == 0 {
				t.Fatalf("%s claims implemented+tested but names no test", testCase.ID)
			}
			for _, name := range names {
				if !wp7TestExists(t, name) {
					t.Fatalf("%s claims a non-existent executable test %q", testCase.ID, name)
				}
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
// the register case order is exactly EW-001..EW-036, IN-001..IN-066.
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
