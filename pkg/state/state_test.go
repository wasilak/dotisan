package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
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

func TestState_GetResourceGroup(t *testing.T) {
	s := NewState()
	s.Resources = []provider.ResourceState{
		{Kind: "BrewPackages", Group: "core-tools", Items: []resource.ItemState{{Name: "ripgrep"}}},
		{Kind: "BrewPackages", Group: "dev-tools", Items: []resource.ItemState{{Name: "jq"}}},
	}

	// Get existing resource
	r, found := s.GetResourceGroup("BrewPackages", "core-tools")
	if !found {
		t.Error("GetResourceGroup() should find existing resource")
	}
	if len(r.Items) != 1 || r.Items[0].Name != "ripgrep" {
		t.Errorf("GetResourceGroup() items = %v, want [ripgrep]", r.Items)
	}

	// Get non-existent resource
	_, found = s.GetResourceGroup("BrewPackages", "nonexistent")
	if found {
		t.Error("GetResourceGroup() should not find non-existent resource")
	}
}

func TestState_SetResourceGroup_New(t *testing.T) {
	s := NewState()

	s.SetResourceGroup(provider.ResourceState{
		Kind:      "BrewPackages",
		Group:     "core-tools",
		Items:     []resource.ItemState{{Name: "ripgrep"}},
		Namespace: "default",
	})

	if len(s.Resources) != 1 {
		t.Errorf("len(Resources) = %d, want 1", len(s.Resources))
	}

	if s.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestState_SetResourceGroup_Update(t *testing.T) {
	s := NewState()
	s.SetResourceGroup(provider.ResourceState{
		Kind:      "BrewPackages",
		Group:    "core-tools",
		Items:    []resource.ItemState{{Name: "ripgrep"}},
		Namespace: "default",
	})

	s.SetResourceGroup(provider.ResourceState{
		Kind:      "BrewPackages",
		Group:    "core-tools",
		Items:    []resource.ItemState{{Name: "ripgrep"}, {Name: "jq"}},
		Namespace: "default",
	})

	if len(s.Resources) != 1 {
		t.Errorf("len(Resources) = %d, want 1", len(s.Resources))
	}

	// Should merge items
	if len(s.Resources[0].Items) != 2 {
		t.Errorf("Resource items = %d, want 2", len(s.Resources[0].Items))
	}
}

func TestState_RemoveResourceGroup(t *testing.T) {
	s := NewState()
	s.SetResourceGroup(provider.ResourceState{
		Kind:      "BrewPackages",
		Group:    "core-tools",
		Items:    []resource.ItemState{{Name: "ripgrep"}},
		Namespace: "default",
	})
	s.SetResourceGroup(provider.ResourceState{
		Kind:      "BrewPackages",
		Group:    "dev-tools",
		Items:    []resource.ItemState{{Name: "jq"}},
		Namespace: "default",
	})

	removed := s.RemoveResourceGroup("BrewPackages", "core-tools")
	if !removed {
		t.Error("RemoveResourceGroup() should return true for existing resource")
	}

	if len(s.Resources) != 1 {
		t.Errorf("len(Resources) = %d, want 1", len(s.Resources))
	}

	// Remove non-existent
	removed = s.RemoveResourceGroup("BrewPackages", "nonexistent")
	if removed {
		t.Error("RemoveResourceGroup() should return false for non-existent resource")
	}
}

func TestLocalBackend_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "state.json")

	backend := NewLocalBackend(path)

	// Create and save state
	state := NewState()
	state.SetResourceGroup(provider.ResourceState{
		Kind:      "BrewPackages",
		Group:     "core-tools",
		Namespace: "default",
		Items: []resource.ItemState{
			{Name: "ripgrep", Version: "15.0.0"},
		},
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

	if loaded.Resources[0].Group != "core-tools" {
		t.Errorf("Resource Group = %q, want %q", loaded.Resources[0].Group, "core-tools")
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