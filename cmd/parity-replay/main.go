// Command parity-replay produces the Native Go result used as the parity
// reference for the Chromium Worker test. It is tooling only: the Compiler core
// is unchanged and the project layout is read through the same
// internal/projectinput loader the reference CLI uses.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/odango-chan/ca-skill-compiler/internal/compiler"
	"github.com/odango-chan/ca-skill-compiler/internal/projectinput"
)

type outputFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type buildView struct {
	Success           bool                  `json:"success"`
	Diagnostics       []compiler.Diagnostic `json:"diagnostics"`
	DependencyClosure []string              `json:"dependencyClosure"`
	OutputFiles       []outputFile          `json:"outputFiles"`
}

type view struct {
	Check   []compiler.Diagnostic         `json:"check"`
	Build   buildView                     `json:"build"`
	Inspect compiler.DependencyInspection `json:"inspect"`
}

func main() {
	args := os.Args[1:]
	project, emitRequest := "", ""
	for _, arg := range args {
		if strings.HasPrefix(arg, "--emit-request=") {
			emitRequest = strings.TrimPrefix(arg, "--emit-request=")
			continue
		}
		if project == "" {
			project = arg
		}
	}
	if project == "" {
		fmt.Fprintln(os.Stderr, "usage: parity-replay <project-dir> [--emit-request=path]")
		os.Exit(2)
	}
	root, err := filepath.Abs(project)
	if err != nil {
		fail(err)
	}
	req, err := projectinput.Load(root)
	if err != nil {
		fail(err)
	}
	if emitRequest != "" {
		encoded, err := json.Marshal(req)
		if err != nil {
			fail(err)
		}
		if err := os.WriteFile(emitRequest, encoded, 0o644); err != nil {
			fail(err)
		}
	}
	check := compiler.Check(req)
	result := compiler.Build(req)
	encoded, err := json.Marshal(view{
		Check:   check,
		Inspect: compiler.InspectDependencies(req),
		Build: buildView{
			Success:           result.Success,
			Diagnostics:       result.Diagnostics,
			DependencyClosure: result.DependencyClosure,
			OutputFiles:       summarize(result.OutputFiles),
		},
	})
	if err != nil {
		fail(err)
	}
	fmt.Println(string(encoded))
}

func summarize(files []compiler.File) []outputFile {
	out := make([]outputFile, 0, len(files))
	for _, file := range files {
		sum := sha256.Sum256(file.Bytes)
		out = append(out, outputFile{Path: file.Path, SHA256: hex.EncodeToString(sum[:])})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
