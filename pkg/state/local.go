package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wasilak/nim/pkg/provider"
)

// LocalBackend implements StateBackend for local JSON file storage.
type LocalBackend struct {
	path string
}

// NewLocalBackend creates a new LocalBackend with the given file path.
func NewLocalBackend(path string) *LocalBackend {
	return &LocalBackend{path: path}
}

// NewLocalBackendWithDefaultPath creates a LocalBackend at the default location.
func NewLocalBackendWithDefaultPath() (*LocalBackend, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	path := filepath.Join(homeDir, ".config", "nim", "state.json")
	return NewLocalBackend(path), nil
}

// Load retrieves the state from the local JSON file.
func (b *LocalBackend) Load(ctx context.Context) (*State, error) {
	// Check if file exists
	if _, err := os.Stat(b.path); os.IsNotExist(err) {
		// Return empty state if file doesn't exist
		return NewState(), nil
	}

	// Read file
	data, err := os.ReadFile(b.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Parse JSON
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Initialize empty resources slice if nil
	if state.Resources == nil {
		state.Resources = []provider.ResourceState{}
	}

	// Normalize item statuses: older state files or providers may not set
	// ItemState.Status. Ensure downstream code can rely on a non-empty
	// status (treat missing as "present"). This avoids surprises in UI
	// rendering and plan/apply logic.
	for ri := range state.Resources {
		for ii := range state.Resources[ri].Items {
			if state.Resources[ri].Items[ii].Status == "" {
				state.Resources[ri].Items[ii].Status = "present"
			}
		}
	}

	return &state, nil
}

// Save persists the state to the local JSON file.
func (b *LocalBackend) Save(ctx context.Context, s *State) error {
	// Ensure directory exists
	dir := filepath.Dir(b.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Normalize missing item statuses before persisting so future loads
	// and other callers can rely on Status being present.
	for ri := range s.Resources {
		for ii := range s.Resources[ri].Items {
			if s.Resources[ri].Items[ii].Status == "" {
				s.Resources[ri].Items[ii].Status = "present"
			}
		}
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to file with restricted permissions (user only)
	if err := os.WriteFile(b.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// Path returns the file path used by this backend.
func (b *LocalBackend) Path() string {
	return b.path
}
