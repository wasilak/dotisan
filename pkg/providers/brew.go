package providers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path"
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
func (p *BrewProvider) Reconcile(ctx context.Context,
	desired []resource.ResourceGroup,
	state []provider.ResourceState,
) provider.GroupPlan {
	plan := provider.GroupPlan{}

	// Track items already processed to avoid duplicates
	processedItems := make(map[string]bool) // key: "group/item"

	// Index state by group name for quick lookup
	stateIndex := provider.IndexStateByGroup(state, resource.KindBrewPackages)

	// Get currently installed packages
	installed, err := p.getInstalledPackages(ctx)
	if err != nil {
		// Can't get installed state, mark all as additions
		for _, group := range desired {
			if group.Kind == resource.KindBrewPackages {
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
		if group.Kind != resource.KindBrewPackages {
			continue
		}

		stateGroup, exists := stateIndex[group.Name]

		if !exists {
			// New group - check which items are already installed vs need installation
			var toInstall, toImport []resource.ResourceItem

			for _, item := range group.Items {
				name := item.Name
				if strings.HasSuffix(name, " (cask)") {
					name = strings.TrimSuffix(name, " (cask)")
				}

				if _, isInstalled := installed[name]; isInstalled {
					// Already installed - needs to be imported
					toImport = append(toImport, item)
				} else {
					// Not installed - needs installation
					toInstall = append(toInstall, item)
				}
			}

			// Add items that need installation
			if len(toInstall) > 0 {
				plan.Additions = append(plan.Additions, provider.GroupAddition{
					Kind:  group.Kind,
					Group: group.Name,
					Items: toInstall,
				})
			}

			// Add items that are already installed (for import)
			if len(toImport) > 0 {
				plan.Additions = append(plan.Additions, provider.GroupAddition{
					Kind:  group.Kind,
					Group: group.Name,
					Items: toImport,
				})

				// Add warning about import
				itemNames := make([]string, 0, len(toImport))
				for _, item := range toImport {
					itemNames = append(itemNames, item.Name)
				}
				plan.Warnings = append(plan.Warnings, provider.PlanWarning{
					GroupID:    fmt.Sprintf("%s/%s", group.Kind, group.Name),
					Severity:   "warning",
					Message:    fmt.Sprintf("Items already installed but not tracked: %s", strings.Join(itemNames, ", ")),
					Suggestion: fmt.Sprintf("dotisan state import %s/%s <item>", group.Kind, group.Name),
				})
			}
		} else {
			// Existing group - compare items
			additions, removals, cleanupItems, modifications, inSync := p.compareGroupItems(
				group, stateGroup, installed,
			)

			if len(additions) > 0 {
				plan.Additions = append(plan.Additions, provider.GroupAddition{
					Kind:  group.Kind,
					Group: group.Name,
					Items: additions,
				})
			}

			// Filter out already processed items and track new ones
			var filteredRemovals []resource.ResourceItem
			for _, item := range removals {
				key := fmt.Sprintf("%s/%s", group.Name, item.Name)
				if !processedItems[key] {
					processedItems[key] = true
					filteredRemovals = append(filteredRemovals, item)
				}
			}
			if len(filteredRemovals) > 0 {
				plan.Removals = append(plan.Removals, provider.GroupRemoval{
					Kind:  group.Kind,
					Group: group.Name,
					Items: filteredRemovals,
				})
				for _, it := range filteredRemovals {
					slog.Debug("plan.removal.append", "group", group.Name, "item", it.Name)
				}
			}

			var filteredCleanup []resource.ResourceItem
			for _, item := range cleanupItems {
				key := fmt.Sprintf("%s/%s", group.Name, item.Name)
				if !processedItems[key] {
					processedItems[key] = true
					filteredCleanup = append(filteredCleanup, item)
				}
			}
			if len(filteredCleanup) > 0 {
				plan.Cleanup = append(plan.Cleanup, provider.GroupCleanup{
					Kind:   group.Kind,
					Group:  group.Name,
					Items:  filteredCleanup,
					Reason: "not_in_config_and_not_installed",
				})
			}

			if len(modifications) > 0 {
				plan.Modifications = append(plan.Modifications, provider.GroupModification{
					Kind:    group.Kind,
					Group:   group.Name,
					Changes: modifications,
				})
			}

			if len(inSync) > 0 && len(additions) == 0 && len(removals) == 0 && len(cleanupItems) == 0 && len(modifications) == 0 {
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
		if group.Kind == resource.KindBrewPackages {
			desiredGroups[group.Name] = true
		}
	}

	for groupName, stateGroup := range stateIndex {
		if !desiredGroups[groupName] {
			// Entire group should be removed - but distinguish between
			// actual removals (installed) and cleanup (not installed)
			var removalItems []resource.ResourceItem
			var cleanupItems []resource.ResourceItem

			for _, item := range stateGroup.Items {
				name := item.Name
				if strings.HasSuffix(name, " (cask)") {
					name = strings.TrimSuffix(name, " (cask)")
				}

				// Check if installed (handle tap packages with base name fallback)
				isInstalled := false
				if _, ok := installed[name]; ok {
					isInstalled = true
				} else if strings.Contains(name, "/") {
					baseName := path.Base(name)
					if _, ok := installed[baseName]; ok {
						isInstalled = true
					}
				}

				key := fmt.Sprintf("%s/%s", groupName, item.Name)
				if processedItems[key] {
					// Skip already processed items
					continue
				}

				if isInstalled {
					removalItems = append(removalItems, resource.ResourceItem{
						Name:    item.Name,
						Version: item.Version,
					})
				} else {
					cleanupItems = append(cleanupItems, resource.ResourceItem{
						Name:    item.Name,
						Version: item.Version,
					})
				}
				processedItems[key] = true
			}

			if len(removalItems) > 0 {
				plan.Removals = append(plan.Removals, provider.GroupRemoval{
					Kind:  resource.KindBrewPackages,
					Group: groupName,
					Items: removalItems,
				})
				for _, it := range removalItems {
					slog.Debug("plan.removal.append.entire_group", "group", groupName, "item", it.Name)
				}
			}

			if len(cleanupItems) > 0 {
				plan.Cleanup = append(plan.Cleanup, provider.GroupCleanup{
					Kind:   resource.KindBrewPackages,
					Group:  groupName,
					Items:  cleanupItems,
					Reason: "not_in_config_and_not_installed",
				})
			}
		}
	}

	return plan
}

// compareGroupItems compares desired group items with state and installed packages.
// Returns additions, removals (installed items to uninstall), cleanup (not installed - state only),
// modifications, and inSync items.
func (p *BrewProvider) compareGroupItems(
	group resource.ResourceGroup,
	stateGroup provider.ResourceState,
	installed map[string]string,
) (additions, removals, cleanup []resource.ResourceItem, modifications []provider.ItemChange, inSync []resource.ItemState) {
	// Index state items by name
	stateItems := make(map[string]resource.ItemState)
	for _, item := range stateGroup.Items {
		stateItems[item.Name] = item
	}

	stateKeys := make([]string, 0, len(stateItems))
	for k := range stateItems {
		stateKeys = append(stateKeys, k)
	}
	installedKeys := make([]string, 0, len(installed))
	for k := range installed {
		installedKeys = append(installedKeys, k)
	}
	slog.Debug("compareGroupItems",
		"group", group.Name,
		"state_items", stateKeys,
		"installed", installedKeys,
	)

	// Check each desired item
	for _, desiredItem := range group.Items {
		name := desiredItem.Name
		// Normalize name: strip (cask) suffix for lookup
		lookupName := name
		isCask := strings.HasSuffix(name, " (cask)")
		if isCask {
			lookupName = strings.TrimSuffix(name, " (cask)")
		}

		// Try to find in installed - check full name first, then base name for tap packages
		isInstalled := false
		installedVersion := ""
		if v, ok := installed[lookupName]; ok {
			isInstalled = true
			installedVersion = v
		} else if strings.Contains(lookupName, "/") {
			// Try base name for tap packages (e.g., "dagger/tap/dagger" -> "dagger")
			baseName := path.Base(lookupName)
			if v, ok := installed[baseName]; ok {
				isInstalled = true
				installedVersion = v
			}
		}

		// Check state - full name and base name for tap packages
		stateItem, inState := stateItems[lookupName]
		if !inState && strings.Contains(lookupName, "/") {
			// Try base name for tap packages in state
			if si, ok := stateItems[path.Base(lookupName)]; ok {
				stateItem = si
				inState = true
			}
		}

		slog.Debug("evaluate.desired_item", "desired", name, "lookup", lookupName, "is_installed", isInstalled, "in_state", inState, "installed_version", installedVersion)

		if !isInstalled {
			// Not installed - needs to be added
			additions = append(additions, desiredItem)
		} else if inState {
			// Installed and tracked - check for modifications
			// Use installed version for comparison
			compareVersion := installedVersion
			if compareVersion == "" {
				compareVersion = stateItem.Version
			}
			if compareVersion != desiredItem.Version && desiredItem.Version != "" {
				modifications = append(modifications, provider.ItemChange{
					ItemName: lookupName,
					OldState: stateItem,
					NewState: resource.ItemState{
						Name:    lookupName,
						Version: desiredItem.Version,
						Status:  "present",
					},
					Diff: fmt.Sprintf("version: %s -> %s", compareVersion, desiredItem.Version),
				})
			} else {
				// In sync - update state with actual version
				stateItem.Version = installedVersion
				inSync = append(inSync, stateItem)
			}
		} else {
			// Installed but not in state - will be imported
			additions = append(additions, desiredItem)
		}
	}

	// Check for items in state but not in desired (removals or cleanup)
	desiredItems := make(map[string]bool)
	for _, item := range group.Items {
		name := item.Name
		if strings.HasSuffix(name, " (cask)") {
			name = strings.TrimSuffix(name, " (cask)")
		}
		desiredItems[name] = true
	}

	desiredKeys := make([]string, 0, len(desiredItems))
	for k := range desiredItems {
		desiredKeys = append(desiredKeys, k)
	}
	slog.Debug("desiredItems", "group", group.Name, "items", desiredKeys)

	for name, stateItem := range stateItems {
		if !desiredItems[name] {
			// Check if installed to determine if removal or cleanup
			lookupName := name
			if strings.HasSuffix(name, " (cask)") {
				lookupName = strings.TrimSuffix(name, " (cask)")
			}

			isInstalled := false
			if _, ok := installed[lookupName]; ok {
				isInstalled = true
			} else if strings.Contains(lookupName, "/") {
				baseName := path.Base(lookupName)
				if _, ok := installed[baseName]; ok {
					isInstalled = true
				}
			}

			if isInstalled {
				removals = append(removals, resource.ResourceItem{
					Name:    name,
					Version: stateItem.Version,
				})
			} else {
				cleanup = append(cleanup, resource.ResourceItem{
					Name:    name,
					Version: stateItem.Version,
				})
			}
		}
	}

	return
}

// getInstalledPackages retrieves currently installed Homebrew packages
func (p *BrewProvider) getInstalledPackages(ctx context.Context) (map[string]string, error) {
	if ctx == nil {
		slog.Warn("brew getInstalledPackages called with nil context; returning empty set")
		return make(map[string]string), nil
	}
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
			_, stderr, err := cmdutil.RunSimple(ctx, "brew", "uninstall", "--cask", name)
			if err != nil {
				// If the formula/cask is not present on this system, treat as no-op
				if strings.Contains(stderr, "No available formula or cask with the name") || strings.Contains(stderr, "is not installed") {
					// Already absent, log and continue
					slog.Warn("package not installed; skipping uninstall", "package", name)
					continue
				}
				// If Homebrew refuses due to dependencies, surface helpful message
				if strings.Contains(stderr, "Refusing to uninstall") {
					return fmt.Errorf("failed to uninstall %s: %s", name, stderr)
				}
				return fmt.Errorf("failed to uninstall cask %s: %s: %w", name, stderr, err)
			}
		} else {
			_, stderr, err := cmdutil.RunSimple(ctx, "brew", "uninstall", name)
			if err != nil {
				// If the formula is not present on this system, treat as no-op
				if strings.Contains(stderr, "No available formula or cask with the name") || strings.Contains(stderr, "is not installed") {
					slog.Warn("package not installed; skipping uninstall", "package", name)
					continue
				}
				// If Homebrew refuses due to dependencies, surface helpful message
				if strings.Contains(stderr, "Refusing to uninstall") {
					// Attempt to list installed dependents to give the user more context
					depsOut, _, _ := cmdutil.RunSimple(ctx, "brew", "uses", "--installed", name)
					hint := strings.TrimSpace(depsOut)
					if hint != "" {
						stderr = stderr + "\nInstalled dependents:\n" + hint
					}
					return fmt.Errorf("failed to uninstall %s: %s", name, stderr)
				}
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

// Import is not supported for BrewProvider (provider-level import removed).
func (p *BrewProvider) Import(ctx context.Context, group string) (provider.ResourceState, error) {
    return provider.ResourceState{}, fmt.Errorf("import not supported for provider homebrew")
}
