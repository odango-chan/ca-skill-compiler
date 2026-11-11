// Package compiler implements stateless, side-effect-free Skill project compilation.
// It neither executes scripts/tools nor opens files, databases, or network sockets.
package compiler

import (
	"bytes"
	"path"
	"sort"
	"strings"
)

const (
	maxFiles    = 512
	maxFileSize = 2 << 20
	maxTotal    = 16 << 20
)

// File is a project-root-relative input or output file.
// The caller owns file IO, authorization, and archive creation.
type File struct {
	Path  string `json:"path"`
	Bytes []byte `json:"bytes"`
}

type Request struct {
	Files   []File   `json:"files"`
	Targets []string `json:"targets,omitempty"`
}

type Diagnostic struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Path     string `json:"path"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Message  string `json:"message"`
}

type Result struct {
	Success     bool         `json:"success"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	OutputFiles []File       `json:"outputFiles"`
}

type LibraryTool struct {
	OperationID           string `json:"operationId"`
	SourcePath            string `json:"sourcePath"`
	SourceStatus          string `json:"sourceStatus"`
	DeclaredEffect        string `json:"declaredEffect,omitempty"`
	Description           string `json:"description,omitempty"`
	InputSchema           any    `json:"inputSchema"`
	OutputSchema          any    `json:"outputSchema,omitempty"`
	ReconcileOutputSchema any    `json:"reconcileOutputSchema,omitempty"`
}

type LibraryResult struct {
	Diagnostics []Diagnostic  `json:"diagnostics"`
	Tools       []LibraryTool `json:"tools"`
}

// InspectLibraries parses only imported, read-only ToolCatalog files and their optional manifest.
func InspectLibraries(req Request) LibraryResult {
	a := newAnalysis(req)
	a.importLibraries()
	return LibraryResult{Diagnostics: a.sortedDiagnostics(), Tools: a.libraryTools()}
}

type analysis struct {
	files               map[string][]byte
	tools               map[string]toolEntry
	skills              map[string]*skillEntry
	diagnostics         []Diagnostic
	generated           map[string][]File
	ambiguousOperations map[string]bool
}

func Check(req Request) []Diagnostic {
	result := analyze(req, false)
	return result.Diagnostics
}

func Build(req Request) Result {
	return analyze(req, true)
}

func analyze(req Request, materialize bool) Result {
	a := newAnalysis(req)
	if len(a.diagnostics) > 0 {
		return a.result(nil)
	}
	a.importLibraries()
	a.parseSkills()
	var selected map[string]bool
	if len(req.Targets) == 0 {
		selected = make(map[string]bool)
		for name := range a.skills {
			selected[name] = true
		}
	} else {
		selected = a.selectSkills(req.Targets)
	}
	a.validateSkills(selected)
	a.filterUnselectedSkillDiagnostics(selected)
	a.compileScripts(selected)
	if !materialize || a.hasErrors() {
		return a.result(nil)
	}
	outputs := make([]File, 0, len(a.files))
	usedLibs := make(map[string]bool)
	for name := range selected {
		prefix := "skills/" + name + "/"
		for filename, content := range a.files {
			if strings.HasPrefix(filename, prefix) {
				outputs = append(outputs, File{Path: filename, Bytes: bytes.Clone(content)})
			}
		}
		if skill := a.skills[name]; skill != nil {
			outputs = append(outputs, a.generated[name]...)
			for filename := range skill.libraryFiles {
				if a.files[filename] != nil {
					usedLibs[filename] = true
				}
			}
			for op := range skill.calls {
				if tool, exists := a.tools[op]; exists {
					usedLibs[tool.source] = true
				}
			}
		}
	}
	for filename := range usedLibs {
		outputs = append(outputs, File{Path: filename, Bytes: bytes.Clone(a.files[filename])})
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Path < outputs[j].Path })
	return a.result(outputs)
}

func (a *analysis) filterUnselectedSkillDiagnostics(selected map[string]bool) {
	kept := a.diagnostics[:0]
	for _, d := range a.diagnostics {
		if strings.HasPrefix(d.Path, "skills/") {
			parts := strings.SplitN(d.Path, "/", 3)
			if len(parts) > 1 && a.skills[parts[1]] != nil && !selected[parts[1]] {
				continue
			}
		}
		kept = append(kept, d)
	}
	a.diagnostics = kept
}

func newAnalysis(req Request) *analysis {
	a := &analysis{files: make(map[string][]byte), tools: make(map[string]toolEntry), skills: make(map[string]*skillEntry), generated: make(map[string][]File), ambiguousOperations: make(map[string]bool)}
	total := 0
	if len(req.Files) > maxFiles {
		a.errorAt("", 0, "INPUT_LIMIT", "too many project files")
	}
	for _, file := range req.Files {
		if !validPath(file.Path) {
			a.errorAt(file.Path, 0, "PATH_INVALID", "file path must be normalized and inside skills/ or lib/")
			continue
		}
		if len(file.Bytes) > maxFileSize {
			a.errorAt(file.Path, 0, "FILE_LIMIT", "file exceeds the maximum size")
			continue
		}
		total += len(file.Bytes)
		if _, exists := a.files[file.Path]; exists {
			a.errorAt(file.Path, 0, "DUPLICATE_PATH", "duplicate project file")
			continue
		}
		a.files[file.Path] = bytes.Clone(file.Bytes)
	}
	if total > maxTotal {
		a.errorAt("", 0, "INPUT_LIMIT", "project exceeds maximum bytes")
	}
	return a
}

func (a *analysis) result(output []File) Result {
	a.sortedDiagnostics()
	if output == nil {
		output = []File{}
	}
	return Result{Success: !a.hasErrors(), Diagnostics: a.diagnostics, OutputFiles: output}
}

func (a *analysis) sortedDiagnostics() []Diagnostic {
	sort.Slice(a.diagnostics, func(i, j int) bool {
		x, y := a.diagnostics[i], a.diagnostics[j]
		if x.Path != y.Path {
			return x.Path < y.Path
		}
		if x.Line != y.Line {
			return x.Line < y.Line
		}
		if x.Column != y.Column {
			return x.Column < y.Column
		}
		if x.Code != y.Code {
			return x.Code < y.Code
		}
		if x.Severity != y.Severity {
			return x.Severity < y.Severity
		}
		return x.Message < y.Message
	})
	if a.diagnostics == nil {
		a.diagnostics = []Diagnostic{}
	}
	return a.diagnostics
}

func (a *analysis) hasErrors() bool {
	for _, d := range a.diagnostics {
		if d.Severity == "error" {
			return true
		}
	}
	return false
}

func (a *analysis) errorAt(filename string, line int, code, message string) {
	a.diagnostics = append(a.diagnostics, Diagnostic{Severity: "error", Code: code, Path: filename, Line: line, Message: message})
}

func (a *analysis) warningAt(filename string, line int, code, message string) {
	a.diagnostics = append(a.diagnostics, Diagnostic{Severity: "warning", Code: code, Path: filename, Line: line, Message: message})
}

func validPath(p string) bool {
	if p == "" || len(p) > 240 || path.Clean(p) != p || strings.HasPrefix(p, "/") ||
		strings.ContainsAny(p, "\\:\x00") || strings.Contains(p, "//") {
		return false
	}
	if !strings.HasPrefix(p, "skills/") && !strings.HasPrefix(p, "lib/") {
		return false
	}
	parts := strings.Split(p, "/")
	for _, part := range parts {
		if part == "." || part == ".." || part == "" {
			return false
		}
	}
	return len(parts) >= 2
}
