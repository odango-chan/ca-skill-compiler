package compiler

// Target-neutral public Compiler API.
//
// This file owns the public BuildRequest / BuildResult / Diagnostic /
// DependencyInspection wire DTOs, the strict JSON transport decoder and the
// exact language-target dispatch. The internal semantic pipeline is
// version-neutral; v1alpha4 remains a frozen historical target dispatched from
// here, while CA-SL v1alpha5 is the active language target.
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

	"github.com/odango-chan/ca-skill-compiler/internal/artifact"
)

const (
	BuildRequestAPIVersion         = "connectionagent.io/compiler-build-request/v1alpha4"
	BuildResultAPIVersion          = "connectionagent.io/compiler-build-result/v1alpha4"
	DependencyInspectionAPIVersion = "connectionagent.io/compiler-dependency-inspection/v1alpha4"
	maxResolvedDependencies        = 128
	maxSelectedTargets             = 256
	maxPathBytes                   = 240
	maxTargetRunes                 = 80
	maxDependencyIDBytes           = 128
	maxOwnerIDRunes                = 160
	maxExportIDRunes               = 240
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

// TrustedArtifactBinding is one v1alpha5 trusted artifact binding: the exact
// dependency snapshot file that holds an immutable protocol or Schema artifact,
// its content dialect and its caller-declared artifact identity. It carries no
// digest: WP1 always recomputes the artifact digest from the raw snapshot bytes.
type TrustedArtifactBinding struct {
	ArtifactRef  string `json:"artifactRef"`
	DependencyID string `json:"dependencyId"`
	SnapshotPath string `json:"snapshotPath"`
	Dialect      string `json:"dialect"`
}

// TrustedSchemaBinding is one v1alpha5 trusted Schema export binding: the exact
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

	// The following fields are v1alpha5-only trusted-context bindings. They are
	// declared on the shared struct so the v1alpha5 target can express the
	// trusted Ingress/Schema artifacts, but the frozen v1alpha4 envelope
	// validation explicitly rejects any non-empty value so the v1alpha4 API stays
	// closed.
	SnapshotBaseURI  string                   `json:"snapshotBaseURI,omitempty"`
	ArtifactBindings []TrustedArtifactBinding `json:"artifactBindings,omitempty"`
	SchemaBindings   []TrustedSchemaBinding   `json:"schemaBindings,omitempty"`
}

// v1alpha5OnlyContextPresent reports whether a TrustedBuildContext carries any
// field that belongs only to the v1alpha5 API.
func (ctx *TrustedBuildContext) v1alpha5OnlyContextPresent() bool {
	if ctx == nil {
		return false
	}
	return ctx.SnapshotBaseURI != "" || len(ctx.ArtifactBindings) > 0 || len(ctx.SchemaBindings) > 0
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

	// IngressOperations is the independent v1alpha5 Ingress Operation inventory.
	// It is empty for the frozen v1alpha4 target, so the v1alpha4 JSON projection
	// is byte-compatible (omitempty). Ingress operations are never mixed into
	// Callables.
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

func Check(req BuildRequest) []Diagnostic {
	// Exact target dispatch before the v1alpha4 envelope validation, which
	// deliberately treats every other target (including v1alpha5) as
	// unsupported. There is no fallback or version sniffing.
	if req.LanguageTarget == V1Alpha5Target {
		return CheckV1Alpha5(req)
	}
	if diagnostics := validateBuildRequest(req); len(diagnostics) > 0 {
		return diagnostics
	}
	return validateV1Alpha4(req).Diagnostics
}

func Build(req BuildRequest) BuildResult {
	if req.LanguageTarget == V1Alpha5Target {
		return BuildV1Alpha5(req)
	}
	if diagnostics := validateBuildRequest(req); len(diagnostics) > 0 {
		return failedBuildResult(diagnostics)
	}
	return validateV1Alpha4(req)
}

// InspectDependencies is a read-only, snapshot-scoped inventory projection. It
// validates the supplied immutable snapshots and trusted dependency/Owner
// bindings, then lists only exports whose identity and Owner can be established.
// It never resolves Skill author references, evaluates Capability Grants or
// authorizes invocation.
func InspectDependencies(req BuildRequest) DependencyInspection {
	// Exact target dispatch before the v1alpha4 snapshot inspection, which
	// deliberately treats every other target (including v1alpha5) as unsupported.
	if req.LanguageTarget == V1Alpha5Target {
		return InspectV1Alpha5Dependencies(req)
	}
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
// listing, generated Script Output Schema identity). It never writes to disk and
// returns nil for an empty or failed result.
//
// v1alpha5 results dispatch to the candidate v1alpha5 artifact contract
// verification; the frozen v1alpha4 artifact verification is unchanged.
func VerifyBuildResult(result BuildResult) error {
	if !result.Success {
		return nil
	}
	files := make([]artifact.File, 0, len(result.OutputFiles))
	for _, file := range result.OutputFiles {
		files = append(files, artifact.File{Path: file.Path, Bytes: file.Bytes})
	}
	if result.APIVersion == BuildResultAPIVersionV1Alpha5 {
		return verifyV1Alpha5ArtifactTree(result, files)
	}
	return artifact.Verify(files)
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
