package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/odango-chan/ca-skill-compiler/internal/compiler"
	"github.com/odango-chan/ca-skill-compiler/internal/projectinput"
)

const usageText = `skillc - CA-SL reference compiler CLI (CA-SL v1alpha5)

Usage:
  skillc check   <PROJECT>                       static check a Skill source project
  skillc check   --request <REQUEST.json>        static check an explicit BuildRequest
  skillc build   <PROJECT> --out <DIR>           compile and write the artifact tree
  skillc build   --request <REQUEST.json> --out <DIR>
  skillc inspect <PROJECT>                       inspect dependency snapshots and bindings
  skillc inspect --request <REQUEST.json>
  skillc build-development --request <REQUEST.json> --out <DIR>
                                                 Development Build (Compile -> Link -> Build)
  skillc verify-development --request <REQUEST.json> --result <RESULT.json>
                                                 verify an actual Development artifact against
                                                 its original request (full deterministic rebuild)
  skillc version                                 print the Compiler product version
  skillc help                                    show this help

Options:
  --request FILE   read a complete request JSON document instead of a project
                   directory (strict decoding; dependencies and trusted build
                   context come from that document)
  --result FILE    read a complete Development BuildResult JSON document
  --out DIR        artifact output directory; must be absent or empty. Defaults
                   to <PROJECT>/output in project mode and is required with
                   --request
  --skill NAME     select one Skill package; repeat for multiple targets
  --json           write machine-readable JSON to stdout instead of human text

Language target:
  connectionagent.io/v1alpha5   the only supported CA-SL language target

API versions:
  build request        connectionagent.io/compiler-build-request/v1alpha5
  build result         connectionagent.io/compiler-build-result/v1alpha5
  dependency inspection connectionagent.io/compiler-dependency-inspection/v1alpha5
  development request  connectionagent.io/compiler-development-build-request/v1alpha1
  development result   connectionagent.io/compiler-development-build-result/v1alpha1
  development artifact connectionagent.io/compiler-development-artifact/v1alpha1

Project layout:
  skills/...                       author Skill source (or files/skills/...)
  snapshots/<dependencyId>/...     immutable dependency snapshot file sets
  trusted.json                     trusted dependency bindings, deployment
                                   bundles and Ingress/Schema artifact bindings

Exit codes:
  0  success
  1  compilation/build artifact failure (diagnostics were reported)
  2  usage, input, or I/O error before or after compilation

The CLI never infers a missing dependency, digest or Owner attribution: an
incomplete trusted project is reported and fails closed. v1alpha4 requests are
rejected like every other unsupported language target.
A Development artifact is LINKED / BUILT / NOT_ADMITTED: it is never an
execution authorization or a Host admission decision.
`

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

type options struct {
	request string
	result  string
	out     string
	skills  []string
	json    bool
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usageText)
		return 2
	}
	command := args[0]
	if command == "help" || command == "-h" || command == "--help" {
		fmt.Fprint(stdout, usageText)
		return 0
	}
	if command == "version" {
		if len(args) > 1 {
			fmt.Fprintln(stderr, "skillc: version takes no arguments")
			return 2
		}
		fmt.Fprintf(stdout, "skillc %s\n", compiler.CompilerVersion)
		fmt.Fprintf(stdout, "language target: %s\n", compiler.V1Alpha5Target)
		fmt.Fprintf(stdout, "build request apiVersion: %s\n", compiler.BuildRequestAPIVersion)
		fmt.Fprintf(stdout, "build result apiVersion: %s\n", compiler.BuildResultAPIVersion)
		fmt.Fprintf(stdout, "dependency inspection apiVersion: %s\n", compiler.DependencyInspectionAPIVersion)
		fmt.Fprintf(stdout, "development request apiVersion: %s\n", compiler.DevelopmentBuildRequestAPIVersion)
		fmt.Fprintf(stdout, "development result apiVersion: %s\n", compiler.DevelopmentBuildResultAPIVersion)
		fmt.Fprintf(stdout, "development artifact apiVersion: %s\n", compiler.DevelopmentArtifactAPIVersion)
		return 0
	}
	if command == "build-development" || command == "verify-development" {
		return runDevelopmentCommand(command, args[1:], stdout, stderr)
	}
	if command != "check" && command != "build" && command != "inspect" {
		fmt.Fprintf(stderr, "skillc: unknown command %q\n\n%s", command, usageText)
		return 2
	}
	opts, positionals, err := parseOptions(command, args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usageText)
			return 0
		}
		fmt.Fprintln(stderr, "skillc:", err)
		return 2
	}
	req, projectRoot, err := loadRequest(opts, positionals)
	if err != nil {
		fmt.Fprintln(stderr, "skillc:", err)
		return 2
	}
	if len(opts.skills) > 0 {
		if len(req.Targets) > 0 {
			fmt.Fprintln(stderr, "skillc: --skill cannot be combined with targets already present in --request")
			return 2
		}
		req.Targets = append([]string(nil), opts.skills...)
	}

	switch command {
	case "check":
		return runCheck(req, opts, stdout, stderr)
	case "inspect":
		return runInspect(req, opts, stdout, stderr)
	default:
		return runBuild(req, opts, projectRoot, stdout, stderr)
	}
}

// parseOptions accepts documented flags before or after the positional PROJECT
// argument while delegating flag syntax to the standard library flag package.
func parseOptions(command string, args []string) (options, []string, error) {
	var opts options
	set := flag.NewFlagSet("skillc "+command, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.StringVar(&opts.request, "request", "", "BuildRequest JSON file")
	set.BoolVar(&opts.json, "json", false, "machine-readable output")
	var skills stringList
	set.Var(&skills, "skill", "selected Skill package")
	if command == "build" {
		set.StringVar(&opts.out, "out", "", "artifact output directory")
	}

	valueFlags := map[string]bool{"--request": true, "-request": true, "--skill": true, "-skill": true}
	if command == "build" {
		valueFlags["--out"] = true
		valueFlags["-out"] = true
	}
	var flagArgs, positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			flagArgs = append(flagArgs, arg)
			if valueFlags[arg] {
				if i+1 >= len(args) {
					return opts, nil, fmt.Errorf("missing value for %s", arg)
				}
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
			continue
		}
		positionals = append(positionals, arg)
	}
	if err := set.Parse(flagArgs); err != nil {
		return opts, nil, err
	}
	opts.skills = skills
	return opts, positionals, nil
}

type stringList []string

func (l *stringList) String() string { return strings.Join(*l, ",") }

func (l *stringList) Set(value string) error {
	if value == "" {
		return fmt.Errorf("Skill name must not be empty")
	}
	*l = append(*l, value)
	return nil
}

// loadRequest produces the frozen BuildRequest input. It either decodes an
// explicit request document or reads the documented project layout; it never
// falls back from one to the other.
func loadRequest(opts options, positionals []string) (compiler.BuildRequest, string, error) {
	if opts.request != "" {
		if len(positionals) > 0 {
			return compiler.BuildRequest{}, "", fmt.Errorf("a PROJECT argument must not be combined with --request")
		}
		raw, err := os.ReadFile(opts.request)
		if err != nil {
			return compiler.BuildRequest{}, "", fmt.Errorf("read BuildRequest: %w", err)
		}
		req, err := compiler.DecodeBuildRequestJSON(raw)
		if err != nil {
			return compiler.BuildRequest{}, "", fmt.Errorf("decode BuildRequest %s: %w", opts.request, err)
		}
		return req, "", nil
	}
	if len(positionals) != 1 {
		return compiler.BuildRequest{}, "", fmt.Errorf("exactly one PROJECT directory or --request FILE is required")
	}
	root := positionals[0]
	info, err := os.Stat(root)
	if err != nil {
		return compiler.BuildRequest{}, "", fmt.Errorf("project directory: %w", err)
	}
	if !info.IsDir() {
		return compiler.BuildRequest{}, "", fmt.Errorf("project path is not a directory: %s", root)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return compiler.BuildRequest{}, "", err
	}
	req, err := projectinput.Load(absolute)
	if err != nil {
		return compiler.BuildRequest{}, "", err
	}
	return req, absolute, nil
}

func runDevelopmentCommand(command string, args []string, stdout, stderr io.Writer) int {
	var opts options
	set := flag.NewFlagSet("skillc "+command, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.StringVar(&opts.request, "request", "", "Development BuildRequest JSON file")
	set.BoolVar(&opts.json, "json", false, "machine-readable output")
	if command == "build-development" {
		set.StringVar(&opts.out, "out", "", "artifact output directory")
	} else {
		set.StringVar(&opts.result, "result", "", "Development BuildResult JSON file")
	}
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usageText)
			return 0
		}
		fmt.Fprintln(stderr, "skillc:", err)
		return 2
	}
	if set.NArg() > 0 {
		fmt.Fprintln(stderr, "skillc: "+command+" takes no positional arguments")
		return 2
	}
	if opts.request == "" {
		fmt.Fprintln(stderr, "skillc: --request is required with "+command)
		return 2
	}
	raw, err := os.ReadFile(opts.request)
	if err != nil {
		fmt.Fprintln(stderr, "skillc: read Development BuildRequest:", err)
		return 2
	}
	req, err := compiler.DecodeDevelopmentBuildRequestJSON(raw)
	if err != nil {
		fmt.Fprintln(stderr, "skillc: decode Development BuildRequest:", err)
		return 2
	}
	if command == "build-development" {
		if opts.out == "" {
			fmt.Fprintln(stderr, "skillc: --out is required with build-development")
			return 2
		}
		result := compiler.BuildDevelopment(req)
		if result.Diagnostics == nil {
			result.Diagnostics = []compiler.Diagnostic{}
		}
		if opts.json {
			if err := writeJSON(stdout, result); err != nil {
				fmt.Fprintln(stderr, "skillc:", err)
				return 2
			}
		} else {
			printDiagnostics(stderr, result.Diagnostics)
		}
		if !result.Success {
			return 1
		}
		if err := writeOutput("", opts.out, result.OutputFiles); err != nil {
			fmt.Fprintln(stderr, "output:", err)
			return 1
		}
		if !opts.json {
			fmt.Fprintf(stdout, "Development build complete: %s / %s / %s / %s, %d file(s), artifact %s -> %s\n",
				result.SourceStatus, result.InterfaceLinkStatus, result.ArtifactStatus, result.ExecutionStatus,
				len(result.OutputFiles), result.ArtifactDigest, opts.out)
		}
		return 0
	}
	if opts.result == "" {
		fmt.Fprintln(stderr, "skillc: --result is required with verify-development")
		return 2
	}
	resultRaw, err := os.ReadFile(opts.result)
	if err != nil {
		fmt.Fprintln(stderr, "skillc: read Development BuildResult:", err)
		return 2
	}
	result, err := compiler.DecodeDevelopmentBuildResultJSON(resultRaw)
	if err != nil {
		fmt.Fprintln(stderr, "skillc: decode Development BuildResult:", err)
		return 2
	}
	if err := compiler.VerifyDevelopment(req, result); err != nil {
		fmt.Fprintln(stderr, "skillc: Development artifact verification failed:", err)
		return 1
	}
	if opts.json {
		if err := writeJSON(stdout, map[string]any{"success": true, "artifactDigest": result.ArtifactDigest, "executionStatus": result.ExecutionStatus}); err != nil {
			fmt.Fprintln(stderr, "skillc:", err)
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "Development artifact verification OK: %s (%s)\n", result.ArtifactDigest, result.ExecutionStatus)
	}
	return 0
}

func runCheck(req compiler.BuildRequest, opts options, stdout, stderr io.Writer) int {
	diagnostics := compiler.Check(req)
	if diagnostics == nil {
		diagnostics = []compiler.Diagnostic{}
	}
	success := !hasErrors(diagnostics)
	if opts.json {
		if err := writeJSON(stdout, diagnostics); err != nil {
			fmt.Fprintln(stderr, "skillc:", err)
			return 2
		}
	} else {
		printDiagnostics(stderr, diagnostics)
		if success {
			fmt.Fprintln(stdout, "Skill project static check OK")
		}
	}
	if !success {
		return 1
	}
	return 0
}

func runBuild(req compiler.BuildRequest, opts options, projectRoot string, stdout, stderr io.Writer) int {
	result := compiler.Build(req)
	if result.Diagnostics == nil {
		result.Diagnostics = []compiler.Diagnostic{}
	}
	if opts.json {
		if err := writeJSON(stdout, result); err != nil {
			fmt.Fprintln(stderr, "skillc:", err)
			return 2
		}
	} else {
		printDiagnostics(stderr, result.Diagnostics)
	}
	if !result.Success {
		return 1
	}
	if err := compiler.VerifyBuildResult(result); err != nil {
		fmt.Fprintln(stderr, "artifact:", err)
		return 1
	}
	out := opts.out
	if out == "" {
		if projectRoot == "" {
			fmt.Fprintln(stderr, "skillc: --out is required with --request")
			return 2
		}
		out = filepath.Join(projectRoot, "output")
	}
	if err := writeOutput(projectRoot, out, result.OutputFiles); err != nil {
		fmt.Fprintln(stderr, "output:", err)
		return 1
	}
	if !opts.json {
		fmt.Fprintf(stdout, "Skill build complete: %d file(s), %d dependenc(ies) -> %s\n", len(result.OutputFiles), len(result.DependencyClosure), out)
	}
	return 0
}

func runInspect(req compiler.BuildRequest, opts options, stdout, stderr io.Writer) int {
	inspection := compiler.InspectDependencies(req)
	diagnostics := inspection.Diagnostics
	if diagnostics == nil {
		diagnostics = []compiler.Diagnostic{}
	}
	success := !hasErrors(diagnostics)
	if opts.json {
		inspection.Diagnostics = diagnostics
		if err := writeJSON(stdout, inspection); err != nil {
			fmt.Fprintln(stderr, "skillc:", err)
			return 2
		}
	} else {
		printDiagnostics(stderr, diagnostics)
		if success {
			printInspection(stdout, inspection)
		}
	}
	if !success {
		return 1
	}
	return 0
}

func printInspection(w io.Writer, inspection compiler.DependencyInspection) {
	fmt.Fprintf(w, "Dependency inspection OK: %d callable(s), %d schema(s), %d ingress operation(s)\n",
		len(inspection.Callables), len(inspection.Schemas), len(inspection.IngressOperations))
	for _, callable := range inspection.Callables {
		fmt.Fprintf(w, "  %-12s %s/%s owner=%s effect=%s\n", callable.Kind, callable.DependencyID, callable.ID, callable.Owner, callable.Effect)
	}
	for _, schema := range inspection.Schemas {
		fmt.Fprintf(w, "  %-12s %s/%s owner=%s\n", "schema", schema.DependencyID, schema.ID, schema.Owner)
	}
	for _, operation := range inspection.IngressOperations {
		fmt.Fprintf(w, "  %-12s %s/%s owner=%s event=%s/%s/%s %s %s\n",
			operation.Kind, operation.DependencyID, operation.ID, operation.Owner,
			operation.EventSource, operation.EventService, operation.EventOperation,
			operation.Method, operation.Path)
	}
}

func printDiagnostics(w io.Writer, diagnostics []compiler.Diagnostic) {
	for _, diagnostic := range diagnostics {
		location := diagnostic.Path
		if location == "" {
			location = "<request>"
		}
		if diagnostic.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, diagnostic.Line)
			if diagnostic.Column > 0 {
				location = fmt.Sprintf("%s:%d", location, diagnostic.Column)
			}
		}
		phase := diagnostic.Phase
		if phase == "" {
			phase = "-"
		}
		fmt.Fprintf(w, "%s: %s %s %s: %s\n", location, diagnostic.Severity, phase, diagnostic.Code, diagnostic.Message)
	}
}

func hasErrors(diagnostics []compiler.Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			return true
		}
	}
	return false
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

// writeOutput writes a successful BuildResult tree. It refuses symlinks,
// nonempty or source-replacing destinations and unsafe paths, so a partial or
// stale Skill package is never presented as a complete build.
func writeOutput(projectRoot, dest string, files []compiler.File) error {
	abs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	if err := rejectSymlinkAncestors(abs); err != nil {
		return err
	}
	if projectRoot != "" {
		root, err := filepath.Abs(projectRoot)
		if err != nil {
			return err
		}
		for _, forbidden := range []string{
			root,
			filepath.Join(root, "skills"),
			filepath.Join(root, "files"),
			filepath.Join(root, "files", "skills"),
		} {
			rel, err := filepath.Rel(forbidden, abs)
			if err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))) {
				return fmt.Errorf("output path may not replace source: %s", dest)
			}
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
		if !validOutputPath(file.Path) {
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

func validOutputPath(value string) bool {
	// The v1alpha5 artifact has exactly one top-level manifest; compiled files
	// live under skills/<skill>/compiled/.
	if value == compiler.V1Alpha5ArtifactManifestPath || value == "host-admission-requirements.json" {
		return true
	}
	if value == "" || len(value) > 240 || !strings.HasPrefix(value, "skills/") ||
		path.Clean(value) != value || strings.ContainsAny(value, "\\:\x00") {
		return false
	}
	parts := strings.Split(value, "/")
	if len(parts) < 3 {
		return false
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}
