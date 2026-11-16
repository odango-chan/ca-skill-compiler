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
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("Chromium parity output missing %q", required)
		}
	}
}
