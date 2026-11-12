package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/odango-chan/ca-skill-compiler/internal/compiler"
)

const (
	maxCLIFileSize  = 2 << 20
	maxCLITotalSize = 16 << 20
	maxCLIFileCount = 512
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) < 2 || (args[0] != "check" && args[0] != "build") {
		fmt.Fprintln(os.Stderr, "usage: skillc check <project> [--skill NAME] | skillc build <project> [--skill NAME] [--out DIRECTORY]")
		return 2
	}
	command := args[0]
	project, out, target := "", "", ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--out", "--skill":
			if i+1 == len(args) {
				fmt.Fprintln(os.Stderr, "missing value for", args[i])
				return 2
			}
			value := args[i+1]
			if args[i] == "--out" {
				out = value
			} else {
				target = value
			}
			i++
		default:
			if strings.HasPrefix(args[i], "-") || project != "" {
				fmt.Fprintln(os.Stderr, "unexpected argument:", args[i])
				return 2
			}
			project = args[i]
		}
	}
	if project == "" {
		fmt.Fprintln(os.Stderr, "project directory is required")
		return 2
	}
	root, err := filepath.Abs(project)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	files, err := readProject(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "input:", err)
		return 1
	}
	req := compiler.BuildRequest{Files: files}
	if target != "" {
		req.Targets = []string{target}
	}
	var result compiler.BuildResult
	if command == "check" {
		result = compiler.BuildResult{Diagnostics: compiler.Check(req)}
		result.Success = true
		for _, d := range result.Diagnostics {
			if d.Severity == "error" {
				result.Success = false
			}
		}
	} else {
		result = compiler.Build(req)
	}
	for _, d := range result.Diagnostics {
		fmt.Fprintf(os.Stderr, "%s:%d: %s %s: %s\n", d.Path, d.Line, d.Severity, d.Code, d.Message)
	}
	if !result.Success {
		return 1
	}
	if command == "check" {
		fmt.Println("Skill project static check OK")
		return 0
	}
	if out == "" {
		out = filepath.Join(root, "output")
	}
	if err := writeOutput(root, out, result.OutputFiles); err != nil {
		fmt.Fprintln(os.Stderr, "output:", err)
		return 1
	}
	fmt.Printf("Skill build complete: %d files\n", len(result.OutputFiles))
	return 0
}

// readProject never reads output/ and never follows symlinks. Source is always
// skills/; imported libraries are an optional sibling lib/ directory.
func readProject(root string) ([]compiler.File, error) {
	var files []compiler.File
	totalSize := int64(0)
	for _, dir := range []string{"skills", "lib"} {
		base := filepath.Join(root, dir)
		info, err := os.Lstat(base)
		if os.IsNotExist(err) && dir == "lib" {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("expected real directory: %s", base)
		}
		err = filepath.WalkDir(base, func(filename string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink is not allowed: %s", filename)
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf("nonregular file: %s", filename)
			}
			rel, err := filepath.Rel(root, filename)
			if err != nil {
				return err
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if info.Size() > maxCLIFileSize {
				return fmt.Errorf("file exceeds %d byte limit: %s", maxCLIFileSize, filename)
			}
			totalSize += info.Size()
			if totalSize > maxCLITotalSize {
				return fmt.Errorf("project exceeds %d byte limit", maxCLITotalSize)
			}
			if len(files) >= maxCLIFileCount {
				return fmt.Errorf("project exceeds %d file limit", maxCLIFileCount)
			}
			b, err := os.ReadFile(filename)
			if err != nil {
				return err
			}
			files = append(files, compiler.File{Path: filepath.ToSlash(rel), Bytes: b})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

// Refuse pre-existing non-empty output instead of accidentally leaving stale
// files from an earlier build in a supposedly complete Skill Package.
func writeOutput(root, dest string, files []compiler.File) error {
	abs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	if err := rejectSymlinkAncestors(abs); err != nil {
		return err
	}
	for _, forbidden := range []string{root, filepath.Join(root, "skills"), filepath.Join(root, "lib")} {
		rel, err := filepath.Rel(forbidden, abs)
		if err == nil && (rel == "." || (forbidden != root && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))) {
			return fmt.Errorf("output path may not replace source or library: %s", dest)
		}
	}
	if info, err := os.Lstat(abs); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output path is not a normal directory: %s", abs)
		}
		existing, err := os.ReadDir(abs)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			return fmt.Errorf("output is nonempty, choose a fresh directory: %s", abs)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, file := range files {
		if strings.HasPrefix(file.Path, "/") || strings.Contains(file.Path, "..") {
			return fmt.Errorf("unsafe output path: %s", file.Path)
		}
		target := filepath.Join(abs, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(target, file.Bytes, 0644); err != nil {
			return err
		}
	}
	return nil
}

func rejectSymlinkAncestors(abs string) error {
	current := filepath.Clean(abs)
	for {
		info, err := os.Lstat(current)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output path traverses a symlink: %s", current)
		}
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}
