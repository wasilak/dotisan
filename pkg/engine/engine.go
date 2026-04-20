// Package engine provides the core orchestration logic for dotisan.
//
// The Engine coordinates all components to execute plan and apply operations:
// - Loads configuration and state
// - Parses resources from dotfiles
// - Finds appropriate providers
// - Reconciles desired vs actual state
// - Executes changes
// - Saves updated state
package engine

import (
	"context"
	"fmt"

	"github.com/wasilak/dotisan/pkg/config"
	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/providers"
	"github.com/wasilak/dotisan/pkg/resource"
	"github.com/wasilak/dotisan/pkg/state"
)

// Engine orchestrates the plan and apply operations.
type Engine struct {
	// Config is the loaded dotisan configuration
	Config *config.Config

	// TemplateContext provides template variables
	TemplateContext *config.TemplateContext

	// StateBackend handles state persistence
	StateBackend state.StateBackend

	// DiffEngine formats diffs for display
	DiffEngine *diff.StyledDiffEngine

	// PlanFormatter formats plan output
	PlanFormatter *diff.PlanFormatter

	// Providers is a map of provider name to Provider instance
	Providers map[string]provider.Provider
}

// NewEngine creates a new Engine with default configuration.
func NewEngine() (*Engine, error) {
	// Load configuration
	cfg, ctx, err := config.LoadComplete()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create state backend
	backend, err := state.NewBackend(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create state backend: %w", err)
	}

	// Create diff engine
	diffEngine := diff.NewStyledEngine()
	planFormatter := diff.NewPlanFormatter()

	// Initialize providers and register them globally
	providerMap := make(map[string]provider.Provider)

	// FileProvider
	fileProvider := providers.NewFileProvider(ctx, diffEngine.Engine, cfg.DotfilesRoot)
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
		Config:          cfg,
		TemplateContext: ctx,
		StateBackend:    backend,
		DiffEngine:      diffEngine,
		PlanFormatter:   planFormatter,
		Providers:       providerMap,
	}, nil
}

// PlanResult contains the result of a plan operation.
type PlanResult struct {
	// Resources is the list of all parsed resources
	Resources []resource.Resource

	// CurrentState is the loaded state
	CurrentState *state.State

	// ProviderPlans is a map of provider name to their plan
	ProviderPlans map[string]provider.Plan

	// TotalAdditions count
	TotalAdditions int

	// TotalModifications count
	TotalModifications int

	// TotalRemovals count
	TotalRemovals int

	// TotalInSync count
	TotalInSync int

	// TotalDrifted count
	TotalDrifted int

	// HasChanges indicates if there are any changes
	HasChanges bool
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

	// Group resources by provider
	resourcesByProvider := e.groupResourcesByProvider(resources)

	// Build state map for quick lookup
	stateMap := make(map[string]provider.ResourceState)
	for _, s := range currentState.Resources {
		stateMap[s.ID] = s
	}

	// Generate plans for each provider
	providerPlans := make(map[string]provider.Plan)
	result := &PlanResult{
		Resources:     resources,
		CurrentState:  currentState,
		ProviderPlans: providerPlans,
	}

	for providerName, provider := range e.Providers {
		providerResources := resourcesByProvider[providerName]
		if len(providerResources) == 0 {
			continue
		}

		// Filter state for this provider
		providerState := e.filterStateForProvider(stateMap, providerName)

		// Reconcile
		plan := provider.Reconcile(providerResources, providerState)
		providerPlans[providerName] = plan

		// Update counts
		result.TotalAdditions += len(plan.Additions)
		result.TotalModifications += len(plan.Modifications)
		result.TotalRemovals += len(plan.Removals)
		result.TotalInSync += len(plan.InSync)
		result.TotalDrifted += len(plan.Drifted)
	}

	result.HasChanges = result.TotalAdditions > 0 || result.TotalModifications > 0 || result.TotalRemovals > 0

	return result, nil
}

// loadResources parses all resource files from the dotfiles directory.
func (e *Engine) loadResources() ([]resource.Resource, error) {
	loader := resource.NewLoader(e.Config.DotfilesRoot, e.TemplateContext)
	return loader.LoadResources()
}

// groupResourcesByProvider groups resources by their provider type.
func (e *Engine) groupResourcesByProvider(resources []resource.Resource) map[string][]resource.Resource {
	grouped := make(map[string][]resource.Resource)

	for _, res := range resources {
		var providerName string
		switch res.(type) {
		case *resource.ManagedFile, *resource.ManagedDirectory:
			providerName = "file"
		case *resource.BrewPackages:
			providerName = "brew"
		case *resource.NpmPackages:
			providerName = "npm"
		case *resource.GoPackages:
			providerName = "go"
		case *resource.CargoPackages:
			providerName = "cargo"
		default:
			continue // Unknown resource type
		}

		grouped[providerName] = append(grouped[providerName], res)
	}

	return grouped
}

// filterStateForProvider filters state entries for a specific provider.
func (e *Engine) filterStateForProvider(stateMap map[string]provider.ResourceState, providerName string) []provider.ResourceState {
	var filtered []provider.ResourceState

	for _, s := range stateMap {
		// State ID format: "provider/namespace/name"
		if len(s.ID) > len(providerName) && s.ID[:len(providerName)] == providerName {
			filtered = append(filtered, s)
		}
	}

	return filtered
}

// DisplayPlan outputs the plan result in a formatted way.
func (e *Engine) DisplayPlan(result *PlanResult) {
	fmt.Println()
	fmt.Println(e.PlanFormatter.FormatSummary(
		result.TotalAdditions,
		result.TotalModifications,
		result.TotalRemovals,
		result.TotalInSync,
	))
	fmt.Println()

	if !result.HasChanges && result.TotalDrifted == 0 {
		fmt.Println("No changes. Your dotfiles are in sync!")
		return
	}

	// Display additions
	for providerName, plan := range result.ProviderPlans {
		for _, res := range plan.Additions {
			fmt.Println(e.PlanFormatter.FormatAddition(
				fmt.Sprintf("%s/%s/%s", providerName, res.GetMetadata().GetNamespace(), res.GetMetadata().Name),
			))
		}
	}

	// Display modifications
	for providerName, plan := range result.ProviderPlans {
		for _, mod := range plan.Modifications {
			resourceID := fmt.Sprintf("%s/%s/%s", providerName, mod.Resource.GetMetadata().GetNamespace(), mod.Resource.GetMetadata().Name)
			fmt.Println(e.PlanFormatter.FormatModification(resourceID, mod.Diff))
		}
	}

	// Display removals
	for providerName, plan := range result.ProviderPlans {
		for _, res := range plan.Removals {
			fmt.Println(e.PlanFormatter.FormatDeletion(
				fmt.Sprintf("%s/%s/%s", providerName, res.GetMetadata().GetNamespace(), res.GetMetadata().Name),
			))
		}
	}

	// Display drifted
	for providerName, plan := range result.ProviderPlans {
		for _, drift := range plan.Drifted {
			resourceID := fmt.Sprintf("%s/%s/%s", providerName, drift.Resource.GetMetadata().GetNamespace(), drift.Resource.GetMetadata().Name)
			fmt.Println(e.PlanFormatter.FormatDrift(resourceID, drift.Description))
		}
	}

	fmt.Println()
}

// ApplyOptions contains options for the apply operation.
type ApplyOptions struct {
	// Confirm indicates if the user has confirmed the apply
	Confirm bool

	// AutoConfirm skips interactive confirmation
	AutoConfirm bool

	// Backup indicates if backups should be created
	Backup bool
}

// Apply executes the planned changes.
func (e *Engine) Apply(ctx context.Context, result *PlanResult, opts ApplyOptions) error {
	// Check if there are changes to apply
	if !result.HasChanges && result.TotalDrifted == 0 {
		fmt.Println("No changes to apply.")
		return nil
	}

	// Display plan first
	e.DisplayPlan(result)

	// Check for confirmation
	if !opts.Confirm && !opts.AutoConfirm {
		fmt.Println("Run with --confirm to apply these changes, or --auto-confirm for non-interactive mode.")
		return nil
	}

	// Execute changes for each provider
	for providerName, plan := range result.ProviderPlans {
		provider, exists := e.Providers[providerName]
		if !exists {
			return fmt.Errorf("provider %s not found", providerName)
		}

		if err := provider.Apply(ctx, plan); err != nil {
			return fmt.Errorf("failed to apply changes for provider %s: %w", providerName, err)
		}
	}

	// Update state with new resource states
	newState := state.NewState()
	for providerName, plan := range result.ProviderPlans {
		// Add in-sync resources to state
		for _, res := range plan.InSync {
			stateEntry := e.resourceToStateEntry(res, providerName)
			newState.SetResource(stateEntry)
		}

		// Add modified resources to state
		for _, mod := range plan.Modifications {
			stateEntry := e.resourceToStateEntry(mod.Resource, providerName)
			newState.SetResource(stateEntry)
		}

		// Add new resources to state
		for _, res := range plan.Additions {
			stateEntry := e.resourceToStateEntry(res, providerName)
			newState.SetResource(stateEntry)
		}
	}

	// Save updated state
	if err := e.StateBackend.Save(ctx, newState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Changes applied successfully!")

	return nil
}

// resourceToStateEntry converts a resource to a state entry.
func (e *Engine) resourceToStateEntry(res resource.Resource, providerName string) provider.ResourceState {
	return provider.ResourceState{
		ID:        fmt.Sprintf("%s/%s/%s", providerName, res.GetMetadata().GetNamespace(), res.GetMetadata().Name),
		Kind:      res.GetKind(),
		Name:      res.GetMetadata().Name,
		Namespace: res.GetMetadata().GetNamespace(),
	}
}
