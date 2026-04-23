package diff

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2/tree"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
	"github.com/wasilak/dotisan/pkg/style"
)

// TreeFormatter renders resources as a 3-level tree: Provider / Resource / Children
type TreeFormatter struct {
	// enumeratorStyle for tree lines (├── └──)
	enumeratorStyle lipgloss.Style
	// providerStyle for provider names (level 1)
	providerStyle lipgloss.Style
	// resourceStyle for resource names (level 2)
	resourceStyle lipgloss.Style
	// itemStyle for children/items (level 3)
	itemStyle lipgloss.Style
	// actionStyle for action text ("will be created", etc.)
	actionStyle lipgloss.Style
}

// NewTreeFormatter creates a new TreeFormatter with default colors
func NewTreeFormatter() *TreeFormatter {
	return &TreeFormatter{
		// Purple/blue for tree lines
		enumeratorStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("63")).MarginRight(1),
		// Green for providers
		providerStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("77")).Bold(true),
		// Yellow/orange for resources
		resourceStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("222")),
		// Pink/magenta for items
		itemStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("212")),
		// Gray for action text
		actionStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	}
}

// SetStyles allows customizing the color scheme
func (f *TreeFormatter) SetStyles(enumerator, provider, resource, item, action lipgloss.Style) {
	f.enumeratorStyle = enumerator
	f.providerStyle = provider
	f.resourceStyle = resource
	f.itemStyle = item
	f.actionStyle = action
}

// resourceGroup represents a resource with its items (for 3-level tree)
type resourceGroup struct {
	provider string
	resource resource.Resource
	items    []string
	action   string
	icon     string
}

// extractItems extracts child items from a resource (packages, modules, etc.)
func extractItems(res resource.Resource) []string {
	switch r := res.(type) {
	case *resource.BrewPackages:
		var items []string
		for _, formula := range r.Spec.Formulae {
			items = append(items, formula.Name)
		}
		for _, cask := range r.Spec.Casks {
			items = append(items, cask.Name+" (cask)")
		}
		return items
	case *resource.NpmPackages:
		var items []string
		for _, pkg := range r.Spec.Packages {
			items = append(items, pkg.Name)
		}
		return items
	case *resource.GoPackages:
		var items []string
		for _, pkg := range r.Spec.Packages {
			items = append(items, pkg.Module)
		}
		return items
	case *resource.CargoPackages:
		var items []string
		for _, pkg := range r.Spec.Packages {
			items = append(items, pkg.Name)
		}
		return items
	case *resource.ManagedFile:
		if r.Spec.SourceFile != "" {
			return []string{r.Spec.SourceFile}
		}
		return []string{"(inline content)"}
	case *resource.ManagedDirectory:
		return []string{r.Spec.SourceDir + " → " + r.Spec.Destination}
	default:
		return nil
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

	// Removals
	if result.TotalRemovals > 0 {
		sections = append(sections, f.formatRemovalsAsTree(result.ProviderPlans))
	}

	// Additions
	if result.TotalAdditions > 0 {
		sections = append(sections, f.formatAdditionsAsTree(result.ProviderPlans))
	}

	// Modifications
	if result.TotalModifications > 0 {
		sections = append(sections, f.formatModificationsAsTree(result.ProviderPlans))
	}

	// Drifted
	if result.TotalDrifted > 0 {
		sections = append(sections, f.formatDriftedAsTree(result.ProviderPlans))
	}

	return strings.Join(sections, "\n\n")
}

// formatAdditionsAsTree formats additions as a tree
func (f *TreeFormatter) formatAdditionsAsTree(providerPlans map[string]provider.Plan) string {
	root := tree.Root(style.Success.Render("Resources to be created")).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(f.enumeratorStyle)

	groups := f.groupByAction(providerPlans, "create")
	for providerName, resources := range groups {
		providerNode := tree.New().
			Root(f.providerStyle.Render(providerName)).
			Enumerator(tree.RoundedEnumerator).
			EnumeratorStyle(f.enumeratorStyle)

		for _, rg := range resources {
			resourceNode := f.buildResourceNode(rg)
			providerNode.Child(resourceNode)
		}

		root.Child(providerNode)
	}

	return root.String()
}

// formatRemovalsAsTree formats removals as a tree
func (f *TreeFormatter) formatRemovalsAsTree(providerPlans map[string]provider.Plan) string {
	root := tree.Root(style.Error.Render("Resources to be removed")).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(f.enumeratorStyle)

	groups := f.groupByAction(providerPlans, "remove")
	for providerName, resources := range groups {
		providerNode := tree.New().
			Root(f.providerStyle.Render(providerName)).
			Enumerator(tree.RoundedEnumerator).
			EnumeratorStyle(f.enumeratorStyle)

		for _, rg := range resources {
			resourceNode := f.buildResourceNode(rg)
			providerNode.Child(resourceNode)
		}

		root.Child(providerNode)
	}

	return root.String()
}

// formatModificationsAsTree formats modifications as a tree
func (f *TreeFormatter) formatModificationsAsTree(providerPlans map[string]provider.Plan) string {
	root := tree.Root(style.Warning.Render("Resources to be modified")).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(f.enumeratorStyle)

	groups := f.groupByAction(providerPlans, "modify")
	for providerName, resources := range groups {
		providerNode := tree.New().
			Root(f.providerStyle.Render(providerName)).
			Enumerator(tree.RoundedEnumerator).
			EnumeratorStyle(f.enumeratorStyle)

		for _, rg := range resources {
			resourceNode := f.buildResourceNode(rg)
			providerNode.Child(resourceNode)
		}

		root.Child(providerNode)
	}

	return root.String()
}

// formatDriftedAsTree formats drifted resources as a tree
func (f *TreeFormatter) formatDriftedAsTree(providerPlans map[string]provider.Plan) string {
	root := tree.Root(style.Warning.Render("Drifted resources (will be restored)")).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(f.enumeratorStyle)

	groups := f.groupByAction(providerPlans, "restore")
	for providerName, resources := range groups {
		providerNode := tree.New().
			Root(f.providerStyle.Render(providerName)).
			Enumerator(tree.RoundedEnumerator).
			EnumeratorStyle(f.enumeratorStyle)

		for _, rg := range resources {
			resourceNode := f.buildResourceNode(rg)
			providerNode.Child(resourceNode)
		}

		root.Child(providerNode)
	}

	return root.String()
}

// groupByAction groups resources by provider and action type
func (f *TreeFormatter) groupByAction(providerPlans map[string]provider.Plan, action string) map[string][]resourceGroup {
	groups := make(map[string][]resourceGroup)

	for providerName, plan := range providerPlans {
		var resources []resourceGroup

		switch action {
		case "create":
			for _, res := range plan.Additions {
				resources = append(resources, resourceGroup{
					provider: providerName,
					resource: res,
					items:    extractItems(res),
					action:   "will be created",
				})
			}
		case "modify":
			for _, mod := range plan.Modifications {
				resources = append(resources, resourceGroup{
					provider: providerName,
					resource: mod.Resource,
					items:    extractItems(mod.Resource),
					action:   "will be updated",
				})
			}
		case "remove":
			for _, res := range plan.Removals {
				resources = append(resources, resourceGroup{
					provider: providerName,
					resource: res,
					items:    extractItems(res),
					action:   "will be destroyed",
				})
			}
		case "restore":
			for _, drift := range plan.Drifted {
				resources = append(resources, resourceGroup{
					provider: providerName,
					resource: drift.Resource,
					items:    extractItems(drift.Resource),
					action:   "will be restored",
				})
			}
		}

		if len(resources) > 0 {
			groups[providerName] = resources
		}
	}

	return groups
}

// buildResourceNode builds a tree node for a resource with its items
func (f *TreeFormatter) buildResourceNode(rg resourceGroup) *tree.Tree {
	resourceName := rg.resource.GetMetadata().Name
	resourceKind := rg.resource.GetKind()

	// Level 2: Resource (Kind/Name)
	resourceLabel := f.resourceStyle.Render(fmt.Sprintf("%s/%s", resourceKind, resourceName))
	resourceNode := tree.New().
		Root(resourceLabel).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(f.enumeratorStyle)

	// Level 3: Items/Children
	if len(rg.items) > 0 {
		for _, item := range rg.items {
			itemLabel := f.itemStyle.Render(item) + " " + f.actionStyle.Render(rg.action)
			resourceNode.Child(itemLabel)
		}
	} else {
		// No children, show action on resource level
		resourceNode.Root(resourceLabel + " " + f.actionStyle.Render(rg.action))
	}

	return resourceNode
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

	// Group by provider
	byProvider := make(map[string][]StateResource)
	for _, res := range resources {
		provider := inferProviderFromKind(res.Kind)
		byProvider[provider] = append(byProvider[provider], res)
	}

	for providerName, providerResources := range byProvider {
		providerNode := tree.New().
			Root(f.providerStyle.Render(providerName)).
			Enumerator(tree.RoundedEnumerator).
			EnumeratorStyle(f.enumeratorStyle)

		for _, res := range providerResources {
			// Level 2: Resource
			resourceLabel := f.resourceStyle.Render(fmt.Sprintf("%s/%s", res.Kind, res.Name))
			resourceNode := tree.New().
				Root(resourceLabel).
				Enumerator(tree.RoundedEnumerator).
				EnumeratorStyle(f.enumeratorStyle)

			// Level 3: ID and Status
			statusLabel := f.itemStyle.Render(res.ID) + " " + f.actionStyle.Render(res.Status)
			resourceNode.Child(statusLabel)

			providerNode.Child(resourceNode)
		}

		root.Child(providerNode)
	}

	return root.String()
}

// inferProviderFromKind maps resource kind to provider name
func inferProviderFromKind(kind string) string {
	switch kind {
	case "ManagedFile", "ManagedDirectory":
		return "file"
	case "BrewPackages":
		return "homebrew"
	case "NpmPackages":
		return "npm"
	case "GoPackages":
		return "go"
	case "CargoPackages":
		return "cargo"
	default:
		return "unknown"
	}
}
