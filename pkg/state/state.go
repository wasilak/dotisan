// Package state provides the StateBackend interface and implementations for dotisan.
//
// State represents the current state of all managed resources as tracked by dotisan.
// The state file is stored at ~/.dotisan/state.json (local) or in S3-compatible storage.
//
// Example state.json:
//
//	{
//	  "version": "1.0",
//	  "created_at": "2024-01-15T10:30:00Z",
//	  "updated_at": "2024-01-15T10:30:00Z",
//	  "resources": [
//	    {
//	      "id": "brew/core-tools/ripgrep",
//	      "kind": "BrewPackages",
//	      "name": "core-tools",
//	      "namespace": "default",
//	      "version": "13.0.0",
//	      "checksum": "sha256:abc123..."
//	    }
//	  ]
//	}
package state

import (
	"context"
	"time"

	"github.com/wasilak/dotisan/pkg/provider"
)

// State represents the complete state of managed resources.
type State struct {
	// Version is the state file format version
	Version string `json:"version"`

	// CreatedAt is when the state file was first created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the state file was last modified
	UpdatedAt time.Time `json:"updated_at"`

	// Resources contains the state of all managed resources
	Resources []provider.ResourceState `json:"resources"`
}

// StateVersion is the current state file format version.
const StateVersion = "1.0"

// NewState creates a new empty State with proper timestamps.
func NewState() *State {
	now := time.Now().UTC()
	return &State{
		Version:   StateVersion,
		CreatedAt: now,
		UpdatedAt: now,
		Resources: []provider.ResourceState{},
	}
}

// GetResource retrieves a resource state by ID.
func (s *State) GetResource(id string) (provider.ResourceState, bool) {
	for _, r := range s.Resources {
		if r.ID == id {
			return r, true
		}
	}
	return provider.ResourceState{}, false
}

// SetResource adds or updates a resource state.
func (s *State) SetResource(r provider.ResourceState) {
	// Update the UpdatedAt timestamp
	s.UpdatedAt = time.Now().UTC()

	// Check if resource already exists
	for i, existing := range s.Resources {
		if existing.ID == r.ID {
			// Update existing resource
			s.Resources[i] = r
			return
		}
	}

	// Add new resource
	s.Resources = append(s.Resources, r)
}

// RemoveResource removes a resource state by ID.
// Returns true if the resource was found and removed.
func (s *State) RemoveResource(id string) bool {
	for i, r := range s.Resources {
		if r.ID == id {
			// Remove by swapping with last and truncating
			s.Resources[i] = s.Resources[len(s.Resources)-1]
			s.Resources = s.Resources[:len(s.Resources)-1]
			s.UpdatedAt = time.Now().UTC()
			return true
		}
	}
	return false
}

// StateBackend is the interface for state storage implementations.
type StateBackend interface {
	// Load retrieves the current state from storage.
	Load(ctx context.Context) (*State, error)

	// Save persists the state to storage.
	Save(ctx context.Context, s *State) error
}
