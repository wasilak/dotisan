//go:build integration
// +build integration

package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/wasilak/nim/pkg/style"
	"github.com/wasilak/nim/pkg/ui"
)

// test-block is an integration-only command that starts a spinner and blocks
// until the root context is cancelled. It is built only with the 'integration'
// build tag so it does not ship in normal CLI builds.
var testBlockCmd = &cobra.Command{
	Use:    "test-block",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		sp := ui.NewSpinner()
		// Use a descriptive cancel message so tests can assert on it.
		stop, _ := sp.StartWithContext(cmd.Context(), style.Info, "Running test block...", "test cancelled")

		// Block until context cancelled (signal from test). Give the spinner a
		// moment to start so output is deterministic.
		select {
		case <-cmd.Context().Done():
		case <-time.After(10 * time.Second):
			// safety: should not happen in tests but avoid hanging forever
			fmt.Println("timeout waiting for cancellation")
		}
		stop()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(testBlockCmd)
}
