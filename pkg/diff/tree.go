package diff

import (
	"strings"

	"charm.land/lipgloss/v2/tree"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/wasilak/dotisan/pkg/provider"
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

// GroupPlanInfo holds the plan for tree rendering
type GroupPlanInfo struct {
	Plan provider.GroupPlan
}

// FormatGroupPlanAsTree renders the entire plan as a 3-level tree
func (f *TreeFormatter) FormatGroupPlanAsTree(info GroupPlanInfo) string {
	var sections []string

	if len(info.Plan.Removals) > 0 {
		sections = append(sections, f.formatRemovals(info.Plan.Removals))
	}
	if len(info.Plan.Additions) > 0 {
		sections = append(sections, f.formatAdditions(info.Plan.Additions))
	}
	if len(info.Plan.Modifications) > 0 {
		sections = append(sections, f.formatModifications(info.Plan.Modifications))
	}
	if len(info.Plan.Drifted) > 0 {
		sections = append(sections, f.formatDrifted(info.Plan.Drifted))
	}

	return strings.Join(sections, "\n\n")
}

func (f *TreeFormatter) formatAdditions(additions []provider.GroupAddition) string {
	root := tree.Root(style.Success.Render("Resources to be created")).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(f.enumeratorStyle)

	// Group by Kind
	byKind := make(map[string][]provider.GroupAddition)
	for _, addition := range additions {
		byKind[addition.Kind] = append(byKind[addition.Kind], addition)
	}

	for kind, groups := range byKind {
		kindNode := tree.New().
			Root(f.kindStyle.Render(kind)).
			Enumerator(tree.RoundedEnumerator).
			EnumeratorStyle(f.enumeratorStyle)

		for _, group := range groups {
			groupNode := tree.New().
				Root(f.nameStyle.Render(group.Group)).
				Enumerator(tree.RoundedEnumerator).
				EnumeratorStyle(f.enumeratorStyle)

			for _, item := range group.Items {
				itemText := f.itemStyle.Render(item.Name) + " " + f.actionStyle.Render("will be created")
				groupNode.Child(itemText)
			}

			kindNode.Child(groupNode)
		}

		root.Child(kindNode)
	}

	return root.String()
}

func (f *TreeFormatter) formatRemovals(removals []provider.GroupRemoval) string {
	root := tree.Root(style.Error.Render("Resources to be removed")).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(f.enumeratorStyle)

	byKind := make(map[string][]provider.GroupRemoval)
	for _, removal := range removals {
		byKind[removal.Kind] = append(byKind[removal.Kind], removal)
	}

	for kind, groups := range byKind {
		kindNode := tree.New().
			Root(f.kindStyle.Render(kind)).
			Enumerator(tree.RoundedEnumerator).
			EnumeratorStyle(f.enumeratorStyle)

		for _, group := range groups {
			groupNode := tree.New().
				Root(f.nameStyle.Render(group.Group)).
				Enumerator(tree.RoundedEnumerator).
				EnumeratorStyle(f.enumeratorStyle)

			for _, item := range group.Items {
				itemText := f.itemStyle.Render(item.Name) + " " + f.actionStyle.Render("will be destroyed")
				groupNode.Child(itemText)
			}

			kindNode.Child(groupNode)
		}

		root.Child(kindNode)
	}

	return root.String()
}

func (f *TreeFormatter) formatModifications(modifications []provider.GroupModification) string {
	root := tree.Root(style.Warning.Render("Resources to be modified")).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(f.enumeratorStyle)

	byKind := make(map[string][]provider.GroupModification)
	for _, mod := range modifications {
		byKind[mod.Kind] = append(byKind[mod.Kind], mod)
	}

	for kind, groups := range byKind {
		kindNode := tree.New().
			Root(f.kindStyle.Render(kind)).
			Enumerator(tree.RoundedEnumerator).
			EnumeratorStyle(f.enumeratorStyle)

		for _, group := range groups {
			groupNode := tree.New().
				Root(f.nameStyle.Render(group.Group)).
				Enumerator(tree.RoundedEnumerator).
				EnumeratorStyle(f.enumeratorStyle)

			for _, change := range group.Changes {
				itemText := f.itemStyle.Render(change.ItemName) + " " + f.actionStyle.Render("will be updated")
				groupNode.Child(itemText)
			}

			kindNode.Child(groupNode)
		}

		root.Child(kindNode)
	}

	return root.String()
}

func (f *TreeFormatter) formatDrifted(drifted []provider.ItemDrift) string {
	root := tree.Root(style.Warning.Render("Drifted resources (will be restored)")).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(f.enumeratorStyle)

	byKind := make(map[string][]provider.ItemDrift)
	for _, drift := range drifted {
		byKind[drift.Kind] = append(byKind[drift.Kind], drift)
	}

	for kind, items := range byKind {
		kindNode := tree.New().
			Root(f.kindStyle.Render(kind)).
			Enumerator(tree.RoundedEnumerator).
			EnumeratorStyle(f.enumeratorStyle)

		for _, drift := range items {
			itemText := f.nameStyle.Render(drift.Group) + "/" + f.itemStyle.Render(drift.Item)
			itemText += " " + f.actionStyle.Render("will be restored")
			kindNode.Child(itemText)
		}

		root.Child(kindNode)
	}

	return root.String()
}

// StateResource represents a resource in state for tree rendering
type StateResource struct {
	Kind   string
	Group  string
	Items  []string
	Status string
}

// FormatStateAsTree renders state list as a 3-level tree
func (f *TreeFormatter) FormatStateAsTree(resources []StateResource) string {
	root := tree.Root(style.Header.Render("Managed Resources")).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(f.enumeratorStyle)

	// Group by Kind
	byKind := make(map[string][]StateResource)
	for _, res := range resources {
		byKind[res.Kind] = append(byKind[res.Kind], res)
	}

	for kind, groups := range byKind {
		kindNode := tree.New().
			Root(f.kindStyle.Render(kind)).
			Enumerator(tree.RoundedEnumerator).
			EnumeratorStyle(f.enumeratorStyle)

		for _, group := range groups {
			groupNode := tree.New().
				Root(f.nameStyle.Render(group.Group)).
				Enumerator(tree.RoundedEnumerator).
				EnumeratorStyle(f.enumeratorStyle)

			for _, item := range group.Items {
				itemText := f.itemStyle.Render(item) + " " + f.actionStyle.Render(group.Status)
				groupNode.Child(itemText)
			}

			kindNode.Child(groupNode)
		}

		root.Child(kindNode)
	}

	return root.String()
}
