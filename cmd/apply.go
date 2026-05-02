package cmd

import (
	"context"
	"github.com/wasilak/dotisan/pkg/output"
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
	outputFormat := output.Format(applyOutputFlag)
	return runPlanApply(ctx, PlanApplyOptions{
		IsApply:      true,
		Confirm:      confirmFlag,
		OutputFormat: string(outputFormat),

		Targets:      applyTargetFlags,
	})
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().BoolVar(&confirmFlag, "confirm", false, "Skip confirmation and apply immediately")
	applyCmd.Flags().StringVarP(&applyOutputFlag, "output", "o", "", "Output format (plain, tree, json)")
	applyCmd.Flags().StringArrayVarP(&applyTargetFlags, "target", "t", nil, "Target specific resources (format: Kind, Kind/Group, or Kind/Group/Item)")
}
