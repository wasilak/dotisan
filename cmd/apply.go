package cmd

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/wasilak/nim/pkg/config"
	"github.com/wasilak/nim/pkg/output"
)

var (
	confirmFlag         bool
	applyOutputFlag     string
	applyTargetFlags    []string
	applyDiffFlag       bool
	applyNamespaceFlag  string
)

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:          "apply",
	SilenceUsage: true,
	Short:        "Apply changes",
	Long:         "Run plan and optionally apply changes to synchronize resources with the desired state.",
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
		ShowDiff:     applyDiffFlag,
		Namespace:    config.GetActiveNamespace(applyNamespaceFlag),
	})
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().BoolVar(&confirmFlag, "confirm", false, "Skip confirmation and apply immediately")
	applyCmd.Flags().StringVarP(&applyOutputFlag, "output", "o", "", "Output format (plain, tree, json)")
	applyCmd.Flags().StringArrayVarP(&applyTargetFlags, "target", "t", nil, "Target specific resources (format: Kind, Kind/Group, or Kind/Group/Item)")
	applyCmd.Flags().BoolVarP(&applyDiffFlag, "diff", "d", false, "Show contextual diffs for file/package changes (unified view)")
	applyCmd.Flags().StringVarP(&applyNamespaceFlag, "namespace", "n", "", "Active namespace for this invocation (overrides NIM_NAMESPACE env var)")
}
