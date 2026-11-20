package compiler

// Development Build public entry points.
//
// BuildDevelopment is the only Development build path. It admits the exact
// request, links the offline interface closure, runs the single Go Core static
// pipeline, records typed Host Admission requirements and assembles a deterministic
// LINKED / BUILT / NOT_ADMITTED artifact. A failed build never returns partial
// output files.
//
// VerifyDevelopment requires the original request and the actual artifact bytes.
// It verifies the artifact in isolation (manifest/file digests, requirement
// identity, execution status) and then rebuilds the artifact from the original
// request with the same generator and compares every byte. Recomputing an outer
// digest after deleting a requirement or editing a Procedure therefore fails.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// BuildDevelopment compiles one Development BuildRequest.
func BuildDevelopment(req DevelopmentBuildRequest) DevelopmentBuildResult {
	return buildDevelopmentInternal(req)
}

func buildDevelopmentInternal(req DevelopmentBuildRequest) DevelopmentBuildResult {
	env, _, diagnostics := validateDevelopmentRequest(req)
	if len(diagnostics) > 0 {
		if developmentSourceFailure(diagnostics) {
			return failedDevelopmentResult(diagnostics, DevelopmentPhaseFailed, DevelopmentPhaseNotRun, DevelopmentPhaseNotRun)
		}
		return failedDevelopmentResult(diagnostics, DevelopmentSourceValidated, DevelopmentPhaseFailed, DevelopmentPhaseNotRun)
	}
	scope, scopeDiagnostics := buildDevelopmentLinkScope(env)
	if len(scopeDiagnostics) > 0 {
		return failedDevelopmentResult(scopeDiagnostics, DevelopmentSourceValidated, DevelopmentPhaseFailed, DevelopmentPhaseNotRun)
	}
	legacy, synthetic := env.legacyRequest()
	collector := newDevelopmentRequirementCollector()
	semantics := newDevelopmentSemantics(env, scope, synthetic, collector)
	// Every inherited pass shares the Development Package-scoped Schema
	// resolver through the private request field; the frozen entry never sets it.
	legacy.development = semantics
	diagnostics = checkSourcesWithDevelopment(legacy, semantics)
	if hasErrorDiagnostic(diagnostics) {
		if developmentSourceFailure(diagnostics) {
			return failedDevelopmentResult(diagnostics, DevelopmentPhaseFailed, DevelopmentPhaseNotRun, DevelopmentPhaseNotRun)
		}
		return failedDevelopmentResult(diagnostics, DevelopmentSourceValidated, DevelopmentPhaseFailed, DevelopmentPhaseNotRun)
	}
	skills, err := collectV1Alpha5ArtifactSkillsWithDevelopment(legacy, semantics)
	if err != nil {
		diagnostics = append(diagnostics, developmentDiagnostic(developmentArtifactInvalid, "", "Development artifact assembly failed: "+err.Error()))
		return failedDevelopmentResult(diagnostics, DevelopmentSourceValidated, DevelopmentInterfaceLinked, DevelopmentPhaseFailed)
	}
	files, _, digest, err := assembleDevelopmentArtifact(env, semantics, skills)
	if err != nil {
		diagnostics = append(diagnostics, developmentDiagnostic(developmentArtifactInvalid, "", "Development artifact assembly failed: "+err.Error()))
		return failedDevelopmentResult(diagnostics, DevelopmentSourceValidated, DevelopmentInterfaceLinked, DevelopmentPhaseFailed)
	}
	result := DevelopmentBuildResult{
		APIVersion:          DevelopmentBuildResultAPIVersion,
		Success:             true,
		SourceStatus:        DevelopmentSourceValidated,
		InterfaceLinkStatus: DevelopmentInterfaceLinked,
		ArtifactStatus:      DevelopmentArtifactBuilt,
		ExecutionStatus:     DevelopmentExecutionStatus,
		Diagnostics:         []Diagnostic{},
		OutputFiles:         files,
		ArtifactDigest:      digest,
	}
	if err := verifyDevelopmentArtifactFiles(result); err != nil {
		diagnostics = append(diagnostics, developmentDiagnostic(developmentArtifactInvalid, "", "Development artifact self-verification failed: "+err.Error()))
		return failedDevelopmentResult(diagnostics, DevelopmentSourceValidated, DevelopmentInterfaceLinked, DevelopmentPhaseFailed)
	}
	return result
}

// developmentSourceFailure reports whether the diagnostics belong to the source
// language stage rather than the Development interface-link stage.
func developmentSourceFailure(diagnostics []Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != "error" {
			continue
		}
		if developmentInterfaceCode(diagnostic.Code) {
			return false
		}
	}
	return true
}

func developmentInterfaceCode(code string) bool {
	if strings.HasPrefix(code, "DEVELOPMENT_") {
		return true
	}
	switch code {
	case "EXPORT_ID_MISSING", "EXPORT_ID_AMBIGUOUS", "SCHEMA_MISSING", "EFFECT_MISMATCH",
		"DEPENDENCY_ID_MISMATCH", "DEPENDENCY_IDENTITY_UNVERIFIED", "DEPENDENCY_OWNER_MISMATCH",
		"DEPENDENCY_EXPORT_OWNER_UNVERIFIED", "CAPABILITY_MAPPING_MISSING", "CAPABILITY_NOT_GRANTED",
		"CAPABILITY_SCOPE_UNRESOLVED", "CAPABILITY_DEFINITION_CONFLICT", "CAPABILITY_MEMBERSHIP_DUPLICATE",
		"EVIDENCE_GUARANTEE_LINK_INVALID", "RECONCILE_CONTRACT_INVALID":
		return true
	}
	return false
}

// VerifyDevelopment verifies a successful Development build result against the
// original request. It verifies the actual artifact bytes first, then rebuilds
// the artifact with the same generator and compares every byte.
func VerifyDevelopment(req DevelopmentBuildRequest, result DevelopmentBuildResult) error {
	if result.APIVersion != DevelopmentBuildResultAPIVersion {
		return fmt.Errorf("unsupported Development BuildResult.apiVersion %q: expected %s", result.APIVersion, DevelopmentBuildResultAPIVersion)
	}
	if !result.Success {
		return fmt.Errorf("cannot verify a failed Development BuildResult")
	}
	if err := verifyDevelopmentArtifactFiles(result); err != nil {
		return err
	}
	rebuilt := buildDevelopmentInternal(req)
	if !rebuilt.Success {
		return fmt.Errorf("Development artifact rebuild failed: %v", rebuilt.Diagnostics)
	}
	if result.SourceStatus != rebuilt.SourceStatus || result.InterfaceLinkStatus != rebuilt.InterfaceLinkStatus ||
		result.ArtifactStatus != rebuilt.ArtifactStatus || result.ExecutionStatus != rebuilt.ExecutionStatus {
		return fmt.Errorf("Development BuildResult status fields differ from the deterministic rebuild")
	}
	if result.ArtifactDigest != rebuilt.ArtifactDigest {
		return fmt.Errorf("Development artifactDigest differs from the deterministic rebuild")
	}
	if len(result.OutputFiles) != len(rebuilt.OutputFiles) {
		return fmt.Errorf("Development artifact file count differs from the deterministic rebuild")
	}
	for index := range rebuilt.OutputFiles {
		want := rebuilt.OutputFiles[index]
		got := result.OutputFiles[index]
		if got.Path != want.Path {
			return fmt.Errorf("Development artifact file order/path differs at %d: %q != %q", index, got.Path, want.Path)
		}
		if string(got.Bytes) != string(want.Bytes) {
			return fmt.Errorf("Development artifact file bytes differ from the deterministic rebuild: %s", want.Path)
		}
	}
	return nil
}

// VerifyDevelopmentJSON decodes both documents strictly and verifies the result.
func VerifyDevelopmentJSON(requestJSON, resultJSON []byte) error {
	req, err := DecodeDevelopmentBuildRequestJSON(requestJSON)
	if err != nil {
		return err
	}
	result, err := DecodeDevelopmentBuildResultJSON(resultJSON)
	if err != nil {
		return err
	}
	return VerifyDevelopment(req, result)
}

// DecodeDevelopmentBuildResultJSON strictly decodes one BuildResult document.
func DecodeDevelopmentBuildResultJSON(data []byte) (DevelopmentBuildResult, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return DevelopmentBuildResult{}, err
	}
	var result DevelopmentBuildResult
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return DevelopmentBuildResult{}, err
	}
	return result, nil
}

// verifyDevelopmentArtifactFiles verifies the artifact in isolation: file paths,
// digest list exactness, JCS manifest identity, execution status, ArtifactDigest
// and the two disjoint requirement documents.
func verifyDevelopmentArtifactFiles(result DevelopmentBuildResult) error {
	if result.ExecutionStatus != DevelopmentExecutionStatus {
		return fmt.Errorf("Development BuildResult.executionStatus must be %s", DevelopmentExecutionStatus)
	}
	byPath := map[string][]byte{}
	for _, file := range result.OutputFiles {
		if !validDevelopmentArtifactPath(file.Path) {
			return fmt.Errorf("Development artifact contains an unsafe path: %q", file.Path)
		}
		if _, exists := byPath[file.Path]; exists {
			return fmt.Errorf("Development artifact contains a duplicate path: %q", file.Path)
		}
		byPath[file.Path] = file.Bytes
	}
	manifestBytes, exists := byPath[developmentManifestFileName]
	if !exists {
		return fmt.Errorf("Development artifact has no %s", developmentManifestFileName)
	}
	canonical, err := canonicalizeJSON(manifestBytes)
	if err != nil {
		return fmt.Errorf("Development manifest is not canonical JSON: %w", err)
	}
	if string(canonical) != string(manifestBytes) {
		return fmt.Errorf("Development manifest is not JCS canonical")
	}
	var manifest developmentArtifactManifest
	manifestDecoder := json.NewDecoder(bytes.NewReader(manifestBytes))
	manifestDecoder.DisallowUnknownFields()
	if err := manifestDecoder.Decode(&manifest); err != nil {
		return fmt.Errorf("Development manifest cannot be decoded: %w", err)
	}
	if manifest.APIVersion != DevelopmentArtifactAPIVersion {
		return fmt.Errorf("Development manifest apiVersion = %q", manifest.APIVersion)
	}
	if manifest.LanguageTarget != V1Alpha5Target || manifest.Profile != DevelopmentProfile {
		return fmt.Errorf("Development manifest language target/profile is invalid")
	}
	if manifest.ExecutionStatus != DevelopmentExecutionStatus {
		return fmt.Errorf("Development manifest executionStatus must be %s", DevelopmentExecutionStatus)
	}
	if manifest.Generator.Name != v1alpha5ArtifactGeneratorName || manifest.Generator.Version != CompilerVersion {
		return fmt.Errorf("Development manifest generator identity is invalid")
	}
	sum := sha256.Sum256(manifestBytes)
	if result.ArtifactDigest != "sha256:"+hex.EncodeToString(sum[:]) {
		return fmt.Errorf("Development artifactDigest does not match the manifest bytes")
	}
	if len(manifest.Files) == 0 {
		return fmt.Errorf("Development manifest files list is empty")
	}
	listed := map[string]bool{}
	for _, entry := range manifest.Files {
		if _, exists := listed[entry.Path]; exists {
			return fmt.Errorf("Development manifest lists a duplicate file: %s", entry.Path)
		}
		listed[entry.Path] = true
		if entry.Path == developmentManifestFileName {
			return fmt.Errorf("Development manifest must not list itself")
		}
		raw, exists := byPath[entry.Path]
		if !exists {
			return fmt.Errorf("Development manifest lists a file that the result does not carry: %s", entry.Path)
		}
		fileSum := sha256.Sum256(raw)
		if entry.SHA256 != "sha256:"+hex.EncodeToString(fileSum[:]) {
			return fmt.Errorf("Development artifact file digest mismatch: %s", entry.Path)
		}
	}
	for path := range byPath {
		if path == developmentManifestFileName {
			continue
		}
		if !listed[path] {
			return fmt.Errorf("Development artifact carries a file the manifest does not list: %s", path)
		}
	}
	requirementsRaw, exists := byPath[developmentRequirementsFileName]
	if !exists {
		return fmt.Errorf("Development artifact has no %s", developmentRequirementsFileName)
	}
	if err := verifyDevelopmentRequirementDocument(requirementsRaw, "host"); err != nil {
		return err
	}
	runtimeRaw, exists := byPath[developmentRuntimeRequirementsFileName]
	if !exists {
		return fmt.Errorf("Development artifact has no %s", developmentRuntimeRequirementsFileName)
	}
	if err := verifyDevelopmentRequirementDocument(runtimeRaw, "runtime"); err != nil {
		return err
	}
	if manifest.Requirements.HostAdmission != developmentRequirementsFileName ||
		manifest.Requirements.RuntimeExecution != developmentRuntimeRequirementsFileName {
		return fmt.Errorf("Development manifest requirement document identities are invalid")
	}
	// The manifest must embed exactly the request-derived interface identity: the
	// lock is normalized and no artifact may claim to be admitted.
	if manifest.Lock.APIVersion != InterfaceDependencyLockAPIVersion {
		return fmt.Errorf("Development manifest lock apiVersion is invalid")
	}
	if err := verifyDevelopmentCapabilityRequirements(manifest.CapabilityRequirements); err != nil {
		return err
	}
	return nil
}

// verifyDevelopmentCapabilityRequirements verifies the structural exactness of
// the static Capability requirements that now form part of the manifest. The
// full request-derived content is already proven by the deterministic rebuild;
// this pass rejects a malformed, ambiguous or inconsistently ordered record even
// when an attacker re-seals the outer digest.
func verifyDevelopmentCapabilityRequirements(requirements []DevelopmentCapabilityRequirement) error {
	if requirements == nil {
		return fmt.Errorf("Development manifest capabilities must be a JSON array")
	}
	last := ""
	for _, requirement := range requirements {
		if requirement.Skill == "" || !validExportID(requirement.CapabilityID) {
			return fmt.Errorf("Development capability requirement has an invalid Skill or Capability identity")
		}
		catalog := requirement.Catalog
		if catalog.Key.PackageID == "" || catalog.Key.Version == "" || catalog.Key.SnapshotDigest == "" {
			return fmt.Errorf("Development capability requirement has an incomplete CapabilityCatalog Package key: %s", requirement.CapabilityID)
		}
		if catalog.Owner == "" || catalog.ExportID == "" || catalog.SourcePath == "" {
			return fmt.Errorf("Development capability requirement has an incomplete CapabilityCatalog export identity: %s", requirement.CapabilityID)
		}
		if catalog.ExportID != requirement.CapabilityID {
			return fmt.Errorf("Development capability requirement Catalog exportId differs from the selected Capability: %s", requirement.CapabilityID)
		}
		if !validDevelopmentCapabilityCatalogPointer(catalog.Pointer) {
			return fmt.Errorf("Development capability requirement has an invalid CapabilityCatalog pointer: %q", catalog.Pointer)
		}
		if requirement.Declaration.Path != "skills/"+requirement.Skill+"/assurance/contract.yaml" {
			return fmt.Errorf("Development capability requirement has a declaration outside the Skill Contract: %s", requirement.Declaration.Path)
		}
		if !validDevelopmentCapabilityPointer(requirement.Declaration.Pointer) {
			return fmt.Errorf("Development capability requirement has an invalid declaration pointer: %q", requirement.Declaration.Pointer)
		}
		if len(requirement.Callables) == 0 {
			return fmt.Errorf("Development capability requirement has no Callable members: %s", requirement.CapabilityID)
		}
		seen := map[string]bool{}
		for _, callable := range requirement.Callables {
			if callable.Key.PackageID == "" || callable.Key.Version == "" || callable.Key.SnapshotDigest == "" ||
				callable.Owner == "" || !validExportID(callable.ExportID) || callable.SourcePath == "" {
				return fmt.Errorf("Development capability requirement has an incomplete Callable member: %s", callable.ExportID)
			}
			if callable.Kind != "operation" && callable.Kind != "observation" {
				return fmt.Errorf("Development capability requirement Callable kind is invalid: %s", callable.Kind)
			}
			if callable.Effect != "READ" && callable.Effect != "MUTATION" {
				return fmt.Errorf("Development capability requirement Callable effect is invalid: %s", callable.Effect)
			}
			identity := callable.ExportID + "\x00" + developmentKeyString(callable.Key) + "\x00" + callable.Kind
			if seen[identity] {
				return fmt.Errorf("Development capability requirement duplicates a Callable member: %s", callable.ExportID)
			}
			seen[identity] = true
		}
		canonical, err := canonicalizeValue(requirement)
		if err != nil {
			return fmt.Errorf("Development capability requirement cannot be canonicalized: %w", err)
		}
		if string(canonical) <= last {
			return fmt.Errorf("Development capability requirements are not in canonical order")
		}
		last = string(canonical)
	}
	return nil
}

// validDevelopmentCapabilityPointer accepts the exact Skill Contract position
// /capabilities/<index> with a canonical array index.
func validDevelopmentCapabilityPointer(pointer string) bool {
	rest, ok := strings.CutPrefix(pointer, "/capabilities/")
	if !ok {
		return false
	}
	return validEvidenceIndex(rest)
}

// validDevelopmentCapabilityCatalogPointer accepts the exact CapabilityCatalog
// position /capabilities/<index>; both documents use the same canonical index
// grammar.
func validDevelopmentCapabilityCatalogPointer(pointer string) bool {
	return validDevelopmentCapabilityPointer(pointer)
}

// verifyDevelopmentRequirementDocument re-derives every requirement ID from its
// typed key and enforces the exact class boundary: the host document carries only
// external pre-dispatch facts and the runtime document carries only
// execution-time guarantees. Deleted, added, edited, reordered or
// cross-class-migrated requirements are rejected.
func verifyDevelopmentRequirementDocument(raw []byte, class string) error {
	label := "host-admission"
	if class == "runtime" {
		label = "runtime-execution"
	}
	canonical, err := canonicalizeJSON(raw)
	if err != nil {
		return fmt.Errorf("Development %s requirements document is not canonical JSON: %w", label, err)
	}
	if string(canonical) != string(raw) {
		return fmt.Errorf("Development %s requirements document is not JCS canonical", label)
	}
	var requirements []DevelopmentHostRequirement
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&requirements); err != nil {
		return fmt.Errorf("Development %s requirements document cannot be decoded: %w", label, err)
	}
	if requirements == nil {
		return fmt.Errorf("Development %s requirements document must be a JSON array", label)
	}
	if len(requirements) > maxDevelopmentRequirements {
		return fmt.Errorf("Development %s requirements document exceeds the requirement count limit", label)
	}
	last := ""
	for _, requirement := range requirements {
		if requirement.RequirementID <= last {
			return fmt.Errorf("Development %s requirements are not sorted by requirementId", label)
		}
		last = requirement.RequirementID
		if developmentRequirementClass(requirement.Kind) != class {
			return fmt.Errorf("Development %s requirement has a kind from the other class: %s", label, requirement.Kind)
		}
		if requirement.Status != developmentRequirementStatus(requirement.Kind) {
			return fmt.Errorf("Development requirement has an invalid status: %s", requirement.Status)
		}
		key := normalizeDevelopmentRequirementKey(developmentRequirementKey{
			Kind: requirement.Kind, Location: requirement.Location, Subjects: requirement.Subjects,
			Independence: requirement.Independence,
		})
		if !developmentRequirementIsNormalized(requirement, key) {
			return fmt.Errorf("Development requirement is not normalized: %s", requirement.RequirementID)
		}
		if err := verifyDevelopmentRequirementLocation(requirement); err != nil {
			return err
		}
		if requirement.Kind == DevelopmentRequirementIndependence {
			if key.Independence == nil {
				return fmt.Errorf("Development independence requirement has no group identity: %s", requirement.RequirementID)
			}
			if requirement.Location.Node == "" {
				return fmt.Errorf("Development independence requirement has no source Predicate: %s", requirement.RequirementID)
			}
			if err := verifyDevelopmentIndependenceGroup(key.Independence); err != nil {
				return err
			}
		} else if requirement.Independence != nil {
			return fmt.Errorf("Development requirement of kind %s must not carry independence detail", requirement.Kind)
		}
		canonicalKey, err := canonicalizeValue(key)
		if err != nil {
			return fmt.Errorf("Development requirement key cannot be canonicalized: %w", err)
		}
		sum := sha256.Sum256(append([]byte(developmentRequirementDigestDomain), canonicalKey...))
		want := "sha256:" + hex.EncodeToString(sum[:])
		if requirement.RequirementID != want {
			return fmt.Errorf("Development requirement identity does not match its typed key: %s", requirement.RequirementID)
		}
	}
	return nil
}

// verifyDevelopmentRequirementLocation enforces that the recorded source position
// is the one the contract fixes for its kind: an Independence requirement points
// at exactly one group JSON Pointer, and an Evidence trust requirement points
// inside the Evidence block of the Predicate named by location.node. A record
// whose location is coarse, dangling or borrowed from another Predicate is
// rejected without consulting the original request.
func verifyDevelopmentRequirementLocation(requirement DevelopmentHostRequirement) error {
	location := requirement.Location
	if location.Skill == "" || location.Path == "" {
		return fmt.Errorf("Development requirement has an incomplete source location: %s", requirement.RequirementID)
	}
	switch requirement.Kind {
	case DevelopmentRequirementIndependence:
		group := requirement.Independence
		if group == nil {
			return fmt.Errorf("Development independence requirement has no group identity: %s", requirement.RequirementID)
		}
		if location.Node == "" || location.Stage != "evidence" {
			return fmt.Errorf("Development independence requirement has no source Predicate stage: %s", requirement.RequirementID)
		}
		want := "/predicates/" + v1alpha5JSONPointerToken(location.Node) + "/evidence/independence/" + itoa(group.GroupIndex)
		if location.Selector != want {
			return fmt.Errorf("Development independence requirement selector %q is not the exact group position", location.Selector)
		}
	case DevelopmentRequirementEvidenceDomain, DevelopmentRequirementEvidenceFact:
		if location.Node == "" || location.Stage != "evidence" {
			return fmt.Errorf("Development evidence trust requirement has no source Predicate stage: %s", requirement.RequirementID)
		}
		if !validDevelopmentEvidenceSelector(location.Node, location.Selector) {
			return fmt.Errorf("Development evidence trust requirement selector %q is not an exact Evidence position", location.Selector)
		}
	case DevelopmentRequirementOwnerTrust:
		// An EvidenceDescriptor export is consumed from the Predicate Evidence
		// block, so its OWNER_TRUST must point at the same exact reference
		// position as the guarantee itself. A CapabilityCatalog owner points at
		// the Skill's own Capability declaration. Callable/Schema Owner
		// requirements use other stages and are not constrained here.
		if location.Stage == "evidence" && !validDevelopmentEvidenceSelector(location.Node, location.Selector) {
			return fmt.Errorf("Development Descriptor Owner Trust selector %q is not an exact Evidence position", location.Selector)
		}
		if location.Stage == "capabilities" {
			if len(requirement.Subjects) != 1 || requirement.Subjects[0].ID != location.Node {
				return fmt.Errorf("Development CapabilityCatalog Owner Trust does not name the exact Capability: %s", requirement.RequirementID)
			}
			if location.Path != "skills/"+location.Skill+"/assurance/contract.yaml" || !validDevelopmentCapabilityPointer(location.Selector) {
				return fmt.Errorf("Development CapabilityCatalog Owner Trust selector %q is not an exact declaration position", location.Selector)
			}
		}
	}
	return nil
}

// validDevelopmentEvidenceSelector accepts exactly the two released Predicate
// Evidence reference positions for the Predicate named by node:
//
//	/predicates/<predicate>/evidence/requires/<parameter>/<index>
//	/predicates/<predicate>/evidence/independence/<group>/members/<member>/guarantee
//
// It rejects a coarse block pointer, a borrowed Predicate, an unreachable member
// position and a non-canonical array index.
func validDevelopmentEvidenceSelector(node, selector string) bool {
	if node == "" {
		return false
	}
	rest, ok := strings.CutPrefix(selector, "/predicates/"+v1alpha5JSONPointerToken(node)+"/evidence/")
	if !ok {
		return false
	}
	segments := strings.Split(rest, "/")
	switch {
	case len(segments) == 3 && segments[0] == "requires":
		return validEvidencePointerToken(segments[1]) && validEvidenceIndex(segments[2])
	case len(segments) == 5 && segments[0] == "independence" && segments[2] == "members" && segments[4] == "guarantee":
		return validEvidenceIndex(segments[1]) && validEvidenceIndex(segments[3])
	default:
		return false
	}
}

func validEvidencePointerToken(token string) bool {
	if token == "" {
		return false
	}
	for i := 0; i < len(token); i++ {
		if token[i] != '~' {
			continue
		}
		if i+1 >= len(token) || (token[i+1] != '0' && token[i+1] != '1') {
			return false
		}
		i++
	}
	return true
}

func validEvidenceIndex(value string) bool {
	if value == "" {
		return false
	}
	index := 0
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
		index = index*10 + int(value[i]-'0')
		if index > maxDevelopmentRequirements {
			return false
		}
	}
	return itoa(index) == value
}

func developmentRequirementIsNormalized(requirement DevelopmentHostRequirement, normalized developmentRequirementKey) bool {
	subjects, err := canonicalizeValue(requirement.Subjects)
	if err != nil || string(subjects) != mustCanonicalize(normalized.Subjects) {
		return false
	}
	independence, err := canonicalizeValue(requirement.Independence)
	if err != nil || string(independence) != mustCanonicalize(normalized.Independence) {
		return false
	}
	return true
}

func mustCanonicalize(value any) string {
	canonical, err := canonicalizeValue(value)
	if err != nil {
		return ""
	}
	return string(canonical)
}

// verifyDevelopmentIndependenceGroup enforces the exact independence shape: one
// identified predicate-local group, at least two distinct member Descriptors,
// distinct Source Domains and at least one Domain Pair, each member carrying an
// exact Interface Package identity.
func verifyDevelopmentIndependenceGroup(group *DevelopmentIndependenceGroup) error {
	if group.GroupIndex < 0 {
		return fmt.Errorf("Development independence requirement has no source group identity")
	}
	if len(group.Members) < 2 {
		return fmt.Errorf("Development independence requirement has fewer than two members")
	}
	domains := map[string]bool{}
	identities := map[string]bool{}
	for _, member := range group.Members {
		if member.DependencyAlias == "" || member.ExportID == "" || member.Domain == "" ||
			member.Key.PackageID == "" || member.Key.Version == "" || member.Key.SnapshotDigest == "" {
			return fmt.Errorf("Development independence member is incomplete: %s", member.ExportID)
		}
		domains[member.Domain] = true
		identity := member.DependencyAlias + "\x00" + member.ExportID + "\x00" + developmentKeyString(member.Key)
		if identities[identity] {
			return fmt.Errorf("Development independence requirement duplicates a member: %s", member.ExportID)
		}
		identities[identity] = true
	}
	if len(domains) < 2 {
		return fmt.Errorf("Development independence requirement has fewer than two Source Domains")
	}
	if len(group.DomainPairs) == 0 {
		return fmt.Errorf("Development independence requirement has no Domain Pair")
	}
	seenPair := map[string]bool{}
	for _, pair := range group.DomainPairs {
		if len(pair) != 2 || pair[0] == "" || pair[1] == "" || pair[0] == pair[1] {
			return fmt.Errorf("Development independence requirement has an invalid Domain Pair")
		}
		if !domains[pair[0]] || !domains[pair[1]] {
			return fmt.Errorf("Development independence Domain Pair does not reference declared member domains")
		}
		if pair[0] >= pair[1] {
			return fmt.Errorf("Development independence Domain Pair is not canonically ordered")
		}
		key := pair[0] + "\x00" + pair[1]
		if seenPair[key] {
			return fmt.Errorf("Development independence requirement duplicates a Domain Pair")
		}
		seenPair[key] = true
	}
	return nil
}
