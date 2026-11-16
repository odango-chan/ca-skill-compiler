// Command parity-replay produces the Native Go result used as the parity
// reference for the Chromium Worker test. It is tooling only: the Compiler core
// is unchanged.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/odango-chan/ca-skill-compiler/internal/compiler"
)

type outputFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type buildView struct {
	Success           bool                  `json:"success"`
	Diagnostics       []compiler.Diagnostic `json:"diagnostics"`
	DependencyClosure []string              `json:"dependencyClosure"`
	OutputFiles       []outputFile          `json:"outputFiles"`
}

type view struct {
	Check   []compiler.Diagnostic         `json:"check"`
	Build   buildView                     `json:"build"`
	Inspect compiler.DependencyInspection `json:"inspect"`
}

type trustedConfig struct {
	DependencyBindings []struct {
		DependencyID     string   `json:"dependencyId"`
		StablePackageID  string   `json:"stablePackageId"`
		AuthorizedOwners []string `json:"authorizedOwners"`
		ExportOwners     []struct {
			ExportKind string `json:"exportKind"`
			ExportID   string `json:"exportId"`
			Owner      string `json:"owner"`
		} `json:"exportOwners"`
	} `json:"dependencyBindings"`
	Deployment struct {
		DeploymentID        string `json:"deploymentId"`
		PublisherID         string `json:"publisherId"`
		AuthorizationRef    string `json:"authorizationRef"`
		CapabilityDirectory string `json:"capabilityDirectory"`
	} `json:"deployment"`
}

func main() {
	args := os.Args[1:]
	project, emitRequest := "", ""
	for _, arg := range args {
		if strings.HasPrefix(arg, "--emit-request=") {
			emitRequest = strings.TrimPrefix(arg, "--emit-request=")
			continue
		}
		if project == "" {
			project = arg
		}
	}
	if project == "" {
		fmt.Fprintln(os.Stderr, "usage: parity-replay <project-dir> [--emit-request=path]")
		os.Exit(2)
	}
	root, err := filepath.Abs(project)
	if err != nil {
		fail(err)
	}
	req, err := buildRequest(root)
	if err != nil {
		fail(err)
	}
	if emitRequest != "" {
		encoded, err := json.Marshal(req)
		if err != nil {
			fail(err)
		}
		if err := os.WriteFile(emitRequest, encoded, 0o644); err != nil {
			fail(err)
		}
	}
	check := compiler.Check(req)
	result := compiler.Build(req)
	encoded, err := json.Marshal(view{
		Check:   check,
		Inspect: compiler.InspectDependencies(req),
		Build: buildView{
			Success:           result.Success,
			Diagnostics:       result.Diagnostics,
			DependencyClosure: result.DependencyClosure,
			OutputFiles:       summarize(result.OutputFiles),
		},
	})
	if err != nil {
		fail(err)
	}
	fmt.Println(string(encoded))
}

// buildRequest supports two layouts:
//   - plain project: skills/... under the given directory (failure-path parity);
//   - trusted project: files/skills/... plus snapshots/<alias>/... plus an
//     optional capability directory and trusted.json bindings (success path).
func buildRequest(root string) (compiler.BuildRequest, error) {
	request := compiler.BuildRequest{
		APIVersion:           compiler.BuildRequestAPIVersion,
		LanguageTarget:       compiler.V1Alpha4Target,
		ResolvedDependencies: []compiler.ResolvedDependency{},
	}
	sourceRoot := root
	if _, err := os.Stat(filepath.Join(root, "files")); err == nil {
		sourceRoot = filepath.Join(root, "files")
	}
	files, err := readProject(sourceRoot)
	if err != nil {
		return request, err
	}
	request.Files = files
	if _, err := os.Stat(filepath.Join(root, "trusted.json")); err != nil {
		return request, nil
	}
	raw, err := os.ReadFile(filepath.Join(root, "trusted.json"))
	if err != nil {
		return request, err
	}
	var trusted trustedConfig
	if err := json.Unmarshal(raw, &trusted); err != nil {
		return request, fmt.Errorf("decode trusted.json: %w", err)
	}
	context := &compiler.TrustedBuildContext{}
	bindings := make([]compiler.TrustedDependencyBinding, 0, len(trusted.DependencyBindings))
	for _, binding := range trusted.DependencyBindings {
		snapshotDir := filepath.Join(root, "snapshots", binding.DependencyID)
		snapshotFiles, err := readSnapshotFiles(snapshotDir)
		if err != nil {
			return request, err
		}
		digest, err := compiler.ComputeDependencySnapshotDigest(snapshotFiles)
		if err != nil {
			return request, err
		}
		request.ResolvedDependencies = append(request.ResolvedDependencies, compiler.ResolvedDependency{
			DependencyID: binding.DependencyID,
			Digest:       digest,
			Files:        snapshotFiles,
		})
		exportOwners := make([]compiler.ExportOwner, 0, len(binding.ExportOwners))
		for _, export := range binding.ExportOwners {
			exportOwners = append(exportOwners, compiler.ExportOwner{ExportKind: export.ExportKind, ExportID: export.ExportID, Owner: export.Owner})
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
	capabilityDir := filepath.Join(root, trusted.Deployment.CapabilityDirectory)
	if trusted.Deployment.CapabilityDirectory != "" {
		capabilityFiles, err := readDirectoryFiles(capabilityDir, "")
		if err != nil {
			return request, err
		}
		digest, err := compiler.ComputeDependencySnapshotDigest(capabilityFiles)
		if err != nil {
			return request, err
		}
		context.CapabilityContracts = &compiler.CapabilityContractsBundle{
			DeploymentID: trusted.Deployment.DeploymentID,
			PublisherID:  trusted.Deployment.PublisherID,
			Digest:       digest,
			Files:        capabilityFiles,
		}
		authorizations := []compiler.DeploymentAuthorization{{
			DeploymentID:     trusted.Deployment.DeploymentID,
			PublisherID:      trusted.Deployment.PublisherID,
			AuthorizationRef: trusted.Deployment.AuthorizationRef,
		}}
		context.DeploymentAuthorizations = &authorizations
	}
	request.TrustedBuildContext = context
	return request, nil
}

func readSnapshotFiles(directory string) ([]compiler.File, error) {
	return readDirectoryFiles(directory, "")
}

func readDirectoryFiles(directory, prefix string) ([]compiler.File, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	files := make([]compiler.File, 0)
	for _, entry := range entries {
		relative := entry.Name()
		if prefix != "" {
			relative = prefix + "/" + entry.Name()
		}
		absolute := filepath.Join(directory, entry.Name())
		if entry.IsDir() {
			nested, err := readDirectoryFiles(absolute, relative)
			if err != nil {
				return nil, err
			}
			files = append(files, nested...)
			continue
		}
		content, err := os.ReadFile(absolute)
		if err != nil {
			return nil, err
		}
		files = append(files, compiler.File{Path: relative, Bytes: content})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func summarize(files []compiler.File) []outputFile {
	out := make([]outputFile, 0, len(files))
	for _, file := range files {
		sum := sha256.Sum256(file.Bytes)
		out = append(out, outputFile{Path: file.Path, SHA256: hex.EncodeToString(sum[:])})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func readProject(root string) ([]compiler.File, error) {
	files := make([]compiler.File, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if !strings.HasPrefix(relative, "skills/") {
			return nil
		}
		content, err := os.ReadFile(path)
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

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
