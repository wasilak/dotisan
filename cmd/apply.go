package cmd

import (
	"charm.land/huh/v2"
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/engine"
	"github.com/wasilak/dotisan/pkg/output"
	"github.com/wasilak/dotisan/pkg/style"

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

		DisplayPlanList(result.ProviderPlans, true)

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
