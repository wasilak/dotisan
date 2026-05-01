package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/wasilak/dotisan/pkg/style"
)

// StateMvOptions contains options for the StateMv operation.
type StateMvOptions struct {
	Source      string
	Destination string
}

// StateMvResult contains the result of a state move operation.
type StateMvResult struct {
	SrcKind  string
	SrcGroup string
	SrcItem  string
	DstKind  string
	DstGroup string
	DstItem  string
	Success  bool
}

// parseStateMvRef parses a state mv reference in the form Kind/Group[Item] or Kind/Group.
// The bracket notation avoids ambiguity when Group itself contains slashes.
func parseStateMvRef(ref string) (kind, group, item string, err error) {
	firstSlash := strings.IndexByte(ref, '/')
	if firstSlash < 0 {
		return "", "", "", fmt.Errorf("invalid reference format: %s (expected Kind/Group or Kind/Group[Item])", ref)
	}
	kind = ref[:firstSlash]
	rest := ref[firstSlash+1:]

	if bracketIdx := strings.IndexByte(rest, '['); bracketIdx >= 0 {
		if !strings.HasSuffix(rest, "]") {
			return "", "", "", fmt.Errorf("invalid reference format: %s (unclosed '[')", ref)
		}
		group = rest[:bracketIdx]
		item = rest[bracketIdx+1 : len(rest)-1]
	} else {
		group = rest
	}
	return kind, group, item, nil
}

// StateMv moves an item from one resource group to another in state only.
// The source item must exist in state, and the destination group must exist in desired config.
func (e *Engine) StateMv(ctx context.Context, opts StateMvOptions) (*StateMvResult, error) {
	// Parse source
	srcKind, srcGroup, srcItem, err := parseStateMvRef(opts.Source)
	if err != nil {
		return nil, fmt.Errorf("invalid source: %w", err)
	}

	// Parse destination
	dstKind, dstGroup, dstItem, err := parseStateMvRef(opts.Destination)
	if err != nil {
		return nil, fmt.Errorf("invalid destination: %w", err)
	}

	// Load current state
	state, err := e.StateBackend.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Load desired resources to validate destination group exists
	resources, err := e.loadResources()
	if err != nil {
		return nil, fmt.Errorf("failed to load desired resources: %w", err)
	}

	// Validate source item exists in state
	srcRes, exists := state.GetResourceGroup(srcKind, srcGroup)
	if !exists {
		return nil, fmt.Errorf("source group %s/%s not found in state", srcKind, srcGroup)
	}

	// Validate source item exists with better error message
	found := false
	var availableItems []string
	for _, item := range srcRes.Items {
		availableItems = append(availableItems, item.Name)
		if item.Name == srcItem {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("source item %q not found in group %s/%s\n\nAvailable items:\n  %s",
			srcItem, srcKind, srcGroup, strings.Join(availableItems, "\n  "))
	}

	// If destination item name not provided, use source item name
	if dstItem == "" {
		dstItem = srcItem
	}

	// Validate source and destination kinds match
	if srcKind != dstKind {
		return nil, fmt.Errorf("cannot move between different kinds: %s → %s", srcKind, dstKind)
	}

	// Validate destination group exists in desired config
	dstGroupExists := false
	var availableGroups []string
	for _, res := range resources {
		if res.GetKind() == dstKind {
			groupName := res.GetMetadata().Name
			availableGroups = append(availableGroups, groupName)
			if groupName == dstGroup {
				dstGroupExists = true
				break
			}
		}
	}
	if !dstGroupExists {
		return nil, fmt.Errorf("destination group %s/%s does not exist in desired configuration\n\nAvailable groups in %s:\n  %s",
			dstKind, dstGroup, dstKind, strings.Join(availableGroups, "\n  "))
	}

	// Validate destination group actually contains the item in desired config
	dstGroupHasItem := false
	var availableItemsInDst []string
	for _, res := range resources {
		if res.GetKind() == dstKind && res.GetMetadata().Name == dstGroup {
			group := res.ToGroup()
			for _, item := range group.Items {
				availableItemsInDst = append(availableItemsInDst, item.Name)
				if item.Name == dstItem {
					dstGroupHasItem = true
					break
				}
			}
			break
		}
	}
	if !dstGroupHasItem {
		return nil, fmt.Errorf("destination group %s/%s does not contain item %q in desired configuration\n\nAvailable items in %s/%s:\n  %s",
			dstKind, dstGroup, dstItem, dstKind, dstGroup, strings.Join(availableItemsInDst, "\n  "))
	}

	// Validate destination doesn't already have the item
	dstRes, exists := state.GetResourceGroup(dstKind, dstGroup)
	if exists {
		for _, item := range dstRes.Items {
			if item.Name == dstItem {
				return nil, fmt.Errorf("destination group %s/%s already contains item %s", dstKind, dstGroup, dstItem)
			}
		}
	}

	// Perform the move
	_, success := state.MoveItem(srcKind, srcGroup, srcItem, dstKind, dstGroup, dstItem)
	if !success {
		return nil, fmt.Errorf("failed to move item (internal error)")
	}

	// Save state
	if err := e.StateBackend.Save(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to save state: %w", err)
	}

	return &StateMvResult{
		SrcKind:  srcKind,
		SrcGroup: srcGroup,
		SrcItem:  srcItem,
		DstKind:  dstKind,
		DstGroup: dstGroup,
		DstItem:  dstItem,
		Success:  true,
	}, nil
}

// DisplayStateMvResult displays the result of a state move operation.
func DisplayStateMvResult(result *StateMvResult) {
	fmt.Println()
	fmt.Printf("%s Successfully moved %s\n", style.IconSuccess, style.Success.Render(result.DstItem))
	fmt.Printf("  From: %s/%s\n", style.Dim.Render(result.SrcKind), style.Dim.Render(result.SrcGroup))
	fmt.Printf("  To:   %s/%s\n", style.Dim.Render(result.DstKind), style.Dim.Render(result.DstGroup))
	fmt.Println()
}
