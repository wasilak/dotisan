package diff

import (
	"github.com/pterm/pterm"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/style"
)

// TreeFormatter renders resources as a 3-level tree: Kind / ResourceName / Items
type TreeFormatter struct {
	enumeratorStyle style.Style
	kindStyle       style.Style
	nameStyle       style.Style
	itemStyle       style.Style
	actionStyle     style.Style
}

// NewTreeFormatter creates a new TreeFormatter with default colors
func NewTreeFormatter() *TreeFormatter {
	return &TreeFormatter{
		enumeratorStyle: style.NewStyle(63),
		kindStyle:       style.NewStyle(pterm.FgGreen, pterm.Bold),
		nameStyle:       style.NewStyle(pterm.FgYellow),
		itemStyle:       style.NewStyle(pterm.FgMagenta),
		actionStyle:     style.NewStyle(pterm.FgGray),
	}
}

// GroupPlanInfo holds the plan for tree rendering
type GroupPlanInfo struct {
	Plan provider.GroupPlan
}

// FormatGroupPlanAsTree renders the entire plan as a 3-level tree using pterm.DefaultTree
func (f *TreeFormatter) FormatGroupPlanAsTree(info GroupPlanInfo) error {
	root := pterm.TreeNode{
		Text: "Managed Resources",
	}

	if len(info.Plan.Removals) > 0 {
		root.Children = append(root.Children, f.formatRemovals(info.Plan.Removals))
	}
	if len(info.Plan.Additions) > 0 {
		root.Children = append(root.Children, f.formatAdditions(info.Plan.Additions))
	}
	if len(info.Plan.Modifications) > 0 {
		root.Children = append(root.Children, f.formatModifications(info.Plan.Modifications))
	}
	if len(info.Plan.Drifted) > 0 {
		root.Children = append(root.Children, f.formatDrifted(info.Plan.Drifted))
	}

	return pterm.DefaultTree.WithRoot(root).Render()
}

func (f *TreeFormatter) formatAdditions(additions []provider.GroupAddition) pterm.TreeNode {
	node := pterm.TreeNode{
		Text: style.Success.Render("Resources to be created"),
	}

	// Group by Kind
	byKind := make(map[string][]provider.GroupAddition)
	for _, addition := range additions {
		byKind[addition.Kind] = append(byKind[addition.Kind], addition)
	}

	for kind, groups := range byKind {
		kindNode := pterm.TreeNode{
			Text: f.kindStyle.Render(kind),
		}

		for _, group := range groups {
			groupNode := pterm.TreeNode{
				Text: f.nameStyle.Render(group.Group),
			}

			for _, item := range group.Items {
				itemText := f.itemStyle.Render(item.Name) + " " + f.actionStyle.Render("will be created")
				groupNode.Children = append(groupNode.Children, pterm.TreeNode{Text: itemText})
			}

			kindNode.Children = append(kindNode.Children, groupNode)
		}

		node.Children = append(node.Children, kindNode)
	}

	return node
}

func (f *TreeFormatter) formatRemovals(removals []provider.GroupRemoval) pterm.TreeNode {
	node := pterm.TreeNode{
		Text: style.Error.Render("Resources to be removed"),
	}

	byKind := make(map[string][]provider.GroupRemoval)
	for _, removal := range removals {
		byKind[removal.Kind] = append(byKind[removal.Kind], removal)
	}

	for kind, groups := range byKind {
		kindNode := pterm.TreeNode{
			Text: f.kindStyle.Render(kind),
		}

		for _, group := range groups {
			groupNode := pterm.TreeNode{
				Text: f.nameStyle.Render(group.Group),
			}

			for _, item := range group.Items {
				itemText := f.itemStyle.Render(item.Name) + " " + f.actionStyle.Render("will be destroyed")
				groupNode.Children = append(groupNode.Children, pterm.TreeNode{Text: itemText})
			}

			kindNode.Children = append(kindNode.Children, groupNode)
		}

		node.Children = append(node.Children, kindNode)
	}

	return node
}

func (f *TreeFormatter) formatModifications(modifications []provider.GroupModification) pterm.TreeNode {
	node := pterm.TreeNode{
		Text: style.Warning.Render("Resources to be modified"),
	}

	byKind := make(map[string][]provider.GroupModification)
	for _, mod := range modifications {
		byKind[mod.Kind] = append(byKind[mod.Kind], mod)
	}

	for kind, groups := range byKind {
		kindNode := pterm.TreeNode{
			Text: f.kindStyle.Render(kind),
		}

		for _, group := range groups {
			groupNode := pterm.TreeNode{
				Text: f.nameStyle.Render(group.Group),
			}

			for _, change := range group.Changes {
				itemText := f.itemStyle.Render(change.ItemName) + " " + f.actionStyle.Render("will be updated")
				groupNode.Children = append(groupNode.Children, pterm.TreeNode{Text: itemText})
			}

			kindNode.Children = append(kindNode.Children, groupNode)
		}

		node.Children = append(node.Children, kindNode)
	}

	return node
}

func (f *TreeFormatter) formatDrifted(drifted []provider.ItemDrift) pterm.TreeNode {
	node := pterm.TreeNode{
		Text: style.Warning.Render("Drifted resources (will be restored)"),
	}

	byKind := make(map[string][]provider.ItemDrift)
	for _, drift := range drifted {
		byKind[drift.Kind] = append(byKind[drift.Kind], drift)
	}

	for kind, items := range byKind {
		kindNode := pterm.TreeNode{
			Text: f.kindStyle.Render(kind),
		}

		for _, drift := range items {
			itemText := f.nameStyle.Render(drift.Group) + "/" + f.itemStyle.Render(drift.Item)
			itemText += " " + f.actionStyle.Render("will be restored")
			kindNode.Children = append(kindNode.Children, pterm.TreeNode{Text: itemText})
		}

		node.Children = append(node.Children, kindNode)
	}

	return node
}

// StateResource represents a resource in state for tree rendering
type StateResource struct {
	Kind   string
	Group  string
	Items  []string
	Status string
}

// FormatStateAsTree renders state list as a 3-level tree using pterm.DefaultTree
func (f *TreeFormatter) FormatStateAsTree(resources []StateResource) error {
	root := pterm.TreeNode{
		Text: style.Header.Render("Managed Resources"),
	}

	// Group by Kind
	byKind := make(map[string][]StateResource)
	for _, res := range resources {
		byKind[res.Kind] = append(byKind[res.Kind], res)
	}

	for kind, groups := range byKind {
		kindNode := pterm.TreeNode{
			Text: f.kindStyle.Render(kind),
		}

		for _, group := range groups {
			groupNode := pterm.TreeNode{
				Text: f.nameStyle.Render(group.Group),
			}

			for _, item := range group.Items {
				itemText := f.itemStyle.Render(item) + " " + f.actionStyle.Render(group.Status)
				groupNode.Children = append(groupNode.Children, pterm.TreeNode{Text: itemText})
			}

			kindNode.Children = append(kindNode.Children, groupNode)
		}

		root.Children = append(root.Children, kindNode)
	}

	return pterm.DefaultTree.WithRoot(root).Render()
}
