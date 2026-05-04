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

// CargoProvider implements the Provider interface for Cargo packages
type CargoProvider struct{}

// NewCargoProvider creates a new CargoProvider.
func NewCargoProvider() *CargoProvider {
	return &CargoProvider{}
}

// Name returns the provider name.
func (p *CargoProvider) Name() string {
	return "cargo"
}

// Available checks if cargo is available on this system.
func (p *CargoProvider) Available() (bool, string) {
	if path := cmdutil.CheckExecutable("cargo"); path == "" {
		return false, "cargo not found in PATH; install Rust from https://rustup.rs/"
	}
	return true, "cargo found"
}

// Reconcile compares the desired resource groups with the current system state.
func (p *CargoProvider) Reconcile(ctx context.Context,
	desired []resource.ResourceGroup,
	state []provider.ResourceState,
) provider.GroupPlan {
	return provider.BaseReconcile(resource.KindCargoPackages, desired, state, p.getInstalledPackages(ctx), nil)
}

func (p *CargoProvider) getInstalledPackages(ctx context.Context) map[string]string {
	if ctx == nil {
		slog.Warn("cargo getInstalledPackages called with nil context; returning empty set")
		return make(map[string]string)
	}
	// List installed crates
	stdout, _, err := cmdutil.RunSimpleFn(ctx, "cargo", "install", "--list")
	if err != nil {
		slog.Warn("cargo getInstalledPackages failed", "err", err)
		return make(map[string]string)
	}

	installed := make(map[string]string)
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Format: "crate_name v1.0.0:"
		if strings.Contains(line, " v") && strings.HasSuffix(line, ":") {
			parts := strings.Split(line, " ")
			if len(parts) >= 2 {
				name := parts[0]
				version := strings.TrimSuffix(parts[1], ":")
				version = strings.TrimPrefix(version, "v")
				installed[name] = version
			}
		}
	}
	return installed
}

// Apply executes the given GroupPlan
func (p *CargoProvider) Apply(ctx context.Context, plan provider.GroupPlan) error {
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

func (p *CargoProvider) applyGroupAddition(ctx context.Context, addition provider.GroupAddition) error {
	for _, item := range addition.Items {
		crate := item.Name
		if item.Version != "" {
			crate = fmt.Sprintf("%s@%s", item.Name, item.Version)
		}
		if _, stderr, err := cmdutil.RunSimpleFn(ctx, "cargo", "install", crate); err != nil {
			return fmt.Errorf("failed to install %s: %s: %w", item.Name, stderr, err)
		}
	}
	return nil
}

func (p *CargoProvider) applyGroupRemoval(ctx context.Context, removal provider.GroupRemoval) error {
	for _, item := range removal.Items {
		if _, stderr, err := cmdutil.RunSimpleFn(ctx, "cargo", "uninstall", item.Name); err != nil {
			return fmt.Errorf("failed to uninstall %s: %s: %w", item.Name, stderr, err)
		}
	}
	return nil
}

func (p *CargoProvider) applyGroupModification(ctx context.Context, modification provider.GroupModification) error {
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

// Import is not supported for cargo (use ImportItem)
func (p *CargoProvider) Import(ctx context.Context, group string) (provider.ResourceState, error) {
	return provider.ResourceState{}, fmt.Errorf("use ImportItem to import specific cargo crates")
}

// ImportItem imports a specific cargo crate
func (p *CargoProvider) ImportItem(ctx context.Context, group string, item string) (provider.ResourceState, error) {
	// Check if crate is installed
	installed := p.getInstalledPackages(ctx)
	if _, isInstalled := installed[item]; !isInstalled {
		return provider.ResourceState{}, fmt.Errorf("crate %s is not installed", item)
	}

	return provider.ResourceState{
		Kind:  resource.KindCargoPackages,
		Group: group,
		Items: []resource.ItemState{
			{
				Name:    item,
				Version: installed[item],
				Status:  "present",
			},
		},
	}, nil
}
