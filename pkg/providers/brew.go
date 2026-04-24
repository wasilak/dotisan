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
	return "homebrew"
}

// Available checks if the brew executable is available on this system.
func (p *BrewProvider) Available() (bool, string) {
	path := cmdutil.CheckExecutable("brew")
	if path == "" {
		return false, "brew not found in PATH; install from https://brew.sh"
	}
	return true, fmt.Sprintf("brew found at %s", path)
}

// Reconcile compares the desired resource groups with the current system state.
func (p *BrewProvider) Reconcile(
	desired []resource.ResourceGroup,
	state []provider.ResourceState,
) provider.GroupPlan {
	plan := provider.GroupPlan{}

	// Index state by group name for quick lookup
	stateIndex := make(map[string]provider.ResourceState)
	for _, s := range state {
		if s.Kind == "BrewPackages" {
			stateIndex[s.Group] = s
		}
	}

	// Get currently installed packages
	installed, err := p.getInstalledPackages()
	if err != nil {
		// Can't get installed state, mark all as additions
		for _, group := range desired {
			if group.Kind == "BrewPackages" {
				plan.Additions = append(plan.Additions, provider.GroupAddition{
					Kind:  group.Kind,
					Group: group.Name,
					Items: group.Items,
				})
			}
		}
		return plan
	}

	// Process each desired group
	for _, group := range desired {
		if group.Kind != "BrewPackages" {
			continue
		}

		stateGroup, exists := stateIndex[group.Name]

		if !exists {
			// New group - all items are additions
			items := filterInstallableItems(group.Items, installed)
			if len(items) > 0 {
				plan.Additions = append(plan.Additions, provider.GroupAddition{
					Kind:  group.Kind,
					Group: group.Name,
					Items: items,
				})
			}
			// Group is in sync if no items need installation
			if len(items) == 0 && len(group.Items) > 0 {
				plan.InSync = append(plan.InSync, provider.GroupState{
					Kind:  group.Kind,
					Group: group.Name,
					Items: itemsToState(group.Items, installed),
				})
			}
		} else {
			// Existing group - compare items
			additions, removals, modifications, inSync := p.compareGroupItems(
				group, stateGroup, installed,
			)

			if len(additions) > 0 {
				plan.Additions = append(plan.Additions, provider.GroupAddition{
					Kind:  group.Kind,
					Group: group.Name,
					Items: additions,
				})
			}

			if len(removals) > 0 {
				plan.Removals = append(plan.Removals, provider.GroupRemoval{
					Kind:  group.Kind,
					Group: group.Name,
					Items: removals,
				})
			}

			if len(modifications) > 0 {
				plan.Modifications = append(plan.Modifications, provider.GroupModification{
					Kind:    group.Kind,
					Group:   group.Name,
					Changes: modifications,
				})
			}

			if len(inSync) > 0 && len(additions) == 0 && len(removals) == 0 && len(modifications) == 0 {
				plan.InSync = append(plan.InSync, provider.GroupState{
					Kind:  group.Kind,
					Group: group.Name,
					Items: inSync,
				})
			}
		}
	}

	// Check for removals (groups in state but not in desired)
	desiredGroups := make(map[string]bool)
	for _, group := range desired {
		if group.Kind == "BrewPackages" {
			desiredGroups[group.Name] = true
		}
	}

	for groupName, stateGroup := range stateIndex {
		if !desiredGroups[groupName] {
			// Entire group should be removed
			items := make([]resource.ResourceItem, 0, len(stateGroup.Items))
			for _, item := range stateGroup.Items {
				items = append(items, resource.ResourceItem{
					Name:    item.Name,
					Version: item.Version,
				})
			}
			plan.Removals = append(plan.Removals, provider.GroupRemoval{
				Kind:  "BrewPackages",
				Group: groupName,
				Items: items,
			})
		}
	}

	return plan
}

// compareGroupItems compares desired group items with state and installed packages
func (p *BrewProvider) compareGroupItems(
	group resource.ResourceGroup,
	stateGroup provider.ResourceState,
	installed map[string]string,
) (additions, removals []resource.ResourceItem, modifications []provider.ItemChange, inSync []resource.ItemState) {
	// Index state items by name
	stateItems := make(map[string]resource.ItemState)
	for _, item := range stateGroup.Items {
		stateItems[item.Name] = item
	}

	// Check each desired item
	for _, desiredItem := range group.Items {
		name := desiredItem.Name
		// Strip (cask) suffix for lookup
		isCask := strings.HasSuffix(name, " (cask)")
		if isCask {
			name = strings.TrimSuffix(name, " (cask)")
		}

		_, inState := stateItems[name]
		_, isInstalled := installed[name]

		if !isInstalled {
			// Not installed - needs to be added
			additions = append(additions, desiredItem)
		} else if inState {
			// Installed and tracked - check for modifications
			stateItem := stateItems[name]
			if stateItem.Version != desiredItem.Version && desiredItem.Version != "" {
				modifications = append(modifications, provider.ItemChange{
					ItemName: name,
					OldState: stateItem,
					NewState: resource.ItemState{
						Name:    name,
						Version: desiredItem.Version,
					},
					Diff: fmt.Sprintf("version: %s -> %s", stateItem.Version, desiredItem.Version),
				})
			} else {
				// In sync
				inSync = append(inSync, stateItem)
			}
		} else {
			// Installed but not in state - will be imported
			additions = append(additions, desiredItem)
		}
	}

	// Check for items in state but not in desired (removals)
	desiredItems := make(map[string]bool)
	for _, item := range group.Items {
		name := item.Name
		if strings.HasSuffix(name, " (cask)") {
			name = strings.TrimSuffix(name, " (cask)")
		}
		desiredItems[name] = true
	}

	for name, stateItem := range stateItems {
		if !desiredItems[name] {
			removals = append(removals, resource.ResourceItem{
				Name:    name,
				Version: stateItem.Version,
			})
		}
	}

	return
}

// filterInstallableItems returns items that are not currently installed
func filterInstallableItems(items []resource.ResourceItem, installed map[string]string) []resource.ResourceItem {
	var result []resource.ResourceItem
	for _, item := range items {
		name := item.Name
		if strings.HasSuffix(name, " (cask)") {
			name = strings.TrimSuffix(name, " (cask)")
		}
		if _, isInstalled := installed[name]; !isInstalled {
			result = append(result, item)
		}
	}
	return result
}

// itemsToState converts ResourceItems to ItemStates with version info
func itemsToState(items []resource.ResourceItem, installed map[string]string) []resource.ItemState {
	var result []resource.ItemState
	for _, item := range items {
		name := item.Name
		if strings.HasSuffix(name, " (cask)") {
			name = strings.TrimSuffix(name, " (cask)")
		}
		version := item.Version
		if version == "" {
			version = installed[name]
		}
		result = append(result, resource.ItemState{
			Name:    item.Name,
			Version: version,
			Status:  "present",
		})
	}
	return result
}

// getInstalledPackages retrieves currently installed Homebrew packages
func (p *BrewProvider) getInstalledPackages() (map[string]string, error) {
	ctx := context.Background()
	packages := make(map[string]string)

	// Get formulae
	stdout, _, err := cmdutil.RunSimple(ctx, "brew", "list", "--formula", "--versions")
	if err == nil {
		lines := strings.Split(stdout, "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				packages[parts[0]] = parts[1]
			}
		}
	}

	// Get casks
	stdout, _, err = cmdutil.RunSimple(ctx, "brew", "list", "--cask", "--versions")
	if err == nil {
		lines := strings.Split(stdout, "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				name := parts[0]
				version := ""
				if len(parts) >= 2 {
					version = parts[1]
				}
				packages[name+" (cask)"] = version
			}
		}
	}

	return packages, nil
}

// Apply executes the given GroupPlan
func (p *BrewProvider) Apply(ctx context.Context, plan provider.GroupPlan) error {
	// Process additions
	for _, addition := range plan.Additions {
		if err := p.applyGroupAddition(ctx, addition); err != nil {
			return fmt.Errorf("failed to add to %s: %w", addition.Group, err)
		}
	}

	// Process removals
	for _, removal := range plan.Removals {
		if err := p.applyGroupRemoval(ctx, removal); err != nil {
			return fmt.Errorf("failed to remove from %s: %w", removal.Group, err)
		}
	}

	// Process modifications
	for _, modification := range plan.Modifications {
		if err := p.applyGroupModification(ctx, modification); err != nil {
			return fmt.Errorf("failed to modify %s: %w", modification.Group, err)
		}
	}

	return nil
}

// applyGroupAddition installs items in a group
func (p *BrewProvider) applyGroupAddition(ctx context.Context, addition provider.GroupAddition) error {
	for _, item := range addition.Items {
		name := item.Name
		isCask := strings.HasSuffix(name, " (cask)")
		if isCask {
			name = strings.TrimSuffix(name, " (cask)")
		}

		if isCask {
			if _, stderr, err := cmdutil.RunSimple(ctx, "brew", "install", "--cask", name); err != nil {
				return fmt.Errorf("failed to install cask %s: %s: %w", name, stderr, err)
			}
		} else {
			if _, stderr, err := cmdutil.RunSimple(ctx, "brew", "install", name); err != nil {
				return fmt.Errorf("failed to install %s: %s: %w", name, stderr, err)
			}
		}
	}
	return nil
}

// applyGroupRemoval uninstalls items from a group
func (p *BrewProvider) applyGroupRemoval(ctx context.Context, removal provider.GroupRemoval) error {
	for _, item := range removal.Items {
		name := item.Name
		isCask := strings.HasSuffix(name, " (cask)")
		if isCask {
			name = strings.TrimSuffix(name, " (cask)")
		}

		if isCask {
			if _, stderr, err := cmdutil.RunSimple(ctx, "brew", "uninstall", "--cask", name); err != nil {
				return fmt.Errorf("failed to uninstall cask %s: %s: %w", name, stderr, err)
			}
		} else {
			if _, stderr, err := cmdutil.RunSimple(ctx, "brew", "uninstall", name); err != nil {
				return fmt.Errorf("failed to uninstall %s: %s: %w", name, stderr, err)
			}
		}
	}
	return nil
}

// applyGroupModification updates items in a group
func (p *BrewProvider) applyGroupModification(ctx context.Context, modification provider.GroupModification) error {
	for _, change := range modification.Changes {
		// For now, reinstall to update version
		name := change.ItemName
		if _, stderr, err := cmdutil.RunSimple(ctx, "brew", "reinstall", name); err != nil {
			return fmt.Errorf("failed to update %s: %s: %w", name, stderr, err)
		}
	}
	return nil
}

// Import discovers an existing package group
func (p *BrewProvider) Import(ctx context.Context, group string) (provider.ResourceState, error) {
	// For BrewProvider, Import doesn't make sense for entire groups
	// Use ImportItem instead
	return provider.ResourceState{}, fmt.Errorf("use ImportItem to import specific packages")
}

// ImportItem imports a specific package
func (p *BrewProvider) ImportItem(ctx context.Context, group string, item string) (provider.ResourceState, error) {
	// Check if package is installed
	stdout, _, err := cmdutil.RunSimple(ctx, "brew", "list", "--versions", item)
	if err != nil {
		// Try cask
		stdout, _, err = cmdutil.RunSimple(ctx, "brew", "list", "--cask", "--versions", item)
		if err != nil {
			return provider.ResourceState{}, fmt.Errorf("package %s is not installed", item)
		}
	}

	// Parse version
	version := ""
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			version = parts[1]
			break
		}
	}

	return provider.ResourceState{
		Kind:      "BrewPackages",
		Group:     group,
		Namespace: "default",
		Items: []resource.ItemState{
			{
				Name:    item,
				Version: version,
				Status:  "present",
			},
		},
	}, nil
}

// brewFormulaInfo represents information about a Homebrew formula
type brewFormulaInfo struct {
	Name     string            `json:"name"`
	Versions map[string]string `json:"versions"`
	Desc     string            `json:"desc"`
}

// getFormulaInfo fetches formula information from the Homebrew API
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
