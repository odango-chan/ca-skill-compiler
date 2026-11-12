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
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

type SuiteFailure struct {
	File         string `json:"file"`
	Group        string `json:"group"`
	Test         string `json:"test"`
	Optional     bool   `json:"optional"`
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
					failure.CompileError = compileErr.Error()
					failure.Error = "schema compilation failed"
					result.addSuiteFailure(failure, false)
					continue
				}
				validationErr := compiled.Validator.Validate(test.Data)
				failure.Actual = validationErr == nil
				if validationErr != nil {
					failure.Error = validationErr.Error()
				}
				result.addSuiteFailure(failure, failure.Actual == test.Valid)
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

func (r *SuiteResult) addSuiteFailure(f SuiteFailure, passed bool) {
	counts := countsFor(r, f.Optional)
	if passed {
		counts.Passed++
		return
	}
	counts.Failed++
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
	if value == "" {
		return true
	}
	valid := true
	for _, term := range strings.Split(value, ",") {
		term = strings.TrimSpace(term)
		rank := -1
		switch {
		case strings.HasPrefix(term, "<="):
			rank = compatibilityRank(strings.TrimSpace(strings.TrimPrefix(term, "<=")))
			valid = rank >= 9
		case strings.HasPrefix(term, "="):
			rank = compatibilityRank(strings.TrimSpace(strings.TrimPrefix(term, "=")))
			valid = rank == 9
		default:
			rank = compatibilityRank(term)
			valid = rank >= 0 && 9 >= rank
		}
		if rank >= 0 && 9 <= rank {
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
