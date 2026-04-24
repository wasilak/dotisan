package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/engine"
	"github.com/wasilak/dotisan/pkg/output"
	"github.com/wasilak/dotisan/pkg/style"

	"github.com/spf13/cobra"
)

var planOutputFlag string

// planCmd represents the plan command
var planCmd = &cobra.Command{
	Use:          "plan",
	SilenceUsage: true,
	Short:        "Show what would change",
	Long: `plan loads the current state, renders all config objects, and calls Reconcile()
on each provider to show a structured diff of what would change.

Output formats:
  plain (default): table view with symbols and colors
  tree:            3-level tree view (Kind / Group / Items)
  json:            machine-readable JSON output`,
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

	// Determine output format
	outputFormat := output.Format(planOutputFlag)
	if outputFormat == "" {
		if eng.Config.UI.Output != "" {
			outputFormat = output.Format(eng.Config.UI.Output)
		} else {
			outputFormat = output.FormatPlain
		}
	}

	// Run plan
	ctx := context.Background()
	result, err := eng.Plan(ctx)
	if err != nil {
		return fmt.Errorf("plan failed: %w", err)
	}

	// Display results based on output format
	switch outputFormat {
	case output.FormatJSON:
		return displayPlanJSON(result)
	case output.FormatTree:
		treeFormatter := diff.NewTreeFormatter()
		for providerName, plan := range result.ProviderPlans {
			if len(plan.Additions) > 0 || len(plan.Removals) > 0 || len(plan.Modifications) > 0 {
				fmt.Printf("\n%s:\n", providerName)
				fmt.Println(treeFormatter.FormatGroupPlanAsTree(diff.GroupPlanInfo{Plan: plan}))
			}
		}
	default:
		// Plain text output
		if !result.HasChanges {
			fmt.Println(style.Info.Render("No changes to apply."))
			return nil
		}

		fmt.Println(style.Header.Render("Plan Summary"))
		fmt.Println()

		for providerName, plan := range result.ProviderPlans {
			// Show all resources, not just changes
			fmt.Printf("%s:\n", providerName)

			for _, addition := range plan.Additions {
				for _, item := range addition.Items {
					fmt.Printf("  %s %s/%s: %s\n", style.IconAdd, addition.Kind, addition.Group, item.Name)
				}
			}

			for _, removal := range plan.Removals {
				for _, item := range removal.Items {
					fmt.Printf("  %s %s/%s: %s\n", style.IconRemove, removal.Kind, removal.Group, item.Name)
				}
			}

			for _, mod := range plan.Modifications {
				for _, change := range mod.Changes {
					fmt.Printf("  %s %s/%s: %s\n", style.IconEdit, mod.Kind, mod.Group, change.ItemName)
				}
			}

			for _, inSync := range plan.InSync {
				for _, item := range inSync.Items {
					fmt.Printf("  %s %s/%s: %s\n", style.IconOK, inSync.Kind, inSync.Group, item.Name)
				}
			}
		}

		fmt.Println()
		fmt.Printf("Plan: %s to add, %s to destroy\n",
			style.Success.Render(fmt.Sprintf("%d", result.TotalAdditions)),
			style.Error.Render(fmt.Sprintf("%d", result.TotalRemovals)))
	}

	return nil
}

func displayPlanJSON(result *engine.PlanResult) error {
	output := map[string]interface{}{
		"summary": map[string]int{
			"additions":     result.TotalAdditions,
			"modifications": result.TotalModifications,
			"removals":      result.TotalRemovals,
			"in_sync":       result.TotalInSync,
			"drifted":       result.TotalDrifted,
		},
		"has_changes": result.HasChanges,
		"providers":   map[string]interface{}{},
	}

	providers := make(map[string]interface{})
	for name, plan := range result.ProviderPlans {
		providers[name] = map[string]interface{}{
			"additions":     plan.Additions,
			"modifications": plan.Modifications,
			"removals":      plan.Removals,
			"in_sync":       plan.InSync,
			"drifted":       plan.Drifted,
		}
	}
	output["providers"] = providers

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func init() {
	rootCmd.AddCommand(planCmd)
	planCmd.Flags().StringVarP(&planOutputFlag, "output", "o", "", "Output format (plain, tree, json)")
}
