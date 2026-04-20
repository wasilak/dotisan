package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ejectCmd represents the eject command
var ejectCmd = &cobra.Command{
	Use:   "eject",
	Short: "Stop managing a resource",
	Long: `eject removes a resource from state AND removes it from config files.
The system is left untouched — opposite of import.

Usage: dotisan eject KIND NAME`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("eject command executed (placeholder)")
	},
}

func init() {
	rootCmd.AddCommand(ejectCmd)
}
