package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wasilak/dotisan/pkg/engine"
	"github.com/wasilak/dotisan/pkg/style"

	"github.com/spf13/cobra"
)

var (
	planJSONFlag bool
)

// planCmd represents the plan command
var planCmd = &cobra.Command{
	Use:          "plan",
	SilenceUsage: true,
	Short:        "Show what would change",
	Long: `plan loads the current state, renders all config objects, and calls Reconcile()
on each provider to show a structured diff of what would change.

Output format (default):
  + green: resource will be added
  ~ yellow: resource will be changed (shows diff)
  - red: resource will be removed
  ! orange: resource has drifted from expected state
  = dim: resource is in sync

Use --json for machine-readable output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlan()
	},
}

// progressModel represents the progress bar model for Bubble Tea
type progressModel struct {
	progress  progress.Model
	percent   float64
	message   string
	result    *engine.PlanResult
	err       error
	done      bool
	eng       *engine.Engine
}

// tickMsg is sent when we want to update the progress
type tickMsg struct{}

func (m progressModel) Init() tea.Cmd {
	return m.tickCmd()
}

func (m progressModel) tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc {
			return m, tea.Quit
		}
		return m, nil

	case tickMsg:
		if m.done {
			return m, tea.Quit
		}
		return m, m.tickCmd()

	case progressMsg:
		m.percent = msg.percent
		m.message = msg.message
		// Note: we don't set m.done here - we wait for resultMsg or errorMsg
		return m, nil

	case resultMsg:
		m.result = msg.result
		m.done = true
		return m, tea.Quit

	case errorMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

func (m progressModel) View() string {
	if m.done {
		return ""
	}

	var s string
	s += "\n"
	s += style.Header.Render("Planning...") + "\n\n"
	s += m.message + "\n"
	s += m.progress.ViewAs(m.percent) + "\n"
	s += fmt.Sprintf("%.0f%%\n", m.percent*100)
	return s
}

// Messages for communicating with the Bubble Tea model
type progressMsg struct {
	percent float64
	message string
}

type resultMsg struct {
	result *engine.PlanResult
}
type errorMsg struct {
	err error
}

func runPlan() error {
	// Create engine
	eng, err := engine.NewEngine()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// For JSON output, skip the progress bar
	if planJSONFlag {
		ctx := context.Background()
		result, err := eng.Plan(ctx, nil)
		if err != nil {
			return fmt.Errorf("plan failed: %w", err)
		}
		return displayJSON(result)
	}

	// Check if we're in an interactive terminal
	if !isTerminal() {
		// Non-interactive mode: run plan without progress bar
		ctx := context.Background()
		result, err := eng.Plan(ctx, func(percent float64, message string) {
			// Simple text progress for non-interactive mode
			if message != "" {
				fmt.Printf("→ %s\n", message)
			}
		})
		if err != nil {
			return fmt.Errorf("plan failed: %w", err)
		}
		eng.DisplayPlan(result)
		return nil
	}

	// Interactive mode: use Bubble Tea progress bar
	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	m := progressModel{
		progress: prog,
		percent:  0,
		message:  "Initializing...",
		eng:      eng,
	}

	// Run Bubble Tea program
	p := tea.NewProgram(m)

	// Run plan in background and send updates
	var result *engine.PlanResult
	var planErr error
	go func() {
		ctx := context.Background()
		result, planErr = eng.Plan(ctx, func(percent float64, message string) {
			p.Send(progressMsg{percent: percent, message: message})
		})

		if planErr != nil {
			p.Send(errorMsg{err: planErr})
		} else {
			p.Send(resultMsg{result: result})
		}
	}()

	// Run the program
	var fallbackUsed bool
	finalModel, err := p.Run()
	if err != nil {
		// Fall back to simple progress on TTY error
		fmt.Printf("→ Running plan...\n")
		ctx := context.Background()
		result, err = eng.Plan(ctx, func(percent float64, message string) {
			if message != "" {
				fmt.Printf("  %s\n", message)
			}
		})
		if err != nil {
			return fmt.Errorf("plan failed: %w", err)
		}
		fallbackUsed = true
	} else {
		// Extract result from the model
		if m, ok := finalModel.(progressModel); ok && m.result != nil {
			result = m.result
		}
		// If model extraction failed, result should already be set by the goroutine
	}

	// Only check planErr if we didn't use the fallback
	if !fallbackUsed && planErr != nil {
		return fmt.Errorf("plan failed: %w", planErr)
	}

	// Check if we have a result
	if result == nil {
		return fmt.Errorf("plan failed: no result returned")
	}

	// Display results
	eng.DisplayPlan(result)

	return nil
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

func displayJSON(result *engine.PlanResult) error {
	output := map[string]interface{}{
		"summary": map[string]int{
			"additions":     result.TotalAdditions,
			"modifications": result.TotalModifications,
			"removals":      result.TotalRemovals,
			"in_sync":       result.TotalInSync,
			"drifted":       result.TotalDrifted,
		},
		"has_changes": result.HasChanges,
		"resources":   []map[string]interface{}{},
	}

	// Build resources list
	resources := []map[string]interface{}{}

	for providerName, plan := range result.ProviderPlans {
		for _, res := range plan.Additions {
			resources = append(resources, map[string]interface{}{
				"action":    "add",
				"provider":  providerName,
				"kind":      res.GetKind(),
				"name":      res.GetMetadata().Name,
				"namespace": res.GetMetadata().GetNamespace(),
			})
		}
		for _, mod := range plan.Modifications {
			resources = append(resources, map[string]interface{}{
				"action":    "modify",
				"provider":  providerName,
				"kind":      mod.Resource.GetKind(),
				"name":      mod.Resource.GetMetadata().Name,
				"namespace": mod.Resource.GetMetadata().GetNamespace(),
				"diff":      mod.Diff,
			})
		}
		for _, res := range plan.Removals {
			resources = append(resources, map[string]interface{}{
				"action":    "remove",
				"provider":  providerName,
				"kind":      res.GetKind(),
				"name":      res.GetMetadata().Name,
				"namespace": res.GetMetadata().GetNamespace(),
			})
		}
		for _, drift := range plan.Drifted {
			resources = append(resources, map[string]interface{}{
				"action":      "drift",
				"provider":    providerName,
				"kind":        drift.Resource.GetKind(),
				"name":        drift.Resource.GetMetadata().Name,
				"namespace":   drift.Resource.GetMetadata().GetNamespace(),
				"description": drift.Description,
				"diff":        drift.Diff,
			})
		}
	}

	output["resources"] = resources

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func init() {
	rootCmd.AddCommand(planCmd)
	planCmd.Flags().BoolVar(&planJSONFlag, "json", false, "Output in JSON format")
}
