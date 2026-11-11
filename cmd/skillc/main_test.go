package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCLIArgumentValidation(t *testing.T) {
	for _, args := range [][]string{{}, {"unknown", "."}, {"check"}, {"build", "--out"}, {"build", "x", "y"}} {
		if code := run(args); code == 0 { t.Fatalf("invalid args accepted: %#v", args) }
	}
}

func TestOutputMustBeFresh(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	if err := os.MkdirAll(output, 0755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(output, "stale.txt"), []byte("stale"), 0644); err != nil { t.Fatal(err) }
	if err := writeOutput(root, output, nil); err == nil { t.Fatal("nonempty output silently accepted") }
	if err := writeOutput(root, filepath.Join(root, "skills"), nil); err == nil { t.Fatal("output would replace skills") }
}

func TestReadProjectRejectsSymlinks(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "skills", "demo")
	if err := os.MkdirAll(src, 0755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("# demo"), 0644); err != nil { t.Fatal(err) }
	if err := os.Symlink("SKILL.md", filepath.Join(src, "alias.md")); err != nil { t.Skipf("symlinks unsupported: %v", err) }
	if _, err := readProject(root); err == nil { t.Fatal("symlink input accepted") }
}
