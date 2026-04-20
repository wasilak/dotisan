package providers

import (
	"context"
	"fmt"
	"strings"

	"dotisan/pkg/cmdutil"
	"dotisan/pkg/provider"
	"dotisan/pkg/resource"
)

// GoProvider implements the Provider interface for Go modules.
type GoProvider struct{}

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
	path := cmdutil.CheckExecutable("go")
	if path == "" {
		return false, "go not found in PATH; install from https://golang.org"
	}
	return true, fmt.Sprintf("go found at %s", path)
}

// Reconcile compares the desired packages with the current system state.
func (p *GoProvider) Reconcile(desired []resource.Resource, state []provider.ResourceState) provider.Plan {
	plan := provider.Plan{}

	// Build state map
	stateMap := make(map[string]provider.ResourceState)
	for _, s := range state {
		stateMap[s.ID] = s
	}

	desiredIDs := make(map[string]bool)

	for _, res := range desired {
		switch r := res.(type) {
		case *resource.GoPackages:
			p.reconcileGoPackages(r, stateMap, &plan, desiredIDs)
		}
	}

	// Check for removals
	for id, s := range stateMap {
		if !desiredIDs[id] && s.Kind == "GoPackages" {
			plan.Removals = append(plan.Removals, &resource.GoPackages{
				BaseResource: resource.BaseResource{
					Kind: s.Kind,
					Metadata: resource.Metadata{
						Name:      s.Name,
						Namespace: s.Namespace,
					},
				},
			})
		}
	}

	return plan
}

// reconcileGoPackages reconciles a single GoPackages resource.
func (p *GoProvider) reconcileGoPackages(
	gp *resource.GoPackages,
	stateMap map[string]provider.ResourceState,
	plan *provider.Plan,
	desiredIDs map[string]bool,
) {
	id := fmt.Sprintf("go/%s/%s", gp.GetMetadata().GetNamespace(), gp.GetMetadata().Name)
	desiredIDs[id] = true

	// Get current installed packages
	installed, err := p.getInstalledPackages()
	if err != nil {
		// Can't get installed state, mark all as additions
		for _, pkg := range gp.Spec.Packages {
			plan.Additions = append(plan.Additions, &resource.GoPackages{
				BaseResource: resource.BaseResource{
					Kind: "GoPackages",
					Metadata: resource.Metadata{
						Name:      pkg.Module,
						Namespace: gp.GetMetadata().GetNamespace(),
					},
				},
				Spec: resource.GoPackagesSpec{
					Packages: []resource.GoPackage{{Module: pkg.Module, Version: pkg.Version}},
				},
			})
		}
		return
	}

	// Check each desired package
	for _, pkg := range gp.Spec.Packages {
		moduleName := pkg.Module
		if !p.isPackageInstalled(moduleName, installed) {
			// Package needs to be installed
			plan.Additions = append(plan.Additions, &resource.GoPackages{
				BaseResource: resource.BaseResource{
					Kind: "GoPackages",
					Metadata: resource.Metadata{
						Name:      moduleName,
						Namespace: gp.GetMetadata().GetNamespace(),
					},
				},
				Spec: resource.GoPackagesSpec{
					Packages: []resource.GoPackage{{Module: moduleName, Version: pkg.Version}},
				},
			})
		}
	}

	// Check if resource is in sync
	if len(plan.Additions) == 0 && len(plan.Modifications) == 0 {
		plan.InSync = append(plan.InSync, gp)
	}
}

// getInstalledPackages retrieves currently installed Go modules.
// Note: Go doesn't have a built-in way to list installed binaries,
// so we check the GOPATH/bin or GOBIN directory.
func (p *GoProvider) getInstalledPackages() (map[string]string, error) {
	// For now, return empty map since Go doesn't track installed modules
	// The user would need to manually check $GOPATH/bin or use `which`
	return make(map[string]string), nil
}

// isPackageInstalled checks if a Go module is installed.
// This checks if the binary exists in GOPATH/bin or PATH.
func (p *GoProvider) isPackageInstalled(module string, installed map[string]string) bool {
	// Extract binary name from module path
	parts := strings.Split(module, "/")
	if len(parts) == 0 {
		return false
	}
	binaryName := parts[len(parts)-1]

	// Check if binary exists in PATH
	ctx := context.Background()
	_, _, err := cmdutil.RunSimple(ctx, "which", binaryName)
	return err == nil
}

// Apply executes the given plan.
func (p *GoProvider) Apply(ctx context.Context, plan provider.Plan) error {
	// Process additions
	for _, res := range plan.Additions {
		if err := p.applyAddition(ctx, res); err != nil {
			return fmt.Errorf("failed to add %s: %w", res.GetMetadata().ResourceID(), err)
		}
	}

	// Process removals
	for _, res := range plan.Removals {
		if err := p.applyRemoval(ctx, res); err != nil {
			return fmt.Errorf("failed to remove %s: %w", res.GetMetadata().ResourceID(), err)
		}
	}

	return nil
}

// applyAddition installs Go modules.
func (p *GoProvider) applyAddition(ctx context.Context, res resource.Resource) error {
	gp, ok := res.(*resource.GoPackages)
	if !ok {
		return fmt.Errorf("not a GoPackages resource")
	}

	// Install each module
	for _, pkg := range gp.Spec.Packages {
		modulePath := pkg.Module
		if pkg.Version != "" && pkg.Version != "latest" {
			modulePath = fmt.Sprintf("%s@%s", pkg.Module, pkg.Version)
		}

		if _, stderr, err := cmdutil.RunSimple(ctx, "go", "install", modulePath); err != nil {
			return fmt.Errorf("failed to install %s: %s: %w", pkg.Module, stderr, err)
		}
	}

	return nil
}

// applyRemoval removes Go binaries.
func (p *GoProvider) applyRemoval(ctx context.Context, res resource.Resource) error {
	gp, ok := res.(*resource.GoPackages)
	if !ok {
		return fmt.Errorf("not a GoPackages resource")
	}

	// Remove each binary
	for _, pkg := range gp.Spec.Packages {
		// Extract binary name from module path
		parts := strings.Split(pkg.Module, "/")
		if len(parts) == 0 {
			continue
		}
		binaryName := parts[len(parts)-1]

		// Find the binary location
		stdout, _, err := cmdutil.RunSimple(ctx, "which", binaryName)
		if err != nil {
			// Binary not found, skip
			continue
		}

		// Remove the binary
		if _, stderr, err := cmdutil.RunSimple(ctx, "rm", stdout); err != nil {
			return fmt.Errorf("failed to remove %s: %s: %w", binaryName, stderr, err)
		}
	}

	return nil
}

// Import discovers an existing module and returns its state.
func (p *GoProvider) Import(ctx context.Context, id string) (provider.ResourceState, error) {
	return provider.ResourceState{}, fmt.Errorf("import not yet implemented")
}
