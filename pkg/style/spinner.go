package style

import (
	"context"
	"fmt"
	"os"

	"github.com/pterm/pterm"
)

// RunWithSpinner runs the provided action while displaying a spinner on TTYs.
// On non-TTY outputs pterm suppresses animation automatically.
//
// parent is a parent context that, when cancelled, will cancel the spinner
// action as well.
func RunWithSpinner(parent context.Context, title string, action func(ctx context.Context) error) error {
	if parent == nil {
		return fmt.Errorf("parent context is nil")
	}

	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	spinner, _ := pterm.DefaultSpinner.WithWriter(os.Stdout).Start(title)

	err := action(ctx)

	if err != nil {
		spinner.Fail(title)
		spinner.Stop() // Ensure cleaned up fully
		return err
	}
	spinner.Success(title)
	spinner.Stop() // Ensure cleaned up fully
	return nil
}
