package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wasilak/dotisan/pkg/provider"
)

func TestNewState(t *testing.T) {
	s := NewState()

	if s.Version != StateVersion {
		t.Errorf("Version = %q, want %q", s.Version, StateVersion)
	}

	if s.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}

	if s.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}

	if s.Resources == nil {
		t.Error("Resources is nil, should be empty slice")
	}
}

func TestState_GetResource(t *testing.T) {
	s := NewState()
	s.Resources = []provider.ResourceState{
		{ID: "test-1", Name: "test1"},
		{ID: "test-2", Name: "test2"},
	}

	// Get existing resource
	r, found := s.GetResource("test-1")
	if !found {
		t.Error("GetResource() should find existing resource")
	}
	if r.Name != "test1" {
		t.Errorf("GetResource() name = %q, want %q", r.Name, "test1")
	}

	// Get non-existent resource
	_, found = s.GetResource("nonexistent")
	if found {
		t.Error("GetResource() should not find non-existent resource")
	}
}

func TestState_SetResource_New(t *testing.T) {
	s := NewState()

	s.SetResource(provider.ResourceState{ID: "test-1", Name: "test1"})

	if len(s.Resources) != 1 {
		t.Errorf("len(Resources) = %d, want 1", len(s.Resources))
	}

	if s.Resources[0].ID != "test-1" {
		t.Errorf("Resource ID = %q, want %q", s.Resources[0].ID, "test-1")
	}

	if s.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestState_SetResource_Update(t *testing.T) {
	s := NewState()
	s.SetResource(provider.ResourceState{ID: "test-1", Name: "old-name"})

	s.SetResource(provider.ResourceState{ID: "test-1", Name: "new-name"})

	if len(s.Resources) != 1 {
		t.Errorf("len(Resources) = %d, want 1", len(s.Resources))
	}

	if s.Resources[0].Name != "new-name" {
		t.Errorf("Resource name = %q, want %q", s.Resources[0].Name, "new-name")
	}
}

func TestState_RemoveResource(t *testing.T) {
	s := NewState()
	s.SetResource(provider.ResourceState{ID: "test-1", Name: "test1"})
	s.SetResource(provider.ResourceState{ID: "test-2", Name: "test2"})

	removed := s.RemoveResource("test-1")
	if !removed {
		t.Error("RemoveResource() should return true for existing resource")
	}

	if len(s.Resources) != 1 {
		t.Errorf("len(Resources) = %d, want 1", len(s.Resources))
	}

	// Remove non-existent
	removed = s.RemoveResource("nonexistent")
	if removed {
		t.Error("RemoveResource() should return false for non-existent resource")
	}
}

func TestLocalBackend_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "state.json")

	backend := NewLocalBackend(path)

	// Create and save state
	state := NewState()
	state.SetResource(provider.ResourceState{
		ID:        "brew/core-tools/ripgrep",
		Kind:      "BrewPackages",
		Name:      "core-tools",
		Namespace: "default",
		Version:   "13.0.0",
		Checksum:  "abc123",
	})

	ctx := context.Background()
	err := backend.Save(ctx, state)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("State file was not created")
	}

	// Load state back
	loaded, err := backend.Load(ctx)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loaded.Version != StateVersion {
		t.Errorf("Version = %q, want %q", loaded.Version, StateVersion)
	}

	if len(loaded.Resources) != 1 {
		t.Fatalf("len(Resources) = %d, want 1", len(loaded.Resources))
	}

	if loaded.Resources[0].ID != "brew/core-tools/ripgrep" {
		t.Errorf("Resource ID = %q, want %q", loaded.Resources[0].ID, "brew/core-tools/ripgrep")
	}
}

func TestLocalBackend_Load_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent", "state.json")

	backend := NewLocalBackend(path)

	ctx := context.Background()
	state, err := backend.Load(ctx)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Should return empty state, not error
	if state.Version != StateVersion {
		t.Errorf("Version = %q, want %q", state.Version, StateVersion)
	}

	if len(state.Resources) != 0 {
		t.Errorf("len(Resources) = %d, want 0", len(state.Resources))
	}
}

func TestLocalBackend_Load_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "state.json")

	// Write invalid JSON
	if err := os.WriteFile(path, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	backend := NewLocalBackend(path)

	ctx := context.Background()
	_, err := backend.Load(ctx)
	if err == nil {
		t.Error("Load() should fail with invalid JSON")
	}
}

func TestLocalBackendWithDefaultPath(t *testing.T) {
	backend, err := NewLocalBackendWithDefaultPath()
	if err != nil {
		t.Fatalf("NewLocalBackendWithDefaultPath() failed: %v", err)
	}

	path := backend.Path()
	if path == "" {
		t.Error("Path() returned empty string")
	}

	// Should contain .config/dotisan/state.json
	if !contains(path, ".config/dotisan") || !contains(path, "state.json") {
		t.Errorf("Path() = %q, should contain .config/dotisan and state.json", path)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
