package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply changes (dry-run unless --confirm)",
	Long: `apply runs plan first, displays the output, then executes changes.

Without --confirm: exits after showing plan (dry-run mode)
With --confirm: executes all changes and updates state
With --auto-confirm: skips interactive confirmation (for CI use)`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("apply command executed (placeholder)")
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
}
