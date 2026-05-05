package ui

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/wasilak/dotisan/pkg/style"
)

// captureOutput temporarily redirects os.Stdout to a pipe and returns a
// function to stop capturing and the captured string.
func captureOutput(f func()) string {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	// run provided function
	f()
	// restore
	_ = w.Close()
	outBytes, _ := io.ReadAll(r)
	os.Stdout = orig
	return string(outBytes)
}

func TestRunWithSpinner_Success(t *testing.T) {
	out := captureOutput(func() {
		ctx := context.Background()
		err := RunWithSpinner(ctx, style.Info, "test success", "cancelled", func(ctx context.Context, publish func(MessageLevel, string)) error {
			// quick successful work
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if strings.Contains(out, "Failed") {
		t.Fatalf("did not expect failure output, got: %q", out)
	}
}

func TestRunWithSpinner_Failure(t *testing.T) {
	out := captureOutput(func() {
		ctx := context.Background()
		err := RunWithSpinner(ctx, style.Info, "test fail", "cancelled", func(ctx context.Context, publish func(MessageLevel, string)) error {
			return errors.New("boom")
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
	if !strings.Contains(out, "Failed") {
		t.Fatalf("expected output to contain 'Failed', got: %q", out)
	}
}

func TestRunWithSpinner_Cancel(t *testing.T) {
	out := captureOutput(func() {
		ctx, cancel := context.WithCancel(context.Background())
		// run spinner in goroutine since we will cancel shortly
		done := make(chan error)
		go func() {
			err := RunWithSpinner(ctx, style.Info, "test cancel", "cancelled", func(ctx context.Context, publish func(MessageLevel, string)) error {
				// wait until context canceled
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(2 * time.Second):
					return nil
				}
			})
			done <- err
		}()
		// give spinner time to start
		time.Sleep(20 * time.Millisecond)
		cancel()
		// wait for run to finish
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatalf("timeout waiting for RunWithSpinner to return after cancel")
		}
	})
	if !strings.Contains(out, "cancelled") {
		t.Fatalf("expected output to contain cancel message 'cancelled', got: %q", out)
	}
}
