package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/wasilak/dotisan/pkg/cmdutil"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

// CargoProvider implements the Provider interface for Cargo packages
type CargoProvider struct{}

// NewCargoProvider creates a new CargoProvider.
func NewCargoProvider() *CargoProvider {
	return &CargoProvider{}
}

// Name returns the provider name.
func (p *CargoProvider) Name() string {
	return "cargo"
}

// Available checks if cargo is available on this system.
func (p *CargoProvider) Available() (bool, string) {
	if path := cmdutil.CheckExecutable("cargo"); path == "" {
		return false, "cargo not found in PATH; install Rust from https://rustup.rs/"
	}
	return true, "cargo found"
}

// Reconcile compares the desired resource groups with the current system state.
func (p *CargoProvider) Reconcile(
	desired []resource.ResourceGroup,
	state []provider.ResourceState,
) provider.GroupPlan {
	plan := provider.GroupPlan{}

	stateIndex := make(map[string]provider.ResourceState)
	for _, s := range state {
		if s.Kind == "CargoPackages" {
			stateIndex[s.Group] = s
		}
	}

	installed := p.getInstalledPackages()

	for _, group := range desired {
		if group.Kind != "CargoPackages" {
			continue
		}

		stateGroup, exists := stateIndex[group.Name]

		if !exists {
			items := filterInstallableCargoItems(group.Items, installed)
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
					Items: cargoItemsToState(group.Items, installed),
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
		if group.Kind == "CargoPackages" {
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
				Kind:  "CargoPackages",
				Group: groupName,
				Items: items,
			})
		}
	}

	return plan
}

func (p *CargoProvider) compareGroupItems(
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

func (p *CargoProvider) getInstalledPackages() map[string]string {
	ctx := context.Background()
	// List installed crates
	stdout, _, err := cmdutil.RunSimple(ctx, "cargo", "install", "--list")
	if err != nil {
		return make(map[string]string)
	}

	installed := make(map[string]string)
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Format: "crate_name v1.0.0:"
		if strings.Contains(line, " v") && strings.HasSuffix(line, ":") {
			parts := strings.Split(line, " ")
			if len(parts) >= 2 {
				name := parts[0]
				version := strings.TrimSuffix(parts[1], ":")
				version = strings.TrimPrefix(version, "v")
				installed[name] = version
			}
		}
	}
	return installed
}

func filterInstallableCargoItems(items []resource.ResourceItem, installed map[string]string) []resource.ResourceItem {
	var result []resource.ResourceItem
	for _, item := range items {
		if _, isInstalled := installed[item.Name]; !isInstalled {
			result = append(result, item)
		}
	}
	return result
}

func cargoItemsToState(items []resource.ResourceItem, installed map[string]string) []resource.ItemState {
	var result []resource.ItemState
	for _, item := range items {
		version := item.Version
		if version == "" {
			version = installed[item.Name]
		}
		result = append(result, resource.ItemState{
			Name:    item.Name,
			Version: version,
			Status:  "present",
		})
	}
	return result
}

// Apply executes the given GroupPlan
func (p *CargoProvider) Apply(ctx context.Context, plan provider.GroupPlan) error {
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

func (p *CargoProvider) applyGroupAddition(ctx context.Context, addition provider.GroupAddition) error {
	for _, item := range addition.Items {
		crate := item.Name
		if item.Version != "" {
			crate = fmt.Sprintf("%s@%s", item.Name, item.Version)
		}
		if _, stderr, err := cmdutil.RunSimple(ctx, "cargo", "install", crate); err != nil {
			return fmt.Errorf("failed to install %s: %s: %w", item.Name, stderr, err)
		}
	}
	return nil
}

func (p *CargoProvider) applyGroupRemoval(ctx context.Context, removal provider.GroupRemoval) error {
	for _, item := range removal.Items {
		if _, stderr, err := cmdutil.RunSimple(ctx, "cargo", "uninstall", item.Name); err != nil {
			return fmt.Errorf("failed to uninstall %s: %s: %w", item.Name, stderr, err)
		}
	}
	return nil
}

func (p *CargoProvider) applyGroupModification(ctx context.Context, modification provider.GroupModification) error {
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

// Import is not supported for cargo (use ImportItem)
func (p *CargoProvider) Import(ctx context.Context, group string) (provider.ResourceState, error) {
	return provider.ResourceState{}, fmt.Errorf("use ImportItem to import specific cargo crates")
}

// ImportItem imports a specific cargo crate
func (p *CargoProvider) ImportItem(ctx context.Context, group string, item string) (provider.ResourceState, error) {
	// Check if crate is installed
	installed := p.getInstalledPackages()
	if _, isInstalled := installed[item]; !isInstalled {
		return provider.ResourceState{}, fmt.Errorf("crate %s is not installed", item)
	}

	return provider.ResourceState{
		Kind:      "CargoPackages",
		Group:     group,
		Namespace: "default",
		Items: []resource.ItemState{
			{
				Name:    item,
				Version: installed[item],
				Status:  "present",
			},
		},
	}, nil
}
