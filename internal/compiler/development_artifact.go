package compiler

// Development artifact assembly.
//
// Layout (Development profile, separate from the frozen v1alpha5 artifact):
//
//	manifest.json
//	host-admission-requirements.json
//	skills/<skill>/compiled/procedure.json
//	skills/<skill>/compiled/ingress-links.json
//
// manifest.json is JCS, carries the exact normalized lock, every linked export
// with its package key, the complete file digest list and the fixed NOT_ADMITTED
// execution status. artifactDigest is the SHA-256 of the JCS manifest bytes. The
// manifest never embeds itself.
//
// The compiled Procedure/Ingress files are the same deterministic projections the
// frozen v1alpha5 artifact produces; the Development manifest wraps them with the
// linked interface identity. host-admission-requirements.json records only the
// external facts the Compiler cannot establish; it never substitutes for a static
// check.

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

type developmentArtifactManifest struct {
	APIVersion      string                       `json:"apiVersion"`
	LanguageTarget  string                       `json:"languageTarget"`
	Profile         string                       `json:"profile"`
	Generator       developmentManifestGenerator `json:"generator"`
	ExecutionStatus string                       `json:"executionStatus"`
	SourceDigest    string                       `json:"sourceDigest"`
	Targets         []string                     `json:"targets"`
	Lock            developmentManifestLock      `json:"lock"`
	Links           []DevelopmentLinkRecord      `json:"links"`
	Files           []developmentManifestFile    `json:"files"`
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
	requirementFile, err := canonicalizeValue(semantics.collector.requirements())
	if err != nil {
		return nil, nil, "", fmt.Errorf("Development host-admission requirements canonicalization: %w", err)
	}
	files = append(files, File{Path: developmentRequirementsFileName, Bytes: requirementFile})

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
	manifest := developmentArtifactManifest{
		APIVersion:      DevelopmentArtifactAPIVersion,
		LanguageTarget:  V1Alpha5Target,
		Profile:         DevelopmentProfile,
		Generator:       developmentManifestGenerator{Name: v1alpha5ArtifactGeneratorName, Version: CompilerVersion},
		ExecutionStatus: DevelopmentExecutionStatus,
		SourceDigest:    sourceDigest,
		Targets:         targets,
		Lock:            canonicalDevelopmentLock(env),
		Links:           links,
		Files:           manifestFiles,
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
	if value == developmentManifestFileName || value == developmentRequirementsFileName {
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
