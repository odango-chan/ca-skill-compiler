package spike

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const SuiteCommit = "38dea932b02986f496345bf007108277777630bf"

type SuiteCounts struct {
	Total              int `json:"total"`
	Passed             int `json:"passed"`
	Failed             int `json:"failed"`
	Skipped            int `json:"skipped"`
	CompileFailures    int `json:"compileFailures"`
	ValidationFailures int `json:"validationFailures"`
}

type SuiteFailure struct {
	File         string `json:"file"`
	Group        string `json:"group"`
	Test         string `json:"test"`
	Optional     bool   `json:"optional"`
	FailureKind  string `json:"failureKind,omitempty"`
	CompileError string `json:"compileError,omitempty"`
	Expected     bool   `json:"expected"`
	Actual       bool   `json:"actual"`
	Error        string `json:"error,omitempty"`
}

type SuiteResult struct {
	Candidate        string         `json:"candidate"`
	RegexMode        string         `json:"regexMode"`
	Commit           string         `json:"commit"`
	Required         SuiteCounts    `json:"required"`
	Optional         SuiteCounts    `json:"optional"`
	CompatibleGroups int            `json:"compatibleGroups"`
	SkippedGroups    int            `json:"skippedGroups"`
	Failures         []SuiteFailure `json:"failures"`
}

type suiteFile struct {
	Tests []suiteGroup `json:"tests"`
}

type suiteGroup struct {
	Description     string         `json:"description"`
	Compatibility   string         `json:"compatibility"`
	Optional        bool           `json:"optional"`
	Tags            []string       `json:"tags"`
	Schema          any            `json:"schema"`
	ExternalSchemas map[string]any `json:"externalSchemas"`
	Tests           []suiteCase    `json:"tests"`
}

type suiteCase struct {
	Description string `json:"description"`
	Data        any    `json:"data"`
	Valid       bool   `json:"valid"`
}

func RunOfficialSuite(c Candidate, suiteRoot, regexMode string) (SuiteResult, error) {
	root := filepath.Join(suiteRoot, "validation", "tests")
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path == filepath.Join(root, "proposals") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".json") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return SuiteResult{}, err
	}
	sort.Strings(files)
	result := SuiteResult{Candidate: c.Name(), Commit: SuiteCommit, RegexMode: regexMode}
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			return result, err
		}
		var document suiteFile
		decoder := json.NewDecoder(bytes.NewReader(b))
		decoder.UseNumber()
		if err := decoder.Decode(&document); err != nil {
			return result, fmt.Errorf("%s: %w", file, err)
		}
		optionalFile := strings.Contains(filepath.ToSlash(file), "/tests/optional/")
		rel, _ := filepath.Rel(root, file)
		for i, group := range document.Tests {
			optional := optionalFile || group.Optional || hasTag(group.Tags, "format-assertion")
			counts := countsFor(&result, optional)
			counts.Total += len(group.Tests)
			if !compatible2020(group.Compatibility) {
				counts.Skipped += len(group.Tests)
				result.SkippedGroups++
				continue
			}
			result.CompatibleGroups++
			rootURI := fmt.Sprintf("https://json-schema-test-suite.invalid/2020-12/%s/%d.json", filepath.ToSlash(rel), i)
			if m, ok := group.Schema.(map[string]any); ok {
				if id, ok := m["$id"].(string); ok && id != "" {
					rootURI = id
				}
			}
			resources := make(map[string]any, len(group.ExternalSchemas)+1)
			resources[rootURI] = group.Schema
			for uri, schema := range group.ExternalSchemas {
				resources[uri] = schema
			}
			assertFormat := hasTag(group.Tags, "format-assertion")
			compiled, compileErr := c.Compile(resources, rootURI, assertFormat, regexMode)
			for _, test := range group.Tests {
				failure := SuiteFailure{File: rel, Group: group.Description, Test: test.Description, Optional: optional, Expected: test.Valid}
				if compileErr != nil {
					failure.FailureKind = "compile"
					failure.CompileError = compileErr.Error()
					failure.Error = "schema compilation failed"
					result.addSuiteFailure(failure)
					continue
				}
				validationErr := compiled.Validator.Validate(test.Data)
				failure.Actual = validationErr == nil
				if validationErr != nil {
					failure.Error = validationErr.Error()
				}
				if failure.Actual == test.Valid {
					counts.Passed++
				} else {
					failure.FailureKind = "validation"
					result.addSuiteFailure(failure)
				}
			}
		}
	}
	return result, nil
}

func countsFor(result *SuiteResult, optional bool) *SuiteCounts {
	if optional {
		return &result.Optional
	}
	return &result.Required
}

func (r *SuiteResult) addSuiteFailure(f SuiteFailure) {
	counts := countsFor(r, f.Optional)
	counts.Failed++
	switch f.FailureKind {
	case "compile":
		counts.CompileFailures++
	case "validation":
		counts.ValidationFailures++
	default:
		panic("suite failure must have a compile or validation kind")
	}
	if len(r.Failures) < 200 {
		r.Failures = append(r.Failures, f)
	}
}

func hasTag(tags []string, wanted string) bool {
	for _, tag := range tags {
		if tag == wanted {
			return true
		}
	}
	return false
}

func compatible2020(value string) bool {
	return compatibleRelease(value, compatibilityRank("2020"))
}

type compatibilityConstraint struct {
	operator string
	version  int
}

// compatibleRelease follows validation/README.md in the pinned
// JSON-Schema-Test-Suite commit. Compatibility tokens describe ordered release
// ranges; the sequence must be evaluated as a whole, including the early break
// at the first upper boundary. Unknown releases are rejected rather than
// silently treated as compatible.
func compatibleRelease(value string, versionUnderTest int) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	terms := strings.Split(value, ",")
	constraints := make([]compatibilityConstraint, 0, len(terms))
	for _, raw := range terms {
		term := strings.TrimSpace(raw)
		if term == "" {
			return false
		}
		operator := ""
		switch {
		case strings.HasPrefix(term, "<="):
			operator = "<="
			term = strings.TrimSpace(strings.TrimPrefix(term, "<="))
		case strings.HasPrefix(term, "="):
			operator = "="
			term = strings.TrimSpace(strings.TrimPrefix(term, "="))
		}
		rank := compatibilityRank(term)
		if rank < 0 {
			return false
		}
		constraints = append(constraints, compatibilityConstraint{operator: operator, version: rank})
	}

	valid := true
	for _, constraint := range constraints {
		switch constraint.operator {
		case "":
			valid = versionUnderTest >= constraint.version
		case "<=":
			valid = versionUnderTest <= constraint.version
		case "=":
			valid = versionUnderTest == constraint.version
		}
		if versionUnderTest <= constraint.version {
			break
		}
	}
	return valid
}

func compatibilityRank(value string) int {
	switch value {
	case "3":
		return 3
	case "4":
		return 4
	case "6":
		return 6
	case "7":
		return 7
	case "2019":
		return 8
	case "2020":
		return 9
	case "2027":
		return 10
	case "2028":
		return 11
	case "9999":
		return 99
	default:
		return -1
	}
}
