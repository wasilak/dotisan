package providers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wasilak/nim/pkg/cmdutil"
	"github.com/wasilak/nim/pkg/provider"
	"github.com/wasilak/nim/pkg/resource"
)

// GoProvider implements the Provider interface for Go packages
type GoProvider struct {
	goBin string
}

// NewGoProvider creates a new GoProvider.
func NewGoProvider() *GoProvider {
	return &GoProvider{}
}

// Name returns the provider name.
func (p *GoProvider) Name() string {
	return "go"
}

// Available checks if the go executable is available on this system.
func (p *GoProvider) Available() (bool, string) {
	path, err := exec.LookPath("go")
	if err != nil {
		return false, "go not found in PATH; install from https://golang.org/dl/"
	}
	p.goBin = path
	return true, fmt.Sprintf("go found at %s", path)
}

// Reconcile compares the desired resource groups with the current system state.
func (p *GoProvider) Reconcile(ctx context.Context,
	desired []resource.ResourceGroup[any],
	state []provider.ResourceState,
) provider.GroupPlan {
	// Go module paths (e.g. golang.org/x/tools/cmd/goimports) install as the
	// last path segment binary name; use that for the installed-map lookup.
	normalizeName := func(name string) string {
		parts := strings.Split(name, "/")
		return parts[len(parts)-1]
	}
	// getInstalledPackages now accepts ctx
	return provider.BaseReconcile(resource.KindGoPackages, desired, state, p.getInstalledPackages(ctx), normalizeName)
}

// getInstalledPackages retrieves currently installed Go packages
func (p *GoProvider) getInstalledPackages(ctx context.Context) map[string]string {
	if ctx == nil {
		slog.Warn("go getInstalledPackages called with nil context; returning empty set")
		return make(map[string]string)
	}
	installed := make(map[string]string)
	goBin := p.goBin
	if goBin == "" {
		goBin = "go"
	}
	stdout, _, err := cmdutil.RunSimpleFn(ctx, goBin, "env", "GOBIN")
	if err != nil {
		slog.Warn("go getInstalledPackages: failed to get GOBIN", "err", err)
		return installed
	}

	goBinPath := strings.TrimSpace(stdout)
	if goBinPath == "" {
		stdout, _, err = cmdutil.RunSimpleFn(ctx, goBin, "env", "GOPATH")
		if err != nil {
			slog.Warn("go getInstalledPackages: failed to get GOPATH", "err", err)
			return installed
		}
		goBinPath = filepath.Join(strings.TrimSpace(stdout), "bin")
	}

	// List binaries in GOBIN
	entries, err := os.ReadDir(goBinPath)
	if err != nil {
		slog.Warn("go getInstalledPackages: failed to read bin dir", "path", goBinPath, "err", err)
		return installed
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err == nil {
				installed[entry.Name()] = fmt.Sprintf("modtime:%d", info.ModTime().Unix())
			}
		}
	}

	return installed
}

// Apply executes the given GroupPlan
func (p *GoProvider) Apply(ctx context.Context, plan provider.GroupPlan) ([]provider.ApplyItemResult, error) {
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

// applyGroupAddition installs Go packages in parallel.
// The Go module cache is safe for concurrent invocations of the go command.
func (p *GoProvider) applyGroupAddition(ctx context.Context, addition provider.GroupAddition) []provider.ApplyItemResult {
	goBin := p.goBin
	if goBin == "" {
		goBin = "go"
	}
	results := make([]provider.ApplyItemResult, len(addition.Items))
	var wg sync.WaitGroup
	for i, item := range addition.Items {
		installPath := fmt.Sprintf("%s@%s", item.Name, item.Version)
		if item.Version == "" || item.Version == "latest" {
			installPath = fmt.Sprintf("%s@latest", item.Name)
		}
		wg.Go(func() {
			var err error
			if _, stderr, e := cmdutil.RunSimpleFn(ctx, goBin, "install", installPath); e != nil {
				err = fmt.Errorf("failed to install %s: %s: %w", item.Name, stderr, e)
			}
			results[i] = provider.ApplyItemResult{Kind: addition.Kind, Group: addition.Group, Item: item.Name, Op: "add", Err: err}
		})
	}
	wg.Wait()
	return results
}

// applyGroupRemoval removes Go binaries in parallel.
func (p *GoProvider) applyGroupRemoval(ctx context.Context, removal provider.GroupRemoval) []provider.ApplyItemResult {
	goBin := p.goBin
	if goBin == "" {
		goBin = "go"
	}

	stdout, _, err := cmdutil.RunSimpleFn(ctx, goBin, "env", "GOBIN")
	if err != nil {
		// Fatal: can't determine where binaries live — fail all items
		results := make([]provider.ApplyItemResult, len(removal.Items))
		fatalErr := fmt.Errorf("failed to get GOBIN: %w", err)
		for i, item := range removal.Items {
			results[i] = provider.ApplyItemResult{Kind: removal.Kind, Group: removal.Group, Item: item.Name, Op: "remove", Err: fatalErr}
		}
		return results
	}

	goBinPath := strings.TrimSpace(stdout)
	if goBinPath == "" {
		stdout, _, err = cmdutil.RunSimpleFn(ctx, goBin, "env", "GOPATH")
		if err != nil {
			results := make([]provider.ApplyItemResult, len(removal.Items))
			fatalErr := fmt.Errorf("failed to get GOPATH: %w", err)
			for i, item := range removal.Items {
				results[i] = provider.ApplyItemResult{Kind: removal.Kind, Group: removal.Group, Item: item.Name, Op: "remove", Err: fatalErr}
			}
			return results
		}
		goBinPath = filepath.Join(strings.TrimSpace(stdout), "bin")
	}

	results := make([]provider.ApplyItemResult, len(removal.Items))
	var wg sync.WaitGroup
	for i, item := range removal.Items {
		parts := strings.Split(item.Name, "/")
		binaryName := parts[len(parts)-1]
		binaryPath := filepath.Join(goBinPath, binaryName)
		wg.Go(func() {
			var err error
			if e := os.Remove(binaryPath); e != nil {
				err = fmt.Errorf("failed to remove %s: %w", binaryName, e)
			}
			results[i] = provider.ApplyItemResult{Kind: removal.Kind, Group: removal.Group, Item: item.Name, Op: "remove", Err: err}
		})
	}
	wg.Wait()
	return results
}

// applyGroupModification updates Go packages (reinstall with new version).
func (p *GoProvider) applyGroupModification(ctx context.Context, modification provider.GroupModification) []provider.ApplyItemResult {
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

// Import not supported for GoProvider
func (p *GoProvider) Import(ctx context.Context, group string) (provider.ResourceState, error) {
	return provider.ResourceState{}, fmt.Errorf("import not supported for provider go")
}
