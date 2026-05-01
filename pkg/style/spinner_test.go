package style

import (
	"context"
	"testing"
	"time"
)

// Test that RunWithSpinner respects cancellation from the parent context.
func TestRunWithSpinner_Cancellation(t *testing.T) {
	t.Parallel()

	parent, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Action blocks until context is cancelled and then returns ctx.Err()
	action := func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
			return nil
		}
	}

	// Cancel shortly after starting RunWithSpinner
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := RunWithSpinner(parent, "testing cancellation", action)
	if err == nil {
		t.Fatalf("expected error due to cancellation, got nil")
	}
	if err != context.Canceled && err != context.DeadlineExceeded {
		t.Fatalf("expected context.Canceled or DeadlineExceeded, got: %v", err)
	}
}
