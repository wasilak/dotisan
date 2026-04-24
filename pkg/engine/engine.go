// Package engine provides the core orchestration logic for dotisan.
package engine

import (
	"context"
	"fmt"

	"github.com/wasilak/dotisan/pkg/config"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/providers"
	"github.com/wasilak/dotisan/pkg/resource"
	"github.com/wasilak/dotisan/pkg/state"
)

// Engine orchestrates the plan and apply operations.
type Engine struct {
	Config       *config.Config
	StateBackend state.StateBackend
	Providers    map[string]provider.Provider
}

// NewEngine creates a new Engine with default configuration.
func NewEngine() (*Engine, error) {
	cfg, _, err := config.LoadComplete()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	backend, err := state.NewBackend(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create state backend: %w", err)
	}

	providerMap := make(map[string]provider.Provider)

	// FileProvider
	fileProvider := providers.NewFileProvider(cfg.DotfilesRoot)
	providerMap["file"] = fileProvider
	provider.Register("file", fileProvider)

	// BrewProvider
	brewProvider := providers.NewBrewProvider()
	providerMap["homebrew"] = brewProvider
	provider.Register("homebrew", brewProvider)

	// NpmProvider
	npmProvider := providers.NewNpmProvider()
	providerMap["npm"] = npmProvider
	provider.Register("npm", npmProvider)

	// GoProvider
	goProvider := providers.NewGoProvider()
	providerMap["go"] = goProvider
	provider.Register("go", goProvider)

	// CargoProvider
	cargoProvider := providers.NewCargoProvider()
	providerMap["cargo"] = cargoProvider
	provider.Register("cargo", cargoProvider)

	return &Engine{
		Config:       cfg,
		StateBackend: backend,
		Providers:    providerMap,
	}, nil
}

// PlanResult contains the result of a plan operation.
type PlanResult struct {
	CurrentState       *state.State
	ProviderPlans      map[string]provider.GroupPlan
	TotalAdditions     int
	TotalModifications int
	TotalRemovals      int
	TotalInSync        int
	TotalDrifted       int
	HasChanges         bool
}

// Plan loads state, parses resources, and generates plans from all providers.
func (e *Engine) Plan(ctx context.Context) (*PlanResult, error) {
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

	// Group resources by provider
	groupsByProvider := e.groupResourcesByProvider(resourceGroups)

	// Generate plans for each provider
	providerPlans := make(map[string]provider.GroupPlan)
	result := &PlanResult{
		CurrentState:  currentState,
		ProviderPlans: providerPlans,
	}

	for providerName, prov := range e.Providers {
		providerGroups := groupsByProvider[providerName]
		if len(providerGroups) == 0 {
			continue
		}

		// Filter state for this provider
		providerState := e.filterStateForProvider(currentState.Resources, providerName)

		// Reconcile
		plan := prov.Reconcile(providerGroups, providerState)
		providerPlans[providerName] = plan

		// Update counts
		result.TotalAdditions += len(plan.Additions)
		result.TotalModifications += len(plan.Modifications)
		result.TotalRemovals += len(plan.Removals)
		result.TotalInSync += len(plan.InSync)
		result.TotalDrifted += len(plan.Drifted)
	}

	result.HasChanges = result.TotalAdditions > 0 || result.TotalModifications > 0 ||
		result.TotalRemovals > 0 || result.TotalDrifted > 0

	return result, nil
}

// loadResources parses all resource files from the dotfiles directory.
func (e *Engine) loadResources() ([]resource.Resource, error) {
	loader := resource.NewLoader(e.Config.DotfilesRoot, nil)
	return loader.LoadResources()
}

// resourcesToGroups converts Resources to ResourceGroups
func (e *Engine) resourcesToGroups(resources []resource.Resource) []resource.ResourceGroup {
	var groups []resource.ResourceGroup
	for _, res := range resources {
		groups = append(groups, res.ToGroup())
	}
	return groups
}

// groupResourcesByProvider groups resource groups by their provider type.
func (e *Engine) groupResourcesByProvider(groups []resource.ResourceGroup) map[string][]resource.ResourceGroup {
	grouped := make(map[string][]resource.ResourceGroup)

	for _, group := range groups {
		var providerName string
		switch group.Kind {
		case "ManagedFile", "ManagedDirectory":
			providerName = "file"
		case "BrewPackages":
			providerName = "homebrew"
		case "NpmPackages":
			providerName = "npm"
		case "GoPackages":
			providerName = "go"
		case "CargoPackages":
			providerName = "cargo"
		default:
			continue
		}

		grouped[providerName] = append(grouped[providerName], group)
	}

	return grouped
}

// filterStateForProvider filters state entries for a specific provider.
func (e *Engine) filterStateForProvider(stateResources []provider.ResourceState, providerName string) []provider.ResourceState {
	var filtered []provider.ResourceState

	providerKinds := make(map[string]bool)
	switch providerName {
	case "file":
		providerKinds["ManagedFile"] = true
		providerKinds["ManagedDirectory"] = true
	case "homebrew":
		providerKinds["BrewPackages"] = true
	case "npm":
		providerKinds["NpmPackages"] = true
	case "go":
		providerKinds["GoPackages"] = true
	case "cargo":
		providerKinds["CargoPackages"] = true
	}

	for _, s := range stateResources {
		if providerKinds[s.Kind] {
			filtered = append(filtered, s)
		}
	}

	return filtered
}

// ApplyOptions contains options for the Apply operation.
type ApplyOptions struct {
	Confirm bool
}

// Apply executes the planned changes.
func (e *Engine) Apply(ctx context.Context, result *PlanResult, opts ApplyOptions) error {
	if !result.HasChanges {
		return nil
	}

	if !opts.Confirm {
		return fmt.Errorf("apply not confirmed")
	}

	// Execute changes for each provider
	for providerName, plan := range result.ProviderPlans {
		prov, exists := e.Providers[providerName]
		if !exists {
			return fmt.Errorf("provider %s not found", providerName)
		}

		if err := prov.Apply(ctx, plan); err != nil {
			return fmt.Errorf("failed to apply changes for provider %s: %w", providerName, err)
		}
	}

	// Update state
	newState := state.NewState()
	for _, plan := range result.ProviderPlans {
		// Add in-sync resources
		for _, inSync := range plan.InSync {
			newState.SetResourceGroup(provider.ResourceState{
				Kind:      inSync.Kind,
				Group:     inSync.Group,
				Items:     inSync.Items,
				Namespace: "default",
			})
		}
		// Add additions
		for _, addition := range plan.Additions {
			items := make([]resource.ItemState, 0, len(addition.Items))
			for _, item := range addition.Items {
				items = append(items, resource.ItemState{
					Name:    item.Name,
					Version: item.Version,
					Status:  "present",
				})
			}
			newState.SetResourceGroup(provider.ResourceState{
				Kind:      addition.Kind,
				Group:     addition.Group,
				Items:     items,
				Namespace: "default",
			})
		}
	}

	if err := e.StateBackend.Save(ctx, newState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// ApplyWithProgress executes the planned changes with progress reporting.
func (e *Engine) ApplyWithProgress(ctx context.Context, result *PlanResult, opts ApplyOptions) error {
	return e.Apply(ctx, result, opts)
}

// DisplayPlan displays the plan result (placeholder for backward compatibility)
func (e *Engine) DisplayPlan(result *PlanResult) {
	// This is now handled by the command layer using output formatters
}
