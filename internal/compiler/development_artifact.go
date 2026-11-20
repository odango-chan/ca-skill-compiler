package compiler

// Development artifact assembly.
//
// Layout (Development profile, separate from the frozen v1alpha5 artifact):
//
//	manifest.json
//	host-admission-requirements.json
//	runtime-execution-requirements.json
//	skills/<skill>/compiled/procedure.json
//	skills/<skill>/compiled/ingress-links.json
//
// manifest.json is JCS, carries the exact normalized lock, every linked export
// with its package key, the complete file digest list, the two requirement
// document identities and the fixed NOT_ADMITTED execution status. It also
// carries the Skill's static Capability requirements: the exact Capability the
// Skill selected, its CapabilityCatalog Package/Owner/export/SourcePath, every
// Callable member and the Skill declaration position. Those records are static
// requirements, never Deployment Grants.
// artifactDigest is the SHA-256 of the JCS manifest bytes. The manifest never
// embeds itself.
//
// The compiled Procedure/Ingress files are the same deterministic projections the
// frozen v1alpha5 artifact produces; the Development manifest wraps them with the
// linked interface identity. host-admission-requirements.json records only the
// external facts the Compiler cannot establish before dispatch.
// runtime-execution-requirements.json records only the execution-time guarantees
// (actual Commit Proof and actual Reconcile) that Build and Host Admission can
// never satisfy. Neither document substitutes for a static check.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

type developmentManifestGenerator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type developmentManifestLockEdge struct {
	Alias string                `json:"alias"`
	Key   DevelopmentPackageKey `json:"key"`
}

type developmentManifestLockPackage struct {
	Key          DevelopmentPackageKey         `json:"key"`
	Dependencies []developmentManifestLockEdge `json:"dependencies"`
}

type developmentManifestLock struct {
	APIVersion string                           `json:"apiVersion"`
	Roots      []DevelopmentLockRoot            `json:"roots"`
	Packages   []developmentManifestLockPackage `json:"packages"`
}

type developmentManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// developmentManifestRequirementDocuments names the two disjoint requirement
// artifacts. Host Admission facts and Runtime Execution guarantees are separate
// contract surfaces: neither file may carry the other class.
type developmentManifestRequirementDocuments struct {
	HostAdmission    string `json:"hostAdmission"`
	RuntimeExecution string `json:"runtimeExecution"`
}

// DevelopmentSourceDeclaration is one exact Skill source position: the author
// file and the JSON Pointer inside it. It is never a "somewhere in the file"
// approximation.
type DevelopmentSourceDeclaration struct {
	Path    string `json:"path"`
	Pointer string `json:"pointer"`
}

// DevelopmentCapabilityCatalogIdentity is the exact CapabilityCatalog export a
// Skill's static Capability requirement was linked against. Owner is the
// publisher declared by the locked Package manifest, never an Owner that the
// Catalog document asserts about itself.
type DevelopmentCapabilityCatalogIdentity struct {
	Key        DevelopmentPackageKey `json:"key"`
	Owner      string                `json:"owner"`
	ExportID   string                `json:"exportId"`
	SourcePath string                `json:"sourcePath"`
	Pointer    string                `json:"pointer"`
}

// DevelopmentCapabilityCallable is one Callable member of the selected
// Capability with its exact Interface Package and export identity.
type DevelopmentCapabilityCallable struct {
	Key        DevelopmentPackageKey `json:"key"`
	Owner      string                `json:"owner"`
	ExportID   string                `json:"exportId"`
	Kind       string                `json:"kind"`
	Effect     string                `json:"effect"`
	SourcePath string                `json:"sourcePath"`
}

// DevelopmentCapabilityRequirement is one Capability the Skill statically
// selects. It records the exact Capability identity rather than a Callable set,
// because several Capabilities may contain the same Callable: only the declared
// Capability is the Skill's authorization requirement. It is a static
// requirement, never a Deployment Grant; it says nothing about what a Host has
// actually authorized.
type DevelopmentCapabilityRequirement struct {
	Skill        string                               `json:"skill"`
	CapabilityID string                               `json:"capabilityId"`
	Catalog      DevelopmentCapabilityCatalogIdentity `json:"catalog"`
	Callables    []DevelopmentCapabilityCallable      `json:"callables"`
	Declaration  DevelopmentSourceDeclaration         `json:"declaration"`
}

// sortedDevelopmentCapabilityRequirements produces the deterministic manifest
// order and rejects duplicate static requirements.
func sortedDevelopmentCapabilityRequirements(requirements []DevelopmentCapabilityRequirement) []DevelopmentCapabilityRequirement {
	seen := map[string]DevelopmentCapabilityRequirement{}
	for _, requirement := range requirements {
		callables := make([]DevelopmentCapabilityCallable, 0, len(requirement.Callables))
		seenCallable := map[string]bool{}
		for _, callable := range requirement.Callables {
			canonical, err := canonicalizeValue(callable)
			if err != nil || seenCallable[string(canonical)] {
				continue
			}
			seenCallable[string(canonical)] = true
			callables = append(callables, callable)
		}
		sort.Slice(callables, func(i, j int) bool {
			if callables[i].ExportID != callables[j].ExportID {
				return callables[i].ExportID < callables[j].ExportID
			}
			if developmentKeyString(callables[i].Key) != developmentKeyString(callables[j].Key) {
				return developmentKeyString(callables[i].Key) < developmentKeyString(callables[j].Key)
			}
			return callables[i].Kind < callables[j].Kind
		})
		requirement.Callables = callables
		if requirement.Callables == nil {
			requirement.Callables = []DevelopmentCapabilityCallable{}
		}
		canonical, err := canonicalizeValue(requirement)
		if err != nil {
			continue
		}
		seen[string(canonical)] = requirement
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	sorted := make([]DevelopmentCapabilityRequirement, 0, len(keys))
	for _, key := range keys {
		sorted = append(sorted, seen[key])
	}
	return sorted
}

type developmentArtifactManifest struct {
	APIVersion             string                                  `json:"apiVersion"`
	LanguageTarget         string                                  `json:"languageTarget"`
	Profile                string                                  `json:"profile"`
	Generator              developmentManifestGenerator            `json:"generator"`
	ExecutionStatus        string                                  `json:"executionStatus"`
	SourceDigest           string                                  `json:"sourceDigest"`
	Targets                []string                                `json:"targets"`
	Lock                   developmentManifestLock                 `json:"lock"`
	Links                  []DevelopmentLinkRecord                 `json:"links"`
	CapabilityRequirements []DevelopmentCapabilityRequirement      `json:"capabilityRequirements"`
	Requirements           developmentManifestRequirementDocuments `json:"requirements"`
	Files                  []developmentManifestFile               `json:"files"`
}

func canonicalDevelopmentLock(env *developmentEnvironment) developmentManifestLock {
	lock := developmentManifestLock{
		APIVersion: InterfaceDependencyLockAPIVersion,
		Roots:      make([]DevelopmentLockRoot, 0, len(env.roots)),
		Packages:   make([]developmentManifestLockPackage, 0, len(env.packages)),
	}
	for _, root := range env.roots {
		lock.Roots = append(lock.Roots, root)
	}
	sort.Slice(lock.Roots, func(i, j int) bool { return lock.Roots[i].Alias < lock.Roots[j].Alias })
	for _, key := range sortedDevelopmentKeys(env.packages) {
		if !env.reachable[key] {
			continue
		}
		entry := developmentManifestLockPackage{Key: key, Dependencies: []developmentManifestLockEdge{}}
		for _, alias := range sortedStringMapKeys(env.dependencyGraph[key]) {
			entry.Dependencies = append(entry.Dependencies, developmentManifestLockEdge{Alias: alias, Key: env.dependencyGraph[key][alias]})
		}
		lock.Packages = append(lock.Packages, entry)
	}
	return lock
}

func sortedDevelopmentLinkRecords(records []DevelopmentLinkRecord) []DevelopmentLinkRecord {
	seen := map[string]DevelopmentLinkRecord{}
	for _, record := range records {
		record.SchemaIDs = dedupeSortedStrings(record.SchemaIDs)
		if record.SchemaIDs == nil {
			record.SchemaIDs = []string{}
		}
		canonical, err := canonicalizeValue(record)
		if err != nil {
			continue
		}
		seen[string(canonical)] = record
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	records = make([]DevelopmentLinkRecord, 0, len(keys))
	for _, key := range keys {
		records = append(records, seen[key])
	}
	return records
}

// assembleDevelopmentArtifact builds the deterministic Development artifact tree.
// Any inconsistency fails the whole build atomically.
func assembleDevelopmentArtifact(env *developmentEnvironment, semantics *developmentSemantics, skills []v1alpha5ArtifactSkill) ([]File, []byte, string, error) {
	legacyFiles, _, err := buildV1Alpha5ArtifactFiles(skills)
	if err != nil {
		return nil, nil, "", err
	}
	files := make([]File, 0, len(legacyFiles)+2)
	for _, file := range legacyFiles {
		if file.path == V1Alpha5ArtifactManifestPath {
			continue
		}
		files = append(files, File{Path: file.path, Bytes: file.bytes})
	}
	sourceDigest, err := v1alpha5SourceSetDigest(env.sourceFiles)
	if err != nil {
		return nil, nil, "", fmt.Errorf("Development source digest: %w", err)
	}
	hostRequirements, err := canonicalizeValue(semantics.collector.hostRequirements())
	if err != nil {
		return nil, nil, "", fmt.Errorf("Development host-admission requirements canonicalization: %w", err)
	}
	files = append(files, File{Path: developmentRequirementsFileName, Bytes: hostRequirements})
	runtimeRequirements, err := canonicalizeValue(semantics.collector.runtimeRequirements())
	if err != nil {
		return nil, nil, "", fmt.Errorf("Development runtime-execution requirements canonicalization: %w", err)
	}
	files = append(files, File{Path: developmentRuntimeRequirementsFileName, Bytes: runtimeRequirements})

	manifestFiles := make([]developmentManifestFile, 0, len(files))
	for _, file := range files {
		sum := sha256.Sum256(file.Bytes)
		manifestFiles = append(manifestFiles, developmentManifestFile{Path: file.Path, SHA256: "sha256:" + hex.EncodeToString(sum[:])})
	}
	sort.Slice(manifestFiles, func(i, j int) bool { return manifestFiles[i].Path < manifestFiles[j].Path })
	targets := append([]string(nil), env.targets...)
	if targets == nil {
		targets = []string{}
	}
	links := sortedDevelopmentLinkRecords(semantics.linkRecords)
	if links == nil {
		links = []DevelopmentLinkRecord{}
	}
	capabilities := sortedDevelopmentCapabilityRequirements(semantics.capabilityRequirements)
	if capabilities == nil {
		capabilities = []DevelopmentCapabilityRequirement{}
	}
	manifest := developmentArtifactManifest{
		APIVersion:             DevelopmentArtifactAPIVersion,
		LanguageTarget:         V1Alpha5Target,
		Profile:                DevelopmentProfile,
		Generator:              developmentManifestGenerator{Name: v1alpha5ArtifactGeneratorName, Version: CompilerVersion},
		ExecutionStatus:        DevelopmentExecutionStatus,
		SourceDigest:           sourceDigest,
		Targets:                targets,
		Lock:                   canonicalDevelopmentLock(env),
		Links:                  links,
		CapabilityRequirements: capabilities,
		Requirements: developmentManifestRequirementDocuments{
			HostAdmission:    developmentRequirementsFileName,
			RuntimeExecution: developmentRuntimeRequirementsFileName,
		},
		Files: manifestFiles,
	}
	manifestBytes, err := canonicalizeValue(manifest)
	if err != nil {
		return nil, nil, "", fmt.Errorf("Development manifest canonicalization: %w", err)
	}
	files = append(files, File{Path: developmentManifestFileName, Bytes: manifestBytes})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	for _, file := range files {
		if !validDevelopmentArtifactPath(file.Path) {
			return nil, nil, "", fmt.Errorf("unsafe Development artifact path: %s", file.Path)
		}
	}
	sum := sha256.Sum256(manifestBytes)
	return files, manifestBytes, "sha256:" + hex.EncodeToString(sum[:]), nil
}

func validDevelopmentArtifactPath(value string) bool {
	if value == developmentManifestFileName || value == developmentRequirementsFileName || value == developmentRuntimeRequirementsFileName {
		return true
	}
	if value == "" || len(value) > 240 || !strings.HasPrefix(value, "skills/") ||
		strings.ContainsAny(value, "\\:\x00") || strings.Contains(value, "//") ||
		strings.Contains(value, "../") || strings.HasSuffix(value, "/") {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}
