package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/wasilak/dotisan/pkg/cmdutil"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

// BrewProvider implements the Provider interface for Homebrew packages.
type BrewProvider struct {
	// httpClient is used for API requests
	httpClient *http.Client
}

// NewBrewProvider creates a new BrewProvider.
func NewBrewProvider() *BrewProvider {
	return &BrewProvider{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the provider name.
func (p *BrewProvider) Name() string {
	return "brew"
}

// Available checks if the brew executable is available on this system.
func (p *BrewProvider) Available() (bool, string) {
	path := cmdutil.CheckExecutable("brew")
	if path == "" {
		return false, "brew not found in PATH; install from https://brew.sh"
	}
	return true, fmt.Sprintf("brew found at %s", path)
}

// Reconcile compares the desired packages with the current system state.
func (p *BrewProvider) Reconcile(desired []resource.Resource, state []provider.ResourceState) provider.Plan {
	plan := provider.Plan{}

	// Build state map
	stateMap := make(map[string]provider.ResourceState)
	for _, s := range state {
		stateMap[s.ID] = s
	}

	desiredIDs := make(map[string]bool)

	for _, res := range desired {
		switch r := res.(type) {
		case *resource.BrewPackages:
			p.reconcileBrewPackages(r, stateMap, &plan, desiredIDs)
		}
	}

	// Check for removals
	for id, s := range stateMap {
		if !desiredIDs[id] && s.Kind == "BrewPackages" {
			plan.Removals = append(plan.Removals, &resource.BrewPackages{
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

// reconcileBrewPackages reconciles a single BrewPackages resource.
func (p *BrewProvider) reconcileBrewPackages(
	bp *resource.BrewPackages,
	stateMap map[string]provider.ResourceState,
	plan *provider.Plan,
	desiredIDs map[string]bool,
) {
	// Build resource ID using kind and name (Terraform-style: files/dirs are for human org only)
	id := fmt.Sprintf("BrewPackages/%s", bp.GetMetadata().Name)
	desiredIDs[id] = true

	// Get current installed packages
	installed, err := p.getInstalledPackages()
	if err != nil {
		// Can't get installed state, skip reconciliation
		return
	}

	// Check taps
	for _, tap := range bp.Spec.Taps {
		tapName := tap.Name
		if !p.isTapInstalled(tapName, installed) {
			// Tap needs to be added
			plan.Additions = append(plan.Additions, &resource.BrewPackages{
				BaseResource: resource.BaseResource{
					Kind: "BrewPackages",
					Metadata: resource.Metadata{
						Name:      fmt.Sprintf("tap-%s", tapName),
						Namespace: bp.GetMetadata().GetNamespace(),
					},
				},
				Spec: resource.BrewPackagesSpec{
					Taps: []resource.Tap{{Name: tapName}},
				},
			})
		}
	}

	// Check formulae
	for _, pkg := range bp.Spec.Formulae {
		pkgName := pkg.Name
		if !p.isPackageInstalled(pkgName, installed) {
			// Package needs to be installed
			plan.Additions = append(plan.Additions, &resource.BrewPackages{
				BaseResource: resource.BaseResource{
					Kind: "BrewPackages",
					Metadata: resource.Metadata{
						Name:      pkgName,
						Namespace: bp.GetMetadata().GetNamespace(),
					},
				},
				Spec: resource.BrewPackagesSpec{
					Formulae: []resource.Package{{Name: pkgName, Version: pkg.Version}},
				},
			})
		}
	}

	// Check casks
	for _, pkg := range bp.Spec.Casks {
		pkgName := pkg.Name
		if !p.isCaskInstalled(pkgName, installed) {
			// Cask needs to be installed
			plan.Additions = append(plan.Additions, &resource.BrewPackages{
				BaseResource: resource.BaseResource{
					Kind: "BrewPackages",
					Metadata: resource.Metadata{
						Name:      pkgName,
						Namespace: bp.GetMetadata().GetNamespace(),
					},
				},
				Spec: resource.BrewPackagesSpec{
					Casks: []resource.Package{{Name: pkgName, Version: pkg.Version}},
				},
			})
		}
	}

	// Drift detection: check if packages in state are still installed
	p.detectDrift(bp, stateMap, plan)

	// Check if resource is in sync
	if len(plan.Additions) == 0 && len(plan.Modifications) == 0 && len(plan.Warnings) == 0 {
		plan.InSync = append(plan.InSync, bp)
	}
}

// detectDrift checks if packages in saved state are still installed on the system.
// If a package was managed by dotisan but is no longer installed, it generates a PlanWarning.
func (p *BrewProvider) detectDrift(
	bp *resource.BrewPackages,
	stateMap map[string]provider.ResourceState,
	plan *provider.Plan,
) {
	// Get the saved state for this BrewPackages resource
	id := fmt.Sprintf("BrewPackages/%s", bp.GetMetadata().Name)
	savedState, exists := stateMap[id]
	if !exists {
		return
	}

	// Extract formulae from saved state
	var formulae []string
	if savedState.Extra != nil {
		if pkgs, ok := savedState.Extra["formulae"].([]interface{}); ok {
			for _, pkg := range pkgs {
				if name, ok := pkg.(string); ok {
					formulae = append(formulae, name)
				}
			}
		}
	}

	if len(formulae) == 0 {
		return
	}

	// Check actual status using brew info
	status, err := p.checkFormulaeStatus(formulae)
	if err != nil {
		return
	}

	// Generate warnings for uninstalled packages
	var driftMsg []string
	for _, formula := range formulae {
		if installed, ok := status[formula]; ok && !installed {
			driftMsg = append(driftMsg, fmt.Sprintf("Package '%s' was uninstalled outside of dotisan", formula))
		}
	}

	if len(driftMsg) > 0 {
		warning := provider.PlanWarning{
			ResourceID: id,
			Severity:   "warning",
			Message:    fmt.Sprintf("BrewPackages/%s has drift. %s", bp.GetMetadata().Name, strings.Join(driftMsg, ". ")),
			Suggestion: "Run 'dotisan apply' to restore",
		}
		plan.Warnings = append(plan.Warnings, warning)
	}
}

// getInstalledPackages retrieves currently installed Homebrew packages.
func (p *BrewProvider) getInstalledPackages() (map[string]string, error) {
	ctx := context.Background()
	stdout, _, err := cmdutil.RunSimple(ctx, "brew", "list", "--formula", "--versions")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed formulae: %w", err)
	}

	packages := make(map[string]string)
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			name := parts[0]
			version := parts[1]
			packages[name] = version
		}
	}

	return packages, nil
}

// checkFormulaeStatus checks the installation status of specific Homebrew formulae.
// It returns a map where the key is the formula name and the value indicates if it's installed.
func (p *BrewProvider) checkFormulaeStatus(formulae []string) (map[string]bool, error) {
	if len(formulae) == 0 {
		return make(map[string]bool), nil
	}

	ctx := context.Background()
	args := append([]string{"info", "--json"}, formulae...)
	stdout, stderr, err := cmdutil.RunSimple(ctx, "brew", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to run brew info: %w (stderr: %s)", err, stderr)
	}

	var results []brewInfoResult
	if err := json.Unmarshal([]byte(stdout), &results); err != nil {
		return nil, fmt.Errorf("failed to parse brew info output: %w", err)
	}

	status := make(map[string]bool)
	for _, result := range results {
		status[result.Name] = len(result.Installed) > 0
	}

	return status, nil
}

// brewInfoResult represents the JSON structure returned by `brew info --json v1`.
type brewInfoResult struct {
	Name      string        `json:"name"`
	Installed []interface{} `json:"installed"`
}

// isTapInstalled checks if a tap is installed.
func (p *BrewProvider) isTapInstalled(tap string, installed map[string]string) bool {
	ctx := context.Background()
	stdout, _, err := cmdutil.RunSimple(ctx, "brew", "tap")
	if err != nil {
		return false
	}

	taps := strings.Split(stdout, "\n")
	for _, t := range taps {
		if strings.TrimSpace(t) == tap {
			return true
		}
	}
	return false
}

// isPackageInstalled checks if a formula is installed.
func (p *BrewProvider) isPackageInstalled(name string, installed map[string]string) bool {
	_, exists := installed[name]
	return exists
}

// isCaskInstalled checks if a cask is installed.
func (p *BrewProvider) isCaskInstalled(name string, installed map[string]string) bool {
	ctx := context.Background()
	stdout, _, err := cmdutil.RunSimple(ctx, "brew", "list", "--cask")
	if err != nil {
		return false
	}

	casks := strings.Split(stdout, "\n")
	for _, c := range casks {
		if strings.TrimSpace(c) == name {
			return true
		}
	}
	return false
}

// Apply executes the given plan.
func (p *BrewProvider) Apply(ctx context.Context, plan provider.Plan) error {
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

// applyAddition installs a package or taps a repository.
func (p *BrewProvider) applyAddition(ctx context.Context, res resource.Resource) error {
	bp, ok := res.(*resource.BrewPackages)
	if !ok {
		return fmt.Errorf("not a BrewPackages resource")
	}

	// Handle taps
	for _, tap := range bp.Spec.Taps {
		if _, stderr, err := cmdutil.RunSimple(ctx, "brew", "tap", tap.Name); err != nil {
			return fmt.Errorf("failed to tap %s: %s: %w", tap.Name, stderr, err)
		}
	}

	// Handle formulae
	for _, pkg := range bp.Spec.Formulae {
		args := []string{"install", pkg.Name}
		if _, stderr, err := cmdutil.RunSimple(ctx, "brew", args...); err != nil {
			return fmt.Errorf("failed to install %s: %s: %w", pkg.Name, stderr, err)
		}
	}

	// Handle casks
	for _, pkg := range bp.Spec.Casks {
		args := []string{"install", "--cask", pkg.Name}
		if _, stderr, err := cmdutil.RunSimple(ctx, "brew", args...); err != nil {
			return fmt.Errorf("failed to install cask %s: %s: %w", pkg.Name, stderr, err)
		}
	}

	return nil
}

// applyRemoval uninstalls a package.
func (p *BrewProvider) applyRemoval(ctx context.Context, res resource.Resource) error {
	bp, ok := res.(*resource.BrewPackages)
	if !ok {
		return fmt.Errorf("not a BrewPackages resource")
	}

	// Uninstall formulae
	for _, pkg := range bp.Spec.Formulae {
		if _, stderr, err := cmdutil.RunSimple(ctx, "brew", "uninstall", pkg.Name); err != nil {
			return fmt.Errorf("failed to uninstall %s: %s: %w", pkg.Name, stderr, err)
		}
	}

	// Uninstall casks
	for _, pkg := range bp.Spec.Casks {
		if _, stderr, err := cmdutil.RunSimple(ctx, "brew", "uninstall", "--cask", pkg.Name); err != nil {
			return fmt.Errorf("failed to uninstall cask %s: %s: %w", pkg.Name, stderr, err)
		}
	}

	return nil
}

// Import discovers an existing package and returns its state.
func (p *BrewProvider) Import(ctx context.Context, id string) (provider.ResourceState, error) {
	// Try regular formula first
	stdout, _, err := cmdutil.RunSimple(ctx, "brew", "list", "--versions", id)
	if err == nil {
		// Regular formula found
		parts := strings.Fields(stdout)
		version := ""
		if len(parts) >= 2 {
			version = parts[1]
		}
		return provider.ResourceState{
			ID:      fmt.Sprintf("BrewPackages/%s", id),
			Kind:    "BrewPackages",
			Name:    id,
			Version: version,
			Extra: map[string]interface{}{
				"type": "formula",
			},
		}, nil
	}

	// Try cask
	stdout, _, err = cmdutil.RunSimple(ctx, "brew", "list", "--cask", "--versions", id)
	if err == nil {
		// Cask found
		parts := strings.Fields(stdout)
		version := ""
		if len(parts) >= 2 {
			version = parts[1]
		}
		return provider.ResourceState{
			ID:      fmt.Sprintf("BrewPackages/%s", id),
			Kind:    "BrewPackages",
			Name:    id,
			Version: version,
			Extra: map[string]interface{}{
				"type": "cask",
			},
		}, nil
	}

	return provider.ResourceState{}, fmt.Errorf("package %s not found (tried both formula and cask)", id)
}

// ImportItem imports a specific package from a BrewPackages resource.
// The itemKey is the package name (e.g., "ripgrep", "fd").
func (p *BrewProvider) ImportItem(ctx context.Context, resourceName string, itemKey string) (provider.ResourceState, error) {
	// Check if the package is installed using brew info
	status, err := p.checkFormulaeStatus([]string{itemKey})
	if err != nil {
		return provider.ResourceState{}, fmt.Errorf("failed to check package status: %w", err)
	}

	installed, ok := status[itemKey]
	if !ok || !installed {
		return provider.ResourceState{}, fmt.Errorf("package %s is not installed", itemKey)
	}

	// Get version info
	stdout, _, err := cmdutil.RunSimple(ctx, "brew", "list", "--versions", itemKey)
	if err != nil {
		return provider.ResourceState{}, fmt.Errorf("failed to get version for %s: %w", itemKey, err)
	}

	version := ""
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 1 && strings.HasPrefix(parts[0], itemKey) {
			if len(parts) >= 2 {
				version = parts[1]
			}
			break
		}
	}

	// Build the ResourceState ID
	stateID := fmt.Sprintf("BrewPackages/%s[%s]", resourceName, itemKey)

	return provider.ResourceState{
		ID:   stateID,
		Kind: "BrewPackages",
		Name: resourceName,
		Extra: map[string]interface{}{
			"formulae": []string{itemKey},
			"version":  version,
		},
	}, nil
}

// getFormulaInfo fetches formula information from the Homebrew API.
func (p *BrewProvider) getFormulaInfo(name string) (*brewFormulaInfo, error) {
	url := fmt.Sprintf("https://formulae.brew.sh/api/formula/%s.json", name)

	resp, err := p.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch formula info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("formula not found: %s", name)
	}

	var info brewFormulaInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode formula info: %w", err)
	}

	return &info, nil
}

// brewFormulaInfo represents information about a Homebrew formula.
type brewFormulaInfo struct {
	Name     string            `json:"name"`
	Versions map[string]string `json:"versions"`
	Desc     string            `json:"desc"`
}
