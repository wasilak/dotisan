package cmd

import (
	"charm.land/huh/v2"
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"

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
		return runApply()
	},
}

func runApply() error {
	// Create engine
	eng, err := engine.NewEngine()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Run plan first
	ctx := context.Background()
	result, err := eng.Plan(ctx, engine.PlanOptions{Targets: applyTargetFlags})
	if err != nil {
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

	// Determine output format
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
				fmt.Println(treeFormatter.FormatGroupPlanAsTree(diff.GroupPlanInfo{Plan: plan}))
			}
		}
	default:
		// Plain text output
		fmt.Println(style.Header.Render("Plan Summary"))
		fmt.Println()

		// Display with Bubbletea Table (Apply Output):
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

		humanPlan := struct{ Items []PlanItem }{Items: flatItems}
		width, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			width = 120
		}
		// Columns: Status, ID, Kind, Group, Name, Info
		table := ui.NewTable([]ui.Column{
			{Title: "Status", Width: 3, Align: lipgloss.Center},
			{Title: "ID", Flex: true},
			{Title: "Kind", Width: 20},
			{Title: "Group", Width: 20},
			{Title: "Name", Width: 20},
			{Title: "Info", Flex: true},
		}, true)
		rows := ui.PlanToRows(&humanPlan)
		table.SetRows(rows)
		fmt.Println(table.RenderPlain(width))

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
		err := style.WithSpinner("Applying changes", func(stop style.StopFunc) error {
			applyErr := eng.Apply(ctx, result, opts)
			if applyErr != nil {
				stop("failed")
			} else {
				stop("done")
			}
			return applyErr
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

		// Fallback to basic prompt if not a TTY
		isTTY := term.IsTerminal(int(os.Stdout.Fd()))
		var confirm bool
		if isTTY {
			err := huh.NewConfirm().
				Title(changeSummary).
				Affirmative("Yes, apply changes").
				Negative("No, cancel").
				Value(&confirm).
				Run()
			if err != nil {
				return fmt.Errorf("confirmation prompt error: %w", err)
			}
		} else {
			fmt.Printf("%s [y/N]: ", changeSummary)
			var resp string
			_, err := fmt.Fscanln(os.Stdin, &resp)
			if err != nil && err.Error() != "unexpected newline" {
				return fmt.Errorf("failed to read confirmation: %w", err)
			}
			resp = strings.TrimSpace(strings.ToLower(resp))
			confirm = (resp == "y" || resp == "yes")
		}
		if !confirm {
			fmt.Println()
			fmt.Println(style.Info.Render("→ Apply cancelled."))
			return nil
		}
		opts.Confirm = true
		// Apply
		err := style.WithSpinner("Applying changes", func(stop style.StopFunc) error {
			applyErr := eng.Apply(ctx, result, opts)
			if applyErr != nil {
				stop("failed")
			} else {
				stop("done")
			}
			return applyErr
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
