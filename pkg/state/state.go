// Package state provides the StateBackend interface and implementations for dotisan.
//
// State represents the current state of all managed resources as tracked by dotisan.
// The state file is stored at ~/.config/dotisan/state.json (local) or in S3-compatible storage.
//
// Example state.json (hierarchical 3-level structure):
//
//	{
//	  "version": "1.0",
//	  "created_at": "2024-01-15T10:30:00Z",
//	  "updated_at": "2024-01-15T10:30:00Z",
//	  "resources": [
//	    {
//	      "kind": "BrewPackages",
//	      "group": "core-tools",
//	      "namespace": "default",
//	      "items": [
//	        {"name": "ripgrep", "version": "14.0.0"},
//	        {"name": "htop", "version": "3.2.0"}
//	      ]
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
// Uses hierarchical 3-level structure: Kind -> Group -> Items
type State struct {
	// Version is the state file format version
	Version string `json:"version"`

	// CreatedAt is when the state file was first created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the state file was last modified
	UpdatedAt time.Time `json:"updated_at"`

	// Resources contains the state of all managed resource groups
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

// GetResourceGroup retrieves a resource group state by kind and group name.
func (s *State) GetResourceGroup(kind, group string) (provider.ResourceState, bool) {
	for _, r := range s.Resources {
		if r.Kind == kind && r.Group == group {
			return r, true
		}
	}
	return provider.ResourceState{}, false
}

// SetResourceGroup adds or updates a resource group state, merging items with existing groups.
func (s *State) SetResourceGroup(r provider.ResourceState) {
	// Update the UpdatedAt timestamp
	s.UpdatedAt = time.Now().UTC()

	// Check if resource already exists
	for i, existing := range s.Resources {
		if existing.Kind == r.Kind && existing.Group == r.Group {
			// Merge items - add new items that don't exist yet
			existingItems := make(map[string]bool)
			for _, item := range s.Resources[i].Items {
				existingItems[item.Name] = true
			}
			for _, newItem := range r.Items {
				if !existingItems[newItem.Name] {
					s.Resources[i].Items = append(s.Resources[i].Items, newItem)
				}
			}
			return
		}
	}

	// Add new resource
	s.Resources = append(s.Resources, r)
}

// RemoveResourceGroup removes a resource group state by kind and group name.
// Returns true if the resource was found and removed.
func (s *State) RemoveResourceGroup(kind, group string) bool {
	for i, r := range s.Resources {
		if r.Kind == kind && r.Group == group {
			// Remove by swapping with last and truncating
			s.Resources[i] = s.Resources[len(s.Resources)-1]
			s.Resources = s.Resources[:len(s.Resources)-1]
			s.UpdatedAt = time.Now().UTC()
			return true
		}
	}
	return false
}

// GetResource retrieves a resource state by ID (legacy method, use GetResourceGroup).
// Deprecated: Use GetResourceGroup(kind, group) instead.
func (s *State) GetResource(id string) (provider.ResourceState, bool) {
	for _, r := range s.Resources {
		// Legacy ID format was "Kind/group[item]" - try to match
		if r.Group == id {
			return r, true
		}
	}
	return provider.ResourceState{}, false
}

// SetResource adds or updates a resource state (legacy method, use SetResourceGroup).
// Deprecated: Use SetResourceGroup(r) instead.
func (s *State) SetResource(r provider.ResourceState) {
	s.SetResourceGroup(r)
}

// RemoveResource removes a resource state by ID (legacy method, use RemoveResourceGroup).
// Deprecated: Use RemoveResourceGroup(kind, group) instead.
func (s *State) RemoveResource(id string) bool {
	// Legacy removal - will match on Group field
	for i, r := range s.Resources {
		if r.Group == id {
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
