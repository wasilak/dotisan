// Package engine provides the core orchestration logic for dotisan.
package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/wasilak/dotisan/pkg/config"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/providers"
	"github.com/wasilak/dotisan/pkg/resource"
	"github.com/wasilak/dotisan/pkg/state"
	"github.com/wasilak/dotisan/pkg/style"
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
		for _, sync := range plan.InSync {
			result.TotalInSync += len(sync.Items)
		}
		for _ = range plan.Drifted {
			result.TotalDrifted++
		}
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
	existingState, err := e.StateBackend.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load existing state: %w", err)
	}

	stateToSave := existingState
	if stateToSave.Resources == nil {
		stateToSave = state.NewState()
	}
	for _, plan := range result.ProviderPlans {
		// Add in-sync resources
		for _, inSync := range plan.InSync {
			stateToSave.SetResourceGroup(provider.ResourceState{
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
			stateToSave.SetResourceGroup(provider.ResourceState{
				Kind:      addition.Kind,
				Group:     addition.Group,
				Items:     items,
				Namespace: "default",
			})
		}
	}

	if err := e.StateBackend.Save(ctx, stateToSave); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// ApplyWithProgress executes the planned changes with progress reporting.
func (e *Engine) ApplyWithProgress(ctx context.Context, result *PlanResult, opts ApplyOptions) error {
	if !result.HasChanges {
		fmt.Println(style.Info.Render("No changes to apply."))
		return nil
	}

	workItems := e.collectWorkItems(result)
	if len(workItems) == 0 {
		fmt.Println(style.Info.Render("No changes to apply."))
		return nil
	}

	totalOps := len(workItems)
	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	m := applyProgressModel{
		progress:   prog,
		current:   0,
		total:     totalOps,
		completed: []resourceResult{},
	}

	p := tea.NewProgram(m)

	var results []resourceResult
	go func() {
		for i, item := range workItems {
			resourceID := fmt.Sprintf("%s/%s/%s", item.kind, item.group, item.item)
			var action string
			switch item.action {
			case "create":
				action = "Creating"
			case "remove":
				action = "Removing"
			case "update":
				action = "Updating"
			case "restore":
				action = "Restoring"
			default:
				action = strings.Title(item.action)
			}

			p.Send(applyProgressMsg{
				resource:  resourceID,
				action:    action,
				completed: results,
				current:   i,
				total:     totalOps,
				percent:   float64(i) / float64(totalOps),
			})

			var err error
			prov := e.Providers[item.provider]

			plan := provider.GroupPlan{}
			switch item.action {
			case "create":
				plan.Additions = []provider.GroupAddition{{
					Kind:  item.kind,
					Group: item.group,
					Items: []resource.ResourceItem{{Name: item.item, Version: item.version}},
				}}
			case "update":
				plan.Modifications = []provider.GroupModification{{
					Kind:  item.kind,
					Group: item.group,
					Changes: []provider.ItemChange{{
						ItemName:  item.item,
						OldState: resource.ItemState{Version: item.oldVersion},
						NewState: resource.ItemState{Version: item.version},
					}},
				}}
			case "remove":
				plan.Removals = []provider.GroupRemoval{{
					Kind:  item.kind,
					Group: item.group,
					Items: []resource.ResourceItem{{Name: item.item}},
				}}
			case "restore":
				plan.Drifted = []provider.ItemDrift{{
					Kind:          item.kind,
					Group:         item.group,
					Item:          item.item,
					ExpectedState: resource.ItemState{Version: item.version},
				}}
			}

			err = prov.Apply(ctx, plan)

			completed := i + 1
			res := resourceResult{
				resource: resourceID,
				action:   item.action,
				success:  err == nil,
				err:      err,
			}
			results = append(results, res)

			p.Send(applyProgressMsg{
				resource:  resourceID,
				action:    action,
				completed: results,
				current:   completed,
				total:     totalOps,
				percent:   float64(completed) / float64(totalOps),
			})
		}

		successCount := 0
		failCount := 0
		for _, r := range results {
			if r.success {
				successCount++
			} else {
				failCount++
			}
		}
		p.Send(applyCompleteMsg{results: results, successCount: successCount, failCount: failCount})
	}()

	if _, err := p.Run(); err != nil {
		return e.Apply(ctx, result, opts)
	}

	successCount := 0
	failCount := 0
	for _, r := range results {
		if r.success {
			successCount++
		} else {
			failCount++
		}
	}

	fmt.Println()
	singular := ""
	if successCount == 1 {
		singular = "s"
	}
	if failCount == 0 {
		fmt.Printf("%s Apply complete! %d resource%s synchronized\n",
			style.IconSuccess, successCount, singular)
	} else if successCount == 0 {
		errMsg := style.ErrorBox.Render(
			style.Error.Render("✖ Apply failed") + "\n\n" +
				fmt.Sprintf("All %d resources failed to apply", failCount),
		)
		fmt.Println(errMsg)
	} else {
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

	if successCount > 0 {
		existingState, err := e.StateBackend.Load(ctx)
		if err != nil {
			return fmt.Errorf("failed to load existing state: %w", err)
		}

		stateToSave := existingState

		if stateToSave.Resources == nil {
			stateToSave = state.NewState()
		}

		for i, item := range workItems {
			if results[i].success {
				stateToSave.SetResourceGroup(provider.ResourceState{
					Kind:      item.kind,
					Group:     item.group,
					Items:     []resource.ItemState{{Name: item.item, Version: item.version, Status: "present"}},
					Namespace: "default",
				})
			}
		}
		if err := e.StateBackend.Save(ctx, stateToSave); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}
	}

	if failCount > 0 {
		return fmt.Errorf("%d resource(s) failed to apply", failCount)
	}

	return nil
}

type workItem struct {
	provider   string
	kind       string
	group      string
	item       string
	oldVersion string
	version    string
	action     string
}

func (e *Engine) collectWorkItems(result *PlanResult) []workItem {
	var items []workItem
	for providerName, plan := range result.ProviderPlans {
		for _, add := range plan.Additions {
			for _, item := range add.Items {
				items = append(items, workItem{
					provider: providerName,
					kind:     add.Kind,
					group:    add.Group,
					item:     item.Name,
					version:  item.Version,
					action:   "create",
				})
			}
		}
		for _, mod := range plan.Modifications {
			for _, change := range mod.Changes {
				items = append(items, workItem{
					provider:   providerName,
					kind:      mod.Kind,
					group:     mod.Group,
					item:      change.ItemName,
					oldVersion: change.OldState.Version,
					version:   change.NewState.Version,
					action:    "update",
				})
			}
		}
		for _, rem := range plan.Removals {
			for _, item := range rem.Items {
				items = append(items, workItem{
					provider: providerName,
					kind:     rem.Kind,
					group:    rem.Group,
					item:     item.Name,
					action:   "remove",
				})
			}
		}
		for _, drift := range plan.Drifted {
			items = append(items, workItem{
				provider:   providerName,
				kind:      drift.Kind,
				group:     drift.Group,
				item:      drift.Item,
				version:   drift.ExpectedState.Version,
				oldVersion: drift.ActualState.Version,
				action:    "restore",
			})
		}
	}
	return items
}

type resourceResult struct {
	resource string
	action   string
	success  bool
	err      error
}

type applyProgressMsg struct {
	resource  string
	action    string
	completed []resourceResult
	current   int
	total     int
	percent   float64
}

type applyCompleteMsg struct {
	results      []resourceResult
	successCount int
	failCount    int
}

type applyTickMsg struct{}

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

	s.WriteString(m.progress.ViewAs(m.percent))
	s.WriteString(fmt.Sprintf("  %d/%d\n", m.current, m.total))

	if m.action != "" && m.resource != "" {
		s.WriteString(fmt.Sprintf("\n%s %s %s\n", style.Dim.Render("→"), style.Info.Render(m.action), style.Dim.Render(m.resource)))
	}

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
