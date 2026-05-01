// Package style provides shared TUI/CLI helpers.
//
// Spinner provides a portable TTY-safe progress indicator
// for CLI workflows using Bubble Tea/Bubbles.
//
// Usage:
//
//	err := spinner.WithSpinner("Applying resources", func(spinner StopFunc) error {
//	  ... // long-running code
//	  return nil
//	})
//
// For non-TTY, falls back to simple log.
package style

import (
	"os"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"fmt"
	"golang.org/x/term"
)

// StopFunc signals spinner to finish.
type StopFunc func(msg string)

// WithSpinner runs a spinner TUI if terminal is interactive, otherwise logs msg at start/end.
// The callback receives a StopFunc it must call to signal completion.
// The callback should return promptly after calling StopFunc.
func WithSpinner(msg string, fn func(stop StopFunc) error) error {
	if !isTTY() {
		fmt.Fprintf(os.Stdout, "%s... ", msg)
		start := time.Now()
		err := fn(func(stopMsg string) {
			// no-op, handled below
		})
		dur := time.Since(start).Truncate(time.Millisecond)
		if err != nil {
			fmt.Fprintf(os.Stdout, "failed: %v\n", err)
			return err
		}
		fmt.Fprintf(os.Stdout, "done (%s)\n", dur)
		return nil
	}
	completed := make(chan struct{})
	var resultErr error
	var doneMsg string

	model := newSpinnerModel(msg)
	p := tea.NewProgram(model)

	// run long-running fn in background
	go func() {
		resultErr = fn(func(stopMsg string) {
			doneMsg = stopMsg
			close(completed)
		})
		close(completed)
	}()

	// spinner loop
	p.Run()
	<-completed
	p.Quit()
	if resultErr != nil {
		fmt.Fprintf(os.Stdout, "\n[✗] %s: %v\n", msg, resultErr)
		return resultErr
	}
	if doneMsg != "" {
		fmt.Fprintf(os.Stdout, "\n[✓] %s: %s\n", msg, doneMsg)
	} else {
		fmt.Fprintf(os.Stdout, "\n[✓] %s\n", msg)
	}
	return nil
}

func isTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// Internal spinner model for Bubble Tea
type spinnerModel struct {
	sp      spinner.Model
	msg     string
	stopped bool
}

func newSpinnerModel(msg string) spinnerModel {
	m := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("205"))))
	return spinnerModel{
		sp:  m,
		msg: msg,
	}
}

func (m spinnerModel) Init() tea.Cmd {
	return func() tea.Msg { return m.sp.Tick() }
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.stopped {
			return m, nil
		}
		var cmd tea.Cmd
		m.sp, cmd = m.sp.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m spinnerModel) View() tea.View {
	if m.stopped {
		return tea.NewView("")
	}
	return tea.NewView(fmt.Sprintf("%s %s", m.sp.View(), m.msg))
}
