package main

import (
	"encoding/json"
	"os"
	"runtime"
	"time"

	"github.com/odango-chan/ca-skill-compiler/tools/schema-engine-spike"
)

type report struct {
	Candidate     string                 `json:"candidate"`
	GoVersion     string                 `json:"goVersion"`
	GOOS          string                 `json:"goos"`
	GOARCH        string                 `json:"goarch"`
	HarnessNanos  int64                  `json:"harnessNanos"`
	Vectors       spike.RunResult        `json:"vectors"`
	Security      []spike.SecurityResult `json:"security"`
	RegexRE2      []spike.RegexResult    `json:"regexDefault"`
	Regex2        []spike.RegexResult    `json:"regexRegexp2"`
	SuiteRegex2   *spike.SuiteResult     `json:"officialSuiteRegexp2,omitempty"`
	OfficialSuite *spike.SuiteResult     `json:"officialSuite,omitempty"`
}

func result() ([]byte, error) {
	started := time.Now()
	fixture, err := spike.LoadEmbeddedFixture()
	if err != nil {
		return nil, err
	}
	candidate := Candidate{}
	r := report{
		Candidate: candidate.Name(),
		GoVersion: runtime.Version(),
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
		Vectors:   spike.RunVectors(candidate, fixture, ""),
		Security:  spike.RunSecurity(candidate),
		RegexRE2:  spike.RunRegex(candidate, ""),
		Regex2:    spike.RunRegex(candidate, "regexp2"),
	}
	r.HarnessNanos = time.Since(started).Nanoseconds()
	if suiteRoot := os.Getenv("SPIKE_SUITE_ROOT"); suiteRoot != "" {
		suite, err := spike.RunOfficialSuite(candidate, suiteRoot, "")
		if err != nil {
			return nil, err
		}
		r.OfficialSuite = &suite
		if os.Getenv("SPIKE_SUITE_REGEX2") == "1" {
			customSuite, err := spike.RunOfficialSuite(candidate, suiteRoot, "regexp2")
			if err != nil {
				return nil, err
			}
			r.SuiteRegex2 = &customSuite
		}
	}
	return json.MarshalIndent(r, "", "  ")
}
