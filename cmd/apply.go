package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/pterm/pterm"

	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/engine"
	"github.com/wasilak/dotisan/pkg/output"
	"github.com/wasilak/dotisan/pkg/style"
	"github.com/wasilak/dotisan/pkg/ui"

	"github.com/spf13/cobra"
)

var (
	confirmFlag      bool
	applyOutputFlag  string
	applyTargetFlags []string
)

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:          "apply",
	SilenceUsage: true,
	Short:        "Apply changes",
	Long: `apply runs plan first, displays the output, then executes changes.

Without --confirm: shows plan and asks for interactive confirmation
With --confirm: executes all changes immediately without prompting`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runApply(cmd.Context())
	},
}

func runApply(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("internal: context is nil")
	}
	// Create engine
	eng, err := engine.NewEngine()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Run plan first (show spinner while planning)
	var result *engine.PlanResult
	var planErr error
	if err = style.RunWithSpinner(ctx, "Planning...", func(ctx context.Context) error {
		result, planErr = eng.Plan(ctx, engine.PlanOptions{Targets: applyTargetFlags})
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

	// Check if there are changes
	if !result.HasChanges {
		fmt.Println(style.Info.Render("No changes to apply."))
		return nil
	}

	// Determine output format (render table as pterm resource table)
	outputFormat := output.Format(applyOutputFlag)
	if outputFormat == "" {
		if eng.Config.UI.Output != "" {
			outputFormat = output.Format(eng.Config.UI.Output)
		} else {
			outputFormat = output.FormatPlain
		}
	}

	// Display plan
	switch outputFormat {
	case output.FormatTree:
		treeFormatter := diff.NewTreeFormatter()
		for providerName, plan := range result.ProviderPlans {
			if len(plan.Additions) > 0 || len(plan.Removals) > 0 || len(plan.Modifications) > 0 {
				fmt.Printf("\n%s:\n", providerName)
				if err := treeFormatter.FormatGroupPlanAsTree(diff.GroupPlanInfo{Plan: plan}); err != nil {
					fmt.Fprintf(os.Stderr, "tree render error: %v\n", err)
				}
			}
		}
	default:
		// Plain text output
		fmt.Println(style.Header.Render("Plan Summary"))
		fmt.Println()

		// Display with resource table
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
						Details:     item.Version,
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

		// Convert flatItems to []ui.ResourceRow explicitly
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

		if err := ui.RenderResourceTable(rows, true); err != nil {
			fmt.Fprintf(os.Stderr, "resource table error: %v\n", err)
		}

		fmt.Println()
		fmt.Printf("Plan: %s to add, %s to destroy\n",
			style.Success.Render(fmt.Sprintf("%d", result.TotalAdditions)),
			style.Error.Render(fmt.Sprintf("%d", result.TotalRemovals)))
	}

	// Apply with options
	opts := engine.ApplyOptions{
		Confirm: confirmFlag,
	}

	// Execute apply based on mode
	if confirmFlag {
		// Non-interactive mode
		err := style.RunWithSpinner(ctx, "Applying changes", func(ctx context.Context) error {
			return eng.Apply(ctx, result, opts)
		})
		if err != nil {
			return fmt.Errorf("apply failed: %w", err)
		}
		fmt.Println(style.IconSuccess + " Changes applied successfully.")
	} else {
		// Interactive mode: ask for confirmation
		totalChanges := result.TotalAdditions + result.TotalModifications + result.TotalRemovals + result.TotalDrifted

		var changeSummary string
		if totalChanges == 1 {
			changeSummary = "Apply 1 change?"
		} else {
			changeSummary = fmt.Sprintf("Apply %d changes?", totalChanges)
		}

		confirm, err := pterm.DefaultInteractiveConfirm.Show(changeSummary)
		if err != nil {
			return fmt.Errorf("confirmation prompt error: %w", err)
		}
		if !confirm {
			fmt.Println()
			fmt.Println(style.Info.Render("→ Apply cancelled."))
			return nil
		}
		opts.Confirm = true
		// Apply
		err = style.RunWithSpinner(ctx, "Applying changes", func(ctx context.Context) error {
			return eng.Apply(ctx, result, opts)
		})
		if err != nil {
			return fmt.Errorf("apply failed: %w", err)
		}
		fmt.Println(style.IconSuccess + " Changes applied successfully.")
	}
	return nil
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().BoolVar(&confirmFlag, "confirm", false, "Skip confirmation and apply immediately")
	applyCmd.Flags().StringVarP(&applyOutputFlag, "output", "o", "", "Output format (plain, tree, json)")
	applyCmd.Flags().StringArrayVarP(&applyTargetFlags, "target", "t", nil, "Target specific resources (format: Kind, Kind/Group, or Kind/Group/Item)")
}
