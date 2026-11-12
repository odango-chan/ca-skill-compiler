package main

import (
	"encoding/json"
	"os"
	"runtime"
	"time"

	"github.com/odango-chan/ca-skill-compiler/tools/schema-engine-spike"
)

type report struct {
	Candidate         string                 `json:"candidate"`
	GoVersion         string                 `json:"goVersion"`
	GOOS              string                 `json:"goos"`
	GOARCH            string                 `json:"goarch"`
	CustomRegexEngine bool                   `json:"customRegexEngineAvailable"`
	HarnessNanos      int64                  `json:"harnessNanos"`
	Vectors           spike.RunResult        `json:"vectors"`
	Security          []spike.SecurityResult `json:"security"`
	RegexDefault      []spike.RegexResult    `json:"regexDefault"`
	OfficialSuite     *spike.SuiteResult     `json:"officialSuite,omitempty"`
}

func result() ([]byte, error) {
	started := time.Now()
	fixture, err := spike.LoadEmbeddedFixture()
	if err != nil {
		return nil, err
	}
	candidate := Candidate{}
	r := report{
		Candidate:         candidate.Name(),
		GoVersion:         runtime.Version(),
		GOOS:              runtime.GOOS,
		GOARCH:            runtime.GOARCH,
		CustomRegexEngine: false,
		Vectors:           spike.RunVectors(candidate, fixture, ""),
		Security:          spike.RunSecurity(candidate),
		RegexDefault:      spike.RunRegex(candidate, ""),
	}
	r.HarnessNanos = time.Since(started).Nanoseconds()
	if suiteRoot := os.Getenv("SPIKE_SUITE_ROOT"); suiteRoot != "" {
		suite, err := spike.RunOfficialSuite(candidate, suiteRoot, "")
		if err != nil {
			return nil, err
		}
		r.OfficialSuite = &suite
	}
	return json.MarshalIndent(r, "", "  ")
}
