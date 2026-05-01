package state

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

func TestLocalBackendLoadNormalizesMissingStatus(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Create a state JSON with an item missing Status
	now := time.Now().UTC()
	s := State{
		Version:   StateVersion,
		CreatedAt: now,
		UpdatedAt: now,
		Resources: []provider.ResourceState{
			{
				Kind:  "BrewPackages",
				Group: "core-tools",
				Items: []resource.ItemState{{Name: "ripgrep"}},
			},
		},
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	backend := NewLocalBackend(path)
	st, err := backend.Load(context.Background())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(st.Resources) != 1 {
		t.Fatalf("unexpected resources count: %d", len(st.Resources))
	}
	if len(st.Resources[0].Items) != 1 {
		t.Fatalf("unexpected items count: %d", len(st.Resources[0].Items))
	}
	if st.Resources[0].Items[0].Status != "present" {
		t.Fatalf("expected status to be 'present', got '%s'", st.Resources[0].Items[0].Status)
	}
}

// no helpers
