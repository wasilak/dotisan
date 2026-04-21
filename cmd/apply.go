package cmd

import (
	"context"
	"fmt"

	"github.com/wasilak/dotisan/pkg/engine"

	"github.com/spf13/cobra"
)

var (
	confirmFlag     bool
	autoConfirmFlag bool
	backupFlag      bool
)

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:          "apply",
	SilenceUsage:  true,
	Short:        "Apply changes (dry-run unless --confirm)",
	Long: `apply runs plan first, displays the output, then executes changes.

Without --confirm: exits after showing plan (dry-run mode)
With --confirm: executes all changes and updates state
With --auto-confirm: skips interactive confirmation (for CI use)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runApply()
	},
}

func runApply() error {
	// Create engine
	eng, err := engine.NewEngine()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Run plan first
	ctx := context.Background()
	result, err := eng.Plan(ctx)
	if err != nil {
		return fmt.Errorf("plan failed: %w", err)
	}

	// Apply with options
	opts := engine.ApplyOptions{
		Confirm:     confirmFlag,
		AutoConfirm: autoConfirmFlag,
		Backup:      backupFlag,
	}

	if err := eng.Apply(ctx, result, opts); err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(applyCmd)

	applyCmd.Flags().BoolVar(&confirmFlag, "confirm", false, "Confirm and apply changes")
	applyCmd.Flags().BoolVar(&autoConfirmFlag, "auto-confirm", false, "Skip confirmation (for CI)")
	applyCmd.Flags().BoolVar(&backupFlag, "backup", false, "Create backups before modifying files")
}
