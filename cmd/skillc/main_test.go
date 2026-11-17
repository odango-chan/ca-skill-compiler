package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/odango-chan/ca-skill-compiler/internal/compiler"
)

func runForTest(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestCLIArgumentValidation(t *testing.T) {
	cases := [][]string{
		{},
		{"unknown", "."},
		{"check"},
		{"build", "--out"},
		{"build", "x", "y"},
		{"check", "--request"},
		{"check", "--bogus", "."},
		{"build", "--request", "request.json"},
		{"check", "--request", "request.json", "."},
	}
	for _, args := range cases {
		if code, _, _ := runForTest(args...); code != 2 {
			t.Fatalf("invalid args %#v returned exit code %d, want 2", args, code)
		}
	}
}

func TestHelpExitsZero(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}} {
		if code, stdout, _ := runForTest(args...); code != 0 || !strings.Contains(stdout, "skillc") {
			t.Fatalf("help args %#v returned code %d stdout %q", args, code, stdout)
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
	if err := writeOutput(root, filepath.Join(root, "files", "skills"), nil); err == nil {
		t.Fatal("output would replace files/skills")
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

func TestWriteOutputRejectsUnsafePaths(t *testing.T) {
	for _, unsafe := range []string{"", "skills", "skills/demo", "../skills/demo/file", "skills\\demo", "/skills/demo/file", "lib/other/file", "skills/demo/../x"} {
		if validOutputPath(unsafe) {
			t.Fatalf("unsafe output path accepted: %q", unsafe)
		}
	}
	if !validOutputPath("skills/demo/SKILL.md") {
		t.Fatal("valid output path rejected")
	}
}

func TestPrintDiagnosticsIncludesPositionAndCode(t *testing.T) {
	var buffer bytes.Buffer
	printDiagnostics(&buffer, []compiler.Diagnostic{{
		Severity: "error",
		Phase:    "C_STATIC",
		Code:     "SOURCE_PATH_INVALID",
		Path:     "skills/demo/SKILL.md",
		Line:     12,
		Column:   5,
		Message:  "bad source path",
	}})
	got := buffer.String()
	for _, want := range []string{"skills/demo/SKILL.md:12:5:", "error", "C_STATIC", "SOURCE_PATH_INVALID", "bad source path"} {
		if !strings.Contains(got, want) {
			t.Fatalf("diagnostic output %q missing %q", got, want)
		}
	}
}
