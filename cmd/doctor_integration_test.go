//go:build integration
// +build integration

package cmd_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wasilak/dotisan/pkg/style"
)

// This integration-style test runs the CLI doctor command in a subprocess
// and asserts that the output contains the checkmark glyph and the ANSI
// sequence defined by the palette's Success role. We run the binary via
// `go run ./cmd doctor` to avoid invoking unexported functions that may
// call os.Exit.
func TestDoctorEmitsColoredCheckmark(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "dotisan-test-home")
	if err != nil {
		t.Fatalf("mkdir temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	// Prepare command with timeout so tests don't hang.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Ensure colors are enabled and CLI reads our isolated HOME
	env := os.Environ()
	env = append(env, "HOME="+tmpHome)
	env = append(env, "TERM=xterm-256color")

	// Build the CLI binary from the repository root and run it. Building first
	// avoids `go run` package resolution issues in tests and produces a stable
	// executable we can run and capture output from.
	tmpBin := filepath.Join(os.TempDir(), fmt.Sprintf("dotisan-test-bin-%d", time.Now().UnixNano()))
    // Build only the cmd package to produce a small binary containing the
    // CLI entrypoint. Building the repository root (".") can attempt to
    // compile unrelated packages and may be killed in constrained test
    // environments. Targeting ./cmd is sufficient for this integration test.
    // Build the cmd package by import path so the command works regardless
    // of the current working directory the test runner uses.
    buildCmd := exec.CommandContext(ctx, "go", "build", "-o", tmpBin, "./cmd")
    buildCmd.Env = env
    // Run the build from the repository root so relative import paths resolve
    // correctly and the produced binary is an executable for the CLI entrypoint.
    buildCmd.Dir = ".."
	if bout, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\noutput:\n%s", err, string(bout))
	}
	defer os.Remove(tmpBin)

    // Execute the CLI via `go run` on the main package. Running `go run` on
    // ./cmd fails because cmd is not a main package. Use the repository root
    // where main.go defines package main that calls cmd.Execute().
    runCmd := exec.CommandContext(ctx, "go", "run", "..", "doctor")
    runCmd.Env = env
    runCmd.Dir = "cmd"
    outBytes, _ := runCmd.CombinedOutput()
    out := string(outBytes)

    // We accept that the command may exit non-zero, but expect styled output.
    // Be tolerant: some environments may alter glyphs or encoding, so accept
    // either the raw checkmark glyph or the StyledIconSuccess which wraps the
    // glyph with the palette's success sequence.
    if !(strings.Contains(out, "✔") || strings.Contains(out, style.StyledIconSuccess)) {
        t.Fatalf("expected output to contain checkmark glyph or styled icon, got:\n%s", out)
    }

    successSeq := style.DefaultColors.Success
    if successSeq == "" {
        t.Fatalf("DefaultColors.Success is empty; palette not initialized in test environment")
    }
    if !strings.Contains(out, successSeq) {
        t.Fatalf("expected output to contain success ANSI sequence %q, output:\n%s", successSeq, out)
    }

    if !strings.Contains(out, style.Reset) {
        t.Fatalf("expected output to contain Reset sequence, output:\n%s", out)
    }
}
