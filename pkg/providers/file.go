// Package providers implements resource providers for dotisan.
//
// Providers are the core abstraction that enables dotisan to manage different
// types of resources:
//   - FileProvider: Manages files and directories (ManagedFile, ManagedDirectory)
//   - BrewProvider: Manages Homebrew packages
//   - NpmProvider: Manages npm packages
//   - GoProvider: Manages Go modules
//   - CargoProvider: Manages Rust crates
//
// Each provider implements the provider.Provider interface.
package providers

import (
	"context"
	"fmt"
	"os"

	"dotisan/pkg/config"
	"dotisan/pkg/diff"
	"dotisan/pkg/provider"
	"dotisan/pkg/resource"
)

// FileProvider implements the Provider interface for ManagedFile and ManagedDirectory resources.
type FileProvider struct {
	// templateContext provides access to template variables
	templateContext *config.TemplateContext

	// diffEngine provides diff generation capabilities
	diffEngine *diff.Engine

	// dotfilesRoot is the path to the dotfiles directory
	dotfilesRoot string
}

// NewFileProvider creates a new FileProvider.
func NewFileProvider(ctx *config.TemplateContext, engine *diff.Engine, dotfilesRoot string) *FileProvider {
	return &FileProvider{
		templateContext: ctx,
		diffEngine:      engine,
		dotfilesRoot:    dotfilesRoot,
	}
}

// Name returns the provider name.
func (p *FileProvider) Name() string {
	return "file"
}

// Available checks if the provider can operate on this system.
// File operations are always available if we have filesystem access.
func (p *FileProvider) Available() (bool, string) {
	// Check if we can read the dotfiles root
	if p.dotfilesRoot != "" {
		if _, err := os.Stat(p.dotfilesRoot); err != nil {
			return false, fmt.Sprintf("dotfiles root not accessible: %s", err)
		}
	}

	// Check if we have write access to home directory for state file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Sprintf("cannot determine home directory: %s", err)
	}

	// Try to stat home directory
	if _, err := os.Stat(homeDir); err != nil {
		return false, fmt.Sprintf("home directory not accessible: %s", err)
	}

	return true, "filesystem operations available"
}

// Reconcile compares the desired resources with the current system state
// and returns a plan of changes needed.
func (p *FileProvider) Reconcile(desired []resource.Resource, state []provider.ResourceState) provider.Plan {
	// TODO: Implement in subtask 7.2 and 7.4
	return provider.Plan{}
}

// Apply executes the given plan.
func (p *FileProvider) Apply(ctx context.Context, plan provider.Plan) error {
	// TODO: Implement in subtask 7.3 and 7.5
	return nil
}

// Import discovers an existing resource on the system and returns its state.
func (p *FileProvider) Import(ctx context.Context, id string) (provider.ResourceState, error) {
	// TODO: Implement import functionality
	return provider.ResourceState{}, fmt.Errorf("import not yet implemented")
}
