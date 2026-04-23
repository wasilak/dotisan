package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/engine"
	"github.com/wasilak/dotisan/pkg/style"

	"github.com/spf13/cobra"
)

var (
	confirmFlag bool
	applyTreeFlag bool
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
	result, err := eng.Plan(ctx, nil)
	if err != nil {
		return fmt.Errorf("plan failed: %w", err)
	}

	// Check if there are changes
	if !result.HasChanges {
		fmt.Println(style.Info.Render("No changes to apply."))
		return nil
	}

	// Apply with options
	opts := engine.ApplyOptions{
		Confirm: confirmFlag,
	}

	// Determine if we should use tree view (flag OR config)
	useTree := applyTreeFlag || eng.Config.UI.Tree

	// Execute apply based on mode
	if confirmFlag {
		// Non-interactive mode: display plan then apply
		if useTree {
			treeFormatter := diff.NewTreeFormatter()
			planInfo := diff.PlanResultInfo{
				ProviderPlans:      result.ProviderPlans,
				TotalAdditions:     result.TotalAdditions,
				TotalModifications: result.TotalModifications,
				TotalRemovals:      result.TotalRemovals,
				TotalDrifted:       result.TotalDrifted,
			}
			fmt.Println(treeFormatter.FormatPlanAsTree(planInfo))
			fmt.Println()
			fmt.Println(eng.PlanFormatter.FormatSummary(result.TotalAdditions, result.TotalModifications, result.TotalRemovals, result.TotalInSync))
		} else {
			eng.DisplayPlan(result)
		}
		if err := eng.Apply(ctx, result, opts); err != nil {
			return fmt.Errorf("apply failed: %w", err)
		}
	} else {
		// Interactive mode: display plan and ask for confirmation
		if useTree {
			treeFormatter := diff.NewTreeFormatter()
			planInfo := diff.PlanResultInfo{
				ProviderPlans:      result.ProviderPlans,
				TotalAdditions:     result.TotalAdditions,
				TotalModifications: result.TotalModifications,
				TotalRemovals:      result.TotalRemovals,
				TotalDrifted:       result.TotalDrifted,
			}
			fmt.Println(treeFormatter.FormatPlanAsTree(planInfo))
			fmt.Println()
			fmt.Println(eng.PlanFormatter.FormatSummary(result.TotalAdditions, result.TotalModifications, result.TotalRemovals, result.TotalInSync))
		} else {
			eng.DisplayPlan(result)
		}
		if !askForConfirmation(result) {
			fmt.Println()
			fmt.Println(style.Info.Render("→ Apply cancelled."))
			return nil
		}
		// Use apply with progress bar
		if err := eng.ApplyWithProgress(ctx, result, opts); err != nil {
			return fmt.Errorf("apply failed: %w", err)
		}
	}

	return nil
}

// askForConfirmation prompts the user to confirm the apply operation
func askForConfirmation(result *engine.PlanResult) bool {
	// Calculate total changes
	totalChanges := result.TotalAdditions + result.TotalModifications + result.TotalRemovals + result.TotalDrifted

	// Build confirmation message
	var changeSummary string
	if totalChanges == 1 {
		changeSummary = "Apply 1 change?"
	} else {
		changeSummary = fmt.Sprintf("Apply %d changes?", totalChanges)
	}

	// Styled confirmation box
	box := style.InfoBox.Render(
		style.Bold.Render(changeSummary) + "\n\n" +
			fmt.Sprintf("%s %s\n", style.Info.Render("[Y]"), style.Dim.Render("Yes, apply changes")) +
			fmt.Sprintf("%s %s", style.Info.Render("[N]"), style.Dim.Render("No, cancel")),
	)

	fmt.Println()
	fmt.Print(box)
	fmt.Print("\n: ")

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func init() {
	rootCmd.AddCommand(applyCmd)

	applyCmd.Flags().BoolVar(&confirmFlag, "confirm", false, "Skip confirmation and apply immediately")
	applyCmd.Flags().BoolVar(&applyTreeFlag, "tree", false, "Render plan output as tree structure")
}
