package compiler

// CA-SL v1alpha5 WP8: candidate artifact-tree verification.
//
// VerifyBuildResult dispatches here for a successful BuildResult stamped with
// BuildResultAPIVersionV1Alpha5. Verification never writes to disk and checks:
// unique normalized paths, exactly one top-level canonical manifest, complete
// file listing with matching sha256, per-Skill source/compiled file coherence,
// ingress-links/manifest self-consistency, dependency-closure identity shape and
// a canonical normalized Procedure document. The frozen v1alpha4 artifact
// verification path (internal/artifact.Verify) is unchanged.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/odango-chan/ca-skill-compiler/internal/artifact"
)

// verifyV1Alpha5ArtifactTree checks one v1alpha5 BuildResult output tree.
func verifyV1Alpha5ArtifactTree(result BuildResult, files []artifact.File) error {
	if result.APIVersion != BuildResultAPIVersionV1Alpha5 {
		return fmt.Errorf("v1alpha5 artifact verification requires BuildResult.apiVersion %q, got %q",
			BuildResultAPIVersionV1Alpha5, result.APIVersion)
	}
	if len(files) == 0 {
		return fmt.Errorf("v1alpha5 artifact tree is empty")
	}
	byPath := make(map[string][]byte, len(files))
	for _, file := range files {
		if file.Path == "" {
			return fmt.Errorf("v1alpha5 artifact contains an empty file path")
		}
		if _, exists := byPath[file.Path]; exists {
			return fmt.Errorf("v1alpha5 artifact contains duplicate file path %q", file.Path)
		}
		byPath[file.Path] = file.Bytes
	}

	manifestBytes, exists := byPath[V1Alpha5ArtifactManifestPath]
	if !exists {
		return fmt.Errorf("v1alpha5 artifact is missing the top-level %s", V1Alpha5ArtifactManifestPath)
	}
	canonical, err := artifact.CanonicalizeJSON(manifestBytes)
	if err != nil {
		return fmt.Errorf("v1alpha5 manifest is not valid JSON: %w", err)
	}
	if !bytes.Equal(canonical, manifestBytes) {
		return fmt.Errorf("v1alpha5 manifest is not canonical JCS")
	}
	var manifest v1alpha5ArtifactManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("v1alpha5 manifest cannot be decoded: %w", err)
	}
	if manifest.APIVersion != V1Alpha5ArtifactAPIVersion {
		return fmt.Errorf("v1alpha5 manifest apiVersion = %q", manifest.APIVersion)
	}
	if manifest.LanguageTarget != V1Alpha5LanguageTarget {
		return fmt.Errorf("v1alpha5 manifest languageTarget = %q", manifest.LanguageTarget)
	}
	if manifest.Generator.Name != v1alpha5ArtifactGeneratorName || manifest.Generator.Version != CompilerVersion {
		return fmt.Errorf("v1alpha5 manifest generator identity is invalid")
	}

	listedFiles := make(map[string]bool, len(manifest.OutputFiles))
	previousPath := ""
	for index, entry := range manifest.OutputFiles {
		if entry.Path == "" || entry.SHA256 == "" {
			return fmt.Errorf("v1alpha5 manifest outputFiles[%d] is incomplete", index)
		}
		if index > 0 && previousPath >= entry.Path {
			return fmt.Errorf("v1alpha5 manifest outputFiles are not strictly sorted at %q", entry.Path)
		}
		previousPath = entry.Path
		if entry.Path == V1Alpha5ArtifactManifestPath {
			return fmt.Errorf("v1alpha5 manifest must not list itself")
		}
		if _, ok := byPath[entry.Path]; !ok {
			return fmt.Errorf("v1alpha5 manifest lists missing file %q", entry.Path)
		}
		sum := sha256Hex(byPath[entry.Path])
		if entry.SHA256 != sum {
			return fmt.Errorf("v1alpha5 manifest digest mismatch for %q", entry.Path)
		}
		listedFiles[entry.Path] = true
	}
	for filePath := range byPath {
		if filePath == V1Alpha5ArtifactManifestPath {
			continue
		}
		if !listedFiles[filePath] {
			return fmt.Errorf("v1alpha5 artifact file %q is not listed by the manifest", filePath)
		}
	}

	if len(manifest.Skills) == 0 {
		return fmt.Errorf("v1alpha5 manifest lists no Skills")
	}
	compiledOwner := make(map[string]string, len(byPath))
	skillNames := make(map[string]bool, len(manifest.Skills))
	previousSkill := ""
	for index, skill := range manifest.Skills {
		if index > 0 && previousSkill >= skill.Name {
			return fmt.Errorf("v1alpha5 manifest skills are not strictly sorted at %q", skill.Name)
		}
		previousSkill = skill.Name
		if !validV1Alpha5ArtifactSkillName(skill.Name) {
			return fmt.Errorf("v1alpha5 manifest Skill name %q is invalid", skill.Name)
		}
		skillNames[skill.Name] = true
		prefix := "skills/" + skill.Name + "/"
		compiledPrefix := prefix + "compiled/"
		if len(skill.SourceFiles) == 0 {
			return fmt.Errorf("v1alpha5 manifest Skill %q lists no source files", skill.Name)
		}
		if !isSHA256Value(skill.SourceDigest) {
			return fmt.Errorf("v1alpha5 manifest Skill %q sourceDigest = %q", skill.Name, skill.SourceDigest)
		}
		for position, sourcePath := range skill.SourceFiles {
			if position > 0 && skill.SourceFiles[position-1] >= sourcePath {
				return fmt.Errorf("v1alpha5 manifest Skill %q sourceFiles are not strictly sorted", skill.Name)
			}
			if !strings.HasPrefix(sourcePath, prefix) || strings.Contains(sourcePath, "..") {
				return fmt.Errorf("v1alpha5 manifest Skill %q lists source file %q outside its tree", skill.Name, sourcePath)
			}
		}
		if len(skill.CompiledFiles) == 0 {
			return fmt.Errorf("v1alpha5 manifest Skill %q lists no compiled files", skill.Name)
		}
		for position, compiledPath := range skill.CompiledFiles {
			if position > 0 && skill.CompiledFiles[position-1] >= compiledPath {
				return fmt.Errorf("v1alpha5 manifest Skill %q compiledFiles are not strictly sorted", skill.Name)
			}
			if !strings.HasPrefix(compiledPath, compiledPrefix) {
				return fmt.Errorf("v1alpha5 manifest Skill %q lists compiled file %q outside its compiled tree", skill.Name, compiledPath)
			}
			if _, ok := byPath[compiledPath]; !ok {
				return fmt.Errorf("v1alpha5 manifest Skill %q lists missing compiled file %q", skill.Name, compiledPath)
			}
			if !listedFiles[compiledPath] {
				return fmt.Errorf("v1alpha5 manifest Skill %q compiled file %q is not listed in outputFiles", skill.Name, compiledPath)
			}
			if owner, ok := compiledOwner[compiledPath]; ok {
				return fmt.Errorf("v1alpha5 compiled file %q is claimed by Skills %q and %q", compiledPath, owner, skill.Name)
			}
			compiledOwner[compiledPath] = skill.Name
		}
		if err := verifyV1Alpha5ManifestSkillFiles(skill, byPath); err != nil {
			return err
		}
		if err := verifyV1Alpha5WaitLinks(skill); err != nil {
			return err
		}
	}
	for path := range byPath {
		if path == V1Alpha5ArtifactManifestPath {
			continue
		}
		if _, ok := compiledOwner[path]; !ok {
			return fmt.Errorf("v1alpha5 artifact file %q is not claimed by any Skill", path)
		}
	}

	if err := verifyV1Alpha5ManifestClosure(manifest.DependencyClosure); err != nil {
		return err
	}
	if err := verifyV1Alpha5ResultClosure(result, manifest); err != nil {
		return err
	}
	return nil
}

func verifyV1Alpha5ManifestSkillFiles(skill v1alpha5ManifestSkill, byPath map[string][]byte) error {
	procedurePath := "skills/" + skill.Name + "/compiled/" + v1alpha5ProcedureFileName
	linksPath := "skills/" + skill.Name + "/compiled/" + v1alpha5IngressLinksFileName
	procedureRaw, ok := byPath[procedurePath]
	if !ok {
		return fmt.Errorf("v1alpha5 manifest Skill %q is missing %s", skill.Name, procedurePath)
	}
	var procedureDocument map[string]json.RawMessage
	if err := json.Unmarshal(procedureRaw, &procedureDocument); err != nil {
		return fmt.Errorf("v1alpha5 Skill %q Procedure document is not a JSON object: %w", skill.Name, err)
	}
	if _, ok := procedureDocument["start"]; !ok {
		return fmt.Errorf("v1alpha5 Skill %q Procedure document has no start", skill.Name)
	}
	if _, ok := procedureDocument["nodes"]; !ok {
		return fmt.Errorf("v1alpha5 Skill %q Procedure document has no nodes", skill.Name)
	}
	canonical, err := artifact.CanonicalizeJSON(procedureRaw)
	if err != nil || !bytes.Equal(canonical, procedureRaw) {
		return fmt.Errorf("v1alpha5 Skill %q Procedure document is not canonical JCS", skill.Name)
	}

	linksRaw, ok := byPath[linksPath]
	if !ok {
		return fmt.Errorf("v1alpha5 manifest Skill %q is missing %s", skill.Name, linksPath)
	}
	canonical, err = artifact.CanonicalizeJSON(linksRaw)
	if err != nil || !bytes.Equal(canonical, linksRaw) {
		return fmt.Errorf("v1alpha5 Skill %q ingress link document is not canonical JCS", skill.Name)
	}
	var document v1alpha5IngressLinksDocument
	if err := json.Unmarshal(linksRaw, &document); err != nil {
		return fmt.Errorf("v1alpha5 Skill %q ingress link document cannot be decoded: %w", skill.Name, err)
	}
	manifestWaitLinks := append([]v1alpha5ManifestWaitLink(nil), skill.WaitLinks...)
	if manifestWaitLinks == nil {
		manifestWaitLinks = []v1alpha5ManifestWaitLink{}
	}
	documentLinks := document.Links
	if documentLinks == nil {
		documentLinks = []v1alpha5ManifestWaitLink{}
	}
	documentBytes, err := json.Marshal(v1alpha5IngressLinksDocument{Links: documentLinks})
	if err != nil {
		return err
	}
	manifestLinksBytes, err := json.Marshal(v1alpha5IngressLinksDocument{Links: manifestWaitLinks})
	if err != nil {
		return err
	}
	if !bytes.Equal(documentBytes, manifestLinksBytes) {
		return fmt.Errorf("v1alpha5 Skill %q ingress link document disagrees with the manifest", skill.Name)
	}
	return nil
}

func verifyV1Alpha5WaitLinks(skill v1alpha5ManifestSkill) error {
	seenNode := make(map[string]bool, len(skill.WaitLinks))
	for index, link := range skill.WaitLinks {
		if link.NodeID == "" || seenNode[link.NodeID] {
			return fmt.Errorf("v1alpha5 Skill %q waitLinks[%d] has a missing or duplicate nodeId", skill.Name, index)
		}
		seenNode[link.NodeID] = true
		if link.Event.Source == "" || link.Event.Service == "" || link.Event.Operation == "" {
			return fmt.Errorf("v1alpha5 Skill %q Wait Node %q event selector is incomplete", skill.Name, link.NodeID)
		}
		if link.Timeout == "" || link.TimeoutRoute == "" {
			return fmt.Errorf("v1alpha5 Skill %q Wait Node %q timeout/route is incomplete", skill.Name, link.NodeID)
		}
		ingress := link.Ingress
		if ingress.DependencyAlias == "" || ingress.CatalogOwner == "" || ingress.SnapshotPath == "" ||
			ingress.OperationID == "" || ingress.NormalizedSchemaRef == "" || !isSHA256Value(ingress.NormalizedSchemaDigest) {
			return fmt.Errorf("v1alpha5 Skill %q Wait Node %q ingress identity is incomplete", skill.Name, link.NodeID)
		}
		response := link.Response
		if response.Mode != "none" && response.Mode != "paired" {
			return fmt.Errorf("v1alpha5 Skill %q Wait Node %q response mode = %q", skill.Name, link.NodeID, response.Mode)
		}
		if response.BoundaryNodeID == "" {
			return fmt.Errorf("v1alpha5 Skill %q Wait Node %q response boundary is empty", skill.Name, link.NodeID)
		}
		if response.Mode == "paired" && response.ProducerStage == "" && response.ResultSchemaRef != "" {
			return fmt.Errorf("v1alpha5 Skill %q Wait Node %q paired response producer is empty", skill.Name, link.NodeID)
		}
		if response.ResultSchemaRef != "" && !isSHA256Value(response.ResultSchemaDigest) {
			return fmt.Errorf("v1alpha5 Skill %q Wait Node %q response result Schema digest is invalid", skill.Name, link.NodeID)
		}
		for position, terminal := range response.EarlyFailureTerminals {
			if position > 0 && response.EarlyFailureTerminals[position-1] >= terminal {
				return fmt.Errorf("v1alpha5 Skill %q Wait Node %q earlyFailureTerminals are not sorted", skill.Name, link.NodeID)
			}
		}
	}
	return nil
}

func verifyV1Alpha5ManifestClosure(closure []v1alpha5ManifestDependency) error {
	previous := ""
	for index, dependency := range closure {
		if dependency.DependencyID == "" || dependency.StablePackageID == "" {
			return fmt.Errorf("v1alpha5 dependency closure[%d] is incomplete", index)
		}
		if index > 0 && previous >= dependency.DependencyID {
			return fmt.Errorf("v1alpha5 dependency closure is not strictly sorted at %q", dependency.DependencyID)
		}
		previous = dependency.DependencyID
		if len(dependency.Exports) == 0 {
			return fmt.Errorf("v1alpha5 dependency closure %q has no exports", dependency.DependencyID)
		}
		for position, export := range dependency.Exports {
			if export.ExportKind == "" || export.ExportID == "" || export.Owner == "" {
				return fmt.Errorf("v1alpha5 dependency closure %q has an incomplete export", dependency.DependencyID)
			}
			if position > 0 {
				left, right := dependency.Exports[position-1], export
				if left.ExportKind > right.ExportKind ||
					(left.ExportKind == right.ExportKind && left.ExportID > right.ExportID) ||
					(left.ExportKind == right.ExportKind && left.ExportID == right.ExportID && left.Owner >= right.Owner) {
					return fmt.Errorf("v1alpha5 dependency closure %q exports are not strictly sorted", dependency.DependencyID)
				}
			}
		}
	}
	return nil
}

func verifyV1Alpha5ResultClosure(result BuildResult, manifest v1alpha5ArtifactManifest) error {
	manifestAliases := make([]string, 0, len(manifest.DependencyClosure))
	for _, dependency := range manifest.DependencyClosure {
		manifestAliases = append(manifestAliases, dependency.DependencyID)
	}
	sort.Strings(manifestAliases)
	if len(result.DependencyClosure) != len(manifestAliases) {
		return fmt.Errorf("v1alpha5 BuildResult dependencyClosure disagrees with the manifest")
	}
	for index := range manifestAliases {
		if result.DependencyClosure[index] != manifestAliases[index] {
			return fmt.Errorf("v1alpha5 BuildResult dependencyClosure disagrees with the manifest")
		}
	}
	return nil
}

func validV1Alpha5ArtifactSkillName(name string) bool {
	if name == "" || len(name) > 80 {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
		default:
			return false
		}
	}
	return true
}

func isSHA256Value(value string) bool {
	const prefix = "sha256:"
	if !strings.HasPrefix(value, prefix) {
		return false
	}
	encoded := value[len(prefix):]
	if len(encoded) != 64 {
		return false
	}
	_, err := hex.DecodeString(encoded)
	return err == nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
