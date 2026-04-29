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

// GroupPlan represents the changes needed to reconcile desired state with actual state.
// Organized by resource groups (3-level hierarchy: Kind -> Group -> Items)
type GroupPlan struct {
	// Additions are groups/items that need to be created
	Additions []GroupAddition

	// Modifications are groups that need item-level updates
	Modifications []GroupModification

	// Removals are groups/items that need to be deleted
	Removals []GroupRemoval

	// Cleanup are items that exist in state but not in config or system.
	// These will be removed from state only (no system changes).
	Cleanup []GroupCleanup

	// InSync are groups that match desired state
	InSync []GroupState

	// Drifted are items that have changed outside of dotisan's management
	Drifted []ItemDrift

	// Warnings are provider-generated advisory messages that do not block apply
	Warnings []PlanWarning
}

// GroupAddition represents items to add within a resource group
type GroupAddition struct {
	Kind  string
	Group string
	Items []resource.ResourceItem
}

// GroupModification represents changes within an existing group
type GroupModification struct {
	Kind    string
	Group   string
	Changes []ItemChange
}

// ItemChange represents a change to a specific item
type ItemChange struct {
	ItemName string
	OldState resource.ItemState
	NewState resource.ItemState
	Diff     string
}

// GroupRemoval represents items to remove from a group
type GroupRemoval struct {
	Kind  string
	Group string
	Items []resource.ResourceItem
}

// GroupCleanup represents items that exist in state but not in config or system.
// These will be removed from state only (no system changes).
type GroupCleanup struct {
	Kind   string
	Group  string
	Items  []resource.ResourceItem
	Reason string // e.g., "not_in_config_and_not_installed"
}

// GroupState represents a group that is in sync
type GroupState struct {
	Kind    string
	Group   string
	Items   []resource.ItemState
	Version string
}

// ItemDrift represents an item that has drifted from expected state
type ItemDrift struct {
	Kind          string
	Group         string
	Item          string
	ExpectedState resource.ItemState
	ActualState   resource.ItemState
	Description   string
	Diff          string
}

// PlanWarning represents a non-blocking advisory produced during reconcile.
// It can point to a resource and optionally include a suggestion (copy-pasteable).
type PlanWarning struct {
	// GroupID is an optional kind/group identifier (e.g., "BrewPackages/core-tools")
	GroupID string

	// ItemID is an optional item identifier (e.g., "ripgrep")
	ItemID string

	// Severity indicates importance: "warning" or "info"
	Severity string

	// Message is a human-friendly description of the issue
	Message string

	// Suggestion is an optional copy-pasteable command or snippet
	Suggestion string
}

// ResourceState represents the state of a resource group as tracked by dotisan.
// Uses 3-level hierarchy: Kind -> Group -> Items
type ResourceState struct {
	// Kind is the resource type (e.g., "BrewPackages")
	Kind string `json:"kind"`

	// Group is the resource group name (e.g., "core-tools")
	Group string `json:"group"`

	// Namespace is the resource namespace
	Namespace string `json:"namespace"`

	// Items are the individual items within this group
	Items []resource.ItemState `json:"items"`

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

	// Reconcile compares the desired resource groups with the current system state
	// and returns a plan of changes needed to reach the desired state.
	// The state parameter contains the previously saved state for these resources.
	Reconcile(desired []resource.ResourceGroup, state []ResourceState) GroupPlan

	// Apply executes the given plan, making actual changes to the system.
	// Returns an error if any operation fails.
	Apply(ctx context.Context, plan GroupPlan) error

	// Import discovers an existing resource on the system and returns its state.
	// This is used by the `state import` command to bring unmanaged resources
	// under dotisan's control.
	Import(ctx context.Context, group string) (ResourceState, error)

	// ImportItem imports a specific item from a list-based resource.
	// The group identifies the resource group (e.g., "core-tools").
	// The item identifies the specific item (e.g., "ripgrep").
	// Returns the ResourceState for the imported item.
	ImportItem(ctx context.Context, group string, item string) (ResourceState, error)
}
