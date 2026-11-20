package compiler

// Development Build API (Draft vertical slice).
//
// This file owns the independent Development Build contract family introduced by
// the next-generation toolchain proposal:
//
//	connectionagent.io/compiler-development-build-request/v1alpha1
//	connectionagent.io/compiler-development-build-result/v1alpha1
//	connectionagent.io/compiler-development-artifact/v1alpha1
//	connectionagent.io/interface-package/v1alpha1
//	connectionagent.io/interface-dependency-lock/v1alpha1
//
// The Development profile consumes Skill source, exact Interface Package
// snapshots (including the formal EvidenceDescriptor export kind) and a
// Resolver-produced Dependency Lock. It performs the same single Go Core
// language validation as the frozen v1alpha5 entry, then links the interface
// closure offline, records the external Host Admission facts and the Runtime
// Execution guarantees that no Compiler input can establish, and assembles a
// deterministic LINKED / BUILT / NOT_ADMITTED artifact. It never downloads,
// resolves, signs, admits or executes anything.
//
// The frozen v2.0.2 BuildRequest/BuildResult/Artifact contract is unchanged;
// the Development family is selected explicitly and rejects every other API
// identity.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const (
	// DevelopmentBuildRequestAPIVersion is the only accepted Development
	// BuildRequest.apiVersion.
	DevelopmentBuildRequestAPIVersion = "connectionagent.io/compiler-development-build-request/v1alpha1"
	// DevelopmentBuildResultAPIVersion is stamped on every Development build
	// result, including fail-closed results.
	DevelopmentBuildResultAPIVersion = "connectionagent.io/compiler-development-build-result/v1alpha1"
	// DevelopmentArtifactAPIVersion is the Development artifact manifest identity.
	DevelopmentArtifactAPIVersion = "connectionagent.io/compiler-development-artifact/v1alpha1"
	// InterfacePackageAPIVersion is the Interface Package manifest identity.
	InterfacePackageAPIVersion = "connectionagent.io/interface-package/v1alpha1"
	// InterfaceDependencyLockAPIVersion is the Dependency Lock identity.
	InterfaceDependencyLockAPIVersion = "connectionagent.io/interface-dependency-lock/v1alpha1"

	// DevelopmentProfile is the only accepted BuildRequest.profile value. A
	// production Host must reject it.
	DevelopmentProfile = "DEVELOPMENT"
	// DevelopmentExecutionStatus is the only executionStatus a Development
	// artifact can carry: the artifact is never admitted to execute by Build.
	DevelopmentExecutionStatus = "NOT_ADMITTED"

	DevelopmentSourceValidated = "VALIDATED"
	DevelopmentInterfaceLinked = "LINKED"
	DevelopmentArtifactBuilt   = "BUILT"
	DevelopmentPhaseFailed     = "FAILED"
	DevelopmentPhaseNotRun     = "NOT_RUN"
	// DevelopmentHostRequirementStatus is the only status of an external fact
	// that the Host must verify before dispatch. It is never a Compiler verdict
	// and never a substitute for a static check.
	DevelopmentHostRequirementStatus = "HOST_VERIFICATION_REQUIRED"
	// DevelopmentRuntimeRequirementStatus is the only status of an execution-time
	// guarantee. It cannot be satisfied by Build or by pre-dispatch admission.
	DevelopmentRuntimeRequirementStatus = "RUNTIME_EXECUTION_REQUIRED"

	DevelopmentRequirementOwnerTrust     = "OWNER_TRUST"
	DevelopmentRequirementEvidenceDomain = "EVIDENCE_DOMAIN_TRUST"
	DevelopmentRequirementEvidenceFact   = "EVIDENCE_FACT_TRUST"
	DevelopmentRequirementIndependence   = "EVIDENCE_INDEPENDENCE_TRUST"
	DevelopmentRequirementCommitProof    = "COMMIT_PROOF"
	DevelopmentRequirementReconcile      = "RECONCILE"

	// developmentRequirementDigestDomain is the single typed-key digest domain for
	// BOTH requirement classes. The domain is deliberately class-neutral: the
	// class is decided by kind, owning document and status together, never by a
	// digest prefix. A Host-specific domain would make Runtime Execution
	// requirements look like part of the Host Admission namespace.
	developmentRequirementDigestDomain     = "CA-SKILL-DEVELOPMENT-REQUIREMENT\x00v1\x00"
	developmentManifestFileName            = "manifest.json"
	developmentRequirementsFileName        = "host-admission-requirements.json"
	developmentRuntimeRequirementsFileName = "runtime-execution-requirements.json"

	// New Development-envelope diagnostics. Static language diagnostics keep
	// the frozen Appendix H identities.
	developmentRequestInvalid    = "DEVELOPMENT_REQUEST_INVALID"
	developmentSnapshotInvalid   = "DEVELOPMENT_SNAPSHOT_INVALID"
	developmentLockInvalid       = "DEVELOPMENT_LOCK_INVALID"
	developmentArtifactInvalid   = "DEVELOPMENT_ARTIFACT_INVALID"
	maxDevelopmentPackages       = 128
	maxDevelopmentExports        = 4096
	maxDevelopmentRequirements   = 4096
	maxDevelopmentVersionRunes   = 128
	maxDevelopmentPublisherRunes = 256
)

// DevelopmentPackageKey is the exact immutable Interface Package identity.
type DevelopmentPackageKey struct {
	PackageID      string `json:"packageId"`
	Version        string `json:"version"`
	SnapshotDigest string `json:"snapshotDigest"`
}

// DevelopmentSnapshot is one exact immutable Interface Package snapshot.
type DevelopmentSnapshot struct {
	Key   DevelopmentPackageKey `json:"key"`
	Files []File                `json:"files"`
}

// DevelopmentLockRoot binds one Skill Contract dependency alias to a package key.
type DevelopmentLockRoot struct {
	Alias string                `json:"alias"`
	Key   DevelopmentPackageKey `json:"key"`
}

// DevelopmentLockEdge is one direct dependency edge with its local alias.
type DevelopmentLockEdge struct {
	Alias string                `json:"alias"`
	Key   DevelopmentPackageKey `json:"key"`
}

// DevelopmentLockPackage records every direct dependency edge of one package.
type DevelopmentLockPackage struct {
	Key          DevelopmentPackageKey `json:"key"`
	Dependencies []DevelopmentLockEdge `json:"dependencies"`
}

// DevelopmentLock is the Resolver-produced dependency lock.
type DevelopmentLock struct {
	APIVersion string                   `json:"apiVersion"`
	Roots      []DevelopmentLockRoot    `json:"roots"`
	Packages   []DevelopmentLockPackage `json:"packages"`
}

// DevelopmentBuildRequest is the only accepted Development Build input.
type DevelopmentBuildRequest struct {
	APIVersion     string                `json:"apiVersion"`
	LanguageTarget string                `json:"languageTarget"`
	Profile        string                `json:"profile"`
	Files          []File                `json:"files"`
	Targets        []string              `json:"targets"`
	Lock           DevelopmentLock       `json:"lock"`
	Snapshots      []DevelopmentSnapshot `json:"snapshots"`
}

// DevelopmentBuildResult is the Development Build output. On failure the phase
// that failed carries FAILED and every later phase carries NOT_RUN; no output
// files and no artifact digest are returned.
type DevelopmentBuildResult struct {
	APIVersion          string       `json:"apiVersion"`
	Success             bool         `json:"success"`
	SourceStatus        string       `json:"sourceStatus"`
	InterfaceLinkStatus string       `json:"interfaceLinkStatus"`
	ArtifactStatus      string       `json:"artifactStatus"`
	ExecutionStatus     string       `json:"executionStatus"`
	Diagnostics         []Diagnostic `json:"diagnostics"`
	OutputFiles         []File       `json:"outputFiles"`
	ArtifactDigest      string       `json:"artifactDigest,omitempty"`
}

// DecodeDevelopmentBuildRequestJSON applies the strict Development request
// contract: an object, no duplicate keys, no unknown fields, no null for
// required fields, no null array elements and no trailing content.
func DecodeDevelopmentBuildRequestJSON(data []byte) (DevelopmentBuildRequest, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return DevelopmentBuildRequest{}, err
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return DevelopmentBuildRequest{}, err
	}
	if object == nil {
		return DevelopmentBuildRequest{}, fmt.Errorf("development BuildRequest must be a JSON object")
	}
	if err := rejectJSONNullProperties(object, "files", "targets", "lock", "snapshots"); err != nil {
		return DevelopmentBuildRequest{}, err
	}
	if err := rejectJSONNullArrayElements(object["snapshots"], "snapshots"); err != nil {
		return DevelopmentBuildRequest{}, err
	}
	if raw, ok := object["lock"]; ok {
		var lock map[string]json.RawMessage
		if err := json.Unmarshal(raw, &lock); err != nil {
			return DevelopmentBuildRequest{}, err
		}
		if lock == nil {
			return DevelopmentBuildRequest{}, fmt.Errorf("lock must be an object")
		}
		if err := rejectJSONNullProperties(lock, "apiVersion", "roots", "packages"); err != nil {
			return DevelopmentBuildRequest{}, err
		}
		if err := rejectJSONNullArrayElements(lock["roots"], "lock.roots"); err != nil {
			return DevelopmentBuildRequest{}, err
		}
		if err := rejectJSONNullArrayElements(lock["packages"], "lock.packages"); err != nil {
			return DevelopmentBuildRequest{}, err
		}
	}
	var req DevelopmentBuildRequest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return DevelopmentBuildRequest{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return DevelopmentBuildRequest{}, io.ErrUnexpectedEOF
		}
		return DevelopmentBuildRequest{}, err
	}
	return req, nil
}

// rejectDuplicateJSONKeys walks the whole JSON document and rejects any object
// that repeats a member name. encoding/json otherwise silently keeps the last
// value, which would let two callers disagree about the same request bytes.
func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return fmt.Errorf("JSON object member name must be a string")
				}
				if seen[key] {
					return fmt.Errorf("duplicate JSON object member %q", key)
				}
				seen[key] = true
				if err := walk(); err != nil {
					return err
				}
			}
			_, err := decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err := decoder.Token()
			return err
		}
		return nil
	}
	if err := walk(); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON content")
		}
		return err
	}
	return nil
}

func failedDevelopmentResult(diagnostics []Diagnostic, source, link, artifact string) DevelopmentBuildResult {
	sortDiagnostics(diagnostics)
	if diagnostics == nil {
		diagnostics = []Diagnostic{}
	}
	return DevelopmentBuildResult{
		APIVersion:          DevelopmentBuildResultAPIVersion,
		Success:             false,
		SourceStatus:        source,
		InterfaceLinkStatus: link,
		ArtifactStatus:      artifact,
		ExecutionStatus:     DevelopmentExecutionStatus,
		Diagnostics:         diagnostics,
		OutputFiles:         []File{},
	}
}

func developmentDiagnostic(code, path, message string) Diagnostic {
	return Diagnostic{
		Severity: "error",
		Phase:    "C_STATIC",
		Code:     code,
		Path:     limitDiagnosticPath(path),
		Message:  limitDiagnosticMessage(message),
	}
}

// developmentRequirementClass returns the artifact class of one requirement
// kind. The two classes are disjoint by construction: an external pre-dispatch
// fact can never migrate into the runtime document and an execution-time
// guarantee can never be satisfied by Host Admission.
func developmentRequirementClass(kind string) string {
	switch kind {
	case DevelopmentRequirementOwnerTrust, DevelopmentRequirementEvidenceDomain,
		DevelopmentRequirementEvidenceFact, DevelopmentRequirementIndependence:
		return "host"
	case DevelopmentRequirementCommitProof, DevelopmentRequirementReconcile:
		return "runtime"
	default:
		return ""
	}
}

func developmentRequirementStatus(kind string) string {
	switch developmentRequirementClass(kind) {
	case "host":
		return DevelopmentHostRequirementStatus
	case "runtime":
		return DevelopmentRuntimeRequirementStatus
	default:
		return ""
	}
}
