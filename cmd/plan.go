package cmd

import (
	"fmt"

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
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("plan command executed (placeholder)")
	},
}

func init() {
	rootCmd.AddCommand(planCmd)
}
