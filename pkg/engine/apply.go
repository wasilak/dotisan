package engine

import (
	"context"
	"fmt"
	"strings"

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

	// STEP 2: Execute provider changes
	type failureEntry struct {
		Resource string
		Err      error
	}
	var failures []failureEntry
	providerSucceeded := make(map[string]bool)

	for providerName, plan := range result.ProviderPlans {
		prov, exists := e.Providers[providerName]
		if !exists {
			dummyErr := fmt.Errorf("provider %s not found", providerName)
			for _, a := range plan.Additions {
				for _, it := range a.Items {
					failures = append(failures, failureEntry{Resource: fmt.Sprintf("%s/%s/%s", a.Kind, a.Group, it.Name), Err: dummyErr})
				}
			}
			for _, m := range plan.Modifications {
				for _, c := range m.Changes {
					failures = append(failures, failureEntry{Resource: fmt.Sprintf("%s/%s/%s", m.Kind, m.Group, c.ItemName), Err: dummyErr})
				}
			}
			for _, r := range plan.Removals {
				for _, it := range r.Items {
					failures = append(failures, failureEntry{Resource: fmt.Sprintf("%s/%s/%s", r.Kind, r.Group, it.Name), Err: dummyErr})
				}
			}
			providerSucceeded[providerName] = false
			continue
		}

		// Create a plan without Cleanup items for the provider (Cleanup = state-only, already handled)
		planForProvider := provider.GroupPlan{
			Additions:     plan.Additions,
			Modifications: plan.Modifications,
			Removals:      plan.Removals,
			InSync:        plan.InSync,
			Drifted:       plan.Drifted,
			Warnings:      plan.Warnings,
		}

		if err := prov.Apply(ctx, planForProvider); err != nil {
			for _, a := range plan.Additions {
				for _, it := range a.Items {
					failures = append(failures, failureEntry{Resource: fmt.Sprintf("%s/%s/%s", a.Kind, a.Group, it.Name), Err: fmt.Errorf("failed to add: %w", err)})
				}
			}
			for _, m := range plan.Modifications {
				for _, c := range m.Changes {
					failures = append(failures, failureEntry{Resource: fmt.Sprintf("%s/%s/%s", m.Kind, m.Group, c.ItemName), Err: fmt.Errorf("failed to modify: %w", err)})
				}
			}
			for _, r := range plan.Removals {
				for _, it := range r.Items {
					failures = append(failures, failureEntry{Resource: fmt.Sprintf("%s/%s/%s", r.Kind, r.Group, it.Name), Err: fmt.Errorf("failed to remove: %w", err)})
				}
			}
			providerSucceeded[providerName] = false
		} else {
			providerSucceeded[providerName] = true
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

	for providerName, plan := range result.ProviderPlans {
		if !providerSucceeded[providerName] {
			continue
		}

		for _, inSync := range plan.InSync {
			stateToSave.SetResourceGroup(provider.ResourceState{
				Kind:      inSync.Kind,
				Group:     inSync.Group,
				Items:     inSync.Items,
				Namespace: "default",
			})
		}
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

	// Display results
	fmt.Println()
	if len(failures) > 0 {
		// Calculate success count
		successCount := 0
		for providerName := range result.ProviderPlans {
			if providerSucceeded[providerName] {
				successCount++
			}
		}

		fmt.Println()
		if successCount == 0 {
			fmt.Println(style.Error.Render("✖ Apply failed"))
			fmt.Println()
			fmt.Printf("All %d resources failed to apply\n", len(failures))
		} else {
			fmt.Println(style.Warning.Render("⚠ Apply completed with errors"))
			fmt.Println()
			fmt.Printf("%s %d succeeded\n", style.IconSuccess, successCount)
			fmt.Printf("%s %d failed\n", style.IconError, len(failures))
		}
		fmt.Println()
		fmt.Println(style.Bold.Render("Failed resources:"))
		fmt.Println()
		for _, f := range failures {
			fmt.Printf("  %s %s\n", style.Error.Render("•"), style.Dim.Render(f.Resource))
			errLines := strings.Split(f.Err.Error(), "\n")
			for _, line := range errLines {
				fmt.Printf("    %s\n", style.Error.Render(line))
			}
			fmt.Println()
		}
		// Return simple error indicator - details already shown
		return fmt.Errorf("apply completed with %d error(s)", len(failures))
	}

	fmt.Printf("%s Apply complete! All resources synchronized\n", style.IconSuccess)
	return nil
}
