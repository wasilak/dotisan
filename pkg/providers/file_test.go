package providers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wasilak/nim/pkg/resource"
)

// minimalStateMock implements the subset of state interface used by FileProvider
type minimalStateMock struct{}

// Note: FileProvider.Reconcile expects []provider.ResourceState for state.
// We only need HasResource-like behaviour via the provided slice, so no methods required.

func TestFileProvider_Reconcile_EmitsWarningForExistingDestination(t *testing.T) {
	tmp := t.TempDir()
	dest := filepath.Join(tmp, ".zshrc")
	if err := os.WriteFile(dest, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write dest: %v", err)
	}

	// Build desired resource group with one ManagedFile item pointing to dest
	item := resource.ResourceItem{
		Name:      "zshrc",
		FileExtra: &resource.FileItemExtra{Destination: dest},
	}
	group := resource.ResourceGroup[any]{
		Kind:  "ManagedFile",
		Name:  "myfiles",
		Items: []resource.ResourceItem{item},
	}

	// empty state (resource not tracked)
	// pass nil state (no saved state) to indicate resource is not tracked
	p := NewFileProvider("")
	plan := p.Reconcile(context.Background(), []resource.ResourceGroup[any]{group}, nil)

	// Expect an addition
	if len(plan.Additions) == 0 {
		t.Fatalf("expected additions when destination exists and resource not in state")
	}

	// Expect a warning
	if len(plan.Warnings) == 0 {
		t.Fatalf("expected warning when destination exists and resource not in state")
	}
	w := plan.Warnings[0]
	if !strings.Contains(w.Message, "Destination file already exists") {
		t.Fatalf("unexpected warning message: %v", w.Message)
	}
	if !strings.Contains(w.Suggestion, "nim state import") {
		t.Fatalf("unexpected suggestion: %v", w.Suggestion)
	}
}
