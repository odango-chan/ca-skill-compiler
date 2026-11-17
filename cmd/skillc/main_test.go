package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCLIArgumentValidation(t *testing.T) {
	for _, args := range [][]string{{}, {"unknown", "."}, {"check"}, {"build", "--out"}, {"build", "x", "y"}} {
		if code := run(args); code == 0 {
			t.Fatalf("invalid args accepted: %#v", args)
		}
	}
}

func TestOutputMustBeFresh(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	if err := os.MkdirAll(output, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "stale.txt"), []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeOutput(root, output, nil); err == nil {
		t.Fatal("nonempty output silently accepted")
	}
	if err := writeOutput(root, filepath.Join(root, "skills"), nil); err == nil {
		t.Fatal("output would replace skills")
	}
}

func TestOutputRejectsSymlinkAncestor(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "linked")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := writeOutput(root, filepath.Join(link, "out"), nil); err == nil {
		t.Fatal("output through symlink ancestor accepted")
	}
}

func TestReadProjectRejectsSymlinks(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "skills", "demo")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("# demo"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("SKILL.md", filepath.Join(src, "alias.md")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if _, err := readProject(root); err == nil {
		t.Fatal("symlink input accepted")
	}
}

func TestReadProjectChecksSizeBeforeAllocating(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "skills", "big")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, "large.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(maxCLIFileSize + 1); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := readProject(root); err == nil {
		t.Fatal("oversized source file was read")
	}
}
