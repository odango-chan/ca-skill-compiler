// Package artifact implements the frozen Compiler Artifact Contract v1alpha4:
// a deterministic virtual file tree with generated Script Output Schemas and one
// generated artifact manifest per Skill. It never writes to disk, never embeds
// dependency bytes and never emits partial output.
package artifact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gowebpki/jcs"
)

const (
	APIVersion     = "connectionagent.io/compiler-artifact/v1alpha4"
	LanguageTarget = "connectionagent.io/v1alpha4"
	GeneratorName  = "ca-skill-compiler"
	// Draft2020 is the exact frozen JSON Schema dialect URI.
	Draft2020 = "https://json-schema.org/draft/2020-12/schema"
)

// File is one output file with exact bytes.
type File struct {
	Path  string
	Bytes []byte
}

// ExportRef is one reachable export reference in a Skill's dependency closure.
type ExportRef struct {
	ExportKind string `json:"exportKind"`
	ExportID   string `json:"exportId"`
	Owner      string `json:"owner"`
}

// DependencyClosureEntry records only logical package and export identity; it
// deliberately carries no digest, URL, revision or credential.
type DependencyClosureEntry struct {
	DependencyID    string      `json:"dependencyId"`
	StablePackageID string      `json:"stablePackageId"`
	Exports         []ExportRef `json:"exports"`
}

// Input is the compiler-supplied content for one Skill artifact.
type Input struct {
	SkillName        string
	Goal             string
	GeneratorVersion string
	GeneratorCommit  string
	SourceFiles      []File
	GeneratedSchemas []File
	Dependency       []DependencyClosureEntry
}

type manifestSkill struct {
	Name string `json:"name"`
	Goal string `json:"goal"`
}

type manifestGenerator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
}

type manifest struct {
	APIVersion        string                   `json:"apiVersion"`
	LanguageTarget    string                   `json:"languageTarget"`
	Skill             manifestSkill            `json:"skill"`
	Generator         manifestGenerator        `json:"generator"`
	SourceFiles       []string                 `json:"sourceFiles"`
	GeneratedFiles    []string                 `json:"generatedFiles"`
	DependencyClosure []DependencyClosureEntry `json:"dependencyClosure"`
}

// ManifestPath returns the generated manifest path for a Skill.
func ManifestPath(skill string) string {
	return "skills/" + skill + "/generated/artifact-manifest.json"
}

// Build assembles the deterministic output tree for one Skill. The result is
// sorted by path; the manifest is JCS-canonicalized. Duplicate paths and
// manifest collisions are hard errors before any output is returned.
func Build(input Input) ([]File, error) {
	if err := validateInput(input); err != nil {
		return nil, err
	}
	generated := make([]File, 0, len(input.GeneratedSchemas))
	for _, file := range input.GeneratedSchemas {
		canonical, err := CanonicalizeJSON(file.Bytes)
		if err != nil {
			return nil, fmt.Errorf("generated Schema %s is not valid JSON: %w", file.Path, err)
		}
		generated = append(generated, File{Path: file.Path, Bytes: canonical})
	}
	sortFiles(generated)

	manifestKeys := make([]string, 0, len(input.Dependency))
	for _, entry := range input.Dependency {
		manifestKeys = append(manifestKeys, entry.DependencyID)
	}
	_ = manifestKeys
	closure, err := normalizeClosure(input.Dependency)
	if err != nil {
		return nil, err
	}

	manifestValue := manifest{
		APIVersion:        APIVersion,
		LanguageTarget:    LanguageTarget,
		Skill:             manifestSkill{Name: input.SkillName, Goal: input.Goal},
		Generator:         manifestGenerator{Name: GeneratorName, Version: input.GeneratorVersion, Commit: input.GeneratorCommit},
		SourceFiles:       filePaths(input.SourceFiles),
		GeneratedFiles:    filePaths(generated),
		DependencyClosure: closure,
	}
	manifestBytes, err := CanonicalizeValue(manifestValue)
	if err != nil {
		return nil, fmt.Errorf("artifact manifest canonicalization failed: %w", err)
	}
	manifestPath := ManifestPath(input.SkillName)
	if collision, ok := collides(append(filePaths(input.SourceFiles), filePaths(generated)...), manifestPath); ok {
		return nil, fmt.Errorf("artifact manifest path collides with output %s", collision)
	}

	files := make([]File, 0, len(input.SourceFiles)+len(generated)+1)
	files = append(files, input.SourceFiles...)
	for _, file := range generated {
		files = append(files, File{Path: file.Path, Bytes: file.Bytes})
	}
	files = append(files, File{Path: manifestPath, Bytes: manifestBytes})
	sortFiles(files)
	return files, nil
}

func validateInput(input Input) error {
	if !validSkillName(input.SkillName) {
		return fmt.Errorf("invalid Skill name %q", input.SkillName)
	}
	if strings.TrimSpace(input.Goal) == "" || len(input.Goal) > 240 {
		return fmt.Errorf("Skill goal is required and must not exceed 240 bytes")
	}
	if strings.TrimSpace(input.GeneratorVersion) == "" || len(input.GeneratorVersion) > 80 {
		return fmt.Errorf("generator version is required and must not exceed 80 bytes")
	}
	if commit := input.GeneratorCommit; commit != "" && len(commit) < 7 {
		return fmt.Errorf("generator commit must be at least 7 characters")
	}
	if len(input.SourceFiles) == 0 {
		return fmt.Errorf("a Skill artifact requires at least one source file")
	}
	seen := map[string]bool{}
	for _, file := range append(append([]File{}, input.SourceFiles...), input.GeneratedSchemas...) {
		if err := validateOutputPath(file.Path); err != nil {
			return err
		}
		if seen[file.Path] {
			return fmt.Errorf("duplicate output path %s", file.Path)
		}
		seen[file.Path] = true
	}
	if seen[ManifestPath(input.SkillName)] {
		return fmt.Errorf("artifact manifest path collides with an output file")
	}
	return nil
}

func validSkillName(name string) bool {
	if name == "" || len(name) > 80 {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
		default:
			return false
		}
	}
	return true
}

func validateOutputPath(path string) error {
	if !strings.HasPrefix(path, "skills/") || len(path) > 240 {
		return fmt.Errorf("output path must be a normalized skills/ path within 240 bytes: %q", path)
	}
	if strings.Contains(path, "\\") || strings.Contains(path, "//") || strings.Contains(path, "../") || strings.HasSuffix(path, "/") {
		return fmt.Errorf("output path is not normalized: %q", path)
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("output path has an invalid segment: %q", path)
		}
	}
	return nil
}

func filePaths(files []File) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	sort.Strings(paths)
	return paths
}

func sortFiles(files []File) {
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
}

func collides(existing []string, candidate string) (string, bool) {
	for _, path := range existing {
		if path == candidate || strings.HasPrefix(path, candidate+"/") || strings.HasPrefix(candidate, path+"/") {
			return path, true
		}
	}
	return "", false
}

func normalizeClosure(entries []DependencyClosureEntry) ([]DependencyClosureEntry, error) {
	out := make([]DependencyClosureEntry, 0, len(entries))
	seenDependency := map[string]bool{}
	for _, entry := range entries {
		if entry.DependencyID == "" || entry.StablePackageID == "" {
			return nil, fmt.Errorf("dependency closure entries require dependencyId and stablePackageId")
		}
		if seenDependency[entry.DependencyID] {
			return nil, fmt.Errorf("duplicate dependency closure entry %s", entry.DependencyID)
		}
		seenDependency[entry.DependencyID] = true
		seenExport := map[ExportRef]bool{}
		exports := make([]ExportRef, 0, len(entry.Exports))
		for _, export := range entry.Exports {
			if export.ExportKind == "" || export.ExportID == "" || export.Owner == "" {
				return nil, fmt.Errorf("export references require exportKind, exportId and owner")
			}
			if seenExport[export] {
				continue
			}
			seenExport[export] = true
			exports = append(exports, export)
		}
		sort.Slice(exports, func(i, j int) bool {
			a, b := exports[i], exports[j]
			if a.ExportKind != b.ExportKind {
				return a.ExportKind < b.ExportKind
			}
			if a.ExportID != b.ExportID {
				return a.ExportID < b.ExportID
			}
			return a.Owner < b.Owner
		})
		out = append(out, DependencyClosureEntry{DependencyID: entry.DependencyID, StablePackageID: entry.StablePackageID, Exports: exports})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DependencyID != out[j].DependencyID {
			return out[i].DependencyID < out[j].DependencyID
		}
		return out[i].StablePackageID < out[j].StablePackageID
	})
	return out, nil
}

// CanonicalizeValue serializes a Go value as compact JSON and applies RFC 8785
// JCS through the frozen OSS implementation.
func CanonicalizeValue(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return CanonicalizeJSON(bytes.TrimSpace(buffer.Bytes()))
}

// CanonicalizeJSON applies RFC 8785 JCS to valid UTF-8 JSON bytes.
func CanonicalizeJSON(data []byte) ([]byte, error) {
	canonical, err := jcs.Transform(data)
	if err != nil {
		return nil, err
	}
	if !json.Valid(canonical) {
		return nil, fmt.Errorf("JCS produced invalid JSON")
	}
	return canonical, nil
}
