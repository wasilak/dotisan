package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/wasilak/dotisan/pkg/cmdutil"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
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
	// Build resource ID using kind and name (Terraform-style: files/dirs are for human org only)
	id := fmt.Sprintf("GoPackages/%s", gp.GetMetadata().Name)
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

	// Check each desired package - add ALL to plan
	for _, pkg := range gp.Spec.Packages {
		moduleName := pkg.Module
		_, isInstalled := installed[moduleName]
		_, inState := stateMap[fmt.Sprintf("GoPackages/%s[%s]", gp.GetMetadata().Name, moduleName)]

		if !isInstalled || !inState {
			// Needs install or not in state
			plan.Additions = append(plan.Additions, &resource.GoPackages{
				BaseResource: resource.BaseResource{
					Kind: "GoPackages",
					Metadata: resource.Metadata{
						Name:      fmt.Sprintf("%s[%s]", gp.GetMetadata().Name, moduleName),
						Namespace: gp.GetMetadata().GetNamespace(),
					},
				},
				Spec: resource.GoPackagesSpec{
					Packages: []resource.GoPackage{{Module: moduleName, Version: pkg.Version}},
				},
			})
		}

		// Mark as desired
		desiredIDs[fmt.Sprintf("GoPackages/%s[%s]", gp.GetMetadata().Name, moduleName)] = true
	}

	// Detect drift: check if modules in saved state are still installed
	p.detectGoDrift(gp, stateMap, plan, installed)

	// Check if resource is in sync
	if len(plan.Additions) == 0 && len(plan.Modifications) == 0 && len(plan.Warnings) == 0 {
		plan.InSync = append(plan.InSync, gp)
	}
}

// detectGoDrift checks if modules in saved state are still installed on the system.
func (p *GoProvider) detectGoDrift(
	gp *resource.GoPackages,
	stateMap map[string]provider.ResourceState,
	plan *provider.Plan,
	installed map[string]string,
) {
	id := fmt.Sprintf("GoPackages/%s", gp.GetMetadata().Name)
	savedState, exists := stateMap[id]
	if !exists {
		return
	}

	var savedModules []string
	if savedState.Extra != nil {
		if mods, ok := savedState.Extra["modules"].(map[string]interface{}); ok {
			for name := range mods {
				savedModules = append(savedModules, name)
			}
		}
	}

	if len(savedModules) == 0 {
		return
	}

	var driftMsg []string
	for _, moduleName := range savedModules {
		if _, stillInstalled := installed[moduleName]; !stillInstalled {
			driftMsg = append(driftMsg, fmt.Sprintf("Module '%s' was removed outside of dotisan", moduleName))
		}
	}

	if len(driftMsg) > 0 {
		warning := provider.PlanWarning{
			ResourceID: id,
			Severity:   "warning",
			Message:   fmt.Sprintf("GoPackages/%s has drift. %s", gp.GetMetadata().Name, strings.Join(driftMsg, ". ")),
			Suggestion: "Run 'dotisan apply' to restore",
		}
		plan.Warnings = append(plan.Warnings, warning)
	}
}

// getInstalledPackages retrieves currently installed Go modules.
func (p *GoProvider) getInstalledPackages() (map[string]string, error) {
	ctx := context.Background()
	stdout, _, err := cmdutil.RunSimple(ctx, "go", "list", "-m", "all")
	if err != nil {
		return nil, fmt.Errorf("failed to list Go modules: %w", err)
	}

	result := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		module := parts[0]
		version := ""
		if len(parts) > 1 {
			version = parts[1]
		}
		result[module] = version
	}

	return result, nil
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
		// When running outside a Go module (from /tmp), go install REQUIRES a version suffix
		// Build module path with @version (use "latest" if no version specified)
		version := pkg.Version
		if version == "" {
			version = "latest"
		}
		modulePath := fmt.Sprintf("%s@%s", pkg.Module, version)

		// Run go install from temp directory to avoid picking up local go.mod
		// This ensures we install the binary globally, not as a project dependency
		_, stderr, err := cmdutil.RunWithDir(ctx, "/tmp", "go", "install", modulePath)
		if err != nil {
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

// Import discovers all installed Go modules and returns their state.
func (p *GoProvider) Import(ctx context.Context, id string) (provider.ResourceState, error) {
	modules, err := p.getInstalledPackages()
	if err != nil {
		return provider.ResourceState{}, fmt.Errorf("failed to list Go modules: %w", err)
	}

	return provider.ResourceState{
		ID:      "GoPackages/global",
		Kind:    "GoPackages",
		Name:    "global",
		Version: "all",
		Extra: map[string]interface{}{
			"modules": modules,
		},
	}, nil
}

func (p *GoProvider) ImportItem(ctx context.Context, resourceName string, itemKey string) (provider.ResourceState, error) {
	return provider.ResourceState{}, fmt.Errorf("ImportItem not implemented for GoProvider")
}
