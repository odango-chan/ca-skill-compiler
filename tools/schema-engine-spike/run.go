package spike

import (
	"fmt"
	"runtime"
	"time"
)

type VectorResult struct {
	Candidate        string `json:"candidate"`
	ID               string `json:"id"`
	Passed           bool   `json:"passed"`
	CompileNanos     int64  `json:"compileNanos"`
	ValidateNanos    int64  `json:"validateNanos"`
	HeapAfterCompile uint64 `json:"heapAfterCompile"`
	Error            string `json:"error,omitempty"`
	Trace            *Trace `json:"trace,omitempty"`
}

type RunResult struct {
	Candidate string         `json:"candidate"`
	RegexMode string         `json:"regexMode"`
	Passed    int            `json:"passed"`
	Failed    int            `json:"failed"`
	Vectors   []VectorResult `json:"vectors"`
}

func RunVectors(c Candidate, fixture Fixture, regexMode string) RunResult {
	result := RunResult{Candidate: c.Name(), RegexMode: regexMode}
	for _, tc := range fixture.Cases {
		started := time.Now()
		compiled, err := c.Compile(fixture.Resources, tc.Root, tc.Mode == "assertion", regexMode)
		compileDuration := time.Since(started)
		if err != nil {
			result.add(VectorResult{Candidate: c.Name(), ID: tc.ID, CompileNanos: compileDuration.Nanoseconds(), Error: err.Error(), Trace: compiled.Trace})
			continue
		}
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		valid := true
		var validationErr string
		validationStarted := time.Now()
		for _, instance := range tc.Valid {
			if err := compiled.Validator.Validate(instance); err != nil {
				valid = false
				validationErr = appendError(validationErr, "expected valid instance: "+err.Error())
			}
		}
		for _, instance := range tc.Invalid {
			if err := compiled.Validator.Validate(instance); err == nil {
				valid = false
				validationErr = appendError(validationErr, "expected invalid instance accepted")
			}
		}
		validationDuration := time.Since(validationStarted)
		result.add(VectorResult{Candidate: c.Name(), ID: tc.ID, Passed: valid, CompileNanos: compileDuration.Nanoseconds(), ValidateNanos: validationDuration.Nanoseconds(), HeapAfterCompile: mem.HeapAlloc, Error: validationErr, Trace: compiled.Trace})
	}
	return result
}

func (r *RunResult) add(v VectorResult) {
	r.Vectors = append(r.Vectors, v)
	if v.Passed {
		r.Passed++
	} else {
		r.Failed++
	}
}

func appendError(existing, next string) string {
	if existing == "" {
		return next
	}
	return fmt.Sprintf("%s; %s", existing, next)
}
