package projectinput

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/odango-chan/ca-skill-compiler/internal/compiler"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestLoadTrustedProjectMatchesBuildRequestContract(t *testing.T) {
	request, err := Load(filepath.Join(repoRoot(t), "web", "parity-success"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if request.APIVersion != compiler.BuildRequestAPIVersion || request.LanguageTarget != compiler.V1Alpha5Target {
		t.Fatalf("unexpected request identity: %+v", request)
	}
	if len(request.Files) == 0 {
		t.Fatal("no author source files loaded")
	}
	for _, file := range request.Files {
		if !strings.HasPrefix(file.Path, "skills/") {
			t.Fatalf("unexpected source path: %s", file.Path)
		}
	}
	if len(request.ResolvedDependencies) != 2 {
		t.Fatalf("resolved dependencies = %d, want 2", len(request.ResolvedDependencies))
	}
	for _, dependency := range request.ResolvedDependencies {
		digest, err := compiler.ComputeDependencySnapshotDigest(dependency.Files)
		if err != nil {
			t.Fatal(err)
		}
		if digest != dependency.Digest {
			t.Fatalf("dependency %s digest = %s, want %s", dependency.DependencyID, dependency.Digest, digest)
		}
	}
	if request.TrustedBuildContext == nil || len(request.TrustedBuildContext.DependencyBindings) != 2 {
		t.Fatalf("trusted bindings not loaded: %+v", request.TrustedBuildContext)
	}
	for _, binding := range request.TrustedBuildContext.DependencyBindings {
		if binding.StablePackageID == "" || len(binding.AuthorizedOwners) == 0 {
			t.Fatalf("incomplete binding: %+v", binding)
		}
	}
	capability := request.TrustedBuildContext.CapabilityContracts
	if capability == nil || len(capability.Files) == 0 {
		t.Fatalf("capability bundle not loaded: %+v", capability)
	}
	digest, err := compiler.ComputeDependencySnapshotDigest(capability.Files)
	if err != nil {
		t.Fatal(err)
	}
	if digest != capability.Digest {
		t.Fatalf("capability digest = %s, want %s", capability.Digest, digest)
	}
	if request.TrustedBuildContext.DeploymentAuthorizations == nil || len(*request.TrustedBuildContext.DeploymentAuthorizations) != 1 {
		t.Fatalf("deployment authorization not loaded")
	}
}

func TestLoadSourceOnlyProject(t *testing.T) {
	request, err := Load(filepath.Join(repoRoot(t), "web", "parity-project"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(request.Files) == 0 {
		t.Fatal("no source files loaded")
	}
	if request.TrustedBuildContext != nil {
		t.Fatal("unexpected trusted context for source-only project")
	}
	if len(request.ResolvedDependencies) != 0 {
		t.Fatal("inferred dependency snapshots for source-only project")
	}
}

func TestLoadFilesSourceLayout(t *testing.T) {
	request, err := Load(filepath.Join(repoRoot(t), "web", "parity-success"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range request.Files {
		if file.Path == "" || strings.HasPrefix(file.Path, "files/") {
			t.Fatalf("files/ source root was not stripped: %s", file.Path)
		}
	}
}

func TestLoadMissingSnapshotIsInputError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "trusted.json"), `{"dependencyBindings":[{"dependencyId":"ca","stablePackageId":"urn:ca","authorizedOwners":["owner"]}]}`)
	writeFile(t, filepath.Join(root, "skills", "demo", "SKILL.md"), "x")
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "snapshots") {
		t.Fatalf("missing snapshot not reported as an input error: %v", err)
	}
}

func TestLoadEmptySnapshotIsInputError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "trusted.json"), `{"dependencyBindings":[{"dependencyId":"ca","stablePackageId":"urn:ca","authorizedOwners":["owner"]}]}`)
	writeFile(t, filepath.Join(root, "skills", "demo", "SKILL.md"), "x")
	if err := os.MkdirAll(filepath.Join(root, "snapshots", "ca"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "no files") {
		t.Fatalf("empty snapshot not reported as an input error: %v", err)
	}
}

func TestLoadRejectsSymlinkedSource(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "skills", "demo")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("SKILL.md", filepath.Join(dir, "alias.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("symlinked source file accepted")
	}
}

func TestLoadRejectsOversizedSourceFile(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "skills", "demo")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, "big.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maxFileSize + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "byte limit") {
		t.Fatalf("oversized source file not rejected: %v", err)
	}
}

func writeFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
