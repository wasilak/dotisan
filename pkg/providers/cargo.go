package providers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/wasilak/nim/pkg/cmdutil"
	"github.com/wasilak/nim/pkg/provider"
	"github.com/wasilak/nim/pkg/resource"
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
	desired []resource.ResourceGroup[any],
	state []provider.ResourceState,
) provider.GroupPlan {
	// Cargo stores crate names with underscores (e.g. fd_find) even when the
	// source crate uses hyphens (fd-find). Normalize to underscores for lookup.
	normalize := func(name string) string {
		return strings.ReplaceAll(name, "-", "_")
	}
	return provider.BaseReconcile(resource.KindCargoPackages, desired, state, p.getInstalledPackages(ctx), normalize)
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

// InstalledForKind implements provider.CoverageProvider.
func (p *CargoProvider) InstalledForKind(ctx context.Context, kind string) (map[string]string, error) {
	if ctx == nil || kind != resource.KindCargoPackages {
		return nil, nil
	}
	return p.getInstalledPackages(ctx), nil
}

// Apply executes the given GroupPlan
func (p *CargoProvider) Apply(ctx context.Context, plan provider.GroupPlan) ([]provider.ApplyItemResult, error) {
	var results []provider.ApplyItemResult
	for _, addition := range plan.Additions {
		results = append(results, p.applyGroupAddition(ctx, addition)...)
	}
	for _, removal := range plan.Removals {
		results = append(results, p.applyGroupRemoval(ctx, removal)...)
	}
	for _, modification := range plan.Modifications {
		results = append(results, p.applyGroupModification(ctx, modification)...)
	}
	return results, nil
}

func (p *CargoProvider) applyGroupAddition(ctx context.Context, addition provider.GroupAddition) []provider.ApplyItemResult {
	crates := make([]string, 0, len(addition.Items))
	crateToName := make(map[string]string, len(addition.Items))
	for _, item := range addition.Items {
		crate := item.Name
		if item.Version != "" {
			crate = fmt.Sprintf("%s@%s", item.Name, item.Version)
		}
		crates = append(crates, crate)
		crateToName[crate] = item.Name
	}
	failed := batchWithFallback(crates, func(ns []string) error {
		args := append([]string{"install"}, ns...)
		_, stderr, err := cmdutil.RunSimpleFn(ctx, "cargo", args...)
		if err != nil {
			if len(ns) == 1 {
				return fmt.Errorf("failed to install %s: %s: %w", ns[0], stderr, err)
			}
			return err
		}
		return nil
	})
	var results []provider.ApplyItemResult
	for i, item := range addition.Items {
		r := provider.ApplyItemResult{Kind: addition.Kind, Group: addition.Group, Item: item.Name, Op: "add"}
		if err, bad := failed[crates[i]]; bad {
			r.Err = err
		}
		results = append(results, r)
	}
	return results
}

func (p *CargoProvider) applyGroupRemoval(ctx context.Context, removal provider.GroupRemoval) []provider.ApplyItemResult {
	names := make([]string, 0, len(removal.Items))
	for _, item := range removal.Items {
		names = append(names, item.Name)
	}
	failed := batchWithFallback(names, func(ns []string) error {
		args := append([]string{"uninstall"}, ns...)
		_, stderr, err := cmdutil.RunSimpleFn(ctx, "cargo", args...)
		if err != nil {
			if len(ns) == 1 {
				return fmt.Errorf("failed to uninstall %s: %s: %w", ns[0], stderr, err)
			}
			return err
		}
		return nil
	})
	var results []provider.ApplyItemResult
	for _, item := range removal.Items {
		r := provider.ApplyItemResult{Kind: removal.Kind, Group: removal.Group, Item: item.Name, Op: "remove"}
		if err, bad := failed[item.Name]; bad {
			r.Err = err
		}
		results = append(results, r)
	}
	return results
}

func (p *CargoProvider) applyGroupModification(ctx context.Context, modification provider.GroupModification) []provider.ApplyItemResult {
	items := make([]resource.ResourceItem, 0, len(modification.Changes))
	for _, change := range modification.Changes {
		items = append(items, resource.ResourceItem{
			Name:    change.ItemName,
			Version: change.NewState.Version,
		})
	}
	results := p.applyGroupAddition(ctx, provider.GroupAddition{
		Kind:  modification.Kind,
		Group: modification.Group,
		Items: items,
	})
	for i := range results {
		results[i].Op = "modify"
	}
	return results
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
