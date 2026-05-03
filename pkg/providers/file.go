package providers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wasilak/dotisan/pkg/planctx"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

// FileProvider implements the Provider interface for ManagedFile
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
	// Diff flag via context
	showDiff := false
	if v := ctx.Value(planctx.PlanShowDiffKey); v != nil {
		if b, ok := v.(bool); ok {
			showDiff = b
		}
	}
	// Index state by kind and group
	stateIndex := make(map[string]map[string]provider.ResourceState)
	for _, s := range state {
		if s.Kind == "ManagedFile" {
			if stateIndex[s.Kind] == nil {
				stateIndex[s.Kind] = make(map[string]provider.ResourceState)
			}
			stateIndex[s.Kind][s.Group] = s
		}
	}

	// Process each desired group
	for _, group := range desired {
		if group.Kind != "ManagedFile" {
			continue
		}

		kindIndex := stateIndex[group.Kind]
		stateGroup, exists := kindIndex[group.Name]

		if !exists {
			// New group - all items are additions (bring them under management even if dest exists)
			plan.Additions = append(plan.Additions, provider.GroupAddition{
				Kind:  group.Kind,
				Group: group.Name,
				Items: group.Items,
			})
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
		if group.Kind == "ManagedFile" {
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

	if showDiff {
		for ai := range plan.Additions {
			for _, item := range plan.Additions[ai].Items {
				source, _ := item.Extra["source"].(string)
				inline, _ := item.Extra["inline"].(string)
				var content []byte
				if source != "" && source != "(inline)" {
					content, _ = os.ReadFile(p.resolveSource(source))
				} else if inline != "" {
					content = []byte(inline)
				}
				if len(content) > 0 {
					if plan.Additions[ai].Contents == nil {
						plan.Additions[ai].Contents = make(map[string]string)
					}
					plan.Additions[ai].Contents[item.Name] = string(content)
				}
			}
		}
		for ri := range plan.Removals {
			for _, item := range plan.Removals[ri].Items {
				dest, _ := item.Extra["destination"].(string)
				if dest == "" {
					dest = item.Name
				}
				content, _ := os.ReadFile(dest)
				if len(content) > 0 {
					if plan.Removals[ri].Contents == nil {
						plan.Removals[ri].Contents = make(map[string]string)
					}
					plan.Removals[ri].Contents[item.Name] = string(content)
				}
			}
		}
		for mi := range plan.Modifications {
			for ci, change := range plan.Modifications[mi].Changes {
				dest, _ := change.NewState.Extra["destination"].(string)
				source, _ := change.NewState.Extra["source"].(string)
				// If inline content was provided in resource, it may be in Extra["inline"].
				inline, _ := change.NewState.Extra["inline"].(string)
				oldContent, newContent := []byte{}, []byte{}
				if dest != "" {
					oldContent, _ = os.ReadFile(dest)
				}
				if source != "" && source != "(inline)" {
					newContent, _ = os.ReadFile(p.resolveSource(source))
				} else if inline != "" {
					newContent = []byte(inline)
				}
				plan.Modifications[mi].Changes[ci].OldContent = string(oldContent)
				plan.Modifications[mi].Changes[ci].NewContent = string(newContent)
			}
		}
	}
	return plan
}

// resolveSource returns the full path for a source value. Absolute paths are
// used as-is; relative paths are joined with the dotfiles root.
func (p *FileProvider) resolveSource(source string) string {
	if filepath.IsAbs(source) {
		return source
	}
	// Relative source paths are resolved against the dotfiles root
	return filepath.Join(p.dotfilesRoot, source)
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

		// Compute desired content checksum (from inline or source)
		desiredHash := ""
		if inline, ok := desiredItem.Extra["inline"].(string); ok && inline != "" {
			h := sha256.Sum256([]byte(inline))
			desiredHash = hex.EncodeToString(h[:])
		} else if source, ok := desiredItem.Extra["source"].(string); ok && source != "" && source != "(inline)" {
			sourcePath := p.resolveSource(source)
			desiredHash = p.hashFile(sourcePath)
		}

		if !destExists {
			additions = append(additions, desiredItem)
			continue
		}

		// Destination exists; compute current hash
		currentHash := p.hashFile(dest)

		if inState {
			if desiredHash != "" {
				if desiredHash != currentHash {
					modifications = append(modifications, provider.ItemChange{
						ItemName: name,
						OldState: stateItem,
						NewState: resource.ItemState{
							Name:     name,
							Checksum: desiredHash,
							Status:   "present",
							Extra:    desiredItem.Extra,
						},
						Diff: "content changed",
					})
				} else {
					inSync = append(inSync, stateItem)
				}
			} else {
				// Fallback: compare current vs saved state
				if currentHash != stateItem.Checksum {
					modifications = append(modifications, provider.ItemChange{
						ItemName: name,
						OldState: stateItem,
						NewState: resource.ItemState{
							Name:     name,
							Checksum: currentHash,
							Status:   "present",
							Extra:    desiredItem.Extra,
						},
						Diff: "content changed",
					})
				} else {
					inSync = append(inSync, stateItem)
				}
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
			sourcePath := p.resolveSource(source)
			if err := p.copyFile(sourcePath, dest); err != nil {
				return fmt.Errorf("failed to copy %s to %s: %w", sourcePath, dest, err)
			}
		} else {
			inline, _ := item.Extra["inline"].(string)
			if err := os.WriteFile(dest, []byte(inline), 0644); err != nil {
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

// Import is intentionally not implemented for FileProvider. ImportItem support
// was removed: provider-level import/export functionality is deprecated and
// handled outside providers.
func (p *FileProvider) Import(ctx context.Context, group string) (provider.ResourceState, error) {
	return provider.ResourceState{}, fmt.Errorf("import not supported for provider file")
}
