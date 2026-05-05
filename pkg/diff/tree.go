package diff

import (
	"context"
	"fmt"
	"strings"

	"github.com/Digital-Shane/treeview/v2"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/style"
)

// pterm fully removed from tree rendering!

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
		enumeratorStyle: style.Header,
		kindStyle:       style.HeaderKindAdd,
		nameStyle:       style.NewStyle(style.DefaultColors.GroupLabel),
		itemStyle:       style.NewStyle(style.DefaultColors.TableCell),
		actionStyle:     style.DiffProvider,
	}
}

// GroupPlanInfo holds the plan for tree rendering
type GroupPlanInfo struct {
	Plan provider.GroupPlan
}

// FormatGroupPlanAsTree renders the entire plan as a 3-level tree using pterm.DefaultTree
func (f *TreeFormatter) FormatGroupPlanAsTree(info GroupPlanInfo) error {
	children := []*treeview.Node[string]{}
	if len(info.Plan.Removals) > 0 {
		children = append(children, f.formatRemovals(info.Plan.Removals))
	}
	if len(info.Plan.Additions) > 0 {
		children = append(children, f.formatAdditions(info.Plan.Additions))
	}
	if len(info.Plan.Modifications) > 0 {
		children = append(children, f.formatModifications(info.Plan.Modifications))
	}
	if len(info.Plan.Drifted) > 0 {
		children = append(children, f.formatDrifted(info.Plan.Drifted))
	}
	root := treeview.NewNode[string]("root", "Managed Resources", "")
	for _, c := range children {
		root.AddChild(c)
	}

	tr := NewTreeRenderer()
	if err := tr.RenderTree(root); err != nil {
		return err
	}
	return nil
}

func (f *TreeFormatter) formatAdditions(additions []provider.GroupAddition) *treeview.Node[string] {
	node := treeview.NewNode[string]("add", style.Success.Render("Resources to be created"), "")
	byKind := make(map[string][]provider.GroupAddition)
	for _, addition := range additions {
		byKind[addition.Kind] = append(byKind[addition.Kind], addition)
	}
	for kind, groups := range byKind {
		kindNode := treeview.NewNode[string]("add-kind-"+kind, f.kindStyle.Render(kind), "")
		for _, group := range groups {
			groupNode := treeview.NewNode[string]("add-group-"+group.Group, f.nameStyle.Render(group.Group), "")
			for _, item := range group.Items {
				itemText := f.itemStyle.Render(item.Name) + " " + f.actionStyle.Render("will be created")
				itemNode := treeview.NewNode[string]("add-item-"+item.Name, itemText, "")
				groupNode.AddChild(itemNode)
			}
			kindNode.AddChild(groupNode)
		}
		node.AddChild(kindNode)
	}
	return node
}

func (f *TreeFormatter) formatRemovals(removals []provider.GroupRemoval) *treeview.Node[string] {
	node := treeview.NewNode[string]("rem", style.Error.Render("Resources to be removed"), "")
	byKind := make(map[string][]provider.GroupRemoval)
	for _, removal := range removals {
		byKind[removal.Kind] = append(byKind[removal.Kind], removal)
	}
	for kind, groups := range byKind {
		kindNode := treeview.NewNode[string]("rem-kind-"+kind, f.kindStyle.Render(kind), "")
		for _, group := range groups {
			groupNode := treeview.NewNode[string]("rem-group-"+group.Group, f.nameStyle.Render(group.Group), "")
			for _, item := range group.Items {
				itemText := f.itemStyle.Render(item.Name) + " " + f.actionStyle.Render("will be destroyed")
				itemNode := treeview.NewNode[string]("rem-item-"+item.Name, itemText, "")
				groupNode.AddChild(itemNode)
			}
			kindNode.AddChild(groupNode)
		}
		node.AddChild(kindNode)
	}
	return node
}

func (f *TreeFormatter) formatModifications(modifications []provider.GroupModification) *treeview.Node[string] {
	node := treeview.NewNode[string]("mod", style.Warning.Render("Resources to be modified"), "")
	byKind := make(map[string][]provider.GroupModification)
	for _, mod := range modifications {
		byKind[mod.Kind] = append(byKind[mod.Kind], mod)
	}
	for kind, groups := range byKind {
		kindNode := treeview.NewNode[string]("mod-kind-"+kind, f.kindStyle.Render(kind), "")
		for _, group := range groups {
			groupNode := treeview.NewNode[string]("mod-group-"+group.Group, f.nameStyle.Render(group.Group), "")
			for _, change := range group.Changes {
				itemText := f.itemStyle.Render(change.ItemName) + " " + f.actionStyle.Render("will be updated")
				itemNode := treeview.NewNode[string]("mod-item-"+change.ItemName, itemText, "")
				groupNode.AddChild(itemNode)
			}
			kindNode.AddChild(groupNode)
		}
		node.AddChild(kindNode)
	}
	return node
}

func (f *TreeFormatter) formatDrifted(drifted []provider.ItemDrift) *treeview.Node[string] {
	node := treeview.NewNode[string]("drift", style.Warning.Render("Drifted resources (will be restored)"), "")
	byKind := make(map[string][]provider.ItemDrift)
	for _, drift := range drifted {
		byKind[drift.Kind] = append(byKind[drift.Kind], drift)
	}
	for kind, items := range byKind {
		kindNode := treeview.NewNode[string]("drift-kind-"+kind, f.kindStyle.Render(kind), "")
		for _, drift := range items {
			itemText := f.nameStyle.Render(drift.Group) + "/" + f.itemStyle.Render(drift.Item) + " " + f.actionStyle.Render("will be restored")
			itemNode := treeview.NewNode[string]("drift-item-"+drift.Group+"/"+drift.Item, itemText, "")
			kindNode.AddChild(itemNode)
		}
		node.AddChild(kindNode)
	}
	return node
}

// StateResource represents a resource in state for tree rendering
type StateResource struct {
	Kind  string
	Group string
	// Items now include optional Info (e.g., version) to display in tree
	Items  []StateItem
	Status string
}

// StateItem represents a named item with optional info (version)
type StateItem struct {
	Name string
	Info string
}

// FormatStateAsTree renders state list as a 3-level tree using pterm.DefaultTree
func (f *TreeFormatter) FormatStateAsTree(resources []StateResource) error {
	// Build a treeview Node hierarchy to get branch glyphs and vertical lines
	root := treeview.NewNode[string]("root", style.Header.Render("Managed Resources"), "")

	if len(resources) == 0 {
		// still render header-only tree
		tr := NewTreeRenderer()
		return tr.RenderTree(root)
	}

	byKind := make(map[string][]StateResource)
	for _, res := range resources {
		byKind[res.Kind] = append(byKind[res.Kind], res)
	}

	for kind, groups := range byKind {
		kindNode := treeview.NewNode[string]("kind-"+kind, f.kindStyle.Render(kind), "")
		for _, group := range groups {
			groupNode := treeview.NewNode[string]("group-"+group.Group, f.nameStyle.Render(group.Group), "")
			for _, item := range group.Items {
				// name + optional info (version) styled the same (TableCell)
				itemText := f.itemStyle.Render(item.Name)
				if item.Info != "" {
					// render version with dedicated VersionColor (use exported wrapper)
					itemText = itemText + " " + style.VersionColor.Render(item.Info)
				}
				// append status (e.g. managed) in action style (dim/gray)
				if group.Status != "" {
					itemText = itemText + " " + f.actionStyle.Render(group.Status)
				}
				itemNode := treeview.NewNode[string]("item-"+item.Name, itemText, "")
				groupNode.AddChild(itemNode)
			}
			kindNode.AddChild(groupNode)
		}
		root.AddChild(kindNode)
	}

	tr := NewTreeRenderer()
	return tr.RenderTree(root)
}

// TreeRenderer is a small adapter that renders treeview nodes using the
// project's palette styles. It avoids importing pkg/ui to prevent cycles.
type TreeRenderer struct {
	kindStyle   style.Style
	groupStyle  style.Style
	itemStyle   style.Style
	actionStyle style.Style
}

func NewTreeRenderer() *TreeRenderer {
	return &TreeRenderer{
		kindStyle:   style.Header,
		groupStyle:  style.TableHeader,
		itemStyle:   style.TableStatusAdd,
		actionStyle: style.DiffProvider,
	}
}

func (tr *TreeRenderer) RenderTree(root *treeview.Node[string]) error {
	tree := treeview.NewTree[string]([]*treeview.Node[string]{root})
	// Ensure all nodes are expanded so Render includes the full hierarchy.
	// treeview renders only "visible" nodes (expanded), and nodes default to
	// collapsed, which produced output containing only the root.
	if err := tree.ExpandAll(context.Background()); err != nil {
		return err
	}

	// Configure provider to show connector lines for deeper nodes by using
	// default node provider; this prints the vertical branch glyphs as in the
	// example.
	// Note: treeview's renderer builds prefixes based on node depth and
	// sibling position; ensuring nodes are expanded is sufficient to produce
	// the vertical lines and branches.

	out, err := tree.Render(context.Background())
	if err != nil {
		return err
	}
	// Dim the branch glyph prefixes for visual contrast. We wrap only the
	// leading branch/indent characters (e.g. "├── ", "│   ", "└── ") with
	// the DimStyle so node text (already styled) remains unchanged.
	var outLines []string
	for _, ln := range strings.Split(out, "\n") {
		if ln == "" {
			outLines = append(outLines, ln)
			continue
		}
		var b strings.Builder
		for _, r := range ln {
			switch r {
			case '─', '│', '├', '└', '┘', '┌', '┐', '┤', '┬', '┴', '┼', '╭', '╮', '╯', '╰', '┝', '┥', '┯', '┷', '┿', '┠', '┨':
				b.WriteString(style.Border.Render(string(r)))
			default:
				b.WriteRune(r)
			}
		}
		outLines = append(outLines, b.String())
	}
	fmt.Println(strings.Join(outLines, "\n"))
	return nil
}
