//go:build integration
// +build integration

package cmd_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestApplyCancellation starts a small test binary that runs a blocking
// command (test-block) and then sends an interrupt to ensure the process
// cancels and prints the configured cancel message.
func TestApplyCancellation(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "dotisan-test-home")
	if err != nil {
		t.Fatalf("mkdir temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	env := os.Environ()
	env = append(env, "HOME="+tmpHome)
	env = append(env, "TERM=xterm-256color")

	// Determine repository root by walking up until we find go.mod.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoRoot := wd
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			break
		}
		repoRoot = parent
	}

	// Build the cmd package first with the integration tag to fetch modules
	// and include the test-only command.
	tmpBin := filepath.Join(os.TempDir(), "dotisan-test-bin-apply-cancel")
	buildCmd := exec.CommandContext(ctx, "go", "build", "-tags=integration", "-o", tmpBin, "./cmd")
	buildCmd.Env = env
	buildCmd.Dir = repoRoot
	if bout, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\noutput:\n%s", err, string(bout))
	}
	defer os.Remove(tmpBin)

	// Now run the main program using `go run` from the repository root. This
	// will execute main.go which calls cmd.Execute(), registering the
	// integration-only test-block command (because we built with -tags=integration).
	runCmd := exec.CommandContext(ctx, "go", "run", "-tags=integration", "./main.go", "test-block")
	runCmd.Env = env
	runCmd.Dir = repoRoot

	// Capture combined stdout+stderr
	var outBuf bytes.Buffer
	runCmd.Stdout = &outBuf
	runCmd.Stderr = &outBuf

	if err := runCmd.Start(); err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	proc := runCmd.Process
	if proc == nil {
		t.Fatalf("process did not start")
	}

	// Wait for the test-block command to start and emit its running message.
	// `go run` may first print module download progress; wait up to 8s for
	// the child process to print the spinner or running text before sending
	// the interrupt.
	started := false
	waitUntil := time.Now().Add(8 * time.Second)
	for time.Now().Before(waitUntil) {
		out := outBuf.String()
		if strings.Contains(out, "Running test block") || strings.Contains(out, "Running test block...") {
			started = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !started {
		// If we didn't detect the running message, proceed but note it in the
		// test output; the process may still be running but slower due to
		// module download. Fall back to a short sleep.
		time.Sleep(500 * time.Millisecond)
	}

	// Send interrupt
	if err := proc.Signal(os.Interrupt); err != nil {
		t.Fatalf("failed to send interrupt: %v", err)
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() { done <- runCmd.Wait() }()
	select {
	case <-ctx.Done():
		t.Fatalf("test timed out")
	case err := <-done:
		_ = err
	}

	out := outBuf.String()
	if !strings.Contains(out, "test cancelled") {
		t.Fatalf("expected output to contain cancel message, got: %s", out)
	}
}
