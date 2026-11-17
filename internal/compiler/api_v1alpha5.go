package compiler

// CA-SL v1alpha5 Compiler API admission.
//
// The v1alpha5 target is versioned independently from the released v1alpha4
// API. Version and language-target selection is exact: Check/Build dispatch on
// req.LanguageTarget == V1Alpha5Target before the v1alpha4 request validation,
// which deliberately rejects the v1alpha5 target. Unknown targets keep failing
// closed through the v1alpha4 LANGUAGE_TARGET_UNSUPPORTED diagnostic; there is
// no fallback, version sniffing or compatibility router.

import (
	"strings"
	"unicode/utf8"
)

const (
	// BuildRequestAPIVersionV1Alpha5 is the only BuildRequest.apiVersion
	// accepted for the v1alpha5 language target.
	BuildRequestAPIVersionV1Alpha5 = "connectionagent.io/compiler-build-request/v1alpha5"
	// BuildResultAPIVersionV1Alpha5 is stamped on every v1alpha5 BuildResult,
	// including fail-closed results.
	BuildResultAPIVersionV1Alpha5 = "connectionagent.io/compiler-build-result/v1alpha5"
	// DependencyInspectionAPIVersionV1Alpha5 reserves the independent
	// dependency-inspection API version for the v1alpha5 target. Ingress
	// inspection types are a later work package.
	DependencyInspectionAPIVersionV1Alpha5 = "connectionagent.io/compiler-dependency-inspection/v1alpha5"
)

// CheckV1Alpha5 is the strict v1alpha5 static Check entry. It returns only real
// diagnostics from the implemented WP2 Wait/Procedure AST rules, the WP5
// Procedure graph rules and the WP3 trigger/correlation binding rules; it never
// fabricates a fail-closed diagnostic. Build remains fail-closed for the
// unimplemented WP4/WP6-WP8 semantics.
func CheckV1Alpha5(req BuildRequest) []Diagnostic {
	if diagnostics := validateBuildRequestV1Alpha5(req); len(diagnostics) > 0 {
		return diagnostics
	}
	return checkV1Alpha5Sources(req)
}

// validateBuildRequestV1Alpha5 is the strict envelope admission for the
// v1alpha5 API: exact API version, exact language target, non-empty file set,
// normalized source paths and the inherited count/size limits. It mirrors the
// v1alpha4 envelope obligations without adding v1alpha4-only fields.
func validateBuildRequestV1Alpha5(req BuildRequest) []Diagnostic {
	var diagnostics []Diagnostic
	add := func(code, filename, message string) {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Phase:    "C_STATIC",
			Code:     code,
			Path:     limitDiagnosticPath(filename),
			Message:  limitDiagnosticMessage(message),
		})
	}

	if req.APIVersion != BuildRequestAPIVersionV1Alpha5 {
		add("REQUEST_INVALID", "", "BuildRequest.apiVersion must equal "+BuildRequestAPIVersionV1Alpha5)
	}
	if req.LanguageTarget != V1Alpha5Target {
		add("LANGUAGE_TARGET_UNSUPPORTED", "", "Unsupported CA-SL target: "+req.LanguageTarget)
	}
	if len(req.Files) == 0 {
		add("SOURCE_ROOT_INVALID", "", "BuildRequest.files must be a nonempty source file array")
	}
	if len(req.Files) > maxFiles || len(req.Targets) > maxSelectedTargets {
		add("INPUT_LIMIT", "", "BuildRequest exceeds the source-file or selected-target count limit")
	}

	seen := make(map[string]bool, len(req.Files))
	seenFolded := make(map[string]bool, len(req.Files))
	skillNames := make(map[string]bool)
	totalBytes := 0
	for _, file := range req.Files {
		if file.Bytes == nil {
			add("REQUEST_INVALID", file.Path, "File.bytes is required and must be a base64 string on the JSON wire; use an empty byte slice for an empty file")
		}
		if !validSourcePath(file.Path) {
			add("PACKAGE_PATH_INVALID", file.Path, "File path must be a normalized UTF-8 POSIX source path rooted at skills/<skill-name>/")
			continue
		}
		if seen[file.Path] {
			add("PACKAGE_PATH_INVALID", file.Path, "Duplicate file path within BuildRequest.files")
		} else if seenFolded[strings.ToLower(file.Path)] {
			add("PACKAGE_PATH_INVALID", file.Path, "Case-folded file path collision within BuildRequest.files")
		}
		seen[file.Path] = true
		seenFolded[strings.ToLower(file.Path)] = true
		if len(file.Bytes) > maxFileSize {
			add("INPUT_LIMIT", file.Path, "File exceeds the 2 MiB raw byte limit")
		}
		totalBytes += len(file.Bytes)
		parts := strings.Split(file.Path, "/")
		skillNames[parts[1]] = true
	}
	if totalBytes > maxTotal {
		add("INPUT_LIMIT", "", "BuildRequest.files exceeds the 16 MiB aggregate raw byte limit")
	}

	for _, target := range req.Targets {
		if target == "" || utf8.RuneCountInString(target) > maxTargetRunes {
			add("REQUEST_INVALID", "", "Target names must contain 1 to 80 Unicode characters")
		}
	}
	seenTargets := make(map[string]bool, len(req.Targets))
	for _, target := range req.Targets {
		if seenTargets[target] {
			add("REQUEST_INVALID", "", "BuildRequest.targets entries must be unique: "+target)
		}
		seenTargets[target] = true
		if len(req.Files) > 0 && !skillNames[target] {
			add("SOURCE_ROOT_INVALID", "", "Selected Skill target does not exist in BuildRequest.files: "+target)
		}
	}

	// WP3: validate the immutable dependency snapshot set and the trusted
	// v1alpha5 build context (including the v1alpha5-only artifact/Schema
	// bindings) before any linking pass runs.
	diagnostics = append(diagnostics, validateV1Alpha5DependencyContext(req)...)

	sortDiagnostics(diagnostics)
	if diagnostics == nil {
		return []Diagnostic{}
	}
	return diagnostics
}

// failedV1Alpha5BuildResult stamps the independent v1alpha5 result API version
// on a failed result without touching the v1alpha4 result contract.
func failedV1Alpha5BuildResult(diagnostics []Diagnostic) BuildResult {
	result := failedBuildResult(diagnostics)
	result.APIVersion = BuildResultAPIVersionV1Alpha5
	return result
}
