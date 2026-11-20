package compiler

// Development Build input admission.
//
// The Resolver hands the Compiler only exact bytes: Skill source, exact
// Interface Package snapshots, a Dependency Lock and optional Assurance Export
// files. This file proves those bytes are self-consistent before any linking
// runs: strict API identity, path/size/count limits, snapshot SHA-256 recompute,
// package.json manifest shape, declared file-set equality, export index identity,
// lock/snapshot equality, exact SemVer versions, closure completeness, acyclicity,
// same-Package-ID single-version and same-version single-digest rules, and
// Contract-alias-to-package-key agreement.
//
// It never downloads, searches, resolves a version or fills a missing edge.

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

const (
	developmentStageSource = 0
	developmentStageLink   = 1
	developmentStageBuild  = 2
)

type developmentPackage struct {
	key         DevelopmentPackageKey
	files       []File
	filesByName map[string][]byte
	manifest    developmentPackageManifest
	rawManifest []byte
}

type developmentPackageDependency struct {
	alias     string
	packageID string
	version   string
}

type developmentPackageFile struct {
	path   string
	sha256 string
}

type developmentArtifactBinding struct {
	artifactRef  string
	dependencyID string
	snapshotPath string
	dialect      string
}

type developmentSchemaBinding struct {
	exportID    string
	artifactRef string
	pointer     string
}

type developmentExport struct {
	kind             string
	id               string
	path             string
	artifactBindings []developmentArtifactBinding
	schemaBindings   []developmentSchemaBinding
}

type developmentPackageManifest struct {
	apiVersion     string
	packageID      string
	version        string
	languageTarget string
	publisher      string
	dependencies   []developmentPackageDependency
	exports        []developmentExport
	files          []developmentPackageFile
}

type developmentEnvironment struct {
	request         DevelopmentBuildRequest
	sourceFiles     []File
	targets         []string
	roots           []DevelopmentLockRoot
	rootByAlias     map[string]DevelopmentPackageKey
	packages        map[DevelopmentPackageKey]*developmentPackage
	packageOrder    []DevelopmentPackageKey
	assuranceByKey  map[DevelopmentPackageKey]*DevelopmentAssurance
	assuranceOrder  []*DevelopmentAssurance
	assuranceFiles  map[DevelopmentPackageKey]map[string][]byte
	dependencyGraph map[DevelopmentPackageKey]map[string]DevelopmentPackageKey
	reachable       map[DevelopmentPackageKey]bool
}

// validateDevelopmentRequest returns the admitted request scope and the stage at
// which admission failed. Diagnostics are sorted and never partial.
func validateDevelopmentRequest(req DevelopmentBuildRequest) (*developmentEnvironment, int, []Diagnostic) {
	var diagnostics []Diagnostic
	add := func(code, path, message string) {
		diagnostics = append(diagnostics, developmentDiagnostic(code, path, message))
	}
	env := &developmentEnvironment{
		request:         req,
		rootByAlias:     map[string]DevelopmentPackageKey{},
		packages:        map[DevelopmentPackageKey]*developmentPackage{},
		assuranceByKey:  map[DevelopmentPackageKey]*DevelopmentAssurance{},
		assuranceFiles:  map[DevelopmentPackageKey]map[string][]byte{},
		dependencyGraph: map[DevelopmentPackageKey]map[string]DevelopmentPackageKey{},
		reachable:       map[DevelopmentPackageKey]bool{},
	}

	if req.APIVersion != DevelopmentBuildRequestAPIVersion {
		add(developmentRequestInvalid, "", "BuildRequest.apiVersion must equal "+DevelopmentBuildRequestAPIVersion)
	}
	if req.LanguageTarget != V1Alpha5Target {
		add(developmentRequestInvalid, "", "BuildRequest.languageTarget must equal "+V1Alpha5Target)
	}
	if req.Profile != DevelopmentProfile {
		add(developmentRequestInvalid, "", "BuildRequest.profile must equal "+DevelopmentProfile)
	}
	if len(req.Targets) > maxSelectedTargets {
		add(developmentRequestInvalid, "", "BuildRequest.targets exceeds the selected-target count limit")
	}
	seenTarget := map[string]bool{}
	for _, target := range req.Targets {
		if target == "" || utf8RuneCount(target) > maxTargetRunes {
			add(developmentRequestInvalid, "", "Target names must contain 1 to 80 Unicode characters")
			continue
		}
		if seenTarget[target] {
			add(developmentRequestInvalid, "", "BuildRequest.targets entries must be unique: "+target)
		}
		seenTarget[target] = true
	}
	env.targets = append([]string(nil), req.Targets...)
	sort.Strings(env.targets)

	// Source files reuse the frozen limits and source-path grammar.
	sourceBytes := 0
	seenSource := map[string]bool{}
	seenFolded := map[string]bool{}
	skillNames := map[string]bool{}
	for _, file := range req.Files {
		if file.Bytes == nil {
			add(developmentRequestInvalid, file.Path, "File.bytes is required and must be a base64 string on the JSON wire")
		}
		if !validSourcePath(file.Path) {
			add(developmentRequestInvalid, file.Path, "File path must be a normalized UTF-8 POSIX source path rooted at skills/<skill-name>/")
			continue
		}
		if seenSource[file.Path] {
			add(developmentRequestInvalid, file.Path, "Duplicate file path within BuildRequest.files")
		} else if seenFolded[strings.ToLower(file.Path)] {
			add(developmentRequestInvalid, file.Path, "Case-folded file path collision within BuildRequest.files")
		}
		seenSource[file.Path] = true
		seenFolded[strings.ToLower(file.Path)] = true
		if len(file.Bytes) > maxFileSize {
			add(developmentRequestInvalid, file.Path, "File exceeds the 2 MiB raw byte limit")
		}
		sourceBytes += len(file.Bytes)
		parts := strings.Split(file.Path, "/")
		skillNames[parts[1]] = true
	}
	if len(req.Files) == 0 {
		add(developmentRequestInvalid, "", "BuildRequest.files must be a nonempty source file array")
	}
	for _, target := range env.targets {
		if len(req.Files) > 0 && !skillNames[target] {
			add(developmentRequestInvalid, "", "Selected Skill target does not exist in BuildRequest.files: "+target)
		}
	}
	env.sourceFiles = append([]File(nil), req.Files...)
	sort.Slice(env.sourceFiles, func(i, j int) bool { return env.sourceFiles[i].Path < env.sourceFiles[j].Path })

	if len(req.Snapshots) == 0 {
		add(developmentSnapshotInvalid, "", "BuildRequest.snapshots must be a nonempty exact snapshot array")
	}
	if len(req.Snapshots) > maxDevelopmentPackages {
		add(developmentSnapshotInvalid, "", "BuildRequest.snapshots exceeds the Interface Package count limit")
	}
	totalBytes := sourceBytes
	packageIDs := map[string]string{}  // packageId -> version
	digestByKey := map[string]string{} // packageId@version -> digest
	for index, snapshot := range req.Snapshots {
		label := "snapshots[" + strconv.Itoa(index) + "]"
		if snapshot.Files == nil {
			add(developmentSnapshotInvalid, label, "snapshot files must be an array")
			continue
		}
		key := snapshot.Key
		if !validStablePackageID(key.PackageID) {
			add(developmentSnapshotInvalid, label, "snapshot key.packageId must be an RFC 3986 absolute URI")
			continue
		}
		if !validDevelopmentVersion(key.Version) {
			add(developmentSnapshotInvalid, label, "snapshot key.version must be an exact SemVer version: "+key.Version)
			continue
		}
		if !validDigest(key.SnapshotDigest) {
			add(developmentSnapshotInvalid, label, "snapshot key.snapshotDigest must be sha256:<64 lowercase hex>")
			continue
		}
		if _, exists := env.packages[key]; exists {
			add(developmentSnapshotInvalid, label, "duplicate Interface Package snapshot key: "+key.PackageID+"@"+key.Version)
			continue
		}
		if existingVersion, exists := packageIDs[key.PackageID]; exists {
			if existingVersion != key.Version {
				add(developmentSnapshotInvalid, label, "the same Package ID is present with more than one version: "+key.PackageID+" "+existingVersion+" and "+key.Version)
				continue
			}
			if existingDigest := digestByKey[key.PackageID+"@"+key.Version]; existingDigest != key.SnapshotDigest {
				add(developmentSnapshotInvalid, label, "the same Package ID and version is present with different snapshot digests: "+key.PackageID+"@"+key.Version)
				continue
			}
			add(developmentSnapshotInvalid, label, "duplicate Interface Package snapshot key: "+key.PackageID+"@"+key.Version)
			continue
		}
		packageIDs[key.PackageID] = key.Version
		digestByKey[key.PackageID+"@"+key.Version] = key.SnapshotDigest

		files, fileBytes, filesOK := validateDevelopmentSnapshotFiles(key, snapshot.Files, add)
		env.packages[key] = &developmentPackage{key: key, files: files, filesByName: fileBytes}
		totalBytes += filesOK
	}
	if totalBytes > maxTotal {
		add(developmentRequestInvalid, "", "the complete Development Build request exceeds the 16 MiB aggregate raw byte limit")
	}

	// Optional Assurance Export input.
	if len(req.Assurance) > maxDevelopmentPackages {
		add(developmentAssuranceInvalid, "", "BuildRequest.assurance exceeds the Assurance Export count limit")
	}
	for index, assurance := range req.Assurance {
		label := "assurance[" + strconv.Itoa(index) + "]"
		if assurance.APIVersion != AssuranceExportAPIVersion {
			add(developmentAssuranceInvalid, label, "Assurance Export apiVersion must equal "+AssuranceExportAPIVersion)
		}
		if _, exists := env.packages[assurance.Key]; !exists {
			add(developmentAssuranceInvalid, label, "Assurance Export key does not match a supplied Interface Package snapshot")
		}
		if _, duplicate := env.assuranceByKey[assurance.Key]; duplicate {
			add(developmentAssuranceInvalid, label, "duplicate Assurance Export for one Interface Package key")
			continue
		}
		files, fileBytes, size := validateDevelopmentAssuranceFiles(assurance.Key, assurance.Files, add)
		if len(assurance.Files) == 0 {
			add(developmentAssuranceInvalid, label, "Assurance Export must contain at least one file")
			continue
		}
		measured, err := ComputeDependencySnapshotDigest(files)
		if err != nil || measured != assurance.Digest {
			add(developmentAssuranceInvalid, label, "Assurance Export digest does not match its original file bytes")
		}
		totalBytes += size
		copyAssurance := assurance
		env.assuranceByKey[assurance.Key] = &copyAssurance
		env.assuranceFiles[assurance.Key] = fileBytes
	}
	if totalBytes > maxTotal {
		add(developmentRequestInvalid, "", "the complete Development Build request exceeds the 16 MiB aggregate raw byte limit")
	}

	// Every snapshot must carry a valid package.json; the manifest is parsed and
	// its declared file list must equal the actual file set exactly.
	for _, key := range sortedDevelopmentKeys(env.packages) {
		pkg := env.packages[key]
		raw, exists := pkg.filesByName["package.json"]
		if !exists {
			add(developmentSnapshotInvalid, key.PackageID, "Interface Package snapshot must contain a root package.json")
			continue
		}
		pkg.rawManifest = raw
		manifest, problems := parseDevelopmentPackageManifest(raw, key)
		for _, problem := range problems {
			add(developmentSnapshotInvalid, key.PackageID+"/package.json", problem)
		}
		pkg.manifest = manifest
	}
	for _, key := range sortedDevelopmentKeys(env.packages) {
		pkg := env.packages[key]
		if pkg.rawManifest == nil {
			continue
		}
		if problems := validateDevelopmentPackageFiles(pkg); len(problems) > 0 {
			for _, problem := range problems {
				add(developmentSnapshotInvalid, key.PackageID, problem)
			}
		}
	}

	// Lock admission: identity, exact keys, roots, edges, closure and Contract
	// alias agreement.
	lock := req.Lock
	if lock.APIVersion != InterfaceDependencyLockAPIVersion {
		add(developmentLockInvalid, "lock", "lock.apiVersion must equal "+InterfaceDependencyLockAPIVersion)
	}
	if len(lock.Roots) == 0 {
		add(developmentLockInvalid, "lock.roots", "lock.roots must be nonempty")
	}
	if len(lock.Roots) > maxSelectedTargets {
		add(developmentLockInvalid, "lock.roots", "lock.roots exceeds the root alias count limit")
	}
	seenRoot := map[string]bool{}
	for _, root := range lock.Roots {
		if !validDependencyID(root.Alias) {
			add(developmentLockInvalid, "lock.roots", "root alias must satisfy the dependency Alias grammar: "+root.Alias)
			continue
		}
		if seenRoot[root.Alias] {
			add(developmentLockInvalid, "lock.roots", "duplicate lock root alias: "+root.Alias)
		}
		seenRoot[root.Alias] = true
		if _, exists := env.packages[root.Key]; !exists {
			add(developmentLockInvalid, "lock.roots", "lock root references an Interface Package snapshot that was not supplied: "+root.Alias)
			continue
		}
		env.roots = append(env.roots, root)
		env.rootByAlias[root.Alias] = root.Key
	}
	sort.Slice(env.roots, func(i, j int) bool { return env.roots[i].Alias < env.roots[j].Alias })

	lockKeys := map[DevelopmentPackageKey]bool{}
	seenPackageKey := map[DevelopmentPackageKey]bool{}
	for _, entry := range lock.Packages {
		if _, exists := env.packages[entry.Key]; !exists {
			add(developmentLockInvalid, "lock.packages", "lock references an Interface Package snapshot that was not supplied: "+entry.Key.PackageID)
			continue
		}
		if seenPackageKey[entry.Key] {
			add(developmentLockInvalid, "lock.packages", "duplicate lock package entry: "+entry.Key.PackageID)
			continue
		}
		seenPackageKey[entry.Key] = true
		lockKeys[entry.Key] = true
		edges := map[string]DevelopmentPackageKey{}
		for _, edge := range entry.Dependencies {
			if !validDependencyID(edge.Alias) {
				add(developmentLockInvalid, "lock.packages", "lock edge alias must satisfy the dependency Alias grammar: "+edge.Alias)
				continue
			}
			if _, duplicate := edges[edge.Alias]; duplicate {
				add(developmentLockInvalid, "lock.packages", "duplicate lock edge alias "+edge.Alias+" in package "+entry.Key.PackageID)
				continue
			}
			if _, exists := env.packages[edge.Key]; !exists {
				add(developmentLockInvalid, "lock.packages", "lock edge references an Interface Package snapshot that was not supplied: "+edge.Alias)
				continue
			}
			if edge.Key.PackageID != entry.Key.PackageID && !validStablePackageID(edge.Key.PackageID) {
				add(developmentLockInvalid, "lock.packages", "lock edge key.packageId must be an RFC 3986 absolute URI")
				continue
			}
			edges[edge.Alias] = edge.Key
		}
		env.dependencyGraph[entry.Key] = edges
	}
	// Every supplied snapshot must be referenced by the lock.
	for _, key := range sortedDevelopmentKeys(env.packages) {
		if !lockKeys[key] {
			add(developmentLockInvalid, "lock.packages", "supplied Interface Package snapshot is not referenced by the lock: "+key.PackageID+"@"+key.Version)
		}
	}
	// Manifest direct dependencies must equal the lock edges exactly.
	for _, key := range sortedDevelopmentKeys(env.packages) {
		pkg := env.packages[key]
		if pkg.rawManifest == nil {
			continue
		}
		edges, locked := env.dependencyGraph[key]
		if !locked {
			continue
		}
		for _, dependency := range pkg.manifest.dependencies {
			target, exists := edges[dependency.alias]
			if !exists {
				add(developmentLockInvalid, key.PackageID+"/package.json", "package manifest dependency has no matching lock edge: "+dependency.alias)
				continue
			}
			if target.PackageID != dependency.packageID || target.Version != dependency.version {
				add(developmentLockInvalid, key.PackageID+"/package.json", "lock edge identity differs from the package manifest dependency: "+dependency.alias)
			}
		}
		for _, alias := range sortedStringMapKeys(edges) {
			found := false
			for _, dependency := range pkg.manifest.dependencies {
				if dependency.alias == alias {
					found = true
					break
				}
			}
			if !found {
				add(developmentLockInvalid, key.PackageID+"/package.json", "lock edge has no matching declared package manifest dependency: "+alias)
			}
		}
	}
	// Roots must be reachable and the lock package set must equal the closure of
	// the roots; no extra package may be reachable from an unselected root.
	for _, root := range env.roots {
		env.reachable[root.Key] = true
	}
	queue := make([]DevelopmentPackageKey, 0, len(env.reachable))
	for key := range env.reachable {
		queue = append(queue, key)
	}
	for len(queue) > 0 {
		sort.Slice(queue, func(i, j int) bool { return developmentKeyString(queue[i]) < developmentKeyString(queue[j]) })
		key := queue[0]
		queue = queue[1:]
		for _, alias := range sortedStringMapKeys(env.dependencyGraph[key]) {
			target := env.dependencyGraph[key][alias]
			if env.reachable[target] {
				continue
			}
			env.reachable[target] = true
			queue = append(queue, target)
		}
	}
	for _, key := range sortedDevelopmentKeys(env.packages) {
		if !env.reachable[key] {
			add(developmentLockInvalid, "lock.packages", "lock package is not reachable from a root: "+key.PackageID)
		}
	}
	if cycle := developmentGraphCycle(env); cycle != "" {
		add(developmentLockInvalid, "lock.packages", "lock dependency graph contains a cycle: "+cycle)
	}
	// Contract dependency aliases must equal lock roots exactly. The Contract is
	// parsed with the shared AST, after which the language pipeline re-proves it;
	// here only exact root identity is established.
	if problems := validateDevelopmentRootsAgainstContracts(env); len(problems) > 0 {
		for _, problem := range problems {
			add(developmentLockInvalid, "", problem)
		}
	}
	// Export index identity is proven before linking so a malformed package
	// cannot contribute a partially valid import surface.
	for _, key := range sortedDevelopmentKeys(env.packages) {
		pkg := env.packages[key]
		if pkg.rawManifest == nil {
			continue
		}
		for _, problem := range validateDevelopmentPackageExports(pkg) {
			add(developmentSnapshotInvalid, key.PackageID+"/package.json", problem)
		}
	}

	if len(diagnostics) > 0 {
		sortDiagnostics(diagnostics)
		return nil, developmentStageLink, diagnostics
	}
	return env, developmentStageSource, nil
}

func sortedDevelopmentKeys(packages map[DevelopmentPackageKey]*developmentPackage) []DevelopmentPackageKey {
	keys := make([]DevelopmentPackageKey, 0, len(packages))
	for key := range packages {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return developmentKeyString(keys[i]) < developmentKeyString(keys[j]) })
	return keys
}

func developmentKeyString(key DevelopmentPackageKey) string {
	return key.PackageID + "\x00" + key.Version + "\x00" + key.SnapshotDigest
}

func validateDevelopmentSnapshotFiles(key DevelopmentPackageKey, files []File, add func(code, path, message string)) ([]File, map[string][]byte, int) {
	ordered := append([]File(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	byName := make(map[string][]byte, len(ordered))
	seen := map[string]bool{}
	seenFolded := map[string]bool{}
	total := 0
	for _, file := range ordered {
		if file.Bytes == nil {
			add(developmentSnapshotInvalid, key.PackageID+"/"+file.Path, "snapshot file bytes are required")
		}
		if !validRelativeSnapshotPath(file.Path) {
			add(developmentSnapshotInvalid, key.PackageID+"/"+file.Path, "snapshot file path must be a normalized UTF-8 POSIX relative path")
			continue
		}
		if seen[file.Path] {
			add(developmentSnapshotInvalid, key.PackageID+"/"+file.Path, "duplicate snapshot file path")
			continue
		} else if seenFolded[strings.ToLower(file.Path)] {
			add(developmentSnapshotInvalid, key.PackageID+"/"+file.Path, "case-folded snapshot file path collision")
			continue
		}
		seen[file.Path] = true
		seenFolded[strings.ToLower(file.Path)] = true
		if len(file.Bytes) > maxFileSize {
			add(developmentSnapshotInvalid, key.PackageID+"/"+file.Path, "snapshot file exceeds the 2 MiB raw byte limit")
		}
		total += len(file.Bytes)
		byName[file.Path] = file.Bytes
	}
	if len(ordered) > maxFiles {
		add(developmentSnapshotInvalid, key.PackageID, "snapshot exceeds the file count limit")
	}
	if _, exists := byName["package.json"]; !exists && len(ordered) > 0 {
		add(developmentSnapshotInvalid, key.PackageID, "Interface Package snapshot must contain a root package.json")
	}
	digest, err := ComputeDependencySnapshotDigest(ordered)
	if err != nil {
		add(developmentSnapshotInvalid, key.PackageID, "snapshot digest cannot be computed: "+err.Error())
	} else if digest != key.SnapshotDigest {
		add(developmentSnapshotInvalid, key.PackageID, "snapshot bytes do not match key.snapshotDigest")
	}
	return ordered, byName, total
}

// validateDevelopmentAssuranceFiles applies the shared snapshot file path/size
// grammar to one Assurance Export file set. Unlike an Interface Package snapshot
// it has no package.json and no key digest to compare against.
func validateDevelopmentAssuranceFiles(key DevelopmentPackageKey, files []File, add func(code, path, message string)) ([]File, map[string][]byte, int) {
	ordered := append([]File(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	byName := make(map[string][]byte, len(ordered))
	seen := map[string]bool{}
	seenFolded := map[string]bool{}
	total := 0
	for _, file := range ordered {
		if file.Bytes == nil {
			add(developmentAssuranceInvalid, key.PackageID+"/"+file.Path, "Assurance Export file bytes are required")
		}
		if !validRelativeSnapshotPath(file.Path) {
			add(developmentAssuranceInvalid, key.PackageID+"/"+file.Path, "Assurance Export file path must be a normalized UTF-8 POSIX relative path")
			continue
		}
		if seen[file.Path] {
			add(developmentAssuranceInvalid, key.PackageID+"/"+file.Path, "duplicate Assurance Export file path")
			continue
		} else if seenFolded[strings.ToLower(file.Path)] {
			add(developmentAssuranceInvalid, key.PackageID+"/"+file.Path, "case-folded Assurance Export file path collision")
			continue
		}
		seen[file.Path] = true
		seenFolded[strings.ToLower(file.Path)] = true
		if len(file.Bytes) > maxFileSize {
			add(developmentAssuranceInvalid, key.PackageID+"/"+file.Path, "Assurance Export file exceeds the 2 MiB raw byte limit")
		}
		total += len(file.Bytes)
		byName[file.Path] = file.Bytes
	}
	if len(ordered) > maxFiles {
		add(developmentAssuranceInvalid, key.PackageID, "Assurance Export exceeds the file count limit")
	}
	return ordered, byName, total
}

// validDevelopmentVersion accepts only an exact SemVer version, never a range,
// wildcard, tag or build metadata.
func validDevelopmentVersion(value string) bool {
	if value == "" || len([]rune(value)) > maxDevelopmentVersionRunes {
		return false
	}
	if strings.ContainsAny(value, "*^~<>=|, ") {
		return false
	}
	core := value
	if index := strings.IndexByte(core, '+'); index >= 0 {
		return false
	}
	if index := strings.IndexByte(core, '-'); index >= 0 {
		if index == 0 || index == len(core)-1 {
			return false
		}
		prerelease := core[index+1:]
		core = core[:index]
		for _, part := range strings.Split(prerelease, ".") {
			if part == "" || !validSemVerIdentifier(part, true) {
				return false
			}
		}
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" || len(part) > 1 && part[0] == '0' {
			return false
		}
		for i := 0; i < len(part); i++ {
			if part[i] < '0' || part[i] > '9' {
				return false
			}
		}
	}
	return true
}

func validSemVerIdentifier(value string, allowNumeric bool) bool {
	allDigits := true
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '-' {
			allDigits = false
			continue
		}
		if c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	if allowNumeric && allDigits && len(value) > 1 && value[0] == '0' {
		return false
	}
	return true
}

func parseDevelopmentPackageManifest(raw []byte, key DevelopmentPackageKey) (developmentPackageManifest, []string) {
	manifest := developmentPackageManifest{}
	var problems []string
	root, err := yaml.Parse(raw)
	if err != nil {
		return manifest, []string{"package.json is not valid JSON/YAML: " + err.Error()}
	}
	root = unalias(root)
	if root == nil || root.Kind != yaml.MappingNode {
		return manifest, []string{"package.json must be a JSON object"}
	}
	allowed := map[string]bool{
		"apiVersion": true, "packageId": true, "version": true, "languageTarget": true,
		"publisher": true, "dependencies": true, "exports": true, "files": true,
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		name := scalar(unalias(root.Content[i]))
		if !allowed[name] {
			problems = append(problems, "package.json contains an unsupported field: "+name)
		}
	}
	manifest.apiVersion = scalar(child(root, "apiVersion"))
	manifest.packageID = scalar(child(root, "packageId"))
	manifest.version = scalar(child(root, "version"))
	manifest.languageTarget = scalar(child(root, "languageTarget"))
	manifest.publisher = scalar(child(root, "publisher"))
	if manifest.apiVersion != InterfacePackageAPIVersion {
		problems = append(problems, "package.json apiVersion must equal "+InterfacePackageAPIVersion)
	}
	if manifest.packageID != key.PackageID {
		problems = append(problems, "package.json packageId must equal the snapshot key packageId")
	}
	if manifest.version != key.Version {
		problems = append(problems, "package.json version must equal the snapshot key version")
	}
	if manifest.languageTarget != V1Alpha5Target {
		problems = append(problems, "package.json languageTarget must equal "+V1Alpha5Target)
	}
	if manifest.publisher == "" || len([]rune(manifest.publisher)) > maxDevelopmentPublisherRunes {
		problems = append(problems, "package.json publisher must be a non-empty publisher identity")
	}
	dependencies := unalias(child(root, "dependencies"))
	if dependencies != nil {
		if dependencies.Kind != yaml.SequenceNode {
			problems = append(problems, "package.json dependencies must be an ordered list")
		} else {
			seenAlias := map[string]bool{}
			for index, node := range dependencies.Content {
				node = unalias(node)
				label := "dependencies[" + strconv.Itoa(index) + "]"
				if node == nil || node.Kind != yaml.MappingNode {
					problems = append(problems, label+" must be a mapping with alias/packageId/version")
					continue
				}
				dependency := developmentPackageDependency{
					alias:     scalar(child(node, "alias")),
					packageID: scalar(child(node, "packageId")),
					version:   scalar(child(node, "version")),
				}
				if !validDependencyID(dependency.alias) {
					problems = append(problems, label+".alias must satisfy the dependency Alias grammar")
				}
				if !validStablePackageID(dependency.packageID) {
					problems = append(problems, label+".packageId must be an RFC 3986 absolute URI")
				}
				if !validDevelopmentVersion(dependency.version) {
					problems = append(problems, label+".version must be an exact SemVer version")
				}
				if seenAlias[dependency.alias] {
					problems = append(problems, label+".alias is duplicated")
				}
				seenAlias[dependency.alias] = true
				manifest.dependencies = append(manifest.dependencies, dependency)
			}
		}
	}
	files := unalias(child(root, "files"))
	if files == nil {
		problems = append(problems, "package.json files must be a non-empty file digest list")
	} else if files.Kind != yaml.SequenceNode {
		problems = append(problems, "package.json files must be an ordered list")
	} else {
		seenPath := map[string]bool{}
		for index, node := range files.Content {
			node = unalias(node)
			label := "files[" + strconv.Itoa(index) + "]"
			if node == nil || node.Kind != yaml.MappingNode {
				problems = append(problems, label+" must be a mapping with path/sha256")
				continue
			}
			entry := developmentPackageFile{path: scalar(child(node, "path")), sha256: scalar(child(node, "sha256"))}
			if !validRelativeSnapshotPath(entry.path) {
				problems = append(problems, label+".path must be a normalized snapshot-relative path")
			}
			if !validDigest(entry.sha256) {
				problems = append(problems, label+".sha256 must be sha256:<64 lowercase hex>")
			}
			if seenPath[entry.path] {
				problems = append(problems, label+".path is duplicated")
			}
			seenPath[entry.path] = true
			manifest.files = append(manifest.files, entry)
		}
	}
	exports := unalias(child(root, "exports"))
	if exports == nil {
		problems = append(problems, "package.json exports must be an ordered list")
	} else if exports.Kind != yaml.SequenceNode {
		problems = append(problems, "package.json exports must be an ordered list")
	} else {
		for index, node := range exports.Content {
			export, exportProblems := parseDevelopmentExport(node, index)
			problems = append(problems, exportProblems...)
			manifest.exports = append(manifest.exports, export)
		}
	}
	return manifest, problems
}

func parseDevelopmentExport(node *yaml.Node, index int) (developmentExport, []string) {
	export := developmentExport{}
	var problems []string
	label := "exports[" + strconv.Itoa(index) + "]"
	node = unalias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return export, []string{label + " must be a mapping with kind/id/path"}
	}
	allowed := map[string]bool{"kind": true, "id": true, "path": true, "artifactBindings": true, "schemaBindings": true}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if name := scalar(unalias(node.Content[i])); !allowed[name] {
			problems = append(problems, label+" contains an unsupported field: "+name)
		}
	}
	export.kind = scalar(child(node, "kind"))
	export.id = scalar(child(node, "id"))
	export.path = scalar(child(node, "path"))
	switch export.kind {
	case "CallableCatalog":
		if !validExportID(export.id) {
			problems = append(problems, label+".id must be a Callable ExportId for CallableCatalog")
		}
	case "Schema":
		if !validStablePackageID(export.id) {
			problems = append(problems, label+".id must be an absolute Schema URI for Schema")
		}
	case "CapabilityCatalog":
		if !validExportID(export.id) {
			problems = append(problems, label+".id must be a CapabilityId for CapabilityCatalog")
		}
	case "IngressCatalog":
		if !validExportID(export.id) {
			problems = append(problems, label+".id must be an Ingress Operation ID for IngressCatalog")
		}
	default:
		problems = append(problems, label+".kind must be CallableCatalog, Schema, CapabilityCatalog or IngressCatalog")
	}
	if !validRelativeSnapshotPath(export.path) {
		problems = append(problems, label+".path must be a normalized snapshot-relative path")
	}
	if raw := unalias(child(node, "artifactBindings")); raw != nil {
		if raw.Kind != yaml.SequenceNode {
			problems = append(problems, label+".artifactBindings must be an ordered list")
		} else {
			for bindingIndex, bindingNode := range raw.Content {
				bindingNode = unalias(bindingNode)
				bindingLabel := label + ".artifactBindings[" + strconv.Itoa(bindingIndex) + "]"
				if bindingNode == nil || bindingNode.Kind != yaml.MappingNode {
					problems = append(problems, bindingLabel+" must be a mapping with artifactRef/dependencyId/snapshotPath/dialect")
					continue
				}
				binding := developmentArtifactBinding{
					artifactRef:  scalar(child(bindingNode, "artifactRef")),
					dependencyID: scalar(child(bindingNode, "dependencyId")),
					snapshotPath: scalar(child(bindingNode, "snapshotPath")),
					dialect:      scalar(child(bindingNode, "dialect")),
				}
				if binding.artifactRef == "" || binding.dependencyID == "" || !validRelativeSnapshotPath(binding.snapshotPath) || binding.dialect == "" {
					problems = append(problems, bindingLabel+" is incomplete")
				}
				if binding.dependencyID != "self" && !validDependencyID(binding.dependencyID) {
					problems = append(problems, bindingLabel+".dependencyId must be a direct dependency alias or self")
				}
				export.artifactBindings = append(export.artifactBindings, binding)
			}
		}
	}
	if raw := unalias(child(node, "schemaBindings")); raw != nil {
		if raw.Kind != yaml.SequenceNode {
			problems = append(problems, label+".schemaBindings must be an ordered list")
		} else {
			for bindingIndex, bindingNode := range raw.Content {
				bindingNode = unalias(bindingNode)
				bindingLabel := label + ".schemaBindings[" + strconv.Itoa(bindingIndex) + "]"
				if bindingNode == nil || bindingNode.Kind != yaml.MappingNode {
					problems = append(problems, bindingLabel+" must be a mapping with exportId/artifactRef/pointer")
					continue
				}
				binding := developmentSchemaBinding{
					exportID:    scalar(child(bindingNode, "exportId")),
					artifactRef: scalar(child(bindingNode, "artifactRef")),
					pointer:     scalar(child(bindingNode, "pointer")),
				}
				if !validStablePackageID(binding.exportID) || binding.artifactRef == "" {
					problems = append(problems, bindingLabel+" is incomplete")
				}
				if binding.pointer != "" && binding.pointer != "#" && !strings.HasPrefix(binding.pointer, "#/") {
					problems = append(problems, bindingLabel+".pointer must be '#' or a JSON Pointer fragment")
				}
				export.schemaBindings = append(export.schemaBindings, binding)
			}
		}
	}
	return export, problems
}

// validateDevelopmentPackageFiles proves the manifest file list is exactly the
// actual snapshot file set (package.json excluded) and that every digest matches
// the original bytes.
func validateDevelopmentPackageFiles(pkg *developmentPackage) []string {
	var problems []string
	declared := map[string]string{}
	for _, entry := range pkg.manifest.files {
		if entry.path == "package.json" {
			problems = append(problems, "package.json must not list itself in files")
			continue
		}
		declared[entry.path] = entry.sha256
	}
	for _, path := range sortedStringMapKeys(declared) {
		raw, exists := pkg.filesByName[path]
		if !exists {
			problems = append(problems, "package.json files[] lists a missing snapshot file: "+path)
			continue
		}
		sum := sha256.Sum256(raw)
		if declared[path] != "sha256:"+hex.EncodeToString(sum[:]) {
			problems = append(problems, "package.json files[] sha256 differs from the actual snapshot bytes: "+path)
		}
	}
	for _, path := range sortedStringMapKeys(pkg.filesByName) {
		if path == "package.json" {
			continue
		}
		if _, exists := declared[path]; !exists {
			problems = append(problems, "snapshot file is not listed in package.json files[]: "+path)
		}
	}
	return problems
}

// schemaBindingPath resolves one declared schema export identity to its snapshot
// artifact path through the package's own artifactBindings/schemaBindings. It is
// used for canonical artifact-root Schema exports whose file-level $id is the
// artifact root identity, not the package export ID.
func (pkg *developmentPackage) schemaBindingPath(exportID string) (string, bool) {
	artifactPaths := map[string]string{}
	for _, export := range pkg.manifest.exports {
		for _, artifact := range export.artifactBindings {
			artifactPaths[artifact.artifactRef] = artifact.snapshotPath
		}
	}
	for _, export := range pkg.manifest.exports {
		for _, binding := range export.schemaBindings {
			if binding.exportID != exportID {
				continue
			}
			if path, exists := artifactPaths[binding.artifactRef]; exists {
				return path, true
			}
		}
	}
	return "", false
}

// validateDevelopmentPackageExports proves that every declared export exists at
// its declared path with the declared identity and that no export identity is
// declared twice.
func validateDevelopmentPackageExports(pkg *developmentPackage) []string {
	var problems []string
	seen := map[string]bool{}
	for _, export := range pkg.manifest.exports {
		key := export.kind + "\x00" + export.id
		if seen[key] {
			problems = append(problems, "duplicate export declaration: "+export.kind+" "+export.id)
			continue
		}
		seen[key] = true
		raw, exists := pkg.filesByName[export.path]
		if !exists {
			problems = append(problems, "declared export path does not exist in the snapshot: "+export.path)
			continue
		}
		switch export.kind {
		case "CallableCatalog":
			root, err := yaml.Parse(raw)
			if err != nil {
				problems = append(problems, "CallableCatalog export is not valid YAML/JSON: "+export.id)
				continue
			}
			catalog, ok := parseToolCatalog(root, export.path)
			if !ok {
				problems = append(problems, "CallableCatalog export does not contain a supported ToolCatalog document: "+export.id)
				continue
			}
			matches := 0
			for _, callable := range catalog.callables {
				if callable.ExportID == export.id {
					matches++
				}
			}
			if matches != 1 {
				problems = append(problems, "CallableCatalog export identity must resolve exactly once in its file: "+export.id)
			}
		case "Schema":
			root, err := yaml.Parse(raw)
			if err != nil {
				problems = append(problems, "Schema export is not valid YAML/JSON: "+export.id)
				continue
			}
			record, ok := parseSchemaExport(root, export.path)
			if !ok || record.ExportID != export.id {
				if boundPath, bound := pkg.schemaBindingPath(export.id); !bound || boundPath != export.path {
					problems = append(problems, "Schema export path does not carry the declared $id: "+export.id)
				}
			}
		case "CapabilityCatalog":
			root, err := yaml.Parse(raw)
			if err != nil {
				problems = append(problems, "CapabilityCatalog export is not valid YAML/JSON: "+export.id)
				continue
			}
			capabilities, capabilityProblems := parseDevelopmentCapabilityDocument(root, export.path)
			problems = append(problems, capabilityProblems...)
			matches := 0
			for _, capability := range capabilities {
				if capability.id == export.id {
					matches++
				}
			}
			if matches != 1 {
				problems = append(problems, "CapabilityCatalog export identity must resolve exactly once in its file: "+export.id)
			}
		case "IngressCatalog":
			root, err := yaml.Parse(raw)
			if err != nil {
				problems = append(problems, "IngressCatalog export is not valid YAML/JSON: "+export.id)
				continue
			}
			if scalar(child(root, "apiVersion")) != "connectionagent.io/ingresscatalog/v1alpha1" {
				problems = append(problems, "IngressCatalog export does not carry the frozen apiVersion: "+export.id)
			}
		}
	}
	return problems
}

// validateDevelopmentRootsAgainstContracts proves every Contract dependency
// alias is a lock root whose package key matches the declared Stable Package ID.
func validateDevelopmentRootsAgainstContracts(env *developmentEnvironment) []string {
	var problems []string
	skillNames := map[string]bool{}
	for _, file := range env.sourceFiles {
		parts := strings.Split(file.Path, "/")
		if len(parts) >= 2 && parts[0] == "skills" {
			skillNames[parts[1]] = true
		}
	}
	selected := map[string]bool{}
	for _, name := range sortedStringMapKeys(skillNames) {
		if len(env.targets) == 0 {
			selected[name] = true
		}
	}
	for _, target := range env.targets {
		selected[target] = true
	}
	for _, name := range sortedStringMapKeys(selected) {
		root, diagnostics := parseDevelopmentSkillContract(name, env.sourceFiles)
		if root == nil || len(diagnostics) > 0 {
			// Language parsing reports the exact source diagnostic later; root
			// agreement is only provable for a readable Contract.
			continue
		}
		declared := map[string]string{}
		dependencies := unalias(child(root, "dependencies"))
		if dependencies == nil {
			continue
		}
		if dependencies.Kind != yaml.MappingNode {
			problems = append(problems, "Skill "+name+" Contract.dependencies must be a mapping")
			continue
		}
		for i := 0; i+1 < len(dependencies.Content); i += 2 {
			alias := scalar(unalias(dependencies.Content[i]))
			value := scalar(unalias(dependencies.Content[i+1]))
			if alias == "" || value == "" {
				continue
			}
			declared[alias] = value
		}
		for _, alias := range sortedStringMapKeys(declared) {
			key, exists := env.rootByAlias[alias]
			if !exists {
				problems = append(problems, "Skill "+name+" Contract dependency alias has no lock root: "+alias)
				continue
			}
			if key.PackageID != declared[alias] {
				problems = append(problems, "Skill "+name+" Contract dependency Stable Package ID differs from the lock root packageId: "+alias)
			}
		}
		for _, root := range env.roots {
			if _, exists := declared[root.Alias]; !exists {
				problems = append(problems, "lock root alias is not declared by any selected Skill Contract: "+root.Alias)
			}
		}
	}
	return problems
}

func parseDevelopmentSkillContract(name string, files []File) (*yaml.Node, []Diagnostic) {
	path := "skills/" + name + "/assurance/contract.yaml"
	var raw []byte
	for _, file := range files {
		if file.Path == path {
			raw = file.Bytes
			break
		}
	}
	if raw == nil {
		return nil, []Diagnostic{developmentDiagnostic(developmentRequestInvalid, path, "Skill Contract is missing")}
	}
	root, err := yaml.Parse(raw)
	if err != nil {
		return nil, []Diagnostic{developmentDiagnostic(developmentRequestInvalid, path, "Skill Contract is not valid YAML")}
	}
	return root, nil
}

// developmentGraphCycle returns a deterministic description of one cycle, or "".
func developmentGraphCycle(env *developmentEnvironment) string {
	state := map[DevelopmentPackageKey]int{} // 0 unvisited, 1 visiting, 2 done
	var stack []DevelopmentPackageKey
	var cycle string
	var visit func(DevelopmentPackageKey)
	visit = func(key DevelopmentPackageKey) {
		if cycle != "" {
			return
		}
		state[key] = 1
		stack = append(stack, key)
		for _, alias := range sortedStringMapKeys(env.dependencyGraph[key]) {
			target := env.dependencyGraph[key][alias]
			switch state[target] {
			case 1:
				parts := make([]string, 0, len(stack))
				start := 0
				for i, entry := range stack {
					if entry == target {
						start = i
						break
					}
				}
				for _, entry := range stack[start:] {
					parts = append(parts, entry.PackageID+"@"+entry.Version)
				}
				parts = append(parts, target.PackageID+"@"+target.Version)
				cycle = strings.Join(parts, " -> ")
			case 0:
				visit(target)
			}
			if cycle != "" {
				return
			}
		}
		stack = stack[:len(stack)-1]
		state[key] = 2
	}
	for _, key := range sortedDevelopmentKeys(env.packages) {
		if state[key] == 0 {
			visit(key)
		}
	}
	return cycle
}
