package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/wasilak/dotisan/pkg/engine"

	"github.com/spf13/cobra"
)

var (
	planJSONFlag bool
)

// planCmd represents the plan command
var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Show what would change",
	Long: `plan loads the current state, renders all config objects, and calls Reconcile()
on each provider to show a structured diff of what would change.

Output format (default):
  + green: resource will be added
  ~ yellow: resource will be changed (shows diff)
  - red: resource will be removed
  ! orange: resource has drifted from expected state
  = dim: resource is in sync

Use --json for machine-readable output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlan()
	},
}

func runPlan() error {
	// Create engine
	eng, err := engine.NewEngine()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Run plan
	ctx := context.Background()
	result, err := eng.Plan(ctx)
	if err != nil {
		return fmt.Errorf("plan failed: %w", err)
	}

	// JSON output
	if planJSONFlag {
		return displayJSON(result)
	}

	// Display results (amazing format)
	eng.DisplayPlan(result)

	return nil
}

func displayJSON(result *engine.PlanResult) error {
	output := map[string]interface{}{
		"summary": map[string]int{
			"additions":     result.TotalAdditions,
			"modifications": result.TotalModifications,
			"removals":      result.TotalRemovals,
			"in_sync":       result.TotalInSync,
			"drifted":       result.TotalDrifted,
		},
		"has_changes": result.HasChanges,
		"resources":   []map[string]interface{}{},
	}

	// Build resources list
	resources := []map[string]interface{}{}

	for providerName, plan := range result.ProviderPlans {
		for _, res := range plan.Additions {
			resources = append(resources, map[string]interface{}{
				"action":    "add",
				"provider":  providerName,
				"kind":      res.GetKind(),
				"name":      res.GetMetadata().Name,
				"namespace": res.GetMetadata().GetNamespace(),
			})
		}
		for _, mod := range plan.Modifications {
			resources = append(resources, map[string]interface{}{
				"action":    "modify",
				"provider":  providerName,
				"kind":      mod.Resource.GetKind(),
				"name":      mod.Resource.GetMetadata().Name,
				"namespace": mod.Resource.GetMetadata().GetNamespace(),
				"diff":      mod.Diff,
			})
		}
		for _, res := range plan.Removals {
			resources = append(resources, map[string]interface{}{
				"action":    "remove",
				"provider":  providerName,
				"kind":      res.GetKind(),
				"name":      res.GetMetadata().Name,
				"namespace": res.GetMetadata().GetNamespace(),
			})
		}
		for _, drift := range plan.Drifted {
			resources = append(resources, map[string]interface{}{
				"action":      "drift",
				"provider":    providerName,
				"kind":        drift.Resource.GetKind(),
				"name":        drift.Resource.GetMetadata().Name,
				"namespace":   drift.Resource.GetMetadata().GetNamespace(),
				"description": drift.Description,
				"diff":        drift.Diff,
			})
		}
	}

	output["resources"] = resources

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func init() {
	rootCmd.AddCommand(planCmd)
	planCmd.Flags().BoolVar(&planJSONFlag, "json", false, "Output in JSON format")
}
