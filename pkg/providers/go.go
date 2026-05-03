package providers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wasilak/dotisan/pkg/cmdutil"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
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
	desired []resource.ResourceGroup,
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
	stdout, _, err := cmdutil.RunSimple(ctx, goBin, "env", "GOBIN")
	if err != nil {
		slog.Warn("go getInstalledPackages: failed to get GOBIN", "err", err)
		return installed
	}

	goBinPath := strings.TrimSpace(stdout)
	if goBinPath == "" {
		stdout, _, err = cmdutil.RunSimple(ctx, goBin, "env", "GOPATH")
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
func (p *GoProvider) Apply(ctx context.Context, plan provider.GroupPlan) error {
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

// applyGroupAddition installs Go packages
func (p *GoProvider) applyGroupAddition(ctx context.Context, addition provider.GroupAddition) error {
	goBin := p.goBin
	if goBin == "" {
		goBin = "go"
	}

	for _, item := range addition.Items {
		module := item.Name
		version := item.Version

		// Build install path
		installPath := module
		if version != "" && version != "latest" {
			installPath = fmt.Sprintf("%s@%s", module, version)
		}

		if _, stderr, err := cmdutil.RunSimple(ctx, goBin, "install", installPath); err != nil {
			return fmt.Errorf("failed to install %s: %s: %w", module, stderr, err)
		}
	}
	return nil
}

// applyGroupRemoval removes Go binaries
func (p *GoProvider) applyGroupRemoval(ctx context.Context, removal provider.GroupRemoval) error {
	// Get GOBIN
	goBin := p.goBin
	if goBin == "" {
		goBin = "go"
	}

	stdout, _, err := cmdutil.RunSimple(ctx, goBin, "env", "GOBIN")
	if err != nil {
		return fmt.Errorf("failed to get GOBIN: %w", err)
	}

	goBinPath := strings.TrimSpace(stdout)
	if goBinPath == "" {
		stdout, _, err = cmdutil.RunSimple(ctx, goBin, "env", "GOPATH")
		if err != nil {
			return fmt.Errorf("failed to get GOPATH: %w", err)
		}
		goBinPath = filepath.Join(strings.TrimSpace(stdout), "bin")
	}

	for _, item := range removal.Items {
		// Extract binary name
		parts := strings.Split(item.Name, "/")
		binaryName := parts[len(parts)-1]

		binaryPath := filepath.Join(goBinPath, binaryName)
		if err := os.Remove(binaryPath); err != nil {
			return fmt.Errorf("failed to remove %s: %w", binaryName, err)
		}
	}
	return nil
}

// applyGroupModification updates Go packages (reinstall)
func (p *GoProvider) applyGroupModification(ctx context.Context, modification provider.GroupModification) error {
	// Reinstall with new version
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

// Import not supported for GoProvider
func (p *GoProvider) Import(ctx context.Context, group string) (provider.ResourceState, error) {
	return provider.ResourceState{}, fmt.Errorf("import not supported for provider go")
}
