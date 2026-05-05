package cmd

import (
    "context"
    "io"
    "os"
    "strings"
    "testing"
    "time"

    "github.com/wasilak/dotisan/pkg/style"
)

// Unit test that captures stdout and runs runDoctor directly. This avoids
// spawning subprocesses which can behave inconsistently under `go test` in
// some environments and still verifies the colored checkmark and ANSI
// sequences are emitted.
func TestRunDoctorEmitsColoredCheckmark(t *testing.T) {
    // isolate HOME
    tmpHome, err := os.MkdirTemp("", "dotisan-test-home")
    if err != nil {
        t.Fatalf("mkdir temp home: %v", err)
    }
    defer os.RemoveAll(tmpHome)
    os.Setenv("HOME", tmpHome)

    // capture stdout
    r, w, err := os.Pipe()
    if err != nil {
        t.Fatalf("pipe: %v", err)
    }
    old := os.Stdout
    os.Stdout = w
    defer func() { os.Stdout = old }()

    // run doctor with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    _ = runDoctor(ctx)

    // close writer and read output
    _ = w.Close()
    outBytes, _ := io.ReadAll(r)
    out := string(outBytes)

    if !(strings.Contains(out, "✔") || strings.Contains(out, style.StyledIconSuccess)) {
        t.Fatalf("expected output to contain checkmark glyph or styled icon, got:\n%s", out)
    }
    if !strings.Contains(out, style.DefaultColors.Success) {
        t.Fatalf("expected output to contain success ANSI sequence %q, output:\n%s", style.DefaultColors.Success, out)
    }
    if !strings.Contains(out, style.Reset) {
        t.Fatalf("expected output to contain Reset sequence, output:\n%s", out)
    }
}
