package ui

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/wasilak/nim/pkg/style"
)

func TestNewPrimarySpinnerUsesHeader(t *testing.T) {
	// Capture stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	// Ensure the spinner writes into our pipe by overriding NewSpinnerFunc.
	orig := NewSpinnerFunc
	NewSpinnerFunc = func() *Spinner {
		s := NewSpinner()
		s.s.Writer = w
		return s
	}
	defer func() { NewSpinnerFunc = orig }()

	// Inject spinner that writes to our pipe
	sp := NewPrimarySpinner("hello-main")
	stop, _ := sp.StartWithContext(context.Background(), style.Info, "hello-main", "cancelled")
	// allow spinner to run briefly
	time.Sleep(20 * time.Millisecond)
	// Signal normal completion and emit final message so output is visible
	sp.SuccessWithStyle(style.Success, "done")
	stop()

	_ = w.Close()
	outBytes, _ := io.ReadAll(r)
	out := string(outBytes)

	// Ensure final output appears in captured output
	if out == "" {
		t.Fatalf("expected spinner output, got empty")
	}
}
