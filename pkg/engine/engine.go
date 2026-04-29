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
	Config          *config.Config
	TemplateContext *config.TemplateContext
	StateBackend    state.StateBackend
	Providers       map[string]provider.Provider
}

// NewEngine creates a new Engine with default configuration.
func NewEngine() (*Engine, error) {
	cfg, ctx, err := config.LoadComplete()
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
		Config:          cfg,
		TemplateContext: ctx,
		StateBackend:    backend,
		Providers:       providerMap,
	}, nil
}

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
}

// Plan loads state, parses resources, and generates plans from all providers.
// It accepts PlanOptions which can be used to target specific resources.
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
                        resourceGroups = append(resourceGroups, resource.ResourceGroup{
                            Kind: s.Kind,
                            Name: s.Group,
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
        CurrentState:  currentState,
        ProviderPlans: providerPlans,
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

        // Reconcile
        plan := prov.Reconcile(providerGroups, providerState)

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
		for _ = range plan.Drifted {
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
func (e *Engine) resourcesToGroups(resources []resource.Resource) []resource.ResourceGroup {
    var groups []resource.ResourceGroup
    for _, res := range resources {
        groups = append(groups, res.ToGroup())
    }
    return groups
}

// filterResourceGroupsByTargets filters resource groups and their items according to targets.
func filterResourceGroupsByTargets(groups []resource.ResourceGroup, targets []TargetMatch) []resource.ResourceGroup {
    var out []resource.ResourceGroup
    for _, g := range groups {
        matched := false
        // Check kind/group/item match for group-level and item-level targeting
        for _, t := range targets {
            // Manually check kind/group first (case-insensitive) so we can
            // handle item-specific targets which require passing the item name
            // into Matches. Using Matches with an empty item would fail when
            // the target specified an item.
            if t.Kind != "" && !strings.EqualFold(t.Kind, g.Kind) {
                continue
            }
            if t.Group != "" && !strings.EqualFold(t.Group, g.Name) {
                continue
            }

            if t.Item == "" {
                // target is kind or kind/group -> keep full group
                matched = true
                break
            }

            // target specifies an item; check if the item exists in group
            for _, it := range g.Items {
                if t.Matches(g.Kind, g.Name, it.Name) {
                    matched = true
                    break
                }
            }
            if matched {
                break
            }
        }
        if matched {
            out = append(out, g)
        }
    }
    return out
}

// matchesKind returns true if the target kind matches the resource kind.
// Only the full/display kind (e.g. "HomebrewPackages") is accepted.
// Matching is case-insensitive.
func matchesKind(targetKind, resourceKind string) bool {
    return strings.EqualFold(targetKind, resourceKind)
}

// (removed kindToFullKind) Full/display kinds are used directly in resources.

// filterPlanByTargets trims a provider.GroupPlan to include only items that match targets.
func filterPlanByTargets(plan provider.GroupPlan, targets []TargetMatch) provider.GroupPlan {
    // Helper to check if a kind/group/item is targeted
    isTargeted := func(kind, group, item string) bool {
        for _, t := range targets {
            // match kind (normalize aliases)
            if t.Kind != "" {
                if !strings.EqualFold(t.Kind, kind) {
                    continue
                }
            }
            if t.Group != "" && t.Group != group {
                continue
            }
            if t.Item != "" && t.Item != item {
                continue
            }
            return true
        }
        return false
    }

    var out provider.GroupPlan

    // Filter additions
    for _, a := range plan.Additions {
        var items []resource.ResourceItem
        for _, it := range a.Items {
            if isTargeted(a.Kind, a.Group, it.Name) {
                items = append(items, it)
            }
        }
        if len(items) > 0 {
            out.Additions = append(out.Additions, provider.GroupAddition{Kind: a.Kind, Group: a.Group, Items: items})
        }
    }

    // Filter modifications
    for _, m := range plan.Modifications {
        var changes []provider.ItemChange
        for _, c := range m.Changes {
            if isTargeted(m.Kind, m.Group, c.ItemName) {
                changes = append(changes, c)
            }
        }
        if len(changes) > 0 {
            out.Modifications = append(out.Modifications, provider.GroupModification{Kind: m.Kind, Group: m.Group, Changes: changes})
        }
    }

    // Filter removals
    for _, r := range plan.Removals {
        var items []resource.ResourceItem
        for _, it := range r.Items {
            if isTargeted(r.Kind, r.Group, it.Name) {
                items = append(items, it)
            }
        }
        if len(items) > 0 {
            out.Removals = append(out.Removals, provider.GroupRemoval{Kind: r.Kind, Group: r.Group, Items: items})
        }
    }

    // Filter cleanup
    for _, c := range plan.Cleanup {
        var items []resource.ResourceItem
        for _, it := range c.Items {
            if isTargeted(c.Kind, c.Group, it.Name) {
                items = append(items, it)
            }
        }
        if len(items) > 0 {
            out.Cleanup = append(out.Cleanup, provider.GroupCleanup{Kind: c.Kind, Group: c.Group, Items: items, Reason: c.Reason})
        }
    }

    // Filter drifted
    for _, d := range plan.Drifted {
        if isTargeted(d.Kind, d.Group, d.Item) {
            out.Drifted = append(out.Drifted, d)
        }
    }

    // InSync: include only if targeted (use items match)
    for _, s := range plan.InSync {
        var items []resource.ItemState
        for _, it := range s.Items {
            if isTargeted(s.Kind, s.Group, it.Name) {
                items = append(items, it)
            }
        }
        if len(items) > 0 {
            out.InSync = append(out.InSync, provider.GroupState{Kind: s.Kind, Group: s.Group, Items: items, Version: s.Version})
        }
    }

    // Warnings are retained if they refer to targeted groups/items
    for _, w := range plan.Warnings {
        if w.GroupID == "" && w.ItemID == "" {
            continue
        }
        // Basic check: if groupID present, split and check
        if w.GroupID != "" {
            parts := strings.SplitN(w.GroupID, "/", 2)
            if len(parts) >= 1 {
                gkind := parts[0]
                ggroup := ""
                if len(parts) == 2 {
                    ggroup = parts[1]
                }
                if isTargeted(gkind, ggroup, w.ItemID) {
                    out.Warnings = append(out.Warnings, w)
                }
            }
        }
    }

    return out
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

// processCleanup removes cleanup items from state (state-only operation, always succeeds)
func (e *Engine) processCleanup(ctx context.Context, result *PlanResult) error {
	existingState, err := e.StateBackend.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load existing state: %w", err)
	}

	stateToSave := existingState
	if stateToSave.Resources == nil {
		return nil // nothing to cleanup
	}

	for _, plan := range result.ProviderPlans {
		for _, cleanup := range plan.Cleanup {
			// Find the resource group in state
			for i, res := range stateToSave.Resources {
				if res.Kind == cleanup.Kind && res.Group == cleanup.Group {
					// Remove cleanup items from this resource group
					newItems := make([]resource.ItemState, 0)
					for _, item := range res.Items {
						shouldKeep := true
						for _, cleanupItem := range cleanup.Items {
							if item.Name == cleanupItem.Name {
								shouldKeep = false
								break
							}
						}
						if shouldKeep {
							newItems = append(newItems, item)
						}
					}
					stateToSave.Resources[i].Items = newItems
					break
				}
			}
		}
	}

	// Save state after cleanup processing
	if err := e.StateBackend.Save(ctx, stateToSave); err != nil {
		return fmt.Errorf("failed to save state after cleanup: %w", err)
	}

	return nil
}

// ApplyWithProgress executes the planned changes with progress reporting.
func (e *Engine) ApplyWithProgress(ctx context.Context, result *PlanResult, opts ApplyOptions) error {
	if !result.HasChanges {
		fmt.Println(style.Info.Render("No changes to apply."))
		return nil
	}

	// STEP 1: Process cleanup items first (state-only, always succeeds)
	// This ensures cleanup is persisted even if other operations fail
	if err := e.processCleanup(ctx, result); err != nil {
		return fmt.Errorf("failed to process cleanup items: %w", err)
	}

	workItems := e.collectWorkItems(result)
	if len(workItems) == 0 {
		// Only cleanup items, no provider operations needed
		fmt.Println(style.Info.Render("Changes applied successfully. (cleanup only)"))
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
		case "cleanup":
			// Cleanup is state-only, already handled in processCleanup
			// This is a no-op here, just mark as success
			completed := i + 1
			res := resourceResult{
				resource: resourceID,
				action:   item.action,
				success:  true,
				err:      nil,
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
			continue
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
		fmt.Println()
		fmt.Println(style.Error.Render("✖ Apply failed"))
		fmt.Println()
		fmt.Printf("All %d resources failed to apply\n", failCount)
		fmt.Println()
		fmt.Println(style.Bold.Render("Failed resources:"))
		fmt.Println()
		for _, res := range results {
			if !res.success {
				fmt.Printf("  %s %s\n", style.Error.Render("•"), style.Dim.Render(res.resource))
				errLines := strings.Split(res.err.Error(), "\n")
				for _, line := range errLines {
					fmt.Printf("    %s\n", style.Error.Render(line))
				}
				fmt.Println()
			}
		}
	} else {
		fmt.Println()
		fmt.Println(style.Warning.Render("⚠ Apply completed with errors"))
		fmt.Println()
		fmt.Printf("%s %d succeeded\n", style.IconSuccess, successCount)
		fmt.Printf("%s %d failed\n", style.IconError, failCount)
		fmt.Println()
		fmt.Println(style.Bold.Render("Failed resources:"))
		fmt.Println()
		for _, res := range results {
			if !res.success {
				// Format error with proper indentation and styling
				fmt.Printf("  %s %s\n", style.Error.Render("•"), style.Dim.Render(res.resource))
				// Wrap error message with indentation
				errLines := strings.Split(res.err.Error(), "\n")
				for _, line := range errLines {
					fmt.Printf("    %s\n", style.Error.Render(line))
				}
				fmt.Println()
			}
		}
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
			if results[i].success && item.action != "cleanup" {
				// Skip cleanup items - they were already processed in processCleanup
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
		// Return a simple error indicator - details already shown in UI
		return fmt.Errorf("apply completed with %d error(s)", failCount)
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
		for _, cleanup := range plan.Cleanup {
			for _, item := range cleanup.Items {
				items = append(items, workItem{
					provider: providerName,
					kind:     cleanup.Kind,
					group:    cleanup.Group,
					item:     item.Name,
					action:   "cleanup",
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
