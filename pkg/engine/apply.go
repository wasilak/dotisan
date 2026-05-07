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

	// succeededItems tracks individual items that applied without error.
	// Keyed by "kind/group/itemName" so that a single failure in a group
	// does not prevent state from being saved for the other items that succeeded.
	type kindGroup struct{ kind, group string }
	succeededItems := make(map[string]bool)

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
				} else {
					succeededItems[fmt.Sprintf("%s/%s/%s", kind, group, it.Name)] = true
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
				} else {
					succeededItems[fmt.Sprintf("%s/%s/%s", kind, group, c.ItemName)] = true
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
				} else {
					succeededItems[fmt.Sprintf("%s/%s/%s", kind, group, it.Name)] = true
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

		if !applyGroup(provName, kind, group) {
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
			applyGroup(provName, kg.kind, kg.group)
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

		// Additions: persist state for each item that succeeded individually.
		for _, addition := range plan.Additions {
			var items []resource.ItemState
			for _, item := range addition.Items {
				if !succeededItems[fmt.Sprintf("%s/%s/%s", addition.Kind, addition.Group, item.Name)] {
					continue
				}
				checksum := ""
				if dest := func() string {
				if item.FileExtra != nil {
					return item.FileExtra.Destination
				}
				return ""
			}(); dest != "" {
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
			if len(items) > 0 {
				stateToSave.SetResourceGroup(provider.ResourceState{
					Kind:  addition.Kind,
					Group: addition.Group,
					Items: items,
				})
			}
		}

		// Modifications: persist state for each item that succeeded individually.
		for _, modification := range plan.Modifications {
			for _, change := range modification.Changes {
				if !succeededItems[fmt.Sprintf("%s/%s/%s", modification.Kind, modification.Group, change.ItemName)] {
					continue
				}
				checksum := ""
				if dest := func() string {
					if change.NewState.FileExtra != nil {
						return change.NewState.FileExtra.Destination
					}
					return ""
				}(); dest != "" {
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

		// Removals: persist state removal for each item that succeeded individually.
		for _, removal := range plan.Removals {
			for _, it := range removal.Items {
				if !succeededItems[fmt.Sprintf("%s/%s/%s", removal.Kind, removal.Group, it.Name)] {
					continue
				}
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
		successCount := len(succeededItems)

		// If caller provided OnMessage, publish a concise summary message so
		// the UI can show it via spinner; otherwise fall back to the existing
		// multi-line output behavior.
		if opts.OnMessage != nil {
			if successCount == 0 {
				opts.OnMessage(MessageLevelError, fmt.Sprintf("Apply failed: %d failed", len(failures)))
			} else {
				msg := fmt.Sprintf("Apply completed with %d succeeded, %d failed", successCount, len(failures))
				if len(skippedGroups) > 0 {
					msg = fmt.Sprintf("%s, %d skipped", msg, len(skippedGroups))
				}
				opts.OnMessage(MessageLevelWarning, msg)
			}
		} else {
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
		}
		// Return simple error indicator - details already shown
		return fmt.Errorf("apply completed with %d error(s)", len(failures))
	}

	// Success path: notify via OnMessage when available for spinner UI.
	if opts.OnMessage != nil {
		// Use OnMessage to surface success to the spinner UI. The spinner
		// run loop will avoid printing duplicate success messages when the
		// last transient message is already a success-level summary.
		opts.OnMessage(MessageLevelSuccess, "Apply complete! All resources synchronized")
		return nil
	}

	// If no OnMessage was provided, print a concise success line once.
	fmt.Println(style.Iconf(style.StyledIconSuccess, style.Success, "Changes applied successfully."))
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
