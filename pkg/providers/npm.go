package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/wasilak/dotisan/pkg/cmdutil"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
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
func (p *NpmProvider) Reconcile(
	desired []resource.ResourceGroup,
	state []provider.ResourceState,
) provider.GroupPlan {
	plan := provider.GroupPlan{}

	stateIndex := make(map[string]provider.ResourceState)
	for _, s := range state {
		if s.Kind == "NpmPackages" {
			stateIndex[s.Group] = s
		}
	}

	installed := p.getInstalledPackages()

	for _, group := range desired {
		if group.Kind != "NpmPackages" {
			continue
		}

		stateGroup, exists := stateIndex[group.Name]

		if !exists {
			items := filterInstallableNpmItems(group.Items, installed)
			if len(items) > 0 {
				plan.Additions = append(plan.Additions, provider.GroupAddition{
					Kind:  group.Kind,
					Group: group.Name,
					Items: items,
				})
			}
			if len(items) == 0 && len(group.Items) > 0 {
				plan.InSync = append(plan.InSync, provider.GroupState{
					Kind:  group.Kind,
					Group: group.Name,
					Items: npmItemsToState(group.Items, installed),
				})
			}
		} else {
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

	desiredGroups := make(map[string]bool)
	for _, group := range desired {
		if group.Kind == "NpmPackages" {
			desiredGroups[group.Name] = true
		}
	}

	for groupName, stateGroup := range stateIndex {
		if !desiredGroups[groupName] {
			items := make([]resource.ResourceItem, 0, len(stateGroup.Items))
			for _, item := range stateGroup.Items {
				items = append(items, resource.ResourceItem{
					Name:    item.Name,
					Version: item.Version,
				})
			}
			plan.Removals = append(plan.Removals, provider.GroupRemoval{
				Kind:  "NpmPackages",
				Group: groupName,
				Items: items,
			})
		}
	}

	return plan
}

func (p *NpmProvider) compareGroupItems(
	group resource.ResourceGroup,
	stateGroup provider.ResourceState,
	installed map[string]string,
) (additions, removals []resource.ResourceItem, modifications []provider.ItemChange, inSync []resource.ItemState) {
	stateItems := make(map[string]resource.ItemState)
	for _, item := range stateGroup.Items {
		stateItems[item.Name] = item
	}

	for _, desiredItem := range group.Items {
		name := desiredItem.Name
		_, inState := stateItems[name]
		_, isInstalled := installed[name]

		if !isInstalled {
			additions = append(additions, desiredItem)
		} else if inState {
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
				inSync = append(inSync, stateItem)
			}
		} else {
			additions = append(additions, desiredItem)
		}
	}

	desiredItems := make(map[string]bool)
	for _, item := range group.Items {
		desiredItems[item.Name] = true
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

func (p *NpmProvider) getInstalledPackages() map[string]string {
	ctx := context.Background()
	stdout, _, err := cmdutil.RunSimple(ctx, "npm", "list", "-g", "--depth=0", "--json")
	if err != nil {
		return make(map[string]string)
	}

	// Simple parsing - look for package names
	installed := make(map[string]string)
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "\"") && strings.Contains(line, "\": {") {
			// Extract package name
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 && parts[1] != "dependencies" && parts[1] != "name" {
				installed[parts[1]] = ""
			}
		}
	}
	return installed
}

func filterInstallableNpmItems(items []resource.ResourceItem, installed map[string]string) []resource.ResourceItem {
	var result []resource.ResourceItem
	for _, item := range items {
		if _, isInstalled := installed[item.Name]; !isInstalled {
			result = append(result, item)
		}
	}
	return result
}

func npmItemsToState(items []resource.ResourceItem, installed map[string]string) []resource.ItemState {
	var result []resource.ItemState
	for _, item := range items {
		result = append(result, resource.ItemState{
			Name:    item.Name,
			Version: item.Version,
			Status:  "present",
		})
	}
	return result
}

// Apply executes the given GroupPlan
func (p *NpmProvider) Apply(ctx context.Context, plan provider.GroupPlan) error {
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

func (p *NpmProvider) applyGroupAddition(ctx context.Context, addition provider.GroupAddition) error {
	for _, item := range addition.Items {
		pkg := item.Name
		if item.Version != "" {
			pkg = fmt.Sprintf("%s@%s", item.Name, item.Version)
		}
		if _, stderr, err := cmdutil.RunSimple(ctx, "npm", "install", "-g", pkg); err != nil {
			return fmt.Errorf("failed to install %s: %s: %w", item.Name, stderr, err)
		}
	}
	return nil
}

func (p *NpmProvider) applyGroupRemoval(ctx context.Context, removal provider.GroupRemoval) error {
	for _, item := range removal.Items {
		if _, stderr, err := cmdutil.RunSimple(ctx, "npm", "uninstall", "-g", item.Name); err != nil {
			return fmt.Errorf("failed to uninstall %s: %s: %w", item.Name, stderr, err)
		}
	}
	return nil
}

func (p *NpmProvider) applyGroupModification(ctx context.Context, modification provider.GroupModification) error {
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

// Import is not supported for npm (use ImportItem)
func (p *NpmProvider) Import(ctx context.Context, group string) (provider.ResourceState, error) {
	return provider.ResourceState{}, fmt.Errorf("use ImportItem to import specific npm packages")
}

// ImportItem imports a specific npm package
func (p *NpmProvider) ImportItem(ctx context.Context, group string, item string) (provider.ResourceState, error) {
	// Check if package is installed
	stdout, _, err := cmdutil.RunSimple(ctx, "npm", "list", "-g", "--depth=0", item)
	if err != nil {
		return provider.ResourceState{}, fmt.Errorf("package %s is not installed globally", item)
	}

	if !strings.Contains(stdout, item) {
		return provider.ResourceState{}, fmt.Errorf("package %s is not installed globally", item)
	}

	return provider.ResourceState{
		Kind:      "NpmPackages",
		Group:     group,
		Namespace: "default",
		Items: []resource.ItemState{
			{
				Name:   item,
				Status: "present",
			},
		},
	}, nil
}
