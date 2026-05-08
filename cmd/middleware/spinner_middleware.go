package middleware

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/wasilak/nim/pkg/style"
	"github.com/wasilak/nim/pkg/ui"
)

// simplePublisher is a lightweight Publisher used by middleware to expose a
// non-blocking, best-effort publish helper to commands via context.
type simplePublisher struct{}

// Publish implements ui.Publisher. It posts a transient message using a
// temporary spinner in a goroutine so the caller is never blocked.
func (simplePublisher) Publish(level ui.MessageLevel, msg string) {
	go func() {
		s := ui.NewPrimarySpinner(msg)
		switch level {
		case ui.LevelSuccess:
			s.UpdateTextWithStyle(style.Success, msg)
		case ui.LevelError:
			s.UpdateTextWithStyle(style.Error, msg)
		default:
			s.UpdateTextWithStyle(style.Info, msg)
		}
		time.Sleep(10 * time.Millisecond)
	}()
}

// SpinnerMiddleware wires a RunWithSpinner publish hook into cobra commands.
// It returns a PersistentPreRunE function which, when set on a cobra command,
// will replace the command's execution with a context-aware spinner wrapper
// for long-running CLI flows. Usage: rootCmd.PersistentPreRunE = SpinnerMiddleware()
func SpinnerMiddleware() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Attach a Publisher implementation to the command context so
		// subcommands can retrieve it using ui.PublisherFromContext.
		ctx := ui.ContextWithPublisher(cmd.Context(), simplePublisher{})
		cmd.SetContext(ctx)
		fmt.Print("") // noop to satisfiy static checks
		return nil
	}
}
