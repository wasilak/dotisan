package providers

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"dotisan/pkg/cmdutil"
	"dotisan/pkg/provider"
	"dotisan/pkg/resource"
)

// CargoProvider implements the Provider interface for Rust crates.
type CargoProvider struct{}

// NewCargoProvider creates a new CargoProvider.
func NewCargoProvider() *CargoProvider {
	return &CargoProvider{}
}

// Name returns the provider name.
func (p *CargoProvider) Name() string {
	return "cargo"
}

// Available checks if the cargo executable is available on this system.
func (p *CargoProvider) Available() (bool, string) {
	path := cmdutil.CheckExecutable("cargo")
	if path == "" {
		return false, "cargo not found in PATH; install Rust from https://rustup.rs"
	}
	return true, fmt.Sprintf("cargo found at %s", path)
}

// Reconcile compares the desired packages with the current system state.
func (p *CargoProvider) Reconcile(desired []resource.Resource, state []provider.ResourceState) provider.Plan {
	plan := provider.Plan{}

	// Build state map
	stateMap := make(map[string]provider.ResourceState)
	for _, s := range state {
		stateMap[s.ID] = s
	}

	desiredIDs := make(map[string]bool)

	for _, res := range desired {
		switch r := res.(type) {
		case *resource.CargoPackages:
			p.reconcileCargoPackages(r, stateMap, &plan, desiredIDs)
		}
	}

	// Check for removals
	for id, s := range stateMap {
		if !desiredIDs[id] && s.Kind == "CargoPackages" {
			plan.Removals = append(plan.Removals, &resource.CargoPackages{
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

// reconcileCargoPackages reconciles a single CargoPackages resource.
func (p *CargoProvider) reconcileCargoPackages(
	cp *resource.CargoPackages,
	stateMap map[string]provider.ResourceState,
	plan *provider.Plan,
	desiredIDs map[string]bool,
) {
	id := fmt.Sprintf("cargo/%s/%s", cp.GetMetadata().GetNamespace(), cp.GetMetadata().Name)
	desiredIDs[id] = true

	// Get current installed packages
	installed, err := p.getInstalledPackages()
	if err != nil {
		// Can't get installed state, mark all as additions
		for _, pkg := range cp.Spec.Packages {
			plan.Additions = append(plan.Additions, &resource.CargoPackages{
				BaseResource: resource.BaseResource{
					Kind: "CargoPackages",
					Metadata: resource.Metadata{
						Name:      pkg.Name,
						Namespace: cp.GetMetadata().GetNamespace(),
					},
				},
				Spec: resource.CargoPackagesSpec{
					Packages: []resource.Package{{Name: pkg.Name, Version: pkg.Version}},
				},
			})
		}
		return
	}

	// Check each desired package
	for _, pkg := range cp.Spec.Packages {
		pkgName := pkg.Name
		if !p.isPackageInstalled(pkgName, installed) {
			// Package needs to be installed
			plan.Additions = append(plan.Additions, &resource.CargoPackages{
				BaseResource: resource.BaseResource{
					Kind: "CargoPackages",
					Metadata: resource.Metadata{
						Name:      pkgName,
						Namespace: cp.GetMetadata().GetNamespace(),
					},
				},
				Spec: resource.CargoPackagesSpec{
					Packages: []resource.Package{{Name: pkgName, Version: pkg.Version}},
				},
			})
		}
	}

	// Check if resource is in sync
	if len(plan.Additions) == 0 && len(plan.Modifications) == 0 {
		plan.InSync = append(plan.InSync, cp)
	}
}

// cargoPackage represents a cargo package from cargo install --list.
type cargoPackage struct {
	Name    string
	Version string
}

// getInstalledPackages retrieves currently installed cargo crates.
func (p *CargoProvider) getInstalledPackages() (map[string]string, error) {
	ctx := context.Background()
	stdout, _, err := cmdutil.RunSimple(ctx, "cargo", "install", "--list")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed crates: %w", err)
	}

	packages := make(map[string]string)

	// Parse output like:
	// ripgrep v13.0.0:
	//     ripgrep
	// fd v8.7.0:
	//     fd
	lines := strings.Split(stdout, "\n")
	var currentPackage string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this is a package header line (contains " v" followed by version and ":")
		// Pattern: package-name v1.2.3:
		re := regexp.MustCompile(`^(\S+)\s+v([\d.]+):`)
		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			currentPackage = matches[1]
			version := matches[2]
			packages[currentPackage] = version
		}
	}

	return packages, nil
}

// isPackageInstalled checks if a crate is installed.
func (p *CargoProvider) isPackageInstalled(name string, installed map[string]string) bool {
	_, exists := installed[name]
	return exists
}

// Apply executes the given plan.
func (p *CargoProvider) Apply(ctx context.Context, plan provider.Plan) error {
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

// applyAddition installs cargo crates.
func (p *CargoProvider) applyAddition(ctx context.Context, res resource.Resource) error {
	cp, ok := res.(*resource.CargoPackages)
	if !ok {
		return fmt.Errorf("not a CargoPackages resource")
	}

	// Install each package
	for _, pkg := range cp.Spec.Packages {
		args := []string{"install", pkg.Name}
		if pkg.Version != "" {
			args = append(args, "--version", pkg.Version)
		}
		if _, stderr, err := cmdutil.RunSimple(ctx, "cargo", args...); err != nil {
			return fmt.Errorf("failed to install %s: %s: %w", pkg.Name, stderr, err)
		}
	}

	return nil
}

// applyRemoval uninstalls cargo crates.
func (p *CargoProvider) applyRemoval(ctx context.Context, res resource.Resource) error {
	cp, ok := res.(*resource.CargoPackages)
	if !ok {
		return fmt.Errorf("not a CargoPackages resource")
	}

	// Uninstall each package
	for _, pkg := range cp.Spec.Packages {
		if _, stderr, err := cmdutil.RunSimple(ctx, "cargo", "uninstall", pkg.Name); err != nil {
			return fmt.Errorf("failed to uninstall %s: %s: %w", pkg.Name, stderr, err)
		}
	}

	return nil
}

// Import discovers an existing crate and returns its state.
func (p *CargoProvider) Import(ctx context.Context, id string) (provider.ResourceState, error) {
	return provider.ResourceState{}, fmt.Errorf("import not yet implemented")
}
