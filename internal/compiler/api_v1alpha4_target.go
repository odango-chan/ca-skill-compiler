package compiler

// Frozen v1alpha4 historical-target envelope admission and shared validation
// helpers. The target-neutral public API lives in api.go; CA-SL v1alpha5 is the
// active language target and shares the same semantic pipeline.

import (
	"path"
	"strings"
	"unicode/utf8"
)

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
	if req.LanguageTarget != V1Alpha4Target {
		add("LANGUAGE_TARGET_UNSUPPORTED", "", "Unsupported CA-SL target: "+req.LanguageTarget)
	}
	if req.Files == nil || len(req.Files) == 0 {
		add("SOURCE_ROOT_INVALID", "", "BuildRequest.files must be a nonempty source file array")
	}
	if req.ResolvedDependencies == nil {
		add("REQUEST_INVALID", "", "BuildRequest.resolvedDependencies is required; use an empty array when there are no dependencies")
	}
	if len(req.Files) > maxFiles || len(req.ResolvedDependencies) > maxResolvedDependencies {
		add("INPUT_LIMIT", "", "BuildRequest exceeds the source-file or dependency-snapshot count limit")
	}
	if len(req.Targets) > maxSelectedTargets {
		add("INPUT_LIMIT", "", "BuildRequest.targets exceeds 256 entries")
	}

	type totals struct {
		files int
		bytes int
	}
	var total totals
	validateFileSet := func(files []File, source bool, scope string) bool {
		valid := true
		if len(files) > maxFiles {
			add("INPUT_LIMIT", "", scope+" exceeds the 512-file per-set limit")
			valid = false
		}
		seen := make(map[string]bool, len(files))
		seenFolded := make(map[string]bool, len(files))
		for _, file := range files {
			total.files++
			if file.Bytes == nil {
				add("REQUEST_INVALID", file.Path, "File.bytes is required and must be a base64 string on the JSON wire; use an empty byte slice for an empty file")
				valid = false
			}
			if !validRelativeSnapshotPath(file.Path) || (source && !validSourcePath(file.Path)) {
				add("PACKAGE_PATH_INVALID", file.Path, "File path must be normalized UTF-8 POSIX relative path within its declared source or snapshot root")
				valid = false
			}
			if seen[file.Path] {
				add("PACKAGE_PATH_INVALID", file.Path, "Duplicate file path within "+scope)
				valid = false
			} else if seenFolded[strings.ToLower(file.Path)] {
				add("PACKAGE_PATH_INVALID", file.Path, "Case-folded file path collision within "+scope)
				valid = false
			}
			seen[file.Path] = true
			seenFolded[strings.ToLower(file.Path)] = true
			if len(file.Bytes) > maxFileSize {
				add("INPUT_LIMIT", file.Path, "File exceeds the 2 MiB raw byte limit")
				valid = false
			}
			total.bytes += len(file.Bytes)
		}
		return valid
	}

	validateFileSet(req.Files, true, "BuildRequest.files")
	dependenciesByID := make(map[string]ResolvedDependency, len(req.ResolvedDependencies))
	for _, dependency := range req.ResolvedDependencies {
		if !validDependencyID(dependency.DependencyID) {
			add("DEPENDENCY_REQUIREMENT_INVALID", "", "ResolvedDependency.dependencyId must be a valid dependency alias")
		}
		if _, exists := dependenciesByID[dependency.DependencyID]; exists {
			add("DEPENDENCY_DUPLICATE", "", "Duplicate resolved dependencyId: "+dependency.DependencyID)
		}
		dependenciesByID[dependency.DependencyID] = dependency
		if len(dependency.Files) == 0 {
			add("DEPENDENCY_IDENTITY_UNVERIFIED", "", "Resolved dependency snapshot must contain at least one file: "+dependency.DependencyID)
		}
		filesValid := validateFileSet(dependency.Files, false, "dependency "+dependency.DependencyID)
		if !validDigest(dependency.Digest) {
			add("DEPENDENCY_IDENTITY_UNVERIFIED", "", "Resolved dependency has an invalid SHA-256 digest: "+dependency.DependencyID)
		} else if filesValid && len(dependency.Files) > 0 {
			actual, err := ComputeDependencySnapshotDigest(dependency.Files)
			if err != nil || actual != dependency.Digest {
				add("DEPENDENCY_IDENTITY_UNVERIFIED", "", "Resolved dependency snapshot digest does not match its original file bytes: "+dependency.DependencyID)
			}
		}
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
	}

	if req.TrustedBuildContext != nil {
		ctx := req.TrustedBuildContext
		if ctx.v1alpha5OnlyContextPresent() {
			add("REQUEST_INVALID", "", "TrustedBuildContext.snapshotBaseURI/artifactBindings/schemaBindings are v1alpha5-only bindings and are rejected for the v1alpha4 target")
		}
		authorizations := []DeploymentAuthorization(nil)
		if ctx.DeploymentAuthorizations != nil {
			authorizations = *ctx.DeploymentAuthorizations
		}
		if ctx.DeploymentAuthorizations != nil && authorizations == nil {
			add("REQUEST_INVALID", "", "TrustedBuildContext.deploymentAuthorizations must be an array, not null")
		}
		if len(ctx.DependencyBindings) > maxResolvedDependencies || len(authorizations) > 128 {
			add("INPUT_LIMIT", "", "TrustedBuildContext exceeds the binding or authorization count limit")
		}
		seenBindings := make(map[string]bool, len(ctx.DependencyBindings))
		for _, binding := range ctx.DependencyBindings {
			if !validDependencyID(binding.DependencyID) || !validDigest(binding.Digest) || !validURIIdentity(binding.StablePackageID) || len(binding.AuthorizedOwners) == 0 || len(binding.AuthorizedOwners) > 128 {
				add("DEPENDENCY_IDENTITY_UNVERIFIED", "", "Trusted dependency binding is incomplete or malformed: "+binding.DependencyID)
			}
			if seenBindings[binding.DependencyID] {
				add("DEPENDENCY_DUPLICATE", "", "Duplicate trusted dependency binding: "+binding.DependencyID)
			}
			seenBindings[binding.DependencyID] = true
			if dependency, exists := dependenciesByID[binding.DependencyID]; exists && binding.Digest != dependency.Digest {
				add("DEPENDENCY_IDENTITY_UNVERIFIED", "", "Trusted dependency binding digest differs from the measured snapshot: "+binding.DependencyID)
			}
			owners := make(map[string]bool, len(binding.AuthorizedOwners))
			for _, owner := range binding.AuthorizedOwners {
				if owner == "" || utf8.RuneCountInString(owner) > maxOwnerIDRunes || owners[owner] {
					add("DEPENDENCY_IDENTITY_UNVERIFIED", "", "Trusted dependency binding contains an invalid or duplicate authorized Owner: "+binding.DependencyID)
				}
				owners[owner] = true
			}
			exports := make(map[string]bool, len(binding.ExportOwners))
			if len(binding.ExportOwners) > 512 {
				add("INPUT_LIMIT", "", "Trusted dependency binding exceeds 512 export-owner records")
			}
			for _, export := range binding.ExportOwners {
				key := export.ExportKind + "\x00" + export.ExportID
				if !validExportKind(export.ExportKind) || export.ExportID == "" || utf8.RuneCountInString(export.ExportID) > maxExportIDRunes || export.Owner == "" || utf8.RuneCountInString(export.Owner) > maxOwnerIDRunes || exports[key] {
					add("DEPENDENCY_EXPORT_OWNER_UNVERIFIED", "", "Trusted dependency binding contains an invalid or ambiguous export-owner record: "+binding.DependencyID)
				}
				exports[key] = true
			}
		}
		if ctx.CapabilityContracts != nil {
			bundle := ctx.CapabilityContracts
			if !validURIIdentity(bundle.DeploymentID) || !validURIIdentity(bundle.PublisherID) || !validDigest(bundle.Digest) || len(bundle.Files) == 0 {
				add("CAPABILITY_INPUT_UNTRUSTED", "", "Trusted CapabilityContracts bundle is incomplete or malformed")
			}
			valid := validateFileSet(bundle.Files, false, "CapabilityContracts bundle")
			if valid && len(bundle.Files) > 0 {
				actual, err := ComputeDependencySnapshotDigest(bundle.Files)
				if err != nil || actual != bundle.Digest {
					add("CAPABILITY_INPUT_DIGEST_MISMATCH", "", "CapabilityContracts bundle digest does not match its original file bytes")
				}
			}
		}
		if ctx.EvidenceTrustBindings != nil {
			bundle := ctx.EvidenceTrustBindings
			if !validURIIdentity(bundle.DeploymentID) || !validURIIdentity(bundle.PublisherID) || !validDigest(bundle.SnapshotDigest) || len(bundle.Files) == 0 {
				add("EVIDENCE_GUARANTEE_LINK_INVALID", "", "EvidenceTrustBindings bundle is incomplete or malformed")
			}
			valid := validateFileSet(bundle.Files, false, "EvidenceTrustBindings bundle")
			if valid && len(bundle.Files) > 0 {
				actual, err := ComputeDependencySnapshotDigest(bundle.Files)
				if err != nil || actual != bundle.SnapshotDigest {
					add("EVIDENCE_GUARANTEE_LINK_INVALID", "", "EvidenceTrustBindings bundle digest does not match its original file bytes")
				}
			}
		}
		if (ctx.CapabilityContracts != nil || ctx.EvidenceTrustBindings != nil) && ctx.DeploymentAuthorizations == nil {
			add("REQUEST_INVALID", "", "TrustedBuildContext.deploymentAuthorizations is required when a trusted deployment bundle is supplied")
		}
		for _, authorization := range authorizations {
			if !validURIIdentity(authorization.DeploymentID) || !validURIIdentity(authorization.PublisherID) || !validURIIdentity(authorization.AuthorizationRef) {
				add("CAPABILITY_INPUT_UNTRUSTED", "", "Deployment authorization binding is incomplete or malformed")
			}
		}
	}

	if total.files > maxFiles || total.bytes > maxTotal {
		add("INPUT_LIMIT", "", "BuildRequest exceeds the aggregate 512-file or 16 MiB raw byte limit")
	}
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
