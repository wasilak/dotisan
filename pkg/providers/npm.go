package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/wasilak/nim/pkg/cmdutil"
	"github.com/wasilak/nim/pkg/provider"
	"github.com/wasilak/nim/pkg/resource"
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
	desired []resource.ResourceGroup[any],
	state []provider.ResourceState,
) provider.GroupPlan {
	return provider.BaseReconcile(resource.KindNpmPackages, desired, state, p.getInstalledPackages(ctx), nil)
}

func (p *NpmProvider) getInstalledPackages(ctx context.Context) map[string]string {
	if ctx == nil {
		slog.Warn("npm getInstalledPackages called with nil context; returning empty set")
		return make(map[string]string)
	}
	stdout, _, err := cmdutil.RunSimpleFn(ctx, "npm", "list", "-g", "--depth=0", "--json")
	if err != nil {
		slog.Warn("npm getInstalledPackages failed", "err", err)
		return make(map[string]string)
	}

	// Parse via JSON — handles scoped packages like @babel/core correctly.
	var parsed struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	installed := make(map[string]string)
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		slog.Warn("npm getInstalledPackages: failed to parse json", "err", err)
		return installed
	}
	for name, dep := range parsed.Dependencies {
		installed[name] = dep.Version
	}
	return installed
}

// InstalledForKind implements provider.CoverageProvider.
func (p *NpmProvider) InstalledForKind(ctx context.Context, kind string) (map[string]string, error) {
	if ctx == nil || kind != resource.KindNpmPackages {
		return nil, nil
	}
	return p.getInstalledPackages(ctx), nil
}

// Apply executes the given GroupPlan
func (p *NpmProvider) Apply(ctx context.Context, plan provider.GroupPlan) ([]provider.ApplyItemResult, error) {
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

func (p *NpmProvider) applyGroupAddition(ctx context.Context, addition provider.GroupAddition) []provider.ApplyItemResult {
	pkgs := make([]string, 0, len(addition.Items))
	// Map pkg string back to item name for result construction
	pkgToName := make(map[string]string, len(addition.Items))
	for _, item := range addition.Items {
		pkg := item.Name
		if item.Version != "" {
			pkg = fmt.Sprintf("%s@%s", item.Name, item.Version)
		}
		pkgs = append(pkgs, pkg)
		pkgToName[pkg] = item.Name
	}
	failed := batchWithFallback(pkgs, func(ns []string) error {
		args := append([]string{"install", "-g"}, ns...)
		_, stderr, err := cmdutil.RunSimpleFn(ctx, "npm", args...)
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
		if err, bad := failed[pkgs[i]]; bad {
			r.Err = err
		}
		results = append(results, r)
	}
	return results
}

func (p *NpmProvider) applyGroupRemoval(ctx context.Context, removal provider.GroupRemoval) []provider.ApplyItemResult {
	names := make([]string, 0, len(removal.Items))
	for _, item := range removal.Items {
		names = append(names, item.Name)
	}
	failed := batchWithFallback(names, func(ns []string) error {
		args := append([]string{"uninstall", "-g"}, ns...)
		_, stderr, err := cmdutil.RunSimpleFn(ctx, "npm", args...)
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

func (p *NpmProvider) applyGroupModification(ctx context.Context, modification provider.GroupModification) []provider.ApplyItemResult {
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
	// Correct the Op field
	for i := range results {
		results[i].Op = "modify"
	}
	return results
}

// Import not supported for npm provider
func (p *NpmProvider) Import(ctx context.Context, group string) (provider.ResourceState, error) {
	return provider.ResourceState{}, fmt.Errorf("import not supported for provider npm")
}
