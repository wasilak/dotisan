package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/engine"
	"github.com/wasilak/dotisan/pkg/output"
	"github.com/wasilak/dotisan/pkg/style"

	"github.com/spf13/cobra"
)

var (
	confirmFlag     bool
	applyOutputFlag string
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
	result, err := eng.Plan(ctx)
	if err != nil {
		return fmt.Errorf("plan failed: %w", err)
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
		if err := eng.Apply(ctx, result, opts); err != nil {
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

		box := style.InfoBox.Render(
			style.Bold.Render(changeSummary) + "\n\n" +
				fmt.Sprintf("%s %s\n", style.Info.Render("[Y]"), style.Dim.Render("Yes, apply changes")) +
				fmt.Sprintf("%s %s", style.Info.Render("[N]"), style.Dim.Render("No, cancel")),
		)

		fmt.Println()
		fmt.Print(box)
		fmt.Print("\n: ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if !strings.HasPrefix(response, "y") {
			fmt.Println()
			fmt.Println(style.Info.Render("→ Apply cancelled."))
			return nil
		}

		opts.Confirm = true

		// Apply with progress
		if err := eng.ApplyWithProgress(ctx, result, opts); err != nil {
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
}
