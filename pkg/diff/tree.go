package diff

import (
	"regexp"
	"strings"

	"charm.land/lipgloss/v2/tree"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
	"github.com/wasilak/dotisan/pkg/style"
)

// TreeFormatter renders resources as a 3-level tree: Kind / ResourceName / Items
type TreeFormatter struct {
	enumeratorStyle lipgloss.Style
	kindStyle       lipgloss.Style
	nameStyle       lipgloss.Style
	itemStyle       lipgloss.Style
	actionStyle     lipgloss.Style
}

// NewTreeFormatter creates a new TreeFormatter with default colors
func NewTreeFormatter() *TreeFormatter {
	return &TreeFormatter{
		enumeratorStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("63")),
		kindStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("77")).Bold(true),
		nameStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("222")),
		itemStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("212")),
		actionStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	}
}

// PlanResultInfo holds the plan counts for tree rendering
type PlanResultInfo struct {
	ProviderPlans      map[string]provider.Plan
	TotalAdditions     int
	TotalModifications int
	TotalRemovals      int
	TotalDrifted       int
}

// FormatPlanAsTree renders the entire plan as a 3-level tree
func (f *TreeFormatter) FormatPlanAsTree(result PlanResultInfo) string {
	var sections []string

	if result.TotalRemovals > 0 {
		sections = append(sections, f.formatSection("Resources to be removed", result.ProviderPlans, "remove", style.Error))
	}
	if result.TotalAdditions > 0 {
		sections = append(sections, f.formatSection("Resources to be created", result.ProviderPlans, "create", style.Success))
	}
	if result.TotalModifications > 0 {
		sections = append(sections, f.formatSection("Resources to be modified", result.ProviderPlans, "modify", style.Warning))
	}
	if result.TotalDrifted > 0 {
		sections = append(sections, f.formatSection("Drifted resources (will be restored)", result.ProviderPlans, "restore", style.Warning))
	}

	return strings.Join(sections, "\n\n")
}

// itemKeyRegex matches resource names with item keys like "core-tools[ripgrep]"
var itemKeyRegex = regexp.MustCompile(`^([^\[]+)\[(.+)\]$`)

// parseResourceName parses a resource name that may contain an item key
// Returns (baseName, itemKey, hasItemKey)
func parseResourceName(name string) (string, string, bool) {
	matches := itemKeyRegex.FindStringSubmatch(name)
	if matches == nil {
		return name, "", false
	}
	return matches[1], matches[2], true
}

func (f *TreeFormatter) formatSection(title string, providerPlans map[string]provider.Plan, actionType string, titleStyle lipgloss.Style) string {
	root := tree.Root(titleStyle.Render(title)).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(f.enumeratorStyle)

	// Collect resources grouped by Kind, then by Name
	// Map: Kind -> Name -> []items
	byKind := make(map[string]map[string][]string)

	for _, plan := range providerPlans {
		switch actionType {
		case "create":
			for _, res := range plan.Additions {
				f.addResourceToMap(res, "will be created", byKind)
			}
		case "remove":
			for _, res := range plan.Removals {
				f.addResourceToMap(res, "will be destroyed", byKind)
			}
		case "modify":
			for _, mod := range plan.Modifications {
				f.addResourceToMap(mod.Resource, "will be updated", byKind)
			}
		case "restore":
			for _, drift := range plan.Drifted {
				f.addResourceToMap(drift.Resource, "will be restored", byKind)
			}
		}
	}

	// Build tree: Kind -> Name -> Items with action
	for kind, namesMap := range byKind {
		kindNode := tree.New().
			Root(f.kindStyle.Render(kind)).
			Enumerator(tree.RoundedEnumerator).
			EnumeratorStyle(f.enumeratorStyle)

		for name, items := range namesMap {
			nameNode := tree.New().
				Root(f.nameStyle.Render(name)).
				Enumerator(tree.RoundedEnumerator).
				EnumeratorStyle(f.enumeratorStyle)

			for _, item := range items {
				nameNode.Child(f.itemStyle.Render(item))
			}

			kindNode.Child(nameNode)
		}

		root.Child(kindNode)
	}

	return root.String()
}

// addResourceToMap adds a resource's items to the nested map structure
func (f *TreeFormatter) addResourceToMap(res resource.Resource, action string, byKind map[string]map[string][]string) {
	kind := res.GetKind()
	fullName := res.GetMetadata().Name

	// Try to parse name as "basename[itemkey]"
	baseName, itemKey, hasItemKey := parseResourceName(fullName)

	// Initialize nested maps if needed
	if byKind[kind] == nil {
		byKind[kind] = make(map[string][]string)
	}

	if hasItemKey {
		// This is an indexed resource - itemKey is the actual item
		byKind[kind][baseName] = append(byKind[kind][baseName], itemKey+" "+action)
	} else {
		// Try to extract items from the resource spec
		items := extractItemsWithAction(res, action)
		if len(items) > 0 {
			byKind[kind][fullName] = append(byKind[kind][fullName], items...)
		} else {
			// No items - use resource name itself
			byKind[kind][fullName] = append(byKind[kind][fullName], fullName+" "+action)
		}
	}
}

// extractItemsWithAction extracts child items from a resource and appends action
func extractItemsWithAction(res resource.Resource, action string) []string {
	switch r := res.(type) {
	case *resource.BrewPackages:
		var items []string
		for _, formula := range r.Spec.Formulae {
			items = append(items, formula.Name+" "+action)
		}
		for _, cask := range r.Spec.Casks {
			items = append(items, cask.Name+" (cask) "+action)
		}
		return items
	case *resource.NpmPackages:
		var items []string
		for _, pkg := range r.Spec.Packages {
			items = append(items, pkg.Name+" "+action)
		}
		return items
	case *resource.GoPackages:
		var items []string
		for _, pkg := range r.Spec.Packages {
			items = append(items, pkg.Module+" "+action)
		}
		return items
	case *resource.CargoPackages:
		var items []string
		for _, pkg := range r.Spec.Packages {
			items = append(items, pkg.Name+" "+action)
		}
		return items
	case *resource.ManagedFile:
		if r.Spec.SourceFile != "" {
			return []string{r.Spec.SourceFile + " " + action}
		}
		return []string{"(inline content) " + action}
	case *resource.ManagedDirectory:
		return []string{r.Spec.SourceDir + " → " + r.Spec.Destination + " " + action}
	default:
		return nil
	}
}

// StateResource represents a resource in state for tree rendering
type StateResource struct {
	Kind   string
	Name   string
	ID     string
	Status string
}

// FormatStateAsTree renders state list as a 3-level tree
func (f *TreeFormatter) FormatStateAsTree(resources []StateResource) string {
	root := tree.Root(style.Header.Render("Managed Resources")).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(f.enumeratorStyle)

	// Group by Kind, then by Name
	byKind := make(map[string]map[string][]string)

	for _, res := range resources {
		// Parse name to extract base name and item key
		baseName, itemKey, hasItemKey := parseResourceName(res.Name)

		if byKind[res.Kind] == nil {
			byKind[res.Kind] = make(map[string][]string)
		}

		if hasItemKey {
			// The itemKey is the actual package/item
			itemText := itemKey
			if res.Status != "" {
				itemText = itemText + " " + res.Status
			}
			byKind[res.Kind][baseName] = append(byKind[res.Kind][baseName], itemText)
		} else {
			// No item key - use the full name
			itemText := res.ID
			if res.Status != "" {
				itemText = itemText + " " + res.Status
			}
			byKind[res.Kind][res.Name] = append(byKind[res.Kind][res.Name], itemText)
		}
	}

	for kind, namesMap := range byKind {
		kindNode := tree.New().
			Root(f.kindStyle.Render(kind)).
			Enumerator(tree.RoundedEnumerator).
			EnumeratorStyle(f.enumeratorStyle)

		for name, items := range namesMap {
			nameNode := tree.New().
				Root(f.nameStyle.Render(name)).
				Enumerator(tree.RoundedEnumerator).
				EnumeratorStyle(f.enumeratorStyle)

			for _, item := range items {
				nameNode.Child(f.itemStyle.Render(item))
			}

			kindNode.Child(nameNode)
		}

		root.Child(kindNode)
	}

	return root.String()
}
