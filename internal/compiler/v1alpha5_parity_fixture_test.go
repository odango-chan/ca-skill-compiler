package compiler

// Regeneration helper for the checked-in synthetic v1alpha5 parity fixture.
//
// The fixture is generated from the same complete synthetic Skill package the
// compiler tests use, so the CLI/Chromium parity harness cannot silently drift
// from the full-language static pipeline. Regenerate it with:
//
//	WRITE_V1ALPHA5_PARITY_FIXTURE=1 go test ./internal/compiler -run TestWriteV1Alpha5ParityFixture -count=1
//
// The test is a no-op in normal runs.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteV1Alpha5ParityFixture(t *testing.T) {
	if os.Getenv("WRITE_V1ALPHA5_PARITY_FIXTURE") != "1" {
		t.Skip("set WRITE_V1ALPHA5_PARITY_FIXTURE=1 to regenerate web/parity-v1alpha5/trusted-build-request.json")
	}
	request := wp3BuildRequest(t, wp3FixtureOptions{})
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("encode trusted v1alpha5 parity fixture: %v", err)
	}
	target := filepath.Join("..", "..", "web", "parity-v1alpha5", "trusted-build-request.json")
	if err := os.WriteFile(target, raw, 0o644); err != nil {
		t.Fatalf("write %s: %v", target, err)
	}
}
