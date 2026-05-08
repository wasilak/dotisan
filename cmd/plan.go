package cmd

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/wasilak/nim/pkg/output"
)

var planOutputFlag string
var planTargetFlags []string
var planDiffFlag bool

// planCmd represents the plan command
var planCmd = &cobra.Command{
	Use:          "plan",
	SilenceUsage: true,
	Short:        "Show what would change",
	Long:         "Compute and display the changes that would be made without applying them. Supports plain, tree, and json outputs.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlan(cmd.Context())
	},
}

func init() {
	planCmd.Flags().StringVarP(&planOutputFlag, "output", "o", "", "Output format (plain, tree, json)")
	planCmd.Flags().StringArrayVarP(&planTargetFlags, "target", "t", nil, "Target specific resources (format: Kind, Kind/Group, or Kind/Group[Item])")
	planCmd.Flags().BoolVarP(&planDiffFlag, "diff", "d", false, "Show contextual diffs for file/package changes (unified view)")
}

func runPlan(ctx context.Context) error {
	outputFormat := output.Format(planOutputFlag)
	return runPlanApply(ctx, PlanApplyOptions{
		IsApply:      false,
		Confirm:      false,
		OutputFormat: string(outputFormat),
		Targets:      planTargetFlags,
		ShowDiff:     planDiffFlag,
	})
}
