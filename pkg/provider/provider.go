// Package provider defines the Provider interface and registry for dotisan.
//
// Providers are the core abstraction that enables dotisan to manage different
// types of resources (files, packages, etc.). Each provider implements the
// Provider interface and registers itself with the global registry.
//
// Example provider implementations:
//   - FileProvider: Manages files and directories
//   - BrewProvider: Manages Homebrew packages
//   - NpmProvider: Manages npm packages
//   - GoProvider: Manages Go modules
//   - CargoProvider: Manages Rust crates
package provider

import (
	"context"

	"github.com/wasilak/dotisan/pkg/resource"
)

// Plan represents the changes needed to reconcile desired state with actual state.
type Plan struct {
	// Additions are resources that need to be created
	Additions []resource.Resource

	// Modifications are resources that need to be updated
	Modifications []Modification

	// Removals are resources that need to be deleted
	Removals []resource.Resource

	// InSync are resources that match desired state
	InSync []resource.Resource

	// Drifted are resources that have changed outside of dotisan's management
	Drifted []Drift
}

// Modification represents a change to an existing resource.
type Modification struct {
	// Resource is the desired state
	Resource resource.Resource

	// OldState is the current state from the system
	OldState ResourceState

	// NewState is the desired state to be applied
	NewState ResourceState

	// Diff is a human-readable description of the changes
	Diff string
}

// Drift represents a resource that has changed outside of dotisan's management.
type Drift struct {
	// Resource is the managed resource
	Resource resource.Resource

	// ExpectedState is what dotisan thinks the state should be
	ExpectedState ResourceState

	// ActualState is what's actually on the system
	ActualState ResourceState

	// Description explains what changed
	Description string
}

// ResourceState represents the state of a resource as tracked by dotisan.
type ResourceState struct {
	// ID uniquely identifies the resource
	ID string `json:"id"`

	// Kind is the resource type
	Kind string `json:"kind"`

	// Name is the resource name
	Name string `json:"name"`

	// Namespace is the resource namespace
	Namespace string `json:"namespace"`

	// Version is the resource version (if applicable)
	Version string `json:"version,omitempty"`

	// Checksum is a hash of the resource content
	Checksum string `json:"checksum,omitempty"`

	// SourceHash is a hash of the source (for files)
	SourceHash string `json:"source_hash,omitempty"`

	// DestHash is a hash of the destination (for files)
	DestHash string `json:"dest_hash,omitempty"`

	// Extra contains provider-specific state data
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// Provider is the interface implemented by all resource providers.
// Each provider knows how to manage a specific type of resource.
type Provider interface {
	// Name returns the provider name (e.g., "brew", "file", "npm")
	Name() string

	// Available checks if the provider can operate on this system.
	// Returns true if available, false with a descriptive message if not.
	Available() (bool, string)

	// Reconcile compares the desired resources with the current system state
	// and returns a plan of changes needed to reach the desired state.
	// The state parameter contains the previously saved state for these resources.
	Reconcile(desired []resource.Resource, state []ResourceState) Plan

	// Apply executes the given plan, making actual changes to the system.
	// Returns an error if any operation fails.
	Apply(ctx context.Context, plan Plan) error

	// Import discovers an existing resource on the system and returns its state.
	// This is used by the `state import` command to bring unmanaged resources
	// under dotisan's control.
	Import(ctx context.Context, id string) (ResourceState, error)
}
