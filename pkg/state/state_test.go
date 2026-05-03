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
		{Kind: resource.KindHomeBrewPackages, Group: "core-tools", Items: []resource.ItemState{{Name: "ripgrep"}}},
		{Kind: resource.KindHomeBrewPackages, Group: "dev-tools", Items: []resource.ItemState{{Name: "jq"}}},
	}

	// Get existing resource
	r, found := s.GetResourceGroup(resource.KindHomeBrewPackages, "core-tools")
	if !found {
		t.Error("GetResourceGroup() should find existing resource")
	}
	if len(r.Items) != 1 || r.Items[0].Name != "ripgrep" {
		t.Errorf("GetResourceGroup() items = %v, want [ripgrep]", r.Items)
	}

	// Get non-existent resource
	_, found = s.GetResourceGroup(resource.KindHomeBrewPackages, "nonexistent")
	if found {
		t.Error("GetResourceGroup() should not find non-existent resource")
	}
}

func TestState_SetResourceGroup_New(t *testing.T) {
	s := NewState()

	s.SetResourceGroup(provider.ResourceState{
		Kind:      resource.KindHomeBrewPackages,
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
		Kind:      resource.KindHomeBrewPackages,
		Group:     "core-tools",
		Items:     []resource.ItemState{{Name: "ripgrep"}},
		Namespace: "default",
	})

	s.SetResourceGroup(provider.ResourceState{
		Kind:      resource.KindHomeBrewPackages,
		Group:     "core-tools",
		Items:     []resource.ItemState{{Name: "ripgrep"}, {Name: "jq"}},
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
		Kind:      resource.KindHomeBrewPackages,
		Group:     "core-tools",
		Items:     []resource.ItemState{{Name: "ripgrep"}},
		Namespace: "default",
	})
	s.SetResourceGroup(provider.ResourceState{
		Kind:      resource.KindHomeBrewPackages,
		Group:     "dev-tools",
		Items:     []resource.ItemState{{Name: "jq"}},
		Namespace: "default",
	})

	removed := s.RemoveResourceGroup(resource.KindHomeBrewPackages, "core-tools")
	if !removed {
		t.Error("RemoveResourceGroup() should return true for existing resource")
	}

	if len(s.Resources) != 1 {
		t.Errorf("len(Resources) = %d, want 1", len(s.Resources))
	}

	// Remove non-existent
	removed = s.RemoveResourceGroup(resource.KindHomeBrewPackages, "nonexistent")
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
		Kind:      resource.KindHomeBrewPackages,
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

func TestState_MoveItem(t *testing.T) {
	s := NewState()

	// Set up initial state with two groups
	s.SetResourceGroup(provider.ResourceState{
		Kind:      resource.KindHomeBrewPackages,
		Group:     "core-tools",
		Namespace: "default",
		Items: []resource.ItemState{
			{Name: "ripgrep", Version: "14.0.0"},
			{Name: "podman", Version: "4.9.0"},
		},
	})
	s.SetResourceGroup(provider.ResourceState{
		Kind:      resource.KindHomeBrewPackages,
		Group:     "homebrew-packages",
		Namespace: "default",
		Items: []resource.ItemState{
			{Name: "zsh", Version: "5.9"},
		},
	})

	// Move podman from core-tools to homebrew-packages
	moved, ok := s.MoveItem(resource.KindHomeBrewPackages, "core-tools", "podman", resource.KindHomeBrewPackages, "homebrew-packages", "podman")
	if !ok {
		t.Fatal("MoveItem() should return true for valid move")
	}
	if moved.Items[0].Name != "podman" {
		t.Errorf("Moved item name = %q, want %q", moved.Items[0].Name, "podman")
	}

	// Verify podman was removed from source group
	srcGroup, exists := s.GetResourceGroup(resource.KindHomeBrewPackages, "core-tools")
	if !exists {
		t.Fatal("Source group should still exist")
	}
	for _, item := range srcGroup.Items {
		if item.Name == "podman" {
			t.Error("podman should be removed from core-tools group")
		}
	}
	if len(srcGroup.Items) != 1 || srcGroup.Items[0].Name != "ripgrep" {
		t.Errorf("core-tools items = %v, want [ripgrep]", srcGroup.Items)
	}

	// Verify podman was added to destination group
	dstGroup, exists := s.GetResourceGroup(resource.KindHomeBrewPackages, "homebrew-packages")
	if !exists {
		t.Fatal("Destination group should exist")
	}
	found := false
	for _, item := range dstGroup.Items {
		if item.Name == "podman" {
			found = true
			break
		}
	}
	if !found {
		t.Error("podman should be in homebrew-packages group")
	}
}

func TestState_MoveItem_ToNewGroup(t *testing.T) {
	s := NewState()

	// Set up initial state with one group
	s.SetResourceGroup(provider.ResourceState{
		Kind:      resource.KindHomeBrewPackages,
		Group:     "core-tools",
		Namespace: "default",
		Items: []resource.ItemState{
			{Name: "podman", Version: "4.9.0"},
		},
	})

	// Move podman to a new group that doesn't exist yet
	_, ok := s.MoveItem(resource.KindHomeBrewPackages, "core-tools", "podman", resource.KindHomeBrewPackages, "new-group", "podman")
	if !ok {
		t.Fatal("MoveItem() should return true for valid move to new group")
	}

	// Verify source group was removed (empty after move)
	_, exists := s.GetResourceGroup(resource.KindHomeBrewPackages, "core-tools")
	if exists {
		t.Error("core-tools group should be removed since it's now empty")
	}

	// Verify destination group was created
	dstGroup, exists := s.GetResourceGroup(resource.KindHomeBrewPackages, "new-group")
	if !exists {
		t.Fatal("new-group should exist")
	}
	if len(dstGroup.Items) != 1 || dstGroup.Items[0].Name != "podman" {
		t.Errorf("new-group items = %v, want [podman]", dstGroup.Items)
	}
}

func TestState_MoveItem_NotFound(t *testing.T) {
	s := NewState()

	// Set up initial state
	s.SetResourceGroup(provider.ResourceState{
		Kind:      resource.KindHomeBrewPackages,
		Group:     "core-tools",
		Namespace: "default",
		Items: []resource.ItemState{
			{Name: "ripgrep", Version: "14.0.0"},
		},
	})

	// Try to move non-existent item
	_, ok := s.MoveItem(resource.KindHomeBrewPackages, "core-tools", "nonexistent", resource.KindHomeBrewPackages, "other-group", "nonexistent")
	if ok {
		t.Error("MoveItem() should return false for non-existent item")
	}

	// Try to move from non-existent group
	_, ok = s.MoveItem(resource.KindHomeBrewPackages, "nonexistent", "ripgrep", resource.KindHomeBrewPackages, "other-group", "ripgrep")
	if ok {
		t.Error("MoveItem() should return false for non-existent group")
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
