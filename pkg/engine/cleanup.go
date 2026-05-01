// Package engine: Cleanup logic extracted from engine.go and apply.go
package engine

import (
	"context"
	"fmt"
)

// processCleanup removes cleanup items from state (state-only operation, always succeeds)
func (e *Engine) processCleanup(ctx context.Context, result *PlanResult) error {
	if result == nil || result.ProviderPlans == nil {
		return nil // nothing to cleanup
	}

	stateChanged := false
	st := result.CurrentState
	for _, plan := range result.ProviderPlans {
		for _, cleanup := range plan.Cleanup {
			for resIdx, res := range st.Resources {
				if res.Kind == cleanup.Kind && res.Group == cleanup.Group {
					newItems := res.Items[:0]
					for _, item := range res.Items {
						shouldRemove := false
						for _, cleanupItem := range cleanup.Items {
							if item.Name == cleanupItem.Name {
								shouldRemove = true
								break
							}
						}
						if !shouldRemove {
							newItems = append(newItems, item)
						} else {
							stateChanged = true
						}
					}
					st.Resources[resIdx].Items = newItems
				}
			}
		}
	}
	if stateChanged {
		if err := e.StateBackend.Save(ctx, st); err != nil {
			return fmt.Errorf("failed to save state after cleanup: %w", err)
		}
	}
	return nil
}
