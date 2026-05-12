package providers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wasilak/nim/pkg/provider"
	"github.com/wasilak/nim/pkg/resource"
)

// minimalStateMock implements the subset of state interface used by FileProvider
type minimalStateMock struct{}

// Note: FileProvider.Reconcile expects []provider.ResourceState for state.
// We only need HasResource-like behaviour via the provided slice, so no methods required.

func TestFileProvider_Apply_UsesMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		wantPerm os.FileMode
	}{
		{"explicit 0755", "0755", 0755},
		{"explicit 0600", "0600", 0600},
		{"empty defaults to 0644", "", 0644},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			dest := filepath.Join(tmp, "testfile")
			p := NewFileProvider(tmp)

			addition := resource.ResourceItem{
				Name: dest,
				FileExtra: &resource.FileItemExtra{
					Source:      "(inline)",
					Inline:      "hello",
					Destination: dest,
					Mode:        tc.mode,
				},
			}
			err := p.applyGroupAddition(context.Background(), groupAdditionFrom(addition))
			if err != nil {
				t.Fatalf("applyGroupAddition: %v", err)
			}

			info, err := os.Stat(dest)
			if err != nil {
				t.Fatalf("stat: %v", err)
			}
			if got := info.Mode().Perm(); got != tc.wantPerm {
				t.Errorf("mode = %04o, want %04o", got, tc.wantPerm)
			}
		})
	}
}

func TestFileProvider_Reconcile_DetectsModeDrift(t *testing.T) {
	tmp := t.TempDir()
	dest := filepath.Join(tmp, "managed")
	content := []byte("content")

	// Write with wrong mode (0600 but desired is 0755)
	if err := os.WriteFile(dest, content, 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	item := resource.ResourceItem{
		Name: dest,
		FileExtra: &resource.FileItemExtra{
			Source:      "(inline)",
			Inline:      string(content),
			Destination: dest,
			Mode:        "0755",
		},
	}
	group := resource.ResourceGroup[any]{
		Kind:  "ManagedFile",
		Name:  "myfiles",
		Items: []resource.ResourceItem{item},
	}

	p := NewFileProvider("")
	// Provide existing state so the item is "tracked"
	state := stateWithItem("ManagedFile", "myfiles", dest, p.hashFile(dest))
	plan := p.Reconcile(context.Background(), []resource.ResourceGroup[any]{group}, state)

	if len(plan.Modifications) == 0 {
		t.Fatalf("expected modification for mode drift, got none (additions=%d inSync=%d)", len(plan.Additions), len(plan.InSync))
	}
	if plan.Modifications[0].Changes[0].Diff != "mode changed" {
		t.Errorf("diff = %q, want %q", plan.Modifications[0].Changes[0].Diff, "mode changed")
	}
}

func TestFileProvider_Apply_FixesModeOnModification(t *testing.T) {
	tmp := t.TempDir()
	dest := filepath.Join(tmp, "managed")

	if err := os.WriteFile(dest, []byte("hello"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	p := NewFileProvider(tmp)
	addition := resource.ResourceItem{
		Name: dest,
		FileExtra: &resource.FileItemExtra{
			Source:      "(inline)",
			Inline:      "hello",
			Destination: dest,
			Mode:        "0755",
		},
	}
	if err := p.applyGroupAddition(context.Background(), groupAdditionFrom(addition)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0755 {
		t.Errorf("mode = %04o, want 0755", got)
	}
}

// helpers

func groupAdditionFrom(item resource.ResourceItem) provider.GroupAddition {
	return provider.GroupAddition{Kind: "ManagedFile", Group: "test", Items: []resource.ResourceItem{item}}
}

func stateWithItem(kind, group, name, checksum string) []provider.ResourceState {
	return []provider.ResourceState{{
		Kind:  kind,
		Group: group,
		Items: []resource.ItemState{{Name: name, Checksum: checksum, Status: "present"}},
	}}
}

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
