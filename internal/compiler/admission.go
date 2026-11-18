package compiler

// Single-version BuildRequest admission.
//
// CA-SL v1alpha5 is the only supported language target. This file owns the
// strict public-API envelope admission and the target-neutral path, identity
// and digest grammar shared by the Compiler Core. There is no version dispatch,
// sniffing or fallback: an apiVersion or languageTarget that is not exactly the
// v1alpha5 identity fails closed here before any source is parsed.

import (
	"path"
	"strings"
	"unicode/utf8"
)

// validateBuildRequest is the single public BuildRequest admission. It mirrors
// the released v1alpha4 envelope obligations, adds the v1alpha5-only trusted
// artifact/Schema bindings and rejects every other API or language version.
func validateBuildRequest(req BuildRequest) []Diagnostic {
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

	if req.APIVersion != BuildRequestAPIVersion {
		add("REQUEST_INVALID", "", "BuildRequest.apiVersion must equal "+BuildRequestAPIVersion)
	}
	if req.LanguageTarget != V1Alpha5Target {
		add("LANGUAGE_TARGET_UNSUPPORTED", "", "Unsupported CA-SL target: "+req.LanguageTarget)
	}
	if len(req.Files) == 0 {
		add("SOURCE_ROOT_INVALID", "", "BuildRequest.files must be a nonempty source file array")
	}
	if len(req.Files) > maxFiles || len(req.Targets) > maxSelectedTargets || len(req.ResolvedDependencies) > maxResolvedDependencies {
		add("INPUT_LIMIT", "", "BuildRequest exceeds the source-file, dependency-snapshot or selected-target count limit")
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

	// Validate the immutable dependency snapshot set and the trusted build
	// context (including the trusted artifact/Schema bindings) before any
	// linking pass runs.
	diagnostics = append(diagnostics, validateV1Alpha5DependencyContext(req)...)

	sortDiagnostics(diagnostics)
	if diagnostics == nil {
		return []Diagnostic{}
	}
	return diagnostics
}

func validRelativeSnapshotPath(p string) bool {
	if p == "" || !utf8.ValidString(p) || len(p) > maxPathBytes || strings.HasPrefix(p, "/") ||
		strings.ContainsAny(p, "\\:\x00") || pathClean(p) != p {
		return false
	}
	parts := strings.Split(p, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func validSourcePath(p string) bool {
	if !validRelativeSnapshotPath(p) {
		return false
	}
	parts := strings.Split(p, "/")
	return len(parts) >= 3 && parts[0] == "skills"
}

func validDependencyID(value string) bool {
	if value == "" || len(value) > maxDependencyIDBytes || !asciiLetter(value[0]) {
		return false
	}
	for i := 1; i < len(value); i++ {
		c := value[i]
		if !asciiLetter(c) && !(c >= '0' && c <= '9') && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func validURIIdentity(value string) bool {
	if value == "" || !asciiLetter(value[0]) {
		return false
	}
	for i := 1; i < len(value); i++ {
		c := value[i]
		if c == ':' {
			return true
		}
		if !asciiLetter(c) && !(c >= '0' && c <= '9') && c != '+' && c != '.' && c != '-' {
			return false
		}
	}
	return false
}

func validDigest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for i := len("sha256:"); i < len(value); i++ {
		c := value[i]
		if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// validExportKind is the inherited trusted export-kind set. The v1alpha5
// IngressCatalog adds the package-scoped ingressOperation kind on top of it.
func validExportKind(value string) bool {
	switch value {
	case "operation", "observation", "schema", "evidenceGuarantee":
		return true
	default:
		return false
	}
}

func asciiLetter(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func pathClean(value string) string {
	// Paths in the transport are POSIX paths on every host platform.
	return path.Clean(value)
}
