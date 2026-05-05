package ui

import (
	"context"
	"github.com/wasilak/dotisan/pkg/style"
	"testing"
	"time"
)

// TestStartWithContextCancellation verifies that the spinner stops and shows
// a failure when the context is cancelled. This is intentionally lightweight
// and avoids depending on exact terminal output. We ensure the returned stop
// function is idempotent.
func TestStartWithContextCancellation(t *testing.T) {
	s := NewSpinner()

	// use a short-lived context
	ctx, cancel := context.WithCancel(context.Background())

	stop := s.StartWithContext(ctx, style.Info, "testing-cancel", "test cancelled")

	// cancel the context and allow goroutine to run
	cancel()
	time.Sleep(50 * time.Millisecond)

	// calling stop should be safe multiple times
	stop()
	stop()
}
