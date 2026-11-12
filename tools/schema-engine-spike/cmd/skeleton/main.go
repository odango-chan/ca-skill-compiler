package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	spike "github.com/odango-chan/ca-skill-compiler/tools/schema-engine-spike"
)

type baseline struct{}
type acceptsAll struct{}

func (baseline) Name() string { return "no-engine-size-baseline" }
func (baseline) Compile(_ map[string]any, _ string, _ bool, _ string) (spike.Compiled, error) {
	return spike.Compiled{Validator: acceptsAll{}, Trace: &spike.Trace{}}, nil
}
func (acceptsAll) Validate(any) error { return nil }

type report struct {
	Candidate    string                 `json:"candidate"`
	GoVersion    string                 `json:"goVersion"`
	GOOS         string                 `json:"goos"`
	GOARCH       string                 `json:"goarch"`
	HarnessNanos int64                  `json:"harnessNanos"`
	Vectors      spike.RunResult        `json:"vectors"`
	Security     []spike.SecurityResult `json:"security"`
	Regex        []spike.RegexResult    `json:"regex"`
}

func main() {
	started := time.Now()
	fixture, err := spike.LoadEmbeddedFixture()
	if err != nil {
		fatal(err)
	}
	candidate := baseline{}
	r := report{
		Candidate:    candidate.Name(),
		GoVersion:    runtime.Version(),
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		Vectors:      spike.RunVectors(candidate, fixture, ""),
		Security:     spike.RunSecurity(candidate),
		Regex:        spike.RunRegex(candidate, ""),
		HarnessNanos: time.Since(started).Nanoseconds(),
	}
	b, err := json.Marshal(r)
	if err != nil {
		fatal(err)
	}
	fmt.Println(string(b))
}

func fatal(err error) {
	fmt.Println(err)
	panic(err)
}
