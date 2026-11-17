// Package compiler implements stateless, side-effect-free Skill project compilation.
// It neither executes scripts/tools nor opens files, databases, or network sockets.
package compiler

import (
	"bytes"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const (
	maxFiles    = 512
	maxFileSize = 2 << 20
	maxTotal    = 16 << 20
)

// File carries original bytes and a logical path. The containing BuildRequest
// field defines whether the path is Skill-rooted or snapshot-relative.
type File struct {
	Path  string `json:"path"`
	Bytes []byte `json:"bytes"`
}

type libraryTool struct {
	OperationID           string `json:"operationId"`
	SourcePath            string `json:"sourcePath"`
	DeclaredEffect        string `json:"declaredEffect"`
	Description           string `json:"description,omitempty"`
	InputSchema           any    `json:"inputSchema"`
	OutputSchema          any    `json:"outputSchema,omitempty"`
	ReconcileOutputSchema any    `json:"reconcileOutputSchema,omitempty"`
}

// The old library inspection entry point is intentionally not exported. The
// v1alpha4 public operation is InspectDependencies in api_v1alpha4.go.

type analysis struct {
	files                 map[string][]byte
	tools                 map[string]toolEntry
	skills                map[string]*skillEntry
	diagnostics           []Diagnostic
	generated             map[string][]File
	ambiguousOperations   map[string]bool
	operationSources      map[string]string
	schemaRefs            map[string]map[string]bool
	procedureInputSchemas map[*yaml.Node]map[string]*yaml.Node
}

func analyze(req BuildRequest, materialize bool) BuildResult {
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
			for root := range skill.schemaRoots {
				a.addSchemaDependencies(root, usedLibs)
			}
		}
	}
	for filename := range usedLibs {
		a.addSchemaDependencies(filename, usedLibs)
	}
	for filename := range usedLibs {
		outputs = append(outputs, File{Path: filename, Bytes: bytes.Clone(a.files[filename])})
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Path < outputs[j].Path })
	if len(outputs) > maxFiles {
		a.errorAt("", 0, "OUTPUT_LIMIT", "BuildResult.outputFiles exceeds 512 files")
		return a.result(nil)
	}
	outputBytes := 0
	for _, file := range outputs {
		if !validPath(file.Path) || len(file.Bytes) > maxFileSize {
			a.errorAt(file.Path, 0, "OUTPUT_LIMIT", "output path or file size exceeds the BuildResult contract")
			return a.result(nil)
		}
		outputBytes += len(file.Bytes)
	}
	if outputBytes > maxTotal {
		a.errorAt("", 0, "OUTPUT_LIMIT", "BuildResult.outputFiles exceeds the total byte limit")
		return a.result(nil)
	}
	return a.result(outputs)
}

func (a *analysis) addSchemaDependencies(root string, usedLibs map[string]bool) {
	seen := map[string]bool{}
	queue := []string{root}
	for len(queue) > 0 {
		filename := queue[0]
		queue = queue[1:]
		if seen[filename] {
			continue
		}
		seen[filename] = true
		for dep := range a.schemaRefs[filename] {
			if strings.HasPrefix(dep, "lib/") {
				usedLibs[dep] = true
			}
			queue = append(queue, dep)
		}
	}
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

func newAnalysis(req BuildRequest) *analysis {
	a := &analysis{files: make(map[string][]byte), tools: make(map[string]toolEntry), skills: make(map[string]*skillEntry), generated: make(map[string][]File), ambiguousOperations: make(map[string]bool), operationSources: make(map[string]string), schemaRefs: make(map[string]map[string]bool), procedureInputSchemas: make(map[*yaml.Node]map[string]*yaml.Node)}
	if len(req.Files) == 0 {
		a.errorAt("", 0, "REQUEST_INVALID", "BuildRequest.files must contain at least one file")
	}
	total := 0
	if len(req.Files) > maxFiles {
		a.errorAt("", 0, "INPUT_LIMIT", "too many project files")
	}
	for _, file := range req.Files {
		if !validPath(file.Path) {
			a.errorAt(file.Path, 0, "PROJECT_PATH_INVALID", "file path must be normalized and inside skills/ or lib/")
			continue
		}
		if len(file.Bytes) > maxFileSize {
			a.errorAt(file.Path, 0, "FILE_LIMIT", "file exceeds the maximum size")
			continue
		}
		total += len(file.Bytes)
		if _, exists := a.files[file.Path]; exists {
			a.errorAt(file.Path, 0, "PROJECT_DUPLICATE_FILE", "duplicate project file")
			continue
		}
		a.files[file.Path] = bytes.Clone(file.Bytes)
	}
	if total > maxTotal {
		a.errorAt("", 0, "INPUT_LIMIT", "project exceeds maximum bytes")
	}
	if len(req.Targets) > 256 {
		a.errorAt("", 0, "REQUEST_INVALID", "BuildRequest.targets exceeds 256 entries")
	}
	seenTargets := make(map[string]bool, len(req.Targets))
	for _, target := range req.Targets {
		if target == "" || utf8.RuneCountInString(target) > 80 {
			a.errorAt("", 0, "REQUEST_INVALID", "target names must contain 1 to 80 characters")
		}
		if seenTargets[target] {
			a.errorAt("", 0, "REQUEST_INVALID", "BuildRequest.targets entries must be unique: "+target)
		}
		seenTargets[target] = true
	}
	return a
}

func (a *analysis) result(output []File) BuildResult {
	a.sortedDiagnostics()
	if output == nil {
		output = []File{}
	}
	return BuildResult{
		APIVersion: BuildResultAPIVersion,
		Success: !a.hasErrors(),
		Diagnostics: a.diagnostics,
		DependencyClosure: []string{},
		OutputFiles: output,
	}
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
	a.diagnostics = append(a.diagnostics, Diagnostic{Severity: "error", Code: code, Path: limitDiagnosticPath(filename), Line: line, Message: limitDiagnosticMessage(message)})
}

func (a *analysis) errorAtNode(filename string, node *yaml.Node, code, message string) {
	span := sourceSpan(filename, node)
	a.diagnostics = append(a.diagnostics, Diagnostic{Severity: "error", Code: code, Path: limitDiagnosticPath(filename), Line: span.Line, Column: span.Column, Message: limitDiagnosticMessage(message)})
}

func (a *analysis) warningAt(filename string, line int, code, message string) {
	a.diagnostics = append(a.diagnostics, Diagnostic{Severity: "warning", Code: code, Path: limitDiagnosticPath(filename), Line: line, Message: limitDiagnosticMessage(message)})
}

func limitDiagnosticPath(filename string) string {
	if utf8.RuneCountInString(filename) <= 240 {
		return filename
	}
	runes := []rune(filename)
	return string(runes[:240])
}

func limitDiagnosticMessage(message string) string {
	if utf8.RuneCountInString(message) <= 8192 {
		return message
	}
	runes := []rune(message)
	return string(runes[:8192])
}

func validPath(p string) bool {
	if p == "" || utf8.RuneCountInString(p) > 240 || path.Clean(p) != p || strings.HasPrefix(p, "/") ||
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
