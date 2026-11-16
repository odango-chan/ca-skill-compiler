package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	BuildRequestAPIVersion          = "connectionagent.io/compiler-build-request/v1alpha4"
	BuildResultAPIVersion           = "connectionagent.io/compiler-build-result/v1alpha4"
	DependencyInspectionAPIVersion  = "connectionagent.io/compiler-dependency-inspection/v1alpha4"
	maxResolvedDependencies         = 128
	maxSelectedTargets              = 256
	maxPathBytes                    = 240
	maxTargetRunes                  = 80
	maxDependencyIDBytes            = 128
	maxOwnerIDRunes                 = 160
	maxExportIDRunes                = 240
)

// BuildRequest is the only accepted Compiler input API. Byte slices use
// base64 on JSON/WASM transports and remain raw bytes for native Go callers.
type BuildRequest struct {
	APIVersion           string                `json:"apiVersion"`
	LanguageTarget       string                `json:"languageTarget"`
	Files                []File                `json:"files"`
	ResolvedDependencies []ResolvedDependency  `json:"resolvedDependencies"`
	Targets              []string              `json:"targets,omitempty"`
	TrustedBuildContext  *TrustedBuildContext  `json:"trustedBuildContext,omitempty"`
}

type ResolvedDependency struct {
	DependencyID string `json:"dependencyId"`
	Digest       string `json:"digest"`
	Files        []File `json:"files"`
}

type TrustedDependencyBinding struct {
	DependencyID    string        `json:"dependencyId"`
	Digest          string        `json:"digest"`
	StablePackageID string        `json:"stablePackageId"`
	AuthorizedOwners []string     `json:"authorizedOwners"`
	ExportOwners    []ExportOwner `json:"exportOwners,omitempty"`
}

type ExportOwner struct {
	ExportKind string `json:"exportKind"`
	ExportID   string `json:"exportId"`
	Owner      string `json:"owner"`
}

type CapabilityContractsBundle struct {
	DeploymentID string `json:"deploymentId"`
	PublisherID  string `json:"publisherId"`
	Digest       string `json:"digest"`
	Files        []File `json:"files"`
}

type EvidenceTrustBindingsBundle struct {
	DeploymentID  string `json:"deploymentId"`
	PublisherID   string `json:"publisherId"`
	SnapshotDigest string `json:"snapshotDigest"`
	Files         []File `json:"files"`
}

type DeploymentAuthorization struct {
	DeploymentID     string `json:"deploymentId"`
	PublisherID      string `json:"publisherId"`
	AuthorizationRef string `json:"authorizationRef"`
}

type TrustedBuildContext struct {
	DependencyBindings       []TrustedDependencyBinding  `json:"dependencyBindings,omitempty"`
	CapabilityContracts      *CapabilityContractsBundle  `json:"capabilityContracts,omitempty"`
	DeploymentAuthorizations *[]DeploymentAuthorization  `json:"deploymentAuthorizations,omitempty"`
	EvidenceTrustBindings    *EvidenceTrustBindingsBundle `json:"evidenceTrustBindings,omitempty"`
}

type Diagnostic struct {
	Severity string `json:"severity"`
	Phase    string `json:"phase"`
	Code     string `json:"code"`
	Path     string `json:"path"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Message  string `json:"message"`
}

type BuildResult struct {
	APIVersion       string       `json:"apiVersion"`
	Success          bool         `json:"success"`
	Diagnostics      []Diagnostic `json:"diagnostics"`
	DependencyClosure []string    `json:"dependencyClosure"`
	OutputFiles      []File       `json:"outputFiles"`
}

type InspectedCallable struct {
	DependencyID       string `json:"dependencyId"`
	Owner              string `json:"owner"`
	Kind               string `json:"kind"`
	ID                 string `json:"id"`
	Effect             string `json:"effect"`
	SourcePath         string `json:"sourcePath"`
	InputSchemaID      string `json:"inputSchemaId,omitempty"`
	QueryContextSchemaID string `json:"queryContextSchemaId,omitempty"`
	OutputSchemaID    string `json:"outputSchemaId"`
	ReceiptSchemaID   string `json:"receiptSchemaId,omitempty"`
}

type InspectedSchema struct {
	DependencyID string `json:"dependencyId"`
	Owner        string `json:"owner"`
	ID           string `json:"id"`
	SourcePath   string `json:"sourcePath"`
}

type DependencyInspection struct {
	APIVersion  string              `json:"apiVersion"`
	Diagnostics []Diagnostic        `json:"diagnostics"`
	Callables   []InspectedCallable `json:"callables"`
	Schemas     []InspectedSchema   `json:"schemas"`
}

// DecodeBuildRequestJSON applies the strict JSON object contract before the
// decoded request reaches the Compiler Core. encoding/json decodes []byte
// fields from base64 and DisallowUnknownFields rejects undeclared properties.
func DecodeBuildRequestJSON(data []byte) (BuildRequest, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return BuildRequest{}, err
	}
	if object == nil {
		return BuildRequest{}, fmt.Errorf("BuildRequest must be a JSON object")
	}
	if err := rejectJSONNullProperties(object, "targets", "trustedBuildContext"); err != nil {
		return BuildRequest{}, err
	}
	if contextJSON, ok := object["trustedBuildContext"]; ok {
		var contextObject map[string]json.RawMessage
		if err := json.Unmarshal(contextJSON, &contextObject); err != nil {
			return BuildRequest{}, err
		}
		if contextObject == nil {
			return BuildRequest{}, fmt.Errorf("trustedBuildContext must be an object")
		}
		if err := rejectJSONNullProperties(contextObject, "dependencyBindings", "capabilityContracts", "deploymentAuthorizations", "evidenceTrustBindings"); err != nil {
			return BuildRequest{}, err
		}
		if bindingsJSON, ok := contextObject["dependencyBindings"]; ok {
			var bindings []json.RawMessage
			if err := json.Unmarshal(bindingsJSON, &bindings); err != nil {
				return BuildRequest{}, err
			}
			for index, bindingJSON := range bindings {
				var binding map[string]json.RawMessage
				if err := json.Unmarshal(bindingJSON, &binding); err != nil {
					return BuildRequest{}, err
				}
				if binding == nil {
					return BuildRequest{}, fmt.Errorf("trustedBuildContext.dependencyBindings[%d] must be an object", index)
				}
				if err := rejectJSONNullProperties(binding, "exportOwners"); err != nil {
					return BuildRequest{}, err
				}
			}
		}
		if bundleJSON, ok := contextObject["capabilityContracts"]; ok {
			var bundle map[string]json.RawMessage
			if err := json.Unmarshal(bundleJSON, &bundle); err != nil {
				return BuildRequest{}, err
			}
			if bundle == nil {
				return BuildRequest{}, fmt.Errorf("trustedBuildContext.capabilityContracts must be an object")
			}
			if err := rejectJSONNullProperties(bundle, "files"); err != nil {
				return BuildRequest{}, err
			}
		}
		if bundleJSON, ok := contextObject["evidenceTrustBindings"]; ok {
			var bundle map[string]json.RawMessage
			if err := json.Unmarshal(bundleJSON, &bundle); err != nil {
				return BuildRequest{}, err
			}
			if bundle == nil {
				return BuildRequest{}, fmt.Errorf("trustedBuildContext.evidenceTrustBindings must be an object")
			}
			if err := rejectJSONNullProperties(bundle, "files"); err != nil {
				return BuildRequest{}, err
			}
		}
	}
	var req BuildRequest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return BuildRequest{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return BuildRequest{}, io.ErrUnexpectedEOF
		}
		return BuildRequest{}, err
	}
	return req, nil
}

func rejectJSONNullProperties(object map[string]json.RawMessage, names ...string) error {
	for _, name := range names {
		if raw, ok := object[name]; ok && bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return fmt.Errorf("%s must not be null when present", name)
		}
	}
	return nil
}

func Check(req BuildRequest) []Diagnostic {
	if diagnostics := validateBuildRequest(req); len(diagnostics) > 0 {
		return diagnostics
	}
	return validateV1Alpha4(req).Diagnostics
}

func Build(req BuildRequest) BuildResult {
	if diagnostics := validateBuildRequest(req); len(diagnostics) > 0 {
		return failedBuildResult(diagnostics)
	}
	return validateV1Alpha4(req)
}

// InspectDependencies exposes the frozen v1alpha4 output shape. The dependency
// linker is implemented in WP3; until then, nonempty snapshots fail closed
// rather than returning a misleading partial inventory.
func InspectDependencies(req BuildRequest) DependencyInspection {
	diagnostics := validateBuildRequest(req)
	if len(diagnostics) == 0 && len(req.ResolvedDependencies) > 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Phase:    "C_STATIC",
			Code:     "DEPENDENCY_IDENTITY_UNVERIFIED",
			Path:     "",
			Message:  "DependencyInspection cannot publish a trusted inventory until the v1alpha4 dependency linker is implemented",
		})
	}
	sortDiagnostics(diagnostics)
	if diagnostics == nil {
		diagnostics = []Diagnostic{}
	}
	return DependencyInspection{
		APIVersion:  DependencyInspectionAPIVersion,
		Diagnostics: diagnostics,
		Callables:   []InspectedCallable{},
		Schemas:     []InspectedSchema{},
	}
}

func failedBuildResult(diagnostics []Diagnostic) BuildResult {
	sortDiagnostics(diagnostics)
	if diagnostics == nil {
		diagnostics = []Diagnostic{}
	}
	return BuildResult{
		APIVersion:        BuildResultAPIVersion,
		Success:           false,
		Diagnostics:      diagnostics,
		DependencyClosure: []string{},
		OutputFiles:      []File{},
	}
}

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
			}
			seen[file.Path] = true
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

func sortDiagnostics(diagnostics []Diagnostic) {
	sort.Slice(diagnostics, func(i, j int) bool {
		a, b := diagnostics[i], diagnostics[j]
		if a.Path != b.Path {
			return bytes.Compare([]byte(a.Path), []byte(b.Path)) < 0
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		rank := func(phase string) int {
			switch phase {
			case "C_STATIC":
				return 0
			case "C_WARNING":
				return 1
			default:
				return 2
			}
		}
		if rank(a.Phase) != rank(b.Phase) {
			return rank(a.Phase) < rank(b.Phase)
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Message < b.Message
	})
}
