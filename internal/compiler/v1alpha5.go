package compiler

// CA-SL v1alpha5 static validation pipeline (WP2 + WP5 slices).
//
// Static source admission only: the v1alpha5 Procedure/Wait AST is parsed and
// validated, then the WP5 Procedure control-flow graph is built and validated
// (Wait entry vertices, EVENT_RECEIVED/TIMEOUT control edges, DAG, exact route
// targets and the final-Postcondition-PASS goal rule). Trigger/correlation
// linking (WP3), wait.output SourcePath (WP4), response boundary post-dominance
// (WP6), conformance mapping (WP7) and artifact assembly (WP8) are not
// implemented yet. Build therefore stays fail-closed through the internal
// residual gate below after source validation succeeds. This is not product or
// language qualification.

import (
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

// v1alpha5ResidualFailClosedMessage marks the internal residual gate that keeps
// Build fail-closed while later work packages are unimplemented. It is not a
// v1alpha5 language diagnostic identity and must disappear as the pipeline is
// completed.
const v1alpha5ResidualFailClosedMessage = "v1alpha5 Node-entry Wait/Procedure AST validation and Procedure graph control-flow analysis are implemented, but WP3-WP8 trigger/correlation linking, wait.output SourcePath, response boundary, dependency/Schema linking and artifact assembly are not yet implemented; compilation fails closed"

// validateV1Alpha5 is the defensive v1alpha5 Build path. It re-runs the strict
// request admission, then source validation, then the internal residual gate.
func validateV1Alpha5(req BuildRequest) BuildResult {
	if diagnostics := validateBuildRequestV1Alpha5(req); len(diagnostics) > 0 {
		return failedV1Alpha5BuildResult(diagnostics)
	}
	diagnostics := checkV1Alpha5Sources(req)
	if hasErrorDiagnostic(diagnostics) {
		return failedV1Alpha5BuildResult(diagnostics)
	}
	diagnostics = append(diagnostics, Diagnostic{
		Severity: "error",
		Phase:    "C_STATIC",
		Code:     "CAPABILITY_INPUT_UNTRUSTED",
		Path:     "",
		Message:  limitDiagnosticMessage("fail-closed: " + v1alpha5ResidualFailClosedMessage),
	})
	return failedV1Alpha5BuildResult(diagnostics)
}

// checkV1Alpha5Sources parses every selected Skill package exactly once and
// returns the v1alpha5 Procedure/Wait diagnostics. Only the Procedure author
// file is consumed by WP2; the other YAML files are still parsed once so
// malformed YAML and duplicate keys surface as YAML_PARSE_ERROR.
func checkV1Alpha5Sources(req BuildRequest) []Diagnostic {
	var diagnostics []Diagnostic
	add := func(file string, n *yaml.Node, code, message string) {
		line, column := 0, 0
		if n != nil {
			line, column = n.Line, n.Column
		}
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Phase:    "C_STATIC",
			Code:     code,
			Path:     limitDiagnosticPath(file),
			Line:     line,
			Column:   column,
			Message:  limitDiagnosticMessage(message),
		})
	}

	files := make(map[string][]byte, len(req.Files))
	skillNames := make(map[string]bool)
	for _, file := range req.Files {
		if _, exists := files[file.Path]; exists {
			continue
		}
		files[file.Path] = file.Bytes
		parts := strings.Split(file.Path, "/")
		if len(parts) < 2 {
			continue
		}
		skillNames[parts[1]] = true
	}
	if len(skillNames) == 0 {
		add("", nil, "SOURCE_ROOT_INVALID", "No logical Skill package rooted at skills/<name>")
		return sortedV1Alpha5Diagnostics(diagnostics)
	}

	selected := make(map[string]bool)
	if len(req.Targets) == 0 {
		for name := range skillNames {
			selected[name] = true
		}
	} else {
		for _, target := range req.Targets {
			if !skillNames[target] {
				add("", nil, "SOURCE_ROOT_INVALID", "Selected Skill target does not exist in BuildRequest.files: "+target)
				continue
			}
			selected[target] = true
		}
	}

	for _, name := range sortedStringMapKeys(selected) {
		base := "skills/" + name + "/"
		procedurePath := base + "assurance/procedure.yaml"

		// Parse every YAML author file exactly once; the Procedure AST and the
		// Wait declaration share these normalized nodes.
		parsedFiles := make(map[string]*yaml.Node)
		for _, filename := range sortedStringMapKeys(files) {
			if strings.HasPrefix(filename, base) &&
				(strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")) {
				parsedFiles[filename] = v1alpha5ParseYAML(filename, files[filename], add)
			}
		}
		if _, exists := parsedFiles[procedurePath]; !exists {
			parsedFiles[procedurePath] = v1alpha5ParseYAML(procedurePath, nil, add)
		}

		skill, skillDiagnostics := parseV1Alpha5SkillAST(name, parsedFiles)
		diagnostics = append(diagnostics, skillDiagnostics...)
		// WP5: validate the v1alpha5 Procedure control-flow graph after the AST
		// parse, in the same static pipeline as the WP2 Wait rules.
		diagnostics = append(diagnostics, validateV1Alpha5ProcedureGraph(procedurePath, skill)...)
	}

	return sortedV1Alpha5Diagnostics(diagnostics)
}

func v1alpha5ParseYAML(file string, data []byte, add func(string, *yaml.Node, string, string)) *yaml.Node {
	if data == nil {
		add(file, nil, "SOURCE_ROOT_INVALID", "Required v1alpha5 author source is missing")
		return nil
	}
	root, err := yaml.Parse(data)
	if err != nil {
		line, column := yaml.ParseErrorLocation(err)
		node := &yaml.Node{Line: line, Column: column}
		code := "YAML_PARSE_ERROR"
		if yaml.IsRootError(err) {
			code = "SOURCE_ROOT_INVALID"
		}
		add(file, node, code, err.Error())
		return nil
	}
	return root
}

func sortedV1Alpha5Diagnostics(diagnostics []Diagnostic) []Diagnostic {
	sortDiagnostics(diagnostics)
	if diagnostics == nil {
		return []Diagnostic{}
	}
	return diagnostics
}
