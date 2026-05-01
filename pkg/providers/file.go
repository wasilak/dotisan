package providers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

// FileProvider implements the Provider interface for ManagedFile and ManagedDirectory
type FileProvider struct {
	dotfilesRoot string
}

// NewFileProvider creates a new FileProvider.
func NewFileProvider(dotfilesRoot string) *FileProvider {
	return &FileProvider{
		dotfilesRoot: dotfilesRoot,
	}
}

// Name returns the provider name.
func (p *FileProvider) Name() string {
	return "file"
}

// Available checks if the provider can operate on this system.
func (p *FileProvider) Available() (bool, string) {
	if p.dotfilesRoot != "" {
		if _, err := os.Stat(p.dotfilesRoot); err != nil {
			return false, fmt.Sprintf("dotfiles root not accessible: %s", err)
		}
	}
	return true, "filesystem operations available"
}

// Reconcile compares the desired resource groups with the current system state.
func (p *FileProvider) Reconcile(ctx context.Context,
	desired []resource.ResourceGroup,
	state []provider.ResourceState,
) provider.GroupPlan {
	plan := provider.GroupPlan{}

	// Index state by kind and group
	stateIndex := make(map[string]map[string]provider.ResourceState)
	for _, s := range state {
		if s.Kind == "ManagedFile" || s.Kind == "ManagedDirectory" {
			if stateIndex[s.Kind] == nil {
				stateIndex[s.Kind] = make(map[string]provider.ResourceState)
			}
			stateIndex[s.Kind][s.Group] = s
		}
	}

	// Process each desired group
	for _, group := range desired {
		if group.Kind != "ManagedFile" && group.Kind != "ManagedDirectory" {
			continue
		}

		kindIndex := stateIndex[group.Kind]
		stateGroup, exists := kindIndex[group.Name]

		if !exists {
			// New group - all items are additions
			items := p.filterExistingItems(group)
			if len(items) > 0 {
				plan.Additions = append(plan.Additions, provider.GroupAddition{
					Kind:  group.Kind,
					Group: group.Name,
					Items: items,
				})
			}
		} else {
			// Existing group - compare items
			additions, removals, modifications, inSync := p.compareGroupItems(group, stateGroup)

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
		if group.Kind == "ManagedFile" || group.Kind == "ManagedDirectory" {
			desiredGroups[group.Kind+"/"+group.Name] = true
		}
	}

	for kind, groups := range stateIndex {
		for groupName := range groups {
			if !desiredGroups[kind+"/"+groupName] {
				stateGroup := stateIndex[kind][groupName]
				items := make([]resource.ResourceItem, 0, len(stateGroup.Items))
				for _, item := range stateGroup.Items {
					items = append(items, resource.ResourceItem{
						Name:    item.Name,
						Version: item.Version,
					})
				}
				plan.Removals = append(plan.Removals, provider.GroupRemoval{
					Kind:  kind,
					Group: groupName,
					Items: items,
				})
			}
		}
	}

	return plan
}

// filterExistingItems returns items that don't exist at destination
func (p *FileProvider) filterExistingItems(group resource.ResourceGroup) []resource.ResourceItem {
	var result []resource.ResourceItem
	for _, item := range group.Items {
		dest, ok := item.Extra["destination"].(string)
		if !ok || dest == "" {
			result = append(result, item)
			continue
		}
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			result = append(result, item)
		}
	}
	return result
}

// compareGroupItems compares desired group items with state
func (p *FileProvider) compareGroupItems(
	group resource.ResourceGroup,
	stateGroup provider.ResourceState,
) (additions, removals []resource.ResourceItem, modifications []provider.ItemChange, inSync []resource.ItemState) {
	stateItems := make(map[string]resource.ItemState)
	for _, item := range stateGroup.Items {
		stateItems[item.Name] = item
	}

	for _, desiredItem := range group.Items {
		name := desiredItem.Name
		dest, _ := desiredItem.Extra["destination"].(string)

		stateItem, inState := stateItems[name]
		destExists := false
		if dest != "" {
			_, err := os.Stat(dest)
			destExists = !os.IsNotExist(err)
		}

		if !destExists {
			additions = append(additions, desiredItem)
		} else if inState {
			// Check if content changed
			currentHash := p.hashFile(dest)
			if currentHash != stateItem.Checksum {
				modifications = append(modifications, provider.ItemChange{
					ItemName: name,
					OldState: stateItem,
					NewState: resource.ItemState{
						Name:     name,
						Checksum: currentHash,
						Status:   "present",
					},
					Diff: "content changed",
				})
			} else {
				inSync = append(inSync, stateItem)
			}
		} else {
			// File exists but not tracked
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

// hashFile computes SHA256 hash of file content
func (p *FileProvider) hashFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Apply executes the given GroupPlan
func (p *FileProvider) Apply(ctx context.Context, plan provider.GroupPlan) error {
	for _, addition := range plan.Additions {
		if err := p.applyGroupAddition(ctx, addition); err != nil {
			return fmt.Errorf("failed to add %s: %w", addition.Group, err)
		}
	}

	for _, removal := range plan.Removals {
		if err := p.applyGroupRemoval(ctx, removal); err != nil {
			return fmt.Errorf("failed to remove %s: %w", removal.Group, err)
		}
	}

	for _, modification := range plan.Modifications {
		if err := p.applyGroupModification(ctx, modification); err != nil {
			return fmt.Errorf("failed to modify %s: %w", modification.Group, err)
		}
	}

	return nil
}

// applyGroupAddition handles file/directory creation
func (p *FileProvider) applyGroupAddition(ctx context.Context, addition provider.GroupAddition) error {
	for _, item := range addition.Items {
		dest, _ := item.Extra["destination"].(string)
		source, _ := item.Extra["source"].(string)

		if dest == "" {
			continue
		}

		// Ensure parent directory exists
		parent := filepath.Dir(dest)
		if err := os.MkdirAll(parent, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", dest, err)
		}

		if source != "" && source != "(inline)" {
			// Copy from source
			sourcePath := filepath.Join(p.dotfilesRoot, source)
			if err := p.copyFile(sourcePath, dest); err != nil {
				return fmt.Errorf("failed to copy %s to %s: %w", sourcePath, dest, err)
			}
		} else {
			// Create empty file or handle inline content
			if err := os.WriteFile(dest, []byte{}, 0644); err != nil {
				return fmt.Errorf("failed to create %s: %w", dest, err)
			}
		}
	}
	return nil
}

// applyGroupRemoval handles file/directory removal
func (p *FileProvider) applyGroupRemoval(ctx context.Context, removal provider.GroupRemoval) error {
	for _, item := range removal.Items {
		dest, _ := item.Extra["destination"].(string)
		if dest == "" {
			dest = item.Name
		}

		if err := os.RemoveAll(dest); err != nil {
			return fmt.Errorf("failed to remove %s: %w", dest, err)
		}
	}
	return nil
}

// applyGroupModification handles file/directory updates
func (p *FileProvider) applyGroupModification(ctx context.Context, modification provider.GroupModification) error {
	// Re-apply the file
	return p.applyGroupAddition(ctx, provider.GroupAddition{
		Kind:  modification.Kind,
		Group: modification.Group,
		Items: func() []resource.ResourceItem {
			var items []resource.ResourceItem
			for _, change := range modification.Changes {
				items = append(items, resource.ResourceItem{
					Name:  change.ItemName,
					Extra: change.NewState.Extra,
				})
			}
			return items
		}(),
	})
}

// copyFile copies a file from src to dst
func (p *FileProvider) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// Import discovers an existing file
func (p *FileProvider) Import(ctx context.Context, group string) (provider.ResourceState, error) {
	return provider.ResourceState{}, fmt.Errorf("use ImportItem to import specific files")
}

// ImportItem imports a specific file
func (p *FileProvider) ImportItem(ctx context.Context, group string, item string) (provider.ResourceState, error) {
	// Check if file exists
	if _, err := os.Stat(item); os.IsNotExist(err) {
		return provider.ResourceState{}, fmt.Errorf("file %s does not exist", item)
	}

	hash := p.hashFile(item)
	return provider.ResourceState{
		Kind:      "ManagedFile",
		Group:     group,
		Namespace: "default",
		Items: []resource.ItemState{
			{
				Name:     item,
				Checksum: hash,
				Status:   "present",
			},
		},
	}, nil
}
