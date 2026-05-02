package engine

import (
	"strings"

	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

// filterResourceGroupsByTargets filters resource groups and their items according to targets.
func filterResourceGroupsByTargets(groups []resource.ResourceGroup, targets []TargetMatch) []resource.ResourceGroup {
	var out []resource.ResourceGroup
	for _, g := range groups {
		matched := false
		// Check kind/group/item match for group-level and item-level targeting
		for _, t := range targets {
			// Manually check kind/group first (case-insensitive) so we can
			// handle item-specific targets which require passing the item name
			// into Matches. Using Matches with an empty item would fail when
			// the target specified an item.
			if t.Kind != "" && !strings.EqualFold(t.Kind, g.Kind) {
				continue
			}
			if t.Group != "" && !strings.EqualFold(t.Group, g.Name) {
				continue
			}

			if t.Item == "" {
				// target is kind or kind/group -> keep full group
				matched = true
				break
			}

			// target specifies an item; check if the item exists in group
			for _, it := range g.Items {
				if t.Matches(g.Kind, g.Name, it.Name) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if matched {
			out = append(out, g)
		}
	}
	return out
}

// matchesKind returns true if the target kind matches the resource kind.
// Only the full/display kind (e.g. "HomebrewPackages") is accepted.
// Matching is case-insensitive.
func matchesKind(targetKind, resourceKind string) bool {
	return strings.EqualFold(targetKind, resourceKind)
}

// (removed kindToFullKind) Full/display kinds are used directly in resources.

// filterPlanByTargets trims a provider.GroupPlan to include only items that match targets.
func filterPlanByTargets(plan provider.GroupPlan, targets []TargetMatch) provider.GroupPlan {
	// Helper to check if a kind/group/item is targeted
	isTargeted := func(kind, group, item string) bool {
		for _, t := range targets {
			// match kind (normalize aliases)
			if t.Kind != "" {
				if !strings.EqualFold(t.Kind, kind) {
					continue
				}
			}
			if t.Group != "" && !strings.EqualFold(t.Group, group) {
				continue
			}
			if t.Item != "" && !strings.EqualFold(t.Item, item) {
				continue
			}
			return true
		}
		return false
	}

	var out provider.GroupPlan

	// Filter additions
	for _, a := range plan.Additions {
		var items []resource.ResourceItem
		for _, it := range a.Items {
			if isTargeted(a.Kind, a.Group, it.Name) {
				items = append(items, it)
			}
		}
		if len(items) > 0 {
			out.Additions = append(out.Additions, provider.GroupAddition{Kind: a.Kind, Group: a.Group, Items: items})
		}
	}

	// Filter modifications
	for _, m := range plan.Modifications {
		var changes []provider.ItemChange
		for _, c := range m.Changes {
			if isTargeted(m.Kind, m.Group, c.ItemName) {
				changes = append(changes, c)
			}
		}
		if len(changes) > 0 {
			out.Modifications = append(out.Modifications, provider.GroupModification{Kind: m.Kind, Group: m.Group, Changes: changes})
		}
	}

	// Filter removals
	for _, r := range plan.Removals {
		var items []resource.ResourceItem
		for _, it := range r.Items {
			if isTargeted(r.Kind, r.Group, it.Name) {
				items = append(items, it)
			}
		}
		if len(items) > 0 {
			out.Removals = append(out.Removals, provider.GroupRemoval{Kind: r.Kind, Group: r.Group, Items: items})
		}
	}

	// Filter cleanup
	for _, c := range plan.Cleanup {
		var items []resource.ResourceItem
		for _, it := range c.Items {
			if isTargeted(c.Kind, c.Group, it.Name) {
				items = append(items, it)
			}
		}
		if len(items) > 0 {
			out.Cleanup = append(out.Cleanup, provider.GroupCleanup{Kind: c.Kind, Group: c.Group, Items: items, Reason: c.Reason})
		}
	}

	// Filter drifted
	for _, d := range plan.Drifted {
		if isTargeted(d.Kind, d.Group, d.Item) {
			out.Drifted = append(out.Drifted, d)
		}
	}

	// InSync: include only if targeted (use items match)
	for _, s := range plan.InSync {
		var items []resource.ItemState
		for _, it := range s.Items {
			if isTargeted(s.Kind, s.Group, it.Name) {
				items = append(items, it)
			}
		}
		if len(items) > 0 {
			out.InSync = append(out.InSync, provider.GroupState{Kind: s.Kind, Group: s.Group, Items: items, Version: s.Version})
		}
	}

	// Warnings are retained if they refer to targeted groups/items
	for _, w := range plan.Warnings {
		if w.GroupID == "" && w.ItemID == "" {
			continue
		}
		// Basic check: if groupID present, split and check
		if w.GroupID != "" {
			parts := strings.SplitN(w.GroupID, "/", 2)
			if len(parts) >= 1 {
				gkind := parts[0]
				ggroup := ""
				if len(parts) == 2 {
					ggroup = parts[1]
				}
				if isTargeted(gkind, ggroup, w.ItemID) {
					out.Warnings = append(out.Warnings, w)
				}
			}
		}
	}

	return out
}
