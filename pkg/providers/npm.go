package providers

import (
	"context"
	"encoding/json"
	"fmt"

	"dotisan/pkg/cmdutil"
	"dotisan/pkg/provider"
	"dotisan/pkg/resource"
)

// NpmProvider implements the Provider interface for npm packages.
type NpmProvider struct{}

// NewNpmProvider creates a new NpmProvider.
func NewNpmProvider() *NpmProvider {
	return &NpmProvider{}
}

// Name returns the provider name.
func (p *NpmProvider) Name() string {
	return "npm"
}

// Available checks if the npm executable is available on this system.
func (p *NpmProvider) Available() (bool, string) {
	path := cmdutil.CheckExecutable("npm")
	if path == "" {
		return false, "npm not found in PATH; install Node.js from https://nodejs.org"
	}
	return true, fmt.Sprintf("npm found at %s", path)
}

// Reconcile compares the desired packages with the current system state.
func (p *NpmProvider) Reconcile(desired []resource.Resource, state []provider.ResourceState) provider.Plan {
	plan := provider.Plan{}

	// Build state map
	stateMap := make(map[string]provider.ResourceState)
	for _, s := range state {
		stateMap[s.ID] = s
	}

	desiredIDs := make(map[string]bool)

	for _, res := range desired {
		switch r := res.(type) {
		case *resource.NpmPackages:
			p.reconcileNpmPackages(r, stateMap, &plan, desiredIDs)
		}
	}

	// Check for removals
	for id, s := range stateMap {
		if !desiredIDs[id] && s.Kind == "NpmPackages" {
			plan.Removals = append(plan.Removals, &resource.NpmPackages{
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

// reconcileNpmPackages reconciles a single NpmPackages resource.
func (p *NpmProvider) reconcileNpmPackages(
	np *resource.NpmPackages,
	stateMap map[string]provider.ResourceState,
	plan *provider.Plan,
	desiredIDs map[string]bool,
) {
	id := fmt.Sprintf("npm/%s/%s", np.GetMetadata().GetNamespace(), np.GetMetadata().Name)
	desiredIDs[id] = true

	// Get current installed packages
	installed, err := p.getInstalledPackages()
	if err != nil {
		// Can't get installed state, skip reconciliation
		return
	}

	// Check each desired package
	for _, pkg := range np.Spec.Packages {
		pkgName := pkg.Name
		if !p.isPackageInstalled(pkgName, installed) {
			// Package needs to be installed
			plan.Additions = append(plan.Additions, &resource.NpmPackages{
				BaseResource: resource.BaseResource{
					Kind: "NpmPackages",
					Metadata: resource.Metadata{
						Name:      pkgName,
						Namespace: np.GetMetadata().GetNamespace(),
					},
				},
				Spec: resource.NpmPackagesSpec{
					Packages: []resource.Package{{Name: pkgName, Version: pkg.Version}},
				},
			})
		}
	}

	// Check if resource is in sync
	if len(plan.Additions) == 0 && len(plan.Modifications) == 0 {
		plan.InSync = append(plan.InSync, np)
	}
}

// npmPackage represents an npm package from npm list output.
type npmPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// npmListOutput represents the JSON output of npm list --json.
type npmListOutput struct {
	Dependencies map[string]npmPackage `json:"dependencies"`
}

// getInstalledPackages retrieves currently installed global npm packages.
func (p *NpmProvider) getInstalledPackages() (map[string]string, error) {
	ctx := context.Background()
	stdout, _, err := cmdutil.RunSimple(ctx, "npm", "list", "-g", "--json")
	if err != nil {
		// npm list returns error when packages are missing, but still outputs valid JSON
		// We continue to parse the output
	}

	var listOutput npmListOutput
	if err := json.Unmarshal([]byte(stdout), &listOutput); err != nil {
		return nil, fmt.Errorf("failed to parse npm list output: %w", err)
	}

	packages := make(map[string]string)
	for name, pkg := range listOutput.Dependencies {
		packages[name] = pkg.Version
	}

	return packages, nil
}

// isPackageInstalled checks if a package is installed.
func (p *NpmProvider) isPackageInstalled(name string, installed map[string]string) bool {
	_, exists := installed[name]
	return exists
}

// Apply executes the given plan.
func (p *NpmProvider) Apply(ctx context.Context, plan provider.Plan) error {
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

// applyAddition installs npm packages.
func (p *NpmProvider) applyAddition(ctx context.Context, res resource.Resource) error {
	np, ok := res.(*resource.NpmPackages)
	if !ok {
		return fmt.Errorf("not a NpmPackages resource")
	}

	// Install each package
	for _, pkg := range np.Spec.Packages {
		args := []string{"install", "-g", pkg.Name}
		if pkg.Version != "" {
			args[2] = fmt.Sprintf("%s@%s", pkg.Name, pkg.Version)
		}
		if _, stderr, err := cmdutil.RunSimple(ctx, "npm", args...); err != nil {
			return fmt.Errorf("failed to install %s: %s: %w", pkg.Name, stderr, err)
		}
	}

	return nil
}

// applyRemoval uninstalls npm packages.
func (p *NpmProvider) applyRemoval(ctx context.Context, res resource.Resource) error {
	np, ok := res.(*resource.NpmPackages)
	if !ok {
		return fmt.Errorf("not a NpmPackages resource")
	}

	// Uninstall each package
	for _, pkg := range np.Spec.Packages {
		if _, stderr, err := cmdutil.RunSimple(ctx, "npm", "uninstall", "-g", pkg.Name); err != nil {
			return fmt.Errorf("failed to uninstall %s: %s: %w", pkg.Name, stderr, err)
		}
	}

	return nil
}

// Import discovers an existing package and returns its state.
func (p *NpmProvider) Import(ctx context.Context, id string) (provider.ResourceState, error) {
	return provider.ResourceState{}, fmt.Errorf("import not yet implemented")
}
