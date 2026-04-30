package providers

import (
	"context"
	"fmt"
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
func (p *GoProvider) Reconcile(
	desired []resource.ResourceGroup,
	state []provider.ResourceState,
) provider.GroupPlan {
	plan := provider.GroupPlan{}

	// Index state by group name
	stateIndex := make(map[string]provider.ResourceState)
	for _, s := range state {
		if s.Kind == "GoPackages" {
			stateIndex[s.Group] = s
		}
	}

	// Get currently installed packages
	installed := p.getInstalledPackages()

	// Process each desired group
	for _, group := range desired {
		if group.Kind != "GoPackages" {
			continue
		}

		stateGroup, exists := stateIndex[group.Name]

		if !exists {
			// New group - check which items are already installed vs need installation
			var toInstall, toImport []resource.ResourceItem

			for _, item := range group.Items {
				// Extract binary name from module path
				parts := strings.Split(item.Name, "/")
				binaryName := parts[len(parts)-1]

				if _, isInstalled := installed[binaryName]; isInstalled {
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
			// Existing group
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

	// Check for removals
	desiredGroups := make(map[string]bool)
	for _, group := range desired {
		if group.Kind == "GoPackages" {
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
				Kind:  "GoPackages",
				Group: groupName,
				Items: items,
			})
		}
	}

	return plan
}

// compareGroupItems compares desired group items with state and installed packages
func (p *GoProvider) compareGroupItems(
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

// getInstalledPackages retrieves currently installed Go packages
func (p *GoProvider) getInstalledPackages() map[string]string {
	installed := make(map[string]string)

	goBin := p.goBin
	if goBin == "" {
		goBin = "go"
	}

	// Get GOBIN or GOPATH/bin
	ctx := context.Background()
	stdout, _, err := cmdutil.RunSimple(ctx, goBin, "env", "GOBIN")
	if err != nil {
		return installed
	}

	goBinPath := strings.TrimSpace(stdout)
	if goBinPath == "" {
		stdout, _, err = cmdutil.RunSimple(ctx, goBin, "env", "GOPATH")
		if err != nil {
			return installed
		}
		goBinPath = filepath.Join(strings.TrimSpace(stdout), "bin")
	}

	// List binaries in GOBIN
	entries, err := os.ReadDir(goBinPath)
	if err != nil {
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

// Import is not supported for Go packages (use ImportItem)
func (p *GoProvider) Import(ctx context.Context, group string) (provider.ResourceState, error) {
	return provider.ResourceState{}, fmt.Errorf("use ImportItem to import specific Go packages")
}

// ImportItem imports a specific Go package
func (p *GoProvider) ImportItem(ctx context.Context, group string, item string) (provider.ResourceState, error) {
	goBin := p.goBin
	if goBin == "" {
		goBin = "go"
	}

	// Check if binary exists in GOBIN
	stdout, _, err := cmdutil.RunSimple(ctx, goBin, "env", "GOBIN")
	if err != nil {
		return provider.ResourceState{}, err
	}

	goBinPath := strings.TrimSpace(stdout)
	if goBinPath == "" {
		stdout, _, err = cmdutil.RunSimple(ctx, goBin, "env", "GOPATH")
		if err != nil {
			return provider.ResourceState{}, err
		}
		goBinPath = filepath.Join(strings.TrimSpace(stdout), "bin")
	}

	// Extract binary name
	parts := strings.Split(item, "/")
	binaryName := parts[len(parts)-1]

	binaryPath := filepath.Join(goBinPath, binaryName)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return provider.ResourceState{}, fmt.Errorf("binary %s not found in GOBIN", binaryName)
	}

	return provider.ResourceState{
		Kind:      "GoPackages",
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
