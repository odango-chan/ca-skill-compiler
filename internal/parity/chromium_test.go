// Package parity holds opt-in cross-runtime acceptance tests that need a real
// browser. They are skipped unless explicitly enabled, so the default Go test
// suite stays hermetic.
package parity

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestChromiumWorkerParity runs the real headless-Chromium Worker parity harness
// (scripts/chromium-parity.mjs) that compares Native Go and browser projections.
// Enable with CHROMIUM_PARITY=1; CHROME_PATH overrides the browser location.
func TestChromiumWorkerParity(t *testing.T) {
	if os.Getenv("CHROMIUM_PARITY") != "1" {
		t.Skip("set CHROMIUM_PARITY=1 to run the real Chromium Worker parity test")
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Fatalf("node is required for the Chromium parity harness: %v", err)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("node", filepath.Join("scripts", "chromium-parity.mjs"))
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	text := string(output)
	t.Logf("chromium parity output:\n%s", text)
	if err != nil {
		t.Fatalf("Chromium Worker parity failed: %v", err)
	}
	for _, required := range []string{
		"ok: check projection identical",
		"ok: build.diagnostics projection identical",
		"ok: inspect projection identical",
		"ok: studio diagnostics/artifact read-only integration identical",
		"ok: Chromium Worker artifact integrity verification PASS",
		"ok: successful-Build artifact byte parity",
		"ok: studio-success.artifactPaths projection identical",
		"PASS: Native and real Chromium Worker projections are identical",
		// v1alpha5 candidate-target WP8 slice: Check diagnostics, successful-Build
		// artifact bytes and linked Ingress metadata must be compared in the real
		// Chromium Worker, not merely in Native Go.
		"ok: v1alpha5 valid Check returns zero diagnostics",
		"ok: v1alpha5 valid Build succeeds with deterministic artifact",
		"ok: v1alpha5 trusted Build succeeds with dependency closure",
		"ok: v1alpha5 invalid WAIT_* diagnostics positioned and deterministic",
		"ok: v1alpha5 Chromium Worker VerifyArtifact PASS",
		"ok: v1alpha5 check projection identical",
		"ok: v1alpha5 build.success projection identical",
		"ok: v1alpha5 build.apiVersion projection identical",
		"ok: v1alpha5 build.diagnostics projection identical",
		"ok: v1alpha5 build.dependencyClosure projection identical",
		"ok: v1alpha5 build.outputFiles.paths projection identical",
		"ok: v1alpha5 successful-Build artifact byte parity",
		"ok: v1alpha5 inspect projection identical",
		"ok: v1alpha5 invalid-diagnostics projection identical",
		"ok: v1alpha5 trusted check projection identical",
		"ok: v1alpha5 trusted build.success projection identical",
		"ok: v1alpha5 trusted build.apiVersion projection identical",
		"ok: v1alpha5 trusted build.diagnostics projection identical",
		"ok: v1alpha5 trusted dependencyClosure projection identical",
		"ok: v1alpha5 trusted build.outputFiles.paths projection identical",
		"ok: v1alpha5 trusted successful-Build artifact byte parity",
		"ok: v1alpha5 trusted linked Ingress metadata identical",
		"PASS: Native and real Chromium Worker v1alpha5 artifact parity",
		"ok: Development LINKED/BUILT/NOT_ADMITTED byte parity",
		"ok: Development reordered input produces identical artifact bytes",
		"ok: Development failure diagnostics identical between Native and Chromium Worker",
		"PASS: Native and real Chromium Worker Development Build parity",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("Chromium parity output missing %q", required)
		}
	}
}
