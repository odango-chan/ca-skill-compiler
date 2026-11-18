package compiler

// Public Compiler API.
//
// This file owns the public BuildRequest / BuildResult / Diagnostic /
// DependencyInspection wire DTOs, the strict JSON transport decoder and the
// four public entry points. CA-SL v1alpha5 is the only supported language
// target: there is exactly one public API version set, one language target and
// one compiler pipeline. Unknown targets fail closed during admission; there is
// no fallback, version sniffing or compatibility router.
//
// The public structs intentionally carry no parser, linker or artifact-version
// types, so the Compiler Core never imports or depends on a specific artifact
// contract version.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

const (
	// BuildRequestAPIVersion is the only accepted BuildRequest.apiVersion.
	BuildRequestAPIVersion = "connectionagent.io/compiler-build-request/v1alpha5"
	// BuildResultAPIVersion is stamped on every BuildResult, including
	// fail-closed results.
	BuildResultAPIVersion = "connectionagent.io/compiler-build-result/v1alpha5"
	// DependencyInspectionAPIVersion is the independent dependency-inspection
	// API version of the single v1alpha5 target.
	DependencyInspectionAPIVersion = "connectionagent.io/compiler-dependency-inspection/v1alpha5"

	maxResolvedDependencies = 128
	maxSelectedTargets      = 256
	maxPathBytes            = 240
	maxTargetRunes          = 80
	maxDependencyIDBytes    = 128
	maxOwnerIDRunes         = 160
	maxExportIDRunes        = 240
)

// BuildRequest is the only accepted Compiler input API. Byte slices use
// base64 on JSON/WASM transports and remain raw bytes for native Go callers.
type BuildRequest struct {
	APIVersion           string               `json:"apiVersion"`
	LanguageTarget       string               `json:"languageTarget"`
	Files                []File               `json:"files"`
	ResolvedDependencies []ResolvedDependency `json:"resolvedDependencies"`
	Targets              []string             `json:"targets,omitempty"`
	TrustedBuildContext  *TrustedBuildContext `json:"trustedBuildContext,omitempty"`
}

type ResolvedDependency struct {
	DependencyID string `json:"dependencyId"`
	Digest       string `json:"digest"`
	Files        []File `json:"files"`
}

type TrustedDependencyBinding struct {
	DependencyID     string        `json:"dependencyId"`
	Digest           string        `json:"digest"`
	StablePackageID  string        `json:"stablePackageId"`
	AuthorizedOwners []string      `json:"authorizedOwners"`
	ExportOwners     []ExportOwner `json:"exportOwners,omitempty"`
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
	DeploymentID   string `json:"deploymentId"`
	PublisherID    string `json:"publisherId"`
	SnapshotDigest string `json:"snapshotDigest"`
	Files          []File `json:"files"`
}

type DeploymentAuthorization struct {
	DeploymentID     string `json:"deploymentId"`
	PublisherID      string `json:"publisherId"`
	AuthorizationRef string `json:"authorizationRef"`
}

// TrustedArtifactBinding is one trusted artifact binding: the exact dependency
// snapshot file that holds an immutable protocol or Schema artifact, its content
// dialect and its caller-declared artifact identity. It carries no digest: the
// Compiler always recomputes the artifact digest from the raw snapshot bytes.
type TrustedArtifactBinding struct {
	ArtifactRef  string `json:"artifactRef"`
	DependencyID string `json:"dependencyId"`
	SnapshotPath string `json:"snapshotPath"`
	Dialect      string `json:"dialect"`
}

// TrustedSchemaBinding is one trusted Schema export binding: the exact
// artifactRef and canonical Schema root pointer (# or #/...) that define one
// exported Schema identity.
type TrustedSchemaBinding struct {
	ExportID    string `json:"exportId"`
	ArtifactRef string `json:"artifactRef"`
	Pointer     string `json:"pointer"`
}

type TrustedBuildContext struct {
	DependencyBindings       []TrustedDependencyBinding   `json:"dependencyBindings,omitempty"`
	CapabilityContracts      *CapabilityContractsBundle   `json:"capabilityContracts,omitempty"`
	DeploymentAuthorizations *[]DeploymentAuthorization   `json:"deploymentAuthorizations,omitempty"`
	EvidenceTrustBindings    *EvidenceTrustBindingsBundle `json:"evidenceTrustBindings,omitempty"`

	// Trusted Ingress/Schema artifact bindings. Every artifact digest is
	// recomputed from the raw dependency snapshot bytes at link time.
	SnapshotBaseURI  string                   `json:"snapshotBaseURI,omitempty"`
	ArtifactBindings []TrustedArtifactBinding `json:"artifactBindings,omitempty"`
	SchemaBindings   []TrustedSchemaBinding   `json:"schemaBindings,omitempty"`
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
	APIVersion        string       `json:"apiVersion"`
	Success           bool         `json:"success"`
	Diagnostics       []Diagnostic `json:"diagnostics"`
	DependencyClosure []string     `json:"dependencyClosure"`
	OutputFiles       []File       `json:"outputFiles"`
}

type InspectedCallable struct {
	DependencyID         string `json:"dependencyId"`
	Owner                string `json:"owner"`
	Kind                 string `json:"kind"`
	ID                   string `json:"id"`
	Effect               string `json:"effect"`
	SourcePath           string `json:"sourcePath"`
	InputSchemaID        string `json:"inputSchemaId,omitempty"`
	QueryContextSchemaID string `json:"queryContextSchemaId,omitempty"`
	OutputSchemaID       string `json:"outputSchemaId"`
	ReceiptSchemaID      string `json:"receiptSchemaId,omitempty"`
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

	// IngressOperations is the independent Ingress Operation inventory. Ingress
	// operations are never mixed into Callables.
	IngressOperations []InspectedIngressOperation `json:"ingressOperations,omitempty"`
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
		if err := rejectJSONNullProperties(contextObject, "dependencyBindings", "capabilityContracts", "deploymentAuthorizations", "evidenceTrustBindings", "snapshotBaseURI", "artifactBindings", "schemaBindings"); err != nil {
			return BuildRequest{}, err
		}
		if bindingsJSON, ok := contextObject["artifactBindings"]; ok {
			if err := rejectJSONNullArrayElements(bindingsJSON, "trustedBuildContext.artifactBindings"); err != nil {
				return BuildRequest{}, err
			}
		}
		if bindingsJSON, ok := contextObject["schemaBindings"]; ok {
			if err := rejectJSONNullArrayElements(bindingsJSON, "trustedBuildContext.schemaBindings"); err != nil {
				return BuildRequest{}, err
			}
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

// rejectJSONNullArrayElements enforces that a JSON array property is an array of
// objects; a null element would otherwise decode into a zero-value struct.
func rejectJSONNullArrayElements(raw json.RawMessage, name string) error {
	var elements []json.RawMessage
	if err := json.Unmarshal(raw, &elements); err != nil {
		return fmt.Errorf("%s must be an array: %w", name, err)
	}
	for index, element := range elements {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(element, &object); err != nil || object == nil {
			return fmt.Errorf("%s[%d] must be an object", name, index)
		}
	}
	return nil
}

// Check runs the complete static validation pipeline for one BuildRequest. It is
// the only Check entry: Build runs exactly the same pipeline and only assembles
// an artifact when Check returns no error diagnostic.
func Check(req BuildRequest) []Diagnostic {
	if diagnostics := validateBuildRequest(req); len(diagnostics) > 0 {
		return diagnostics
	}
	return checkSources(req)
}

// Build compiles one BuildRequest. It re-runs the strict admission and the full
// static pipeline, then assembles the deterministic artifact tree only when no
// error diagnostic was produced. A failed Build never returns partial output
// files.
func Build(req BuildRequest) BuildResult {
	if diagnostics := validateBuildRequest(req); len(diagnostics) > 0 {
		return failedBuildResult(diagnostics)
	}
	diagnostics := checkSources(req)
	if hasErrorDiagnostic(diagnostics) {
		return failedBuildResult(diagnostics)
	}
	result, err := assembleSuccessResult(req)
	if err != nil {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Phase:    "C_STATIC",
			Code:     "CAPABILITY_INPUT_UNTRUSTED",
			Path:     "",
			Message:  limitDiagnosticMessage("fail-closed: artifact assembly failed: " + err.Error()),
		})
		return failedBuildResult(diagnostics)
	}
	return result
}

// InspectDependencies is a read-only, snapshot-scoped inventory projection. It
// validates the supplied immutable snapshots and trusted dependency/Owner
// bindings, then lists only exports whose identity and Owner can be
// established. It never resolves Skill author references, evaluates Capability
// Grants or authorizes invocation.
func InspectDependencies(req BuildRequest) DependencyInspection {
	return inspectDependencies(req)
}

func failedBuildResult(diagnostics []Diagnostic) BuildResult {
	sortDiagnostics(diagnostics)
	if diagnostics == nil {
		diagnostics = []Diagnostic{}
	}
	return BuildResult{
		APIVersion:        BuildResultAPIVersion,
		Success:           false,
		Diagnostics:       diagnostics,
		DependencyClosure: []string{},
		OutputFiles:       []File{},
	}
}

// VerifyBuildResult checks the structural integrity of a successful BuildResult
// artifact tree (unique paths, canonical manifest per Skill, complete file
// listing, source digests, Schema closure identity). It never writes to disk
// and returns nil for an empty or failed result.
//
// A successful result must carry the single v1alpha5 BuildResult API version.
// Any other result version is rejected rather than verified against a
// different artifact contract.
func VerifyBuildResult(result BuildResult) error {
	if !result.Success {
		return nil
	}
	if result.APIVersion != BuildResultAPIVersion {
		return fmt.Errorf("unsupported BuildResult.apiVersion %q: expected %s", result.APIVersion, BuildResultAPIVersion)
	}
	files := make([]File, 0, len(result.OutputFiles))
	for _, file := range result.OutputFiles {
		files = append(files, File{Path: file.Path, Bytes: file.Bytes})
	}
	return verifyV1Alpha5ArtifactTree(result, files)
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
