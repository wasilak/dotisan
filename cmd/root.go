package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dotisan",
	Short: "Declarative dotfiles management CLI",
	Long: `dotisan is a declarative dotfiles management CLI tool written in Go.

It treats a local developer environment like Terraform treats cloud infrastructure:
declare desired state in version-controlled config files, compute a diff against
current state, and apply changes — including removals.

Unlike chezmoi which applies changes forward but never cleans up, dotisan tracks
managed resources explicitly and handles removals as first-class operations.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to dotisan!")
		fmt.Println("Run 'dotisan --help' to see available commands.")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
}
