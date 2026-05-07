package provider

import (
	"fmt"
	"strings"

	"github.com/wasilak/dotisan/pkg/resource"
)

// IndexStateByGroup indexes state entries by group name, filtered to a single kind.
func IndexStateByGroup(state []ResourceState, kind string) map[string]ResourceState {
	index := make(map[string]ResourceState)
	for _, s := range state {
		if s.Kind == kind {
			index[s.Group] = s
		}
	}
	return index
}

// CompareGroupItems compares desired items against state and installed packages.
// Returns additions, removals, modifications, and in-sync items.
// normalizeName converts a desired item name to the lookup key used in installed;
// pass nil for identity.
func CompareGroupItems(
	group resource.ResourceGroup[any],
	stateGroup ResourceState,
	installed map[string]string,
	normalizeName func(string) string,
) (additions, removals []resource.ResourceItem, modifications []ItemChange, inSync []resource.ItemState) {
	if normalizeName == nil {
		normalizeName = func(name string) string { return name }
	}
	stateItems := make(map[string]resource.ItemState)
	for _, item := range stateGroup.Items {
		stateItems[item.Name] = item
	}

	for _, desiredItem := range group.Items {
		name := desiredItem.Name
		_, inState := stateItems[name]
		_, isInstalled := installed[normalizeName(name)]

		if inState {
			// State is the source of truth — if managed, treat as present regardless
			// of whether the binary scan found it (binary may be installed via a
			// different mechanism, e.g. brew).
			stateItem := stateItems[name]
			if stateItem.Version != desiredItem.Version && desiredItem.Version != "" && desiredItem.Version != "latest" {
				modifications = append(modifications, ItemChange{
					ItemName: name,
					OldState: stateItem,
					NewState: resource.ItemState{Name: name, Version: desiredItem.Version, Status: "present"},
					Diff:     fmt.Sprintf("version: %s -> %s", stateItem.Version, desiredItem.Version),
				})
			} else {
				inSync = append(inSync, stateItem)
			}
		} else if !isInstalled {
			additions = append(additions, desiredItem)
		} else {
			// Installed but not yet tracked in state — needs import.
			additions = append(additions, desiredItem)
		}
	}

	desiredItems := make(map[string]bool)
	for _, item := range group.Items {
		desiredItems[item.Name] = true
	}

	for name, stateItem := range stateItems {
		if !desiredItems[name] {
			removals = append(removals, resource.ResourceItem{Name: name, Version: stateItem.Version})
		}
	}

	return
}

// BaseReconcile implements the common reconciliation pattern for simple providers
// (npm, cargo, go). The normalizeName func converts a desired item name to the
// lookup key used in the installed map; pass nil for identity (name unchanged).
func BaseReconcile(
	kind string,
	desired []resource.ResourceGroup[any],
	stateResources []ResourceState,
	installed map[string]string,
	normalizeName func(string) string,
) GroupPlan {
	if normalizeName == nil {
		normalizeName = func(name string) string { return name }
	}

	plan := GroupPlan{}
	stateIndex := IndexStateByGroup(stateResources, kind)

	for _, group := range desired {
		if group.Kind != kind {
			continue
		}

		stateGroup, exists := stateIndex[group.Name]

		if !exists {
			var toInstall, toImport []resource.ResourceItem
			for _, item := range group.Items {
				if _, isInstalled := installed[normalizeName(item.Name)]; isInstalled {
					toImport = append(toImport, item)
				} else {
					toInstall = append(toInstall, item)
				}
			}

			if len(toInstall) > 0 {
				plan.Additions = append(plan.Additions, GroupAddition{
					Kind: group.Kind, Group: group.Name, Items: toInstall,
				})
			}

			if len(toImport) > 0 {
				plan.Additions = append(plan.Additions, GroupAddition{
					Kind: group.Kind, Group: group.Name, Items: toImport,
				})
				itemNames := make([]string, 0, len(toImport))
				for _, item := range toImport {
					itemNames = append(itemNames, item.Name)
				}
				plan.Warnings = append(plan.Warnings, PlanWarning{
					GroupID:    fmt.Sprintf("%s/%s", group.Kind, group.Name),
					Severity:   "warning",
					Message:    fmt.Sprintf("Items already installed but not tracked: %s", strings.Join(itemNames, ", ")),
					Suggestion: fmt.Sprintf("dotisan state import %s/%s[<item>]", group.Kind, group.Name),
				})
			}
		} else {
			additions, removals, modifications, inSync := CompareGroupItems(group, stateGroup, installed, normalizeName)

			if len(additions) > 0 {
				plan.Additions = append(plan.Additions, GroupAddition{
					Kind: group.Kind, Group: group.Name, Items: additions,
				})
			}
			if len(removals) > 0 {
				plan.Removals = append(plan.Removals, GroupRemoval{
					Kind: group.Kind, Group: group.Name, Items: removals,
				})
			}
			if len(modifications) > 0 {
				plan.Modifications = append(plan.Modifications, GroupModification{
					Kind: group.Kind, Group: group.Name, Changes: modifications,
				})
			}
			if len(inSync) > 0 && len(additions) == 0 && len(removals) == 0 && len(modifications) == 0 {
				plan.InSync = append(plan.InSync, GroupState{
					Kind: group.Kind, Group: group.Name, Items: inSync,
				})
			}
		}
	}

	// Groups in state but absent from desired → whole-group removals
	desiredGroups := make(map[string]bool)
	for _, group := range desired {
		if group.Kind == kind {
			desiredGroups[group.Name] = true
		}
	}
	for groupName, stateGroup := range stateIndex {
		if !desiredGroups[groupName] {
			items := make([]resource.ResourceItem, 0, len(stateGroup.Items))
			for _, item := range stateGroup.Items {
				items = append(items, resource.ResourceItem{Name: item.Name, Version: item.Version})
			}
			plan.Removals = append(plan.Removals, GroupRemoval{
				Kind: kind, Group: groupName, Items: items,
			})
		}
	}

	return plan
}
