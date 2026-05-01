package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"

	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/engine"
	"github.com/wasilak/dotisan/pkg/output"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
	"github.com/wasilak/dotisan/pkg/style"

	"github.com/spf13/cobra"
)

var planOutputFlag string
var planTargetFlags []string

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
	result, err := eng.Plan(ctx, engine.PlanOptions{Targets: planTargetFlags})
	if err != nil {
		return fmt.Errorf("plan failed: %w", err)
	}

	// Print warnings for unmatched targets
	if len(result.UnmatchedTargets) > 0 {
		for _, t := range result.UnmatchedTargets {
			fmt.Fprintf(os.Stderr, "%s target %q did not match any resources\n", style.Warning.Render("Warning:"), t)
		}
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

		// Only display changes (skipEmpty=true ensures ✓ in-sync not shown)
			DisplayPlanList(result.ProviderPlans, true)

		fmt.Println()
		planParts := []string{
			fmt.Sprintf("%s to add", style.Success.Render(fmt.Sprintf("%d", result.TotalAdditions))),
			fmt.Sprintf("%s to destroy", style.Error.Render(fmt.Sprintf("%d", result.TotalRemovals))),
		}
		if result.TotalCleanup > 0 {
			planParts = append(planParts, fmt.Sprintf("%s cleanup (will be removed from state)", style.Dim.Render(fmt.Sprintf("%d", result.TotalCleanup))))
		}
		fmt.Printf("Plan: %s\n", strings.Join(planParts, ", "))
	}

	return nil
}

func displayPlanJSON(result *engine.PlanResult) error {
	output := map[string]interface{}{
		"summary": map[string]int{
			"additions":     result.TotalAdditions,
			"modifications": result.TotalModifications,
			"removals":      result.TotalRemovals,
			"cleanup":       result.TotalCleanup,
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
			"cleanup":       plan.Cleanup,
			"in_sync":       plan.InSync,
			"drifted":       plan.Drifted,
		}
	}
	output["providers"] = providers

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// Note: resources use the full/display kind names (e.g. HomebrewPackages).

func DisplayPlanList(plans map[string]provider.GroupPlan, skipEmpty bool) {
	emptyEnumerator := func(l list.Items, i int) string {
		return ""
	}

	for providerName, plan := range plans {
		// Group items by group name and action type
		groups := make(map[string]map[string][]resource.ResourceItem)
		for _, add := range plan.Additions {
			if groups[add.Group] == nil {
				groups[add.Group] = make(map[string][]resource.ResourceItem)
			}
			groups[add.Group]["add"] = append(groups[add.Group]["add"], add.Items...)
		}
		for _, rem := range plan.Removals {
			if groups[rem.Group] == nil {
				groups[rem.Group] = make(map[string][]resource.ResourceItem)
			}
			groups[rem.Group]["remove"] = append(groups[rem.Group]["remove"], rem.Items...)
		}
		for _, cleanup := range plan.Cleanup {
			if groups[cleanup.Group] == nil {
				groups[cleanup.Group] = make(map[string][]resource.ResourceItem)
			}
			groups[cleanup.Group]["cleanup"] = append(groups[cleanup.Group]["cleanup"], cleanup.Items...)
		}
		for _, mod := range plan.Modifications {
			if groups[mod.Group] == nil {
				groups[mod.Group] = make(map[string][]resource.ResourceItem)
			}
			for _, change := range mod.Changes {
				groups[mod.Group]["modify"] = append(groups[mod.Group]["modify"], resource.ResourceItem{
					Name:    change.ItemName,
					Version: change.NewState.Version,
				})
			}
		}
		for _, drift := range plan.Drifted {
			if groups[drift.Group] == nil {
				groups[drift.Group] = make(map[string][]resource.ResourceItem)
			}
			groups[drift.Group]["drift"] = append(groups[drift.Group]["drift"], resource.ResourceItem{
				Name: drift.Item,
			})
		}

		// In plan output, do NOT include InSync items — only changes matter; see Terraform-like behavior.

		// Build nested list for each group
		var groupListArgs []any
		for groupName, actions := range groups {
			var itemStrings []string

			// Additions
			for _, item := range actions["add"] {
				itemStrings = append(itemStrings, "     "+style.ItemSuccess.Render(style.StyledIconAdd+" "+item.Name))
			}
			// Removals
			for _, item := range actions["remove"] {
				itemStrings = append(itemStrings, "     "+style.ItemError.Render(style.IconError+" "+item.Name))
			}
			// Cleanup
			for _, item := range actions["cleanup"] {
				itemStrings = append(itemStrings, "     "+style.Dim.Render(style.IconTrashBin+" "+item.Name+" (cleanup — will be removed from state)"))
			}
			// Modifications
			for _, item := range actions["modify"] {
				itemStrings = append(itemStrings, "     "+style.ItemWarning.Render(style.IconWarning+" "+item.Name))
			}
			// In-sync
			for _, item := range actions["sync"] {
				itemStrings = append(itemStrings, "     "+style.ItemSuccess.Render(style.IconSuccess+" "+item.Name))
			}
			// Drift
			for _, item := range actions["drift"] {
				itemStrings = append(itemStrings, "     "+style.ItemWarning.Render(style.IconWarning+" "+item.Name))
			}

			if len(itemStrings) == 0 {
				continue
			}

			// If skipping empty, verify there are actual changes (not just InSync)
			if skipEmpty {
				hasChanges := len(actions["add"]) > 0 || len(actions["remove"]) > 0 ||
					len(actions["cleanup"]) > 0 || len(actions["modify"]) > 0 || len(actions["drift"]) > 0
				if !hasChanges {
					continue
				}
			}

			// Create child list for this group
			childListArgs := make([]any, len(itemStrings)*2)
			for i, s := range itemStrings {
				childListArgs[i*2] = s
			}
			childList := list.New(childListArgs...).
				Enumerator(emptyEnumerator)

			groupListArgs = append(groupListArgs, " "+groupName, childList)
		}

		if len(groupListArgs) == 0 {
			continue
		}

		// Determine full kind name from first non-empty group. Resources
		// already use the full/display kind names (e.g. "HomebrewPackages"),
		// so use the kind directly.
		fullKind := providerName
		if len(plan.Additions) > 0 {
			fullKind = plan.Additions[0].Kind
		} else if len(plan.Removals) > 0 {
			fullKind = plan.Removals[0].Kind
		} else if len(plan.Cleanup) > 0 {
			fullKind = plan.Cleanup[0].Kind
		} else if len(plan.Modifications) > 0 {
			fullKind = plan.Modifications[0].Kind
		} else if len(plan.InSync) > 0 {
			fullKind = plan.InSync[0].Kind
		}

		// Build parent list with all groups
		providerList := list.New(groupListArgs...).
			Enumerator(emptyEnumerator)

		// Print top-level kind separately, then groups list
		lipgloss.Println(fullKind)
		lipgloss.Println(providerList)
	}
}

func init() {
	rootCmd.AddCommand(planCmd)
	planCmd.Flags().StringVarP(&planOutputFlag, "output", "o", "", "Output format (plain, tree, json)")
	planCmd.Flags().StringArrayVarP(&planTargetFlags, "target", "t", nil, "Target specific resources (format: Kind, Kind/Group, or Kind/Group/Item)")
}
