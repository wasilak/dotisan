package style

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

// RunWithSpinner runs the provided action while displaying a lightweight
// spinner on TTYs. The spinner implementation deliberately avoids Bubble
// Tea / terminal queries to prevent escape sequences from leaking into the
// user's shell. On non-TTY, it falls back to simple start/done logging.
//
// parent is a parent context that, when cancelled, will cancel the spinner
// action as well. This allows cancellation to propagate from a top-level
// command context (e.g., root command signal handling).
func RunWithSpinner(parent context.Context, title string, action func(ctx context.Context) error) error {
	// Require caller to pass a non-nil parent context so cancellation is
	// explicit. Using context.Background() here would create hidden cancellation
	// sources and defeat signal propagation from the top-level command.
	if parent == nil {
		return fmt.Errorf("parent context is nil")
	}

	if !isTTY() {
		fmt.Fprintf(os.Stdout, "%s... ", title)
		start := time.Now()

		// Create cancellable context derived from parent. Cancellation from the
		// parent (e.g. rootCtx) will propagate here; no per-spinner signal
		// handling is necessary.
		ctx, cancel := context.WithCancel(parent)
		defer cancel()

		err := action(ctx)

		dur := time.Since(start).Truncate(time.Millisecond)
		if err != nil {
			fmt.Fprintf(os.Stdout, "failed: %v\n", err)
			return err
		}
		fmt.Fprintf(os.Stdout, "done (%s)\n", dur)
		return nil
	}

	frames := []string{"|", "/", "-", "\\"}
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	// create cancellable context derived from parent. Cancellation from the
	// parent (e.g. rootCtx) will propagate here; avoid per-spinner signal
	// handlers to prevent duplicated signal listeners.
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- action(ctx)
	}()

	i := 0
	for {
		select {
		case err := <-done:
			// clear the spinner line
			fmt.Fprintf(os.Stdout, "\r%s\r", "                                                                 ")
			if err != nil {
				fmt.Fprintf(os.Stdout, "[✗] %s: %v\n", title, err)
			} else {
				fmt.Fprintf(os.Stdout, "[✓] %s\n", title)
			}
			return err
		case <-ticker.C:
			frame := frames[i%len(frames)]
			fmt.Fprintf(os.Stdout, "\r%s %s", frame, title)
			i++
		}
	}
}

func isTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}
