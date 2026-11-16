package compiler

// WP2 acceptance guard: exactly one YAML parser is allowed. No legacy yaml.v3
// import may exist in production or Compiler packages.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoLegacyYAMLParser(t *testing.T) {
	module, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if strings.Contains(string(module), "gopkg.in/yaml.v3") {
		t.Fatalf("legacy gopkg.in/yaml.v3 must not be a module dependency")
	}
	root := "../.."
	offenders := make([]string, 0)
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "single_parser_test.go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(content), "gopkg.in/yaml.v3") {
			offenders = append(offenders, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk source tree: %v", err)
	}
	if len(offenders) != 0 {
		t.Fatalf("legacy YAML parser imports found: %v", offenders)
	}
}
