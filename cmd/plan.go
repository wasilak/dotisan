package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/wasilak/dotisan/pkg/engine"

	"github.com/spf13/cobra"
)

// planCmd represents the plan command
var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Show what would change",
	Long: `plan loads the current state, renders all config objects, and calls Reconcile()
on each provider to show a structured diff of what would change.

Output format:
  + green: resource will be added
  ~ yellow: resource will be changed (shows diff)
  - red: resource will be removed
  = dim: resource is in sync`,
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

	// Display results
	eng.DisplayPlan(result)

	// Exit with error code if there are drifted resources
	if result.TotalDrifted > 0 {
		fmt.Fprintln(os.Stderr, "\nWarning: Some resources have drifted from their expected state.")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(planCmd)
}
