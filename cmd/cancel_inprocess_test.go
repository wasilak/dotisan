package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/wasilak/dotisan/pkg/style"
	"github.com/wasilak/dotisan/pkg/ui"
)

// TestCancelInProcess runs a temporary command in-process that starts a
// spinner and blocks until the context is cancelled. We cancel the context
// and assert the spinner emitted the cancel message. This avoids flakiness
// of spawning subprocesses and is suitable for CI.
func TestCancelInProcess(t *testing.T) {
	// Create a temporary command that uses the command's context
	testCmd := &cobra.Command{
		Use:   "test-block-local",
		Short: "local test block command",
		RunE: func(cmd *cobra.Command, args []string) error {
			sp := ui.NewSpinner()
			stop, _ := sp.StartWithContext(cmd.Context(), style.Info, "Running local test block...", "test cancelled")
			<-cmd.Context().Done()
			// allow spinner goroutine to process
			time.Sleep(20 * time.Millisecond)
			stop()
			return nil
		},
	}

	// Register and ensure removal after test
	rootCmd.AddCommand(testCmd)
	defer rootCmd.RemoveCommand(testCmd)

	// Capture stdout
	oldOut := os.Stdout
	oldErr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = oldOut
		os.Stderr = oldErr
	}()

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	// Execute the command with a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rootCmd.SetArgs([]string{"test-block-local"})
	errc := make(chan error, 1)
	go func() {
		errc <- rootCmd.ExecuteContext(ctx)
	}()

	// Let the command start and spinner spin a bit
	time.Sleep(100 * time.Millisecond)

	// Cancel and wait for command to exit
	cancel()
	select {
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for command to exit")
	case <-errc:
	}

	// Close writer so reader goroutine finishes
	_ = w.Close()
	<-done

	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("test cancelled")) {
		t.Fatalf("expected cancel message in output, got: %q", out)
	}
}
