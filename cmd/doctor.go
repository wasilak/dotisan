package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system prerequisites",
	Long: `doctor checks each provider's Available() status, state backend connectivity,
config file validity, and template rendering. Reports issues and suggests fixes.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("doctor command executed (placeholder)")
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
