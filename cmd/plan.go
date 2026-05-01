package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	_ "charm.land/lipgloss/v2"
	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/engine"
	"github.com/wasilak/dotisan/pkg/output"
	"github.com/wasilak/dotisan/pkg/style"
	"github.com/wasilak/dotisan/pkg/ui"
	"golang.org/x/term"

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
		return runPlan(cmd.Context())
	},
}

func runPlan(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("internal: context is nil")
	}
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

	// Run plan (show spinner while planning)
	var result *engine.PlanResult
	var planErr error
	if err = style.RunWithSpinner(ctx, "Planning...", func(ctx context.Context) error {
		result, planErr = eng.Plan(ctx, engine.PlanOptions{Targets: planTargetFlags})
		return planErr
	}); err != nil {
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

		// Display using Bubbletea Table with new theme (shared UI)
		type PlanItem struct {
			Action      string
			Name        string
			Kind        string
			Region      string
			Explanation string
			Details     string
		}

		type Plan struct{ Items []PlanItem }

		var flatItems []PlanItem
		for _, groupPlan := range result.ProviderPlans {
			for _, add := range groupPlan.Additions {
				for _, item := range add.Items {
					flatItems = append(flatItems, PlanItem{
						Action:      "add",
						Name:        item.Name,
						Kind:        add.Kind,
						Region:      add.Group,
						Details:     item.Version, // version detail if wanted
						Explanation: "",
					})
				}
			}
			for _, rem := range groupPlan.Removals {
				for _, item := range rem.Items {
					flatItems = append(flatItems, PlanItem{
						Action:      "remove",
						Name:        item.Name,
						Kind:        rem.Kind,
						Region:      rem.Group,
						Details:     item.Version,
						Explanation: "",
					})
				}
			}
			for _, cl := range groupPlan.Cleanup {
				for _, item := range cl.Items {
					flatItems = append(flatItems, PlanItem{
						Action:      "cleanup",
						Name:        item.Name,
						Kind:        cl.Kind,
						Region:      cl.Group,
						Details:     item.Version,
						Explanation: "will be removed from state",
					})
				}
			}
			for _, mod := range groupPlan.Modifications {
				for _, ch := range mod.Changes {
					flatItems = append(flatItems, PlanItem{
						Action:      "update",
						Name:        ch.ItemName,
						Kind:        mod.Kind,
						Region:      mod.Group,
						Details:     ch.NewState.Version,
						Explanation: "",
					})
				}
			}
			for _, drift := range groupPlan.Drifted {
				flatItems = append(flatItems, PlanItem{
					Action:      "drift",
					Name:        drift.Item,
					Kind:        "",
					Region:      "",
					Details:     "",
					Explanation: "actual vs expected drift",
				})
			}
		}

		// Convert flatItems (local typed PlanItem) to []ui.ResourceRow explicitly
		rows := make([]ui.ResourceRow, 0, len(flatItems))
		for _, it := range flatItems {
			var id string
			if it.Kind != "" && it.Region != "" && it.Name != "" {
				id = fmt.Sprintf("%s/%s[%s]", it.Kind, it.Region, it.Name)
			} else if it.Kind != "" && it.Region != "" {
				id = fmt.Sprintf("%s/%s", it.Kind, it.Region)
			} else {
				id = it.Name
			}
			info := it.Explanation
			if info == "" {
				info = it.Details
			}
			rows = append(rows, ui.ResourceRow{
				Status: it.Action,
				ID:     id,
				Kind:   it.Kind,
				Group:  it.Region,
				Name:   it.Name,
				Info:   info,
			})
		}

		// Render table
		width, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			width = 120
		}
		fmt.Println(ui.RenderResourceTable(width, rows, true))

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
