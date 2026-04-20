package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// stateCmd represents the state command
var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "State management commands",
	Long: `state provides subcommands for managing the state file:

  import  - Bring existing resource under management
  remove  - Drop from state without touching system
  list    - Show all managed resources + status
  pull    - Fetch state from remote backend
  push    - Write local state to remote backend`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("state command executed (placeholder)")
		fmt.Println("Use 'state import', 'state remove', 'state list', 'state pull', or 'state push'")
	},
}

func init() {
	rootCmd.AddCommand(stateCmd)
}
