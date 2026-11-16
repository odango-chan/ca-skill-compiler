// Command parity-replay produces the Native Go result used as the parity
// reference for the Chromium Worker test. It is tooling only: the Compiler core
// is unchanged.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/odango-chan/ca-skill-compiler/internal/compiler"
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
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: parity-replay <project-dir>")
		os.Exit(2)
	}
	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		fail(err)
	}
	files, err := readProject(root)
	if err != nil {
		fail(err)
	}
	req := compiler.BuildRequest{
		APIVersion:           compiler.BuildRequestAPIVersion,
		LanguageTarget:       compiler.V1Alpha4Target,
		Files:                files,
		ResolvedDependencies: []compiler.ResolvedDependency{},
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

func readProject(root string) ([]compiler.File, error) {
	files := make([]compiler.File, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if !strings.HasPrefix(relative, "skills/") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, compiler.File{Path: relative, Bytes: content})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
