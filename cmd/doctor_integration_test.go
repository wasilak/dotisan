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
    buildCmd := exec.CommandContext(ctx, "go", "build", "-o", tmpBin, ".")
    buildCmd.Env = env
    buildCmd.Dir = ".."
    if bout, err := buildCmd.CombinedOutput(); err != nil {
        t.Fatalf("go build failed: %v\noutput:\n%s", err, string(bout))
    }
    defer os.Remove(tmpBin)

    // Run the built binary
    runCmd := exec.CommandContext(ctx, tmpBin, "doctor")
    runCmd.Env = env
    outBytes, _ := runCmd.CombinedOutput()
    out := string(outBytes)

    // We accept that the command may exit non-zero (warnings/errors),
    // but still expect it to emit styled output including the checkmark
    // glyph and the palette success sequence.
    if !strings.Contains(out, "✔") {
        t.Fatalf("expected output to contain checkmark glyph, got:\n%s", out)
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
