package providers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/wasilak/dotisan/pkg/cmdutil"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

// NpmProvider implements the Provider interface for npm packages
type NpmProvider struct{}

// NewNpmProvider creates a new NpmProvider.
func NewNpmProvider() *NpmProvider {
	return &NpmProvider{}
}

// Name returns the provider name.
func (p *NpmProvider) Name() string {
	return "npm"
}

// Available checks if npm is available on this system.
func (p *NpmProvider) Available() (bool, string) {
	if path := cmdutil.CheckExecutable("npm"); path == "" {
		return false, "npm not found in PATH; install Node.js from https://nodejs.org/"
	}
	return true, "npm found"
}

// Reconcile compares the desired resource groups with the current system state.
func (p *NpmProvider) Reconcile(ctx context.Context,
	desired []resource.ResourceGroup,
	state []provider.ResourceState,
) provider.GroupPlan {
	return provider.BaseReconcile(resource.KindNpmPackages, desired, state, p.getInstalledPackages(ctx), nil)
}

func (p *NpmProvider) getInstalledPackages(ctx context.Context) map[string]string {
	if ctx == nil {
		slog.Warn("npm getInstalledPackages called with nil context; returning empty set")
		return make(map[string]string)
	}
	stdout, _, err := cmdutil.RunSimple(ctx, "npm", "list", "-g", "--depth=0", "--json")
	if err != nil {
		slog.Warn("npm getInstalledPackages failed", "err", err)
		return make(map[string]string)
	}

	// Simple parsing - look for package names
	installed := make(map[string]string)
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "\"") && strings.Contains(line, "\": {") {
			// Extract package name
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 && parts[1] != "dependencies" && parts[1] != "name" {
				installed[parts[1]] = ""
			}
		}
	}
	return installed
}

// Apply executes the given GroupPlan
func (p *NpmProvider) Apply(ctx context.Context, plan provider.GroupPlan) error {
	for _, addition := range plan.Additions {
		if err := p.applyGroupAddition(ctx, addition); err != nil {
			return fmt.Errorf("failed to add to %s: %w", addition.Group, err)
		}
	}

	for _, removal := range plan.Removals {
		if err := p.applyGroupRemoval(ctx, removal); err != nil {
			return fmt.Errorf("failed to remove from %s: %w", removal.Group, err)
		}
	}

	for _, modification := range plan.Modifications {
		if err := p.applyGroupModification(ctx, modification); err != nil {
			return fmt.Errorf("failed to modify %s: %w", modification.Group, err)
		}
	}

	return nil
}

func (p *NpmProvider) applyGroupAddition(ctx context.Context, addition provider.GroupAddition) error {
	for _, item := range addition.Items {
		pkg := item.Name
		if item.Version != "" {
			pkg = fmt.Sprintf("%s@%s", item.Name, item.Version)
		}
		if _, stderr, err := cmdutil.RunSimple(ctx, "npm", "install", "-g", pkg); err != nil {
			return fmt.Errorf("failed to install %s: %s: %w", item.Name, stderr, err)
		}
	}
	return nil
}

func (p *NpmProvider) applyGroupRemoval(ctx context.Context, removal provider.GroupRemoval) error {
	for _, item := range removal.Items {
		if _, stderr, err := cmdutil.RunSimple(ctx, "npm", "uninstall", "-g", item.Name); err != nil {
			return fmt.Errorf("failed to uninstall %s: %s: %w", item.Name, stderr, err)
		}
	}
	return nil
}

func (p *NpmProvider) applyGroupModification(ctx context.Context, modification provider.GroupModification) error {
	return p.applyGroupAddition(ctx, provider.GroupAddition{
		Kind:  modification.Kind,
		Group: modification.Group,
		Items: func() []resource.ResourceItem {
			var items []resource.ResourceItem
			for _, change := range modification.Changes {
				items = append(items, resource.ResourceItem{
					Name:    change.ItemName,
					Version: change.NewState.Version,
				})
			}
			return items
		}(),
	})
}

// Import not supported for npm provider
func (p *NpmProvider) Import(ctx context.Context, group string) (provider.ResourceState, error) {
	return provider.ResourceState{}, fmt.Errorf("import not supported for provider npm")
}
