// Package engine: Plan logic extracted from engine.go
package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/wasilak/dotisan/pkg/graph"
	"github.com/wasilak/dotisan/pkg/planctx"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
	"github.com/wasilak/dotisan/pkg/state"
)

// PlanResult contains the result of a plan operation.
type PlanResult struct {
	CurrentState       *state.State
	ProviderPlans      map[string]provider.GroupPlan
	TotalAdditions     int
	TotalModifications int
	TotalRemovals      int
	TotalCleanup       int
	TotalInSync        int
	TotalDrifted       int
	HasChanges         bool
	// UnmatchedTargets are targets provided by the user that didn't match any resource
	UnmatchedTargets []string
	// DependencyOrder is the topological order of resources by NodeID.
	// Resources earlier in the slice must be applied before those later.
	DependencyOrder []graph.NodeID

	// DAG is the validated dependency graph built during planning.
	// Used by Apply to check per-node dependencies for skip propagation.
	DAG *graph.DAG
}

// Plan loads state, parses resources, and generates plans from all providers.
// It accepts PlanOptions which can be used to target specific resources.
// Note: Plan show-diff context key lives in pkg/planctx to avoid
// import cycles between engine and providers.
func (e *Engine) Plan(ctx context.Context, opts PlanOptions) (*PlanResult, error) {
	// Load current state
	currentState, err := e.StateBackend.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Parse all resources from dotfiles
	resources, err := e.loadResources()
	if err != nil {
		return nil, fmt.Errorf("failed to load resources: %w", err)
	}

	// Convert resources to groups
	resourceGroups := e.resourcesToGroups(resources)

	// Build dependency graph and validate it before proceeding.
	dag, err := graph.Build(resources)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}
	dependencyOrder, err := dag.TopologicalOrder()
	if err != nil {
		return nil, fmt.Errorf("dependency error: %w", err)
	}

	// If targets provided, parse them. We'll not filter out resourceGroups here
	// (desired) because targets may refer to state-only groups. Instead we will
	// augment resourceGroups with synthetic groups based on state when needed
	// and detect unmatched targets against both desired resources and state.
	var targetMatches []TargetMatch
	var unmatched []string
	if len(opts.Targets) > 0 {
		targetMatches = ParseTargets(opts.Targets)

		// For each parsed target, check if it matches any desired group/item.
		// If it doesn't but exists in state, add a synthetic empty desired group
		// so providers will produce plans for removals, and mark as matched.
		for i, raw := range opts.Targets {
			t := targetMatches[i]
			found := false

			// Check desired groups
			for _, g := range resourceGroups {
				if t.Matches(g.Kind, g.Name, "") {
					if t.Item == "" {
						found = true
						break
					}
					for _, it := range g.Items {
						if strings.EqualFold(it.Name, t.Item) {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
			}

			if found {
				continue
			}

			// Check state resources
			for _, s := range currentState.Resources {
				if t.Kind != "" && !strings.EqualFold(t.Kind, s.Kind) {
					continue
				}
				if t.Group != "" && !strings.EqualFold(t.Group, s.Group) {
					continue
				}

				if t.Item == "" {
					// match at group level
					found = true
				} else {
					for _, it := range s.Items {
						if strings.EqualFold(it.Name, t.Item) {
							found = true
							break
						}
					}
				}

				if found {
					// If desired didn't contain this group, add a synthetic empty
					// group so provider.Reconcile will compute removals for items
					// present in state but not desired.
					existsInDesired := false
					for _, g := range resourceGroups {
						if strings.EqualFold(g.Kind, s.Kind) && strings.EqualFold(g.Name, s.Group) {
							existsInDesired = true
							break
						}
					}
					if !existsInDesired {
						resourceGroups = append(resourceGroups, resource.ResourceGroup[any]{
							Kind:  s.Kind,
							Name:  s.Group,
							Items: []resource.ResourceItem{},
						})
					}
					break
				}
			}

			if !found {
				unmatched = append(unmatched, raw)
			}
		}
	}

	// Group resources by provider
	groupsByProvider := e.groupResourcesByProvider(resourceGroups)

	// Generate plans for each provider
	providerPlans := make(map[string]provider.GroupPlan)
	result := &PlanResult{
		CurrentState:    currentState,
		ProviderPlans:   providerPlans,
		DependencyOrder: dependencyOrder,
		DAG:             dag,
	}

	if len(unmatched) > 0 {
		result.UnmatchedTargets = unmatched
	}

	for providerName, prov := range e.Providers {
		providerGroups := groupsByProvider[providerName]
		if len(providerGroups) == 0 {
			continue
		}

		// Filter state for this provider
		providerState := e.filterStateForProvider(currentState.Resources, providerName)

		// Inject ShowDiff flag into context
		ctxWithDiff := ctx
		if opts.ShowDiff {
			ctxWithDiff = context.WithValue(ctx, planctx.PlanShowDiffKey, true)
		} else {
			ctxWithDiff = context.WithValue(ctx, planctx.PlanShowDiffKey, false)
		}

		// Reconcile (pass ctx so providers can perform cancellable operations)
		plan := prov.Reconcile(ctxWithDiff, providerGroups, providerState)

		// If targets provided, further filter plan items to item-level targets
		if len(targetMatches) > 0 {
			plan = filterPlanByTargets(plan, targetMatches)
		}
		providerPlans[providerName] = plan

		// Update counts - sum individual items within each plan group
		for _, add := range plan.Additions {
			result.TotalAdditions += len(add.Items)
		}
		for _, mod := range plan.Modifications {
			result.TotalModifications += len(mod.Changes)
		}
		for _, rem := range plan.Removals {
			result.TotalRemovals += len(rem.Items)
		}
		for _, cleanup := range plan.Cleanup {
			result.TotalCleanup += len(cleanup.Items)
		}
		for _, sync := range plan.InSync {
			result.TotalInSync += len(sync.Items)
		}
		for range plan.Drifted {
			result.TotalDrifted++
		}
	}

	result.HasChanges = result.TotalAdditions > 0 || result.TotalModifications > 0 ||
		result.TotalRemovals > 0 || result.TotalCleanup > 0 || result.TotalDrifted > 0

	return result, nil
}

// loadResources parses all resource files from the dotfiles directory.
func (e *Engine) loadResources() ([]resource.Resource, error) {
	loader := resource.NewLoader(e.Config.DotfilesRoot, e.TemplateContext)
	return loader.LoadResources()
}

// resourcesToGroups converts Resources to ResourceGroups
func (e *Engine) resourcesToGroups(resources []resource.Resource) []resource.ResourceGroup[any] {
	var groups []resource.ResourceGroup[any]
	for _, res := range resources {
		groups = append(groups, res.ToGroup())
	}
	return groups
}

// groupResourcesByProvider groups resource groups by their provider type.
func (e *Engine) groupResourcesByProvider(groups []resource.ResourceGroup[any]) map[string][]resource.ResourceGroup[any] {
	grouped := make(map[string][]resource.ResourceGroup[any])

	for _, group := range groups {
		// Look up provider by kind from the registry. If not registered, skip.
		if provName, ok := provider.ProviderNameForKind(group.Kind); ok {
			grouped[provName] = append(grouped[provName], group)
		}
	}

	return grouped
}

// filterStateForProvider filters state entries for a specific provider.
func (e *Engine) filterStateForProvider(stateResources []provider.ResourceState, providerName string) []provider.ResourceState {
	var filtered []provider.ResourceState

	// Build a reverse mapping: kind -> providerName is in registry. Filter
	// stateResources by asking the registry which provider handles each kind.
	for _, s := range stateResources {
		if provName, ok := provider.ProviderNameForKind(s.Kind); ok && provName == providerName {
			filtered = append(filtered, s)
		}
	}

	return filtered
}
