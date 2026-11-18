// Package projectinput loads a local Compiler project directory into the
// BuildRequest v1alpha5 transport.
//
// It is the single implementation of the on-disk project layout used by the
// reference CLI (cmd/skillc) and the parity tooling (cmd/parity-replay). It does
// not define a second Compiler input contract: every field it produces is a
// BuildRequest v1alpha5 field, and dependency snapshot digests are computed with
// the Compiler's ComputeDependencySnapshotDigest algorithm.
//
// Supported layout (all paths relative to the project root):
//
//	skills/...                          author Skill source (or files/skills/...)
//	snapshots/<dependencyId>/...        immutable dependency snapshot file sets
//	<capabilityDirectory>/...           trusted CapabilityContracts bundle
//	<evidenceDirectory>/...             trusted EvidenceTrustBindings bundle
//	trusted.json                        trusted metadata naming the above
//
// trusted.json is a local convenience description for trusted callers; it is
// not a frozen wire contract. Machine callers that already hold a trusted
// BuildRequest JSON document should use --request instead.
package projectinput

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/odango-chan/ca-skill-compiler/internal/compiler"
)

const (
	// maxFileSize is the per-file raw byte limit of the BuildRequest contract.
	maxFileSize = 2 << 20
	// maxTotalSize is the aggregate raw byte limit of the BuildRequest contract.
	maxTotalSize = 16 << 20
	// maxFileCount is the aggregate file-count limit of the BuildRequest contract.
	maxFileCount = 512
)

// TrustedConfig is the local trusted-input description read from trusted.json.
// The caller is responsible for only supplying values produced by its
// authenticated trust store; the Compiler does not make an untrusted file
// trustworthy.
type TrustedConfig struct {
	DependencyBindings []DependencyBinding `json:"dependencyBindings"`
	Deployment         Deployment          `json:"deployment"`

	// SnapshotBaseURI and ArtifactBindings/SchemaBindings describe the trusted
	// Ingress/Schema artifacts a v1alpha5 Wait-bearing Skill links against. They
	// are pass-through declarations: the Compiler recomputes every digest from
	// the referenced immutable snapshot bytes.
	SnapshotBaseURI  string            `json:"snapshotBaseURI"`
	ArtifactBindings []ArtifactBinding `json:"artifactBindings"`
	SchemaBindings   []SchemaBinding   `json:"schemaBindings"`
}

// DependencyBinding identifies one immutable dependency snapshot and the
// trusted Owner attribution for its exports.
type DependencyBinding struct {
	DependencyID     string        `json:"dependencyId"`
	StablePackageID  string        `json:"stablePackageId"`
	AuthorizedOwners []string      `json:"authorizedOwners"`
	ExportOwners     []ExportOwner `json:"exportOwners"`
}

// ExportOwner is the trusted Owner of one dependency export.
type ExportOwner struct {
	ExportKind string `json:"exportKind"`
	ExportID   string `json:"exportId"`
	Owner      string `json:"owner"`
}

// ArtifactBinding binds one declared artifact identity to an exact file inside a
// dependency snapshot.
type ArtifactBinding struct {
	ArtifactRef  string `json:"artifactRef"`
	DependencyID string `json:"dependencyId"`
	SnapshotPath string `json:"snapshotPath"`
	Dialect      string `json:"dialect"`
}

// SchemaBinding binds one exported Schema identity to a canonical root pointer
// inside a declared trusted artifact.
type SchemaBinding struct {
	ExportID    string `json:"exportId"`
	ArtifactRef string `json:"artifactRef"`
	Pointer     string `json:"pointer"`
}

// Deployment locates the trusted deployment bundles and authorization record.
type Deployment struct {
	DeploymentID        string `json:"deploymentId"`
	PublisherID         string `json:"publisherId"`
	AuthorizationRef    string `json:"authorizationRef"`
	CapabilityDirectory string `json:"capabilityDirectory"`
	EvidenceDirectory   string `json:"evidenceDirectory"`
}

// Load builds a BuildRequest from the project directory at root.
//
// It never infers a dependency snapshot, digest or Owner attribution: every
// snapshot and trusted binding comes from explicitly named files. A missing
// snapshot directory or trusted bundle is a hard input error instead of an
// empty inferred value, so a caller cannot get a false successful build from an
// incomplete project.
func Load(root string) (compiler.BuildRequest, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return compiler.BuildRequest{}, err
	}
	loader := &fileLoader{}
	request := compiler.BuildRequest{
		APIVersion:           compiler.BuildRequestAPIVersion,
		LanguageTarget:       compiler.V1Alpha5Target,
		ResolvedDependencies: []compiler.ResolvedDependency{},
	}
	sourceRoot := absolute
	if info, err := os.Stat(filepath.Join(absolute, "files")); err == nil && info.IsDir() {
		sourceRoot = filepath.Join(absolute, "files")
	}
	files, err := loader.readSkillFiles(sourceRoot)
	if err != nil {
		return compiler.BuildRequest{}, err
	}
	request.Files = files

	trustedPath := filepath.Join(absolute, "trusted.json")
	raw, err := os.ReadFile(trustedPath)
	if os.IsNotExist(err) {
		return request, nil
	}
	if err != nil {
		return compiler.BuildRequest{}, fmt.Errorf("read trusted.json: %w", err)
	}
	var trusted TrustedConfig
	if err := json.Unmarshal(raw, &trusted); err != nil {
		return compiler.BuildRequest{}, fmt.Errorf("decode trusted.json: %w", err)
	}

	context := &compiler.TrustedBuildContext{}
	bindings := make([]compiler.TrustedDependencyBinding, 0, len(trusted.DependencyBindings))
	for _, binding := range trusted.DependencyBindings {
		if binding.DependencyID == "" {
			return compiler.BuildRequest{}, fmt.Errorf("trusted.json dependency binding is missing dependencyId")
		}
		snapshotDir := filepath.Join(absolute, "snapshots", binding.DependencyID)
		snapshotFiles, err := loader.readSnapshotFiles(snapshotDir)
		if err != nil {
			return compiler.BuildRequest{}, fmt.Errorf("dependency %q: %w", binding.DependencyID, err)
		}
		if len(snapshotFiles) == 0 {
			return compiler.BuildRequest{}, fmt.Errorf("dependency %q: snapshot directory contains no files: %s", binding.DependencyID, snapshotDir)
		}
		digest, err := compiler.ComputeDependencySnapshotDigest(snapshotFiles)
		if err != nil {
			return compiler.BuildRequest{}, fmt.Errorf("dependency %q: %w", binding.DependencyID, err)
		}
		request.ResolvedDependencies = append(request.ResolvedDependencies, compiler.ResolvedDependency{
			DependencyID: binding.DependencyID,
			Digest:       digest,
			Files:        snapshotFiles,
		})
		exportOwners := make([]compiler.ExportOwner, 0, len(binding.ExportOwners))
		for _, export := range binding.ExportOwners {
			exportOwners = append(exportOwners, compiler.ExportOwner{
				ExportKind: export.ExportKind,
				ExportID:   export.ExportID,
				Owner:      export.Owner,
			})
		}
		bindings = append(bindings, compiler.TrustedDependencyBinding{
			DependencyID:     binding.DependencyID,
			Digest:           digest,
			StablePackageID:  binding.StablePackageID,
			AuthorizedOwners: binding.AuthorizedOwners,
			ExportOwners:     exportOwners,
		})
	}
	context.DependencyBindings = bindings

	hasDeploymentBundle := false
	if trusted.Deployment.CapabilityDirectory != "" {
		files, err := loader.readDirectoryFiles(filepath.Join(absolute, trusted.Deployment.CapabilityDirectory))
		if err != nil {
			return compiler.BuildRequest{}, fmt.Errorf("capability bundle: %w", err)
		}
		if len(files) == 0 {
			return compiler.BuildRequest{}, fmt.Errorf("capability bundle directory contains no files: %s", trusted.Deployment.CapabilityDirectory)
		}
		digest, err := compiler.ComputeDependencySnapshotDigest(files)
		if err != nil {
			return compiler.BuildRequest{}, fmt.Errorf("capability bundle: %w", err)
		}
		context.CapabilityContracts = &compiler.CapabilityContractsBundle{
			DeploymentID: trusted.Deployment.DeploymentID,
			PublisherID:  trusted.Deployment.PublisherID,
			Digest:       digest,
			Files:        files,
		}
		hasDeploymentBundle = true
	}
	if trusted.Deployment.EvidenceDirectory != "" {
		files, err := loader.readDirectoryFiles(filepath.Join(absolute, trusted.Deployment.EvidenceDirectory))
		if err != nil {
			return compiler.BuildRequest{}, fmt.Errorf("evidence trust bundle: %w", err)
		}
		if len(files) == 0 {
			return compiler.BuildRequest{}, fmt.Errorf("evidence trust bundle directory contains no files: %s", trusted.Deployment.EvidenceDirectory)
		}
		digest, err := compiler.ComputeDependencySnapshotDigest(files)
		if err != nil {
			return compiler.BuildRequest{}, fmt.Errorf("evidence trust bundle: %w", err)
		}
		context.EvidenceTrustBindings = &compiler.EvidenceTrustBindingsBundle{
			DeploymentID:   trusted.Deployment.DeploymentID,
			PublisherID:    trusted.Deployment.PublisherID,
			SnapshotDigest: digest,
			Files:          files,
		}
		hasDeploymentBundle = true
	}
	if hasDeploymentBundle {
		authorizations := []compiler.DeploymentAuthorization{{
			DeploymentID:     trusted.Deployment.DeploymentID,
			PublisherID:      trusted.Deployment.PublisherID,
			AuthorizationRef: trusted.Deployment.AuthorizationRef,
		}}
		context.DeploymentAuthorizations = &authorizations
	}

	context.SnapshotBaseURI = trusted.SnapshotBaseURI
	for _, binding := range trusted.ArtifactBindings {
		context.ArtifactBindings = append(context.ArtifactBindings, compiler.TrustedArtifactBinding{
			ArtifactRef:  binding.ArtifactRef,
			DependencyID: binding.DependencyID,
			SnapshotPath: binding.SnapshotPath,
			Dialect:      binding.Dialect,
		})
	}
	for _, binding := range trusted.SchemaBindings {
		context.SchemaBindings = append(context.SchemaBindings, compiler.TrustedSchemaBinding{
			ExportID:    binding.ExportID,
			ArtifactRef: binding.ArtifactRef,
			Pointer:     binding.Pointer,
		})
	}

	request.TrustedBuildContext = context
	return request, nil
}

// fileLoader enforces the BuildRequest resource limits while reading, before
// the Compiler core parses anything.
type fileLoader struct {
	count int
	bytes int64
}

func (l *fileLoader) admit(info fs.FileInfo, name string) error {
	if info.Size() > maxFileSize {
		return fmt.Errorf("file exceeds %d byte limit: %s", maxFileSize, name)
	}
	l.count++
	if l.count > maxFileCount {
		return fmt.Errorf("project exceeds %d file limit", maxFileCount)
	}
	l.bytes += info.Size()
	if l.bytes > maxTotalSize {
		return fmt.Errorf("project exceeds %d byte total limit", maxTotalSize)
	}
	return nil
}

// readSkillFiles reads only author Skill source under skills/ and never follows
// a symbolic link.
func (l *fileLoader) readSkillFiles(root string) ([]compiler.File, error) {
	files := make([]compiler.File, 0)
	err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) && name == root {
				return nil
			}
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not allowed: %s", name)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("nonregular file: %s", name)
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if !strings.HasPrefix(relative, "skills/") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if err := l.admit(info, name); err != nil {
			return err
		}
		content, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		files = append(files, compiler.File{Path: relative, Bytes: content})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// readSnapshotFiles reads a dependency snapshot, whose file paths are relative
// to the snapshot root rather than to the project.
func (l *fileLoader) readSnapshotFiles(directory string) ([]compiler.File, error) {
	return l.readDirectoryFiles(directory)
}

func (l *fileLoader) readDirectoryFiles(directory string) ([]compiler.File, error) {
	info, err := os.Lstat(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory does not exist: %s", directory)
		}
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("expected a real directory: %s", directory)
	}
	files := make([]compiler.File, 0)
	err = filepath.WalkDir(directory, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not allowed: %s", name)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("nonregular file: %s", name)
		}
		relative, err := filepath.Rel(directory, name)
		if err != nil {
			return err
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if err := l.admit(fileInfo, name); err != nil {
			return err
		}
		content, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		files = append(files, compiler.File{Path: filepath.ToSlash(relative), Bytes: content})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}
