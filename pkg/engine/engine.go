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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wasilak/dotisan/pkg/config"
	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/providers"
	"github.com/wasilak/dotisan/pkg/resource"
	"github.com/wasilak/dotisan/pkg/state"
	"github.com/wasilak/dotisan/pkg/style"
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

	// TotalWarnings is the aggregated number of warnings from providers
	TotalWarnings int

	// HasChanges indicates if there are any changes
	HasChanges bool
}

// ProgressFunc is called during plan execution to report progress (0.0 to 1.0)
type ProgressFunc func(percent float64, message string)

// Plan loads state, parses resources, and generates plans from all providers.
// If progressFn is provided, it will be called with progress updates (0.0 to 1.0).
func (e *Engine) Plan(ctx context.Context, progressFn ProgressFunc) (*PlanResult, error) {
	// Define progress steps
	totalSteps := 4
	currentStep := 0

	updateProgress := func(message string) {
		currentStep++
		if progressFn != nil {
			percent := float64(currentStep) / float64(totalSteps)
			progressFn(percent, message)
		}
	}

	// Load current state
	updateProgress("Loading state...")
	currentState, err := e.StateBackend.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Parse all resources from dotfiles
	updateProgress("Loading resources...")
	resources, err := e.loadResources()
	if err != nil {
		return nil, fmt.Errorf("failed to load resources: %w", err)
	}

	// Group resources by provider
	updateProgress("Grouping resources...")
	resourcesByProvider := e.groupResourcesByProvider(resources)

	// Build state map for quick lookup
	stateMap := make(map[string]provider.ResourceState)
	for _, s := range currentState.Resources {
		stateMap[s.ID] = s
	}

	// Generate plans for each provider
	updateProgress("Generating plans...")
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
		result.TotalWarnings += len(plan.Warnings)
	}

	// Check for orphaned state resources (in state but not in config)
	// These should be marked for removal - Terraform-style workflow
	orphanedRemovals := e.findOrphanedStateResources(currentState.Resources, resourcesByProvider)
	if len(orphanedRemovals) > 0 {
		for _, orphaned := range orphanedRemovals {
			result.TotalRemovals++

			// Get provider name from the resource's kind
			var providerName string
			switch orphaned.(type) {
			case *resource.ManagedFile, *resource.ManagedDirectory:
				providerName = "file"
			case *resource.BrewPackages:
				providerName = "homebrew"
			case *resource.NpmPackages:
				providerName = "npm"
			case *resource.GoPackages:
				providerName = "go"
			case *resource.CargoPackages:
				providerName = "cargo"
			default:
				continue
			}

			// Update plan with orphaned resource
			if plan, exists := providerPlans[providerName]; exists {
				// Add to existing plan's removals
				plan.Removals = append(plan.Removals, orphaned)
				providerPlans[providerName] = plan
			} else {
				// Create new plan with just this removal
				providerPlans[providerName] = provider.Plan{
					Removals: []resource.Resource{orphaned},
				}
			}
		}
	}

	result.HasChanges = result.TotalAdditions > 0 || result.TotalModifications > 0 || result.TotalRemovals > 0 || result.TotalDrifted > 0

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
			providerName = "homebrew"
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
// Maps provider names to the kinds they handle (e.g., "file" -> ["ManagedFile", "ManagedDirectory"])
func (e *Engine) filterStateForProvider(stateMap map[string]provider.ResourceState, providerName string) []provider.ResourceState {
	var filtered []provider.ResourceState

	// Map provider to the kinds it handles
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

	for _, s := range stateMap {
		// Check if this state's kind belongs to this provider
		if providerKinds[s.Kind] {
			filtered = append(filtered, s)
		}
	}

	return filtered
}

// findOrphanedStateResources finds state entries that are not in the config.
// These are resources that were previously managed but no longer have YAML definitions.
// Returns the orphaned resources converted to resource.Resource for display.
func (e *Engine) findOrphanedStateResources(stateResources []provider.ResourceState, resourcesByProvider map[string][]resource.Resource) []resource.Resource {
	// Build set of config resource IDs using kind/name (Terraform-style)
	// Also track indexed IDs for list-based resources
	configIDs := make(map[string]bool)
	parentIDs := make(map[string]bool)
	for _, providerResources := range resourcesByProvider {
		for _, res := range providerResources {
			parentID := fmt.Sprintf("%s/%s", res.GetKind(), res.GetMetadata().Name)
			parentIDs[parentID] = true
			configIDs[parentID] = true
		}
	}

	// Find state resources NOT in config (orphaned) and convert to Resource
	var orphaned []resource.Resource
	for _, s := range stateResources {
		isOrphaned := !configIDs[s.ID]
		if isOrphaned {
			// Check if it's an indexed item of a parent resource
			parentID := fmt.Sprintf("%s/%s", s.Kind, s.Name)
			if parentIDs[parentID] {
				// It's an item of a resource in config - not orphaned
				isOrphaned = false
			}
		}
		if isOrphaned {
			// Convert ResourceState to appropriate Resource type
			orphanedRes := e.stateToResource(s)
			if orphanedRes != nil {
				orphaned = append(orphaned, orphanedRes)
			}
		}
	}

	return orphaned
}

// stateToResource converts a ResourceState to a Resource object.
// This is used for orphaned resources that need to be displayed in plan output.
func (e *Engine) stateToResource(s provider.ResourceState) resource.Resource {
	// Create base resource with metadata
	base := resource.BaseResource{
		APIVersion: resource.SupportedAPIVersion,
		Kind:       s.Kind,
		Metadata: resource.Metadata{
			Name:      s.Name,
			Namespace: s.Namespace,
		},
	}

	switch s.Kind {
	case "ManagedFile":
		return &resource.ManagedFile{BaseResource: base}
	case "ManagedDirectory":
		return &resource.ManagedDirectory{BaseResource: base}
	case "BrewPackages", "homebrew":
		return &resource.BrewPackages{BaseResource: base}
	case "NpmPackages":
		return &resource.NpmPackages{BaseResource: base}
	case "GoPackages":
		return &resource.GoPackages{BaseResource: base}
	case "CargoPackages":
		return &resource.CargoPackages{BaseResource: base}
	default:
		// For unknown kinds, return nil (won't be shown in plan)
		return nil
	}
}

// getProviderNameFromStateID extracts provider name from state ID.
// State ID format: "provider/namespace/name"
func (e *Engine) getProviderNameFromStateID(stateID string) string {
	parts := strings.Split(stateID, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// DisplayPlan outputs the plan result in a formatted way (Terraform-style).
// Summary is at the end, with clear descriptions and breathing space.
func (e *Engine) DisplayPlan(result *PlanResult) {
	// Show sections with clear headers and breathing space

	// 1. Removals (orphaned resources - will be deleted)
	if result.TotalRemovals > 0 {
		fmt.Println()
		fmt.Println(e.PlanFormatter.FormatSectionHeader("Resources to be removed"))
		for _, plan := range result.ProviderPlans {
			for _, res := range plan.Removals {
				resourceID := fmt.Sprintf("%s/%s", res.GetKind(), res.GetMetadata().Name)
				fmt.Println()
				fmt.Println(e.PlanFormatter.FormatRemovalDetailed(resourceID))
			}
		}
	}

	// 2. Additions (new resources - will be created)
	if result.TotalAdditions > 0 {
		fmt.Println()
		fmt.Println(e.PlanFormatter.FormatSectionHeader("Resources to be created"))
		for _, plan := range result.ProviderPlans {
			for _, res := range plan.Additions {
				resourceID := fmt.Sprintf("%s/%s", res.GetKind(), res.GetMetadata().Name)
				fmt.Println(e.PlanFormatter.FormatAdditionDetailed(resourceID))
			}
		}
	}

	// 3. Modifications (resources that will be changed)
	if result.TotalModifications > 0 {
		fmt.Println()
		fmt.Println(e.PlanFormatter.FormatSectionHeader("Resources to be modified"))
		for _, plan := range result.ProviderPlans {
			for _, mod := range plan.Modifications {
				resourceID := fmt.Sprintf("%s/%s", mod.Resource.GetKind(), mod.Resource.GetMetadata().Name)
				fmt.Println(e.PlanFormatter.FormatModificationDetailed(resourceID))
				if mod.Diff != "" {
					fmt.Println()
					fmt.Println(e.PlanFormatter.FormatDiff(mod.Diff))
				}
			}
		}
	}

	// 4. Drifted resources (changed outside of dotisan)
	if result.TotalDrifted > 0 {
		fmt.Println()
		fmt.Println(e.PlanFormatter.FormatSectionHeader("Drifted resources (manual changes detected)"))
		for _, plan := range result.ProviderPlans {
			for _, drift := range plan.Drifted {
				resourceID := fmt.Sprintf("%s/%s", drift.Resource.GetKind(), drift.Resource.GetMetadata().Name)
				fmt.Println(e.PlanFormatter.FormatDriftDetailed(resourceID))
				if drift.Diff != "" {
					fmt.Println()
					fmt.Println(e.PlanFormatter.FormatDiff(drift.Diff))
				}
			}
		}
	}

	// 5. Warnings (non-blocking advisory messages)
	if result.TotalWarnings > 0 {
		fmt.Println()
		fmt.Println(e.PlanFormatter.FormatSectionHeader("Warnings and Advisories"))
		for _, plan := range result.ProviderPlans {
			for _, w := range plan.Warnings {
				// Render severity and message; include suggestion if present
				sev := strings.ToUpper(w.Severity)
				header := fmt.Sprintf("%s: %s", sev, w.Message)
				fmt.Println("  ", e.PlanFormatter.FormatActionReason(header))
				if w.Suggestion != "" {
					fmt.Println("\t", w.Suggestion)
				}
			}
		}
	}

	// SUMMARY at the end (like Terraform)
	fmt.Println()
	fmt.Println()
	// Print summary plus warnings count if present
	summary := e.PlanFormatter.FormatSummary(
		result.TotalAdditions,
		result.TotalModifications,
		result.TotalRemovals,
		result.TotalInSync,
	)
	if result.TotalWarnings > 0 {
		// Append warnings summary on its own line for visibility
		fmt.Println(e.PlanFormatter.FormatWarningsSummary(result.TotalWarnings))
	}
	fmt.Println(summary)

	// No changes message - now includes drift check
	if !result.HasChanges {
		fmt.Println()
		fmt.Println(e.PlanFormatter.FormatNoChanges())
	}

	fmt.Println()
}

// ApplyOptions contains options for the apply operation.
type ApplyOptions struct {
	// Confirm indicates if the user has confirmed the apply
	Confirm bool
}

// resourceResult tracks the result of applying a single resource
type resourceResult struct {
	resource string
	action   string
	success  bool
	err      error
}

// applyProgressMsg is sent to update apply progress
type applyProgressMsg struct {
	resource  string
	action    string
	completed []resourceResult
	current   int
	total     int
	percent   float64
}

// applyCompleteMsg is sent when apply is complete
type applyCompleteMsg struct {
	successCount int
	failCount    int
	results      []resourceResult
}

// applyTickMsg is sent periodically to update the UI
type applyTickMsg struct{}

// applyProgressModel represents the progress UI for apply
type applyProgressModel struct {
	progress  progress.Model
	current   int
	total     int
	percent   float64
	resource  string
	action    string
	completed []resourceResult
	done      bool
}

func (m applyProgressModel) Init() tea.Cmd {
	return m.tickCmd()
}

func (m applyProgressModel) tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
		return applyTickMsg{}
	})
}

func (m applyProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		return m, nil

	case applyTickMsg:
		if m.done {
			return m, tea.Quit
		}
		return m, m.tickCmd()

	case applyProgressMsg:
		m.current = msg.current
		m.total = msg.total
		m.percent = msg.percent
		m.resource = msg.resource
		m.action = msg.action
		m.completed = msg.completed
		return m, nil

	case applyCompleteMsg:
		m.completed = msg.results
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

func (m applyProgressModel) View() string {
	if m.done {
		return ""
	}

	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(style.Header.Render("Applying changes..."))
	s.WriteString("\n\n")

	// Progress bar
	s.WriteString(m.progress.ViewAs(m.percent))
	s.WriteString(fmt.Sprintf("  %d/%d\n", m.current, m.total))

	// Current operation
	if m.action != "" && m.resource != "" {
		s.WriteString(fmt.Sprintf("\n%s %s %s\n", style.Dim.Render("→"), style.Info.Render(m.action), style.Dim.Render(m.resource)))
	}

	// Recently completed items (last 3) with status
	if len(m.completed) > 0 {
		s.WriteString("\n")
		start := len(m.completed) - 3
		if start < 0 {
			start = 0
		}
		for _, res := range m.completed[start:] {
			if res.success {
				s.WriteString(fmt.Sprintf("  %s %s\n", style.IconSuccess, style.Dim.Render(res.resource)))
			} else {
				s.WriteString(fmt.Sprintf("  %s %s\n", style.IconError, style.Error.Render(res.resource)))
			}
		}
	}

	return s.String()
}

// ApplyWithProgress executes apply with a progress bar and continues on failures
func (e *Engine) ApplyWithProgress(ctx context.Context, result *PlanResult, opts ApplyOptions) error {
	if !result.HasChanges {
		fmt.Println(style.Info.Render("No changes to apply."))
		return nil
	}

	// Collect all resources to process
	type workItem struct {
		provider string
		resource resource.Resource
		action   string // "create", "update", "remove", "restore"
		state    provider.ResourceState
	}

	var workItems []workItem
	for providerName, plan := range result.ProviderPlans {
		for _, res := range plan.Additions {
			workItems = append(workItems, workItem{providerName, res, "create", provider.ResourceState{}})
		}
		for _, mod := range plan.Modifications {
			workItems = append(workItems, workItem{providerName, mod.Resource, "update", mod.OldState})
		}
		for _, res := range plan.Removals {
			workItems = append(workItems, workItem{providerName, res, "remove", provider.ResourceState{}})
		}
		for _, drift := range plan.Drifted {
			workItems = append(workItems, workItem{providerName, drift.Resource, "restore", drift.ExpectedState})
		}
	}

	if len(workItems) == 0 {
		fmt.Println(style.Info.Render("No changes to apply."))
		return nil
	}

	totalOps := len(workItems)

	// Create progress model
	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	m := applyProgressModel{
		progress:  prog,
		current:   0,
		total:     totalOps,
		percent:   0,
		completed: []resourceResult{},
	}

	// Create Bubble Tea program
	p := tea.NewProgram(m)

	// Execute apply in background
	var results []resourceResult
	completedOps := 0

	go func() {
		for _, item := range workItems {
			resourceID := fmt.Sprintf("%s/%s", item.resource.GetKind(), item.resource.GetMetadata().Name)

			// Send "starting" message
			action := strings.Title(item.action + "ing")
			p.Send(applyProgressMsg{
				resource:  resourceID,
				action:    action,
				completed: results,
				current:   completedOps,
				total:     totalOps,
				percent:   float64(completedOps) / float64(totalOps),
			})

			// Execute the action
			var err error
			prov := e.Providers[item.provider]

			switch item.action {
			case "create":
				err = prov.Apply(ctx, provider.Plan{Additions: []resource.Resource{item.resource}})
			case "update":
				err = prov.Apply(ctx, provider.Plan{Modifications: []provider.Modification{{
					Resource: item.resource,
					OldState: item.state,
				}}})
			case "remove":
				err = prov.Apply(ctx, provider.Plan{Removals: []resource.Resource{item.resource}})
			case "restore":
				err = prov.Apply(ctx, provider.Plan{Drifted: []provider.Drift{{
					Resource:      item.resource,
					ExpectedState: item.state,
				}}})
			}

			// Record result
			completedOps++
			res := resourceResult{
				resource: resourceID,
				action:   item.action,
				success:  err == nil,
				err:      err,
			}
			results = append(results, res)

			// Send completion update
			p.Send(applyProgressMsg{
				resource:  resourceID,
				action:    action,
				completed: results,
				current:   completedOps,
				total:     totalOps,
				percent:   float64(completedOps) / float64(totalOps),
			})
		}

		p.Send(applyCompleteMsg{results: results, successCount: 0, failCount: 0})
	}()

	// Run the program
	if _, err := p.Run(); err != nil {
		// Fallback to non-interactive apply
		return e.Apply(ctx, result, opts)
	}

	// Count successes and failures
	successCount := 0
	failCount := 0
	for _, res := range results {
		if res.success {
			successCount++
		} else {
			failCount++
		}
	}

	// Show summary with lipgloss styling
	fmt.Println()
	
	if failCount == 0 {
		// All succeeded
		fmt.Printf("%s Apply complete! %d resource%s synchronized\n", 
			style.IconSuccess, successCount, plural(successCount))
	} else if successCount == 0 {
		// All failed
		fmt.Println(style.ErrorBox.Render(
			style.Error.Render("✖ Apply failed") + "\n\n" +
			fmt.Sprintf("All %d resources failed to apply", failCount),
		))
	} else {
		// Mixed results
		var summary strings.Builder
		summary.WriteString(style.Warning.Render("⚠ Apply completed with errors") + "\n\n")
		summary.WriteString(fmt.Sprintf("%s %d succeeded\n", style.IconSuccess, successCount))
		summary.WriteString(fmt.Sprintf("%s %d failed\n\n", style.IconError, failCount))
		summary.WriteString(style.Bold.Render("Failed resources:"))
		summary.WriteString("\n")
		for _, res := range results {
			if !res.success {
				summary.WriteString(fmt.Sprintf("  • %s: %s\n", res.resource, res.err))
			}
		}
		fmt.Println(style.WarningBox.Render(summary.String()))
	}

	// Update and save state for successful operations only
	if successCount > 0 {
		newState := state.NewState()
		for i, item := range workItems {
			if results[i].success {
				stateEntry := e.resourceToStateEntry(item.resource, item.provider)
				newState.SetResource(stateEntry)
			}
		}

		if err := e.StateBackend.Save(ctx, newState); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}
	}

	// Return error if any failed
	if failCount > 0 {
		return fmt.Errorf("%d resource(s) failed to apply", failCount)
	}

	return nil
}

// Apply executes the planned changes.
func (e *Engine) Apply(ctx context.Context, result *PlanResult, opts ApplyOptions) error {
	// Check if there are changes to apply (HasChanges now includes drift)
	if !result.HasChanges {
		fmt.Println("No changes to apply.")
		return nil
	}

	// Check for confirmation (should already be confirmed by cmd/apply.go)
	if !opts.Confirm {
		return fmt.Errorf("apply not confirmed: this should not happen, use --confirm flag")
	}

	// Execute changes for each provider
	totalResources := len(result.ProviderPlans)
	currentProvider := 0
	for providerName, plan := range result.ProviderPlans {
		currentProvider++

		// Progress header
		fmt.Printf("\n[%d/%d] Processing %s...\n", currentProvider, totalResources, providerName)

		provider, exists := e.Providers[providerName]
		if !exists {
			return fmt.Errorf("provider %s not found", providerName)
		}

		// Track resource-level progress
		currentRes := 0

		// Show additions
		for _, res := range plan.Additions {
			currentRes++
			fmt.Printf("  %s Creating %s/%s\n", style.IconAdd, res.GetKind(), res.GetMetadata().Name)
		}

		// Show modifications
		for _, mod := range plan.Modifications {
			currentRes++
			fmt.Printf("  %s Updating %s/%s\n", style.IconEdit, mod.Resource.GetKind(), mod.Resource.GetMetadata().Name)
		}

		// Show removals
		for _, res := range plan.Removals {
			currentRes++
			fmt.Printf("  %s Removing %s/%s\n", style.IconRemove, res.GetKind(), res.GetMetadata().Name)
		}

		// Apply changes
		if err := provider.Apply(ctx, plan); err != nil {
			return fmt.Errorf("failed to apply changes for provider %s: %w", providerName, err)
		}

		if currentRes > 0 {
			fmt.Printf("  %s %d changes applied\n", style.IconSuccess, currentRes)
		}
	}

	// Update state with new resource states
	newState := state.NewState()
	totalChanges := 0
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
			totalChanges++
		}

		// Add new resources to state
		for _, res := range plan.Additions {
			stateEntry := e.resourceToStateEntry(res, providerName)
			newState.SetResource(stateEntry)
			totalChanges++
		}

		// Add drift-restored resources to state (they were applied as additions internally)
		for _, drift := range plan.Drifted {
			stateEntry := e.resourceToStateEntry(drift.Resource, providerName)
			newState.SetResource(stateEntry)
			totalChanges++
		}
	}

	// Save updated state
	if err := e.StateBackend.Save(ctx, newState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	fmt.Println()
	fmt.Printf("%s Apply complete! %d resource%s synchronized\n", style.IconSuccess, totalChanges, plural(totalChanges))

	return nil
}

func plural(n int) string {
	if n != 1 {
		return "s"
	}
	return ""
}

// resourceToStateEntry converts a resource to a state entry.
// For file resources, it calculates checksums of the content.
func (e *Engine) resourceToStateEntry(res resource.Resource, providerName string) provider.ResourceState {
	// Use kind/name for ID (Terraform-style: files/dirs are for human org only)
	stateEntry := provider.ResourceState{
		ID:        fmt.Sprintf("%s/%s", res.GetKind(), res.GetMetadata().Name),
		Kind:      res.GetKind(),
		Name:      res.GetMetadata().Name,
		Namespace: res.GetMetadata().GetNamespace(),
	}

	// Calculate checksums for file resources
	switch r := res.(type) {
	case *resource.ManagedFile:
		var content string
		if r.Spec.Source != "" {
			// Inline content - render with template if enabled
			content = r.Spec.Source
			if r.Spec.Template {
				engine := config.NewTemplateEngine(e.TemplateContext)
				rendered, err := engine.RenderTemplate("inline", content)
				if err == nil {
					content = rendered
				}
			}
		} else if r.Spec.SourceFile != "" {
			// External file - path is relative to resources/ subdirectory
			sourcePath := filepath.Join(e.Config.DotfilesRoot, "resources", r.Spec.SourceFile)
			data, err := e.renderSourceFile(sourcePath, r.Spec.Template)
			if err == nil {
				content = data
			}
		}

		if content != "" {
			hash := sha256.Sum256([]byte(content))
			stateEntry.DestHash = hex.EncodeToString(hash[:])
		}

		// Store the mode in extra
		stateEntry.Extra = map[string]interface{}{
			"mode": r.Spec.Mode,
		}

	case *resource.ManagedDirectory:
		stateEntry.Extra = map[string]interface{}{
			"recursive": r.Spec.Recursive,
			"clean":     r.Spec.Clean,
		}

	case *resource.NpmPackages:
		pkgs := make(map[string]string)
		for _, p := range r.Spec.Packages {
			pkgs[p.Name] = p.Version
		}
		stateEntry.Extra = map[string]interface{}{
			"packages": pkgs,
		}

	case *resource.GoPackages:
		mods := make(map[string]string)
		for _, m := range r.Spec.Packages {
			mods[m.Module] = m.Version
		}
		stateEntry.Extra = map[string]interface{}{
			"modules": mods,
		}
	}

	return stateEntry
}

// renderSourceFile reads and optionally templates a source file.
func (e *Engine) resourceToSourceEntry(res resource.Resource, providerName string) provider.ResourceState {
	// Use kind/name for ID (Terraform-style: files/dirs are for human org only)
	stateEntry := provider.ResourceState{
		ID:        fmt.Sprintf("%s/%s", res.GetKind(), res.GetMetadata().Name),
		Kind:      res.GetKind(),
		Name:      res.GetMetadata().Name,
		Namespace: res.GetMetadata().GetNamespace(),
	}

	// Calculate checksums for file resources
	switch r := res.(type) {
	case *resource.ManagedFile:
		var content string
		if r.Spec.Source != "" {
			// Inline content - render with template if enabled
			content = r.Spec.Source
			if r.Spec.Template {
				engine := config.NewTemplateEngine(e.TemplateContext)
				rendered, err := engine.RenderTemplate("inline", content)
				if err == nil {
					content = rendered
				}
			}
		} else if r.Spec.SourceFile != "" {
			// External file - path is relative to resources/ subdirectory
			sourcePath := filepath.Join(e.Config.DotfilesRoot, "resources", r.Spec.SourceFile)
			data, err := e.renderSourceFile(sourcePath, r.Spec.Template)
			if err == nil {
				content = data
			}
		}

		if content != "" {
			hash := sha256.Sum256([]byte(content))
			stateEntry.DestHash = hex.EncodeToString(hash[:])
		}

		// Store the mode in extra
		stateEntry.Extra = map[string]interface{}{
			"mode": r.Spec.Mode,
		}

	case *resource.ManagedDirectory:
		stateEntry.Extra = map[string]interface{}{
			"recursive": r.Spec.Recursive,
			"clean":     r.Spec.Clean,
		}

	case *resource.NpmPackages:
		pkgs := make(map[string]string)
		for _, p := range r.Spec.Packages {
			pkgs[p.Name] = p.Version
		}
		stateEntry.Extra = map[string]interface{}{
			"packages": pkgs,
		}

	case *resource.GoPackages:
		mods := make(map[string]string)
		for _, m := range r.Spec.Packages {
			mods[m.Module] = m.Version
		}
		stateEntry.Extra = map[string]interface{}{
			"modules": mods,
		}
	}

	return stateEntry
}

// renderSourceFile reads and optionally templates a source file.
func (e *Engine) renderSourceFile(sourcePath string, useTemplate bool) (string, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}

	content := string(data)

	if useTemplate && e.TemplateContext != nil {
		engine := config.NewTemplateEngine(e.TemplateContext)
		return engine.RenderTemplate(sourcePath, content)
	}

	return content, nil
}
