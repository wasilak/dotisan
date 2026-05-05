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

// This integration-style test runs runDoctor in-process and asserts the
// output contains the checkmark glyph and the ANSI sequence defined by the
// palette's Success role. Running in-process avoids flakiness of spawning
// subprocesses while still exercising the CLI output with integration tag.
func TestDoctorEmitsColoredCheckmark(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "dotisan-test-home")
	if err != nil {
		t.Fatalf("mkdir temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)
	os.Setenv("HOME", tmpHome)
	os.Setenv("TERM", "xterm-256color")

	// capture stdout/stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	oldOut := os.Stdout
	oldErr := os.Stderr
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = oldOut
		os.Stderr = oldErr
	}()

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
