package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/wasilak/dotisan/pkg/graph"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
	"github.com/wasilak/dotisan/pkg/state"
	"github.com/wasilak/dotisan/pkg/style"
)

// Apply executes the planned changes.
func (e *Engine) Apply(ctx context.Context, result *PlanResult, opts ApplyOptions) error {
	if !result.HasChanges {
		return nil
	}

	if !opts.Confirm {
		return fmt.Errorf("apply not confirmed")
	}

	// Load existing state first
	existingState, err := e.StateBackend.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load existing state: %w", err)
	}

	stateToSave := existingState
	if stateToSave.Resources == nil {
		stateToSave = state.NewState()
	}

	// STEP 1: Process cleanup items first (state-only, always succeeds)
	// These are items that exist in state but not in config and not installed
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

	// Save state after cleanup processing (cleanup always succeeds)
	if err := e.StateBackend.Save(ctx, stateToSave); err != nil {
		return fmt.Errorf("failed to save state after cleanup: %w", err)
	}

	// STEP 2: Execute provider changes in dependency order.
	type failureEntry struct {
		Resource string
		Err      error
	}
	var failures []failureEntry

	// failedNodes tracks NodeIDs that failed or were skipped; used to propagate
	// skips to dependents.
	failedNodes := make(map[graph.NodeID]bool)

	// succeededGroups tracks (kind, group) pairs that applied without error;
	// only those groups get their state updated in STEP 3.
	type kindGroup struct{ kind, group string }
	succeededGroups := make(map[kindGroup]bool)

	// parseNodeID splits a NodeID of the form "Kind/Group" into its two parts.
	parseNodeID := func(id graph.NodeID) (kind, group string) {
		s := string(id)
		parts := strings.SplitN(s, "/", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
		return "", ""
	}

	// applyGroup executes all plan entries (adds, modifies, removes) for a single
	// (kind, group) pair and returns whether all operations succeeded.
	applyGroup := func(providerName, kind, group string) bool {
		prov, exists := e.Providers[providerName]
		plan := result.ProviderPlans[providerName]
		if !exists {
			dummyErr := fmt.Errorf("provider %s not found", providerName)
			for _, a := range plan.Additions {
				if a.Kind != kind || a.Group != group {
					continue
				}
				for _, it := range a.Items {
					failures = append(failures, failureEntry{Resource: fmt.Sprintf("%s/%s/%s", kind, group, it.Name), Err: dummyErr})
				}
			}
			return false
		}

		ok := true
		for _, a := range plan.Additions {
			if a.Kind != kind || a.Group != group {
				continue
			}
			for _, it := range a.Items {
				if opts.OnItemStart != nil {
					opts.OnItemStart(kind, group, it.Name)
				}
				singlePlan := provider.GroupPlan{
					Additions: []provider.GroupAddition{{Kind: kind, Group: group, Items: []resource.ResourceItem{it}}},
				}
				err := prov.Apply(ctx, singlePlan)
				if opts.OnItemComplete != nil {
					opts.OnItemComplete(kind, group, it.Name, err)
				}
				if err != nil {
					failures = append(failures, failureEntry{Resource: fmt.Sprintf("%s/%s/%s", kind, group, it.Name), Err: fmt.Errorf("failed to add: %w", err)})
					ok = false
				}
			}
		}
		for _, m := range plan.Modifications {
			if m.Kind != kind || m.Group != group {
				continue
			}
			for _, c := range m.Changes {
				if opts.OnItemStart != nil {
					opts.OnItemStart(kind, group, c.ItemName)
				}
				singlePlan := provider.GroupPlan{
					Modifications: []provider.GroupModification{{Kind: kind, Group: group, Changes: []provider.ItemChange{c}}},
				}
				err := prov.Apply(ctx, singlePlan)
				if opts.OnItemComplete != nil {
					opts.OnItemComplete(kind, group, c.ItemName, err)
				}
				if err != nil {
					failures = append(failures, failureEntry{Resource: fmt.Sprintf("%s/%s/%s", kind, group, c.ItemName), Err: fmt.Errorf("failed to modify: %w", err)})
					ok = false
				}
			}
		}
		for _, r := range plan.Removals {
			if r.Kind != kind || r.Group != group {
				continue
			}
			for _, it := range r.Items {
				if opts.OnItemStart != nil {
					opts.OnItemStart(kind, group, it.Name)
				}
				singlePlan := provider.GroupPlan{
					Removals: []provider.GroupRemoval{{Kind: kind, Group: group, Items: []resource.ResourceItem{it}}},
				}
				err := prov.Apply(ctx, singlePlan)
				if opts.OnItemComplete != nil {
					opts.OnItemComplete(kind, group, it.Name, err)
				}
				if err != nil {
					failures = append(failures, failureEntry{Resource: fmt.Sprintf("%s/%s/%s", kind, group, it.Name), Err: fmt.Errorf("failed to remove: %w", err)})
					ok = false
				}
			}
		}
		return ok
	}

	// Pass 1: process groups in topological (dependency) order.
	processedGroups := make(map[kindGroup]bool)
	for _, nodeID := range result.DependencyOrder {
		kind, group := parseNodeID(nodeID)
		if kind == "" {
			continue
		}
		kg := kindGroup{kind, group}
		processedGroups[kg] = true

		// Check if any direct dependency failed; skip this group if so.
		depFailed := false
		if result.DAG != nil {
			for _, dep := range result.DAG.DependenciesOf(nodeID) {
				if failedNodes[dep] {
					depFailed = true
					break
				}
			}
		}

		provName, ok := provider.ProviderNameForKind(kind)
		if !ok {
			continue // no provider — no plan entries for this kind
		}

		if depFailed {
			failedNodes[nodeID] = true
			// Record the skip in the provider plan for visibility.
			plan := result.ProviderPlans[provName]
			plan.Skipped = append(plan.Skipped, provider.GroupSkip{
				Kind:   kind,
				Group:  group,
				Reason: fmt.Sprintf("dependency failed"),
			})
			result.ProviderPlans[provName] = plan
			continue
		}

		if applyGroup(provName, kind, group) {
			succeededGroups[kg] = true
		} else {
			failedNodes[nodeID] = true
		}
	}

	// Pass 2: process any groups not covered by DependencyOrder (e.g. synthetic
	// target groups added after DAG building).
	for provName, plan := range result.ProviderPlans {
		allKGs := make(map[kindGroup]bool)
		for _, a := range plan.Additions {
			allKGs[kindGroup{a.Kind, a.Group}] = true
		}
		for _, m := range plan.Modifications {
			allKGs[kindGroup{m.Kind, m.Group}] = true
		}
		for _, r := range plan.Removals {
			allKGs[kindGroup{r.Kind, r.Group}] = true
		}
		for kg := range allKGs {
			if processedGroups[kg] {
				continue
			}
			processedGroups[kg] = true
			if applyGroup(provName, kg.kind, kg.group) {
				succeededGroups[kg] = true
			}
		}
	}

	// STEP 3: Update state with successful provider operations
	existingState, err = e.StateBackend.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to reload state after provider operations: %w", err)
	}
	stateToSave = existingState
	if stateToSave.Resources == nil {
		stateToSave = state.NewState()
	}

	for _, plan := range result.ProviderPlans {
		// InSync groups always update state (they require no apply action).
		for _, inSync := range plan.InSync {
			stateToSave.SetResourceGroup(provider.ResourceState{
				Kind:  inSync.Kind,
				Group: inSync.Group,
				Items: inSync.Items,
			})
		}

		// Additions: only persist state for groups that succeeded.
		for _, addition := range plan.Additions {
			if !succeededGroups[kindGroup{addition.Kind, addition.Group}] {
				continue
			}
			items := make([]resource.ItemState, 0, len(addition.Items))
			for _, item := range addition.Items {
				checksum := ""
				if dest, ok := item.Extra["destination"].(string); ok && dest != "" {
					if data, err := readFileMaybe(dest); err == nil {
						h := sha256.Sum256(data)
						checksum = hex.EncodeToString(h[:])
					}
				}
				items = append(items, resource.ItemState{
					Name:     item.Name,
					Version:  item.Version,
					Status:   "present",
					Checksum: checksum,
				})
			}
			stateToSave.SetResourceGroup(provider.ResourceState{
				Kind:  addition.Kind,
				Group: addition.Group,
				Items: items,
			})
		}

		// Modifications: only persist state for groups that succeeded.
		for _, modification := range plan.Modifications {
			if !succeededGroups[kindGroup{modification.Kind, modification.Group}] {
				continue
			}
			for _, change := range modification.Changes {
				checksum := ""
				if dest, ok := change.NewState.Extra["destination"].(string); ok && dest != "" {
					if data, err := readFileMaybe(dest); err == nil {
						h := sha256.Sum256(data)
						checksum = hex.EncodeToString(h[:])
					}
				}

				updated := false
				for gi := range stateToSave.Resources {
					r := &stateToSave.Resources[gi]
					if r.Kind != modification.Kind || r.Group != modification.Group {
						continue
					}
					for ii := range r.Items {
						if r.Items[ii].Name == change.ItemName {
							r.Items[ii].Checksum = checksum
							r.Items[ii].Status = "present"
							updated = true
							break
						}
					}
					break
				}

				if !updated {
					stateToSave.SetResourceGroup(provider.ResourceState{
						Kind:  modification.Kind,
						Group: modification.Group,
						Items: []resource.ItemState{
							{Name: change.ItemName, Version: "", Checksum: checksum, Status: "present"},
						},
					})
				}
			}
		}

		// Removals: only update state for groups that succeeded.
		for _, removal := range plan.Removals {
			if !succeededGroups[kindGroup{removal.Kind, removal.Group}] {
				continue
			}
			for _, it := range removal.Items {
				stateToSave.RemoveResourceItem(removal.Kind, removal.Group, it.Name)
			}
		}
	}

	if err := e.StateBackend.Save(ctx, stateToSave); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Collect all skipped groups across provider plans.
	var skippedGroups []provider.GroupSkip
	for _, plan := range result.ProviderPlans {
		skippedGroups = append(skippedGroups, plan.Skipped...)
	}

	// Display results
	fmt.Println()
	if len(failures) > 0 {
		successCount := len(succeededGroups)

		fmt.Println()
		if successCount == 0 {
			fmt.Println(style.Error.Render("✖ Apply failed"))
			fmt.Println()
			fmt.Printf("All %d resources failed to apply\n", len(failures))
		} else {
			fmt.Println(style.Warning.Render("⚠ Apply completed with errors"))
			fmt.Println()
            fmt.Println(style.Iconf(style.StyledIconSuccess, style.Success, "%d succeeded", successCount))
            fmt.Println(style.Iconf(style.StyledIconError, style.Error, "%d failed", len(failures)))
			if len(skippedGroups) > 0 {
                fmt.Println(style.Iconf(style.IconWarning, style.Warning, "%d skipped (dependency failed)", len(skippedGroups)))
			}
		}
		fmt.Println()
		fmt.Println(style.Bold.Render("Failed resources:"))
		fmt.Println()
		for _, f := range failures {
			fmt.Printf("  %s %s\n", style.Error.Render("•"), style.DimStyle.Render(f.Resource))
			errLines := strings.Split(f.Err.Error(), "\n")
			for _, line := range errLines {
				fmt.Printf("    %s\n", style.Error.Render(line))
			}
			fmt.Println()
		}
		if len(skippedGroups) > 0 {
			fmt.Println(style.Bold.Render("Skipped resources:"))
			fmt.Println()
			for _, s := range skippedGroups {
				fmt.Printf("  %s %s/%s — %s\n",
					style.Warning.Render("•"),
					style.DimStyle.Render(s.Kind),
					style.DimStyle.Render(s.Group),
					s.Reason,
				)
			}
			fmt.Println()
		}
		// Return simple error indicator - details already shown
		return fmt.Errorf("apply completed with %d error(s)", len(failures))
	}

    fmt.Println(style.Iconf(style.StyledIconSuccess, style.Success, "Apply complete! All resources synchronized"))
	return nil
}

// readFileMaybe attempts to read a file and returns its bytes or an error.
// It wraps os.Open + io.ReadAll to make testing and error handling clearer.
func readFileMaybe(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
