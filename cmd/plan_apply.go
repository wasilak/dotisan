package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	// pterm import removed. TODO: migrate all CLI UI calls to palette-based toolkit.
	"github.com/wasilak/dotisan/pkg/engine"
	"github.com/wasilak/dotisan/pkg/output"
	"github.com/wasilak/dotisan/pkg/style"
	"github.com/wasilak/dotisan/pkg/ui"
)

type PlanApplyOptions struct {
	IsApply      bool
	Confirm      bool
	OutputFormat string
	Targets      []string
	ShowDiff     bool // If true, print contextual (syntax) diffs in output
}

func runPlanApply(ctx context.Context, opts PlanApplyOptions) error {
	if ctx == nil {
		return fmt.Errorf("internal: context is nil")
	}
	eng, err := engine.NewEngine()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Run plan (show spinner while planning)
	var result *engine.PlanResult
	var planErr error
	// TODO: Replace with spinner from toolkit; for now, print and wait.
	fmt.Print("Planning...")
	result, planErr = eng.Plan(ctx, engine.PlanOptions{Targets: opts.Targets, ShowDiff: opts.ShowDiff})
	fmt.Println()
	if planErr != nil {
		return fmt.Errorf("plan failed: %w", planErr)
	}

	// Print warnings for unmatched targets
	if len(result.UnmatchedTargets) > 0 {
		for _, t := range result.UnmatchedTargets {
			fmt.Fprintf(os.Stderr, "%s target %q did not match any resources\n", style.Warning.Render("Warning:"), t)
		}
	}
	// Early exit if nothing to do
	allEmpty := true
	for _, groupPlan := range result.ProviderPlans {
		if len(groupPlan.Additions) > 0 || len(groupPlan.Removals) > 0 || len(groupPlan.Modifications) > 0 || len(groupPlan.Cleanup) > 0 || len(groupPlan.Drifted) > 0 {
			allEmpty = false
			break
		}
	}
	if allEmpty {
		// Show the friendly no-changes card for both plan and apply flows.
		// Previously apply printed a plain message while plan used the
		// celebratory UI; unify behaviour so users see the same output.
		ui.RenderNoChanges()
		return nil
	}
	if !result.HasChanges {
		ui.RenderNoChanges()
		return nil
	}

	// Plan display
	if err := ui.DisplayPlanResult(result, output.Format(opts.OutputFormat), opts.ShowDiff); err != nil {
		return fmt.Errorf("plan output failed: %w", err)
	}

	if !opts.IsApply {
		return nil
	}

	// ===== Handle confirmation (before progress setup!) =====
	confirmed := opts.Confirm
	if !confirmed {
		fmt.Fprint(os.Stdout, "\n") // ensure visual separation, required for some terminals
		totalChanges := result.TotalAdditions + result.TotalModifications + result.TotalRemovals + result.TotalDrifted
		var changeSummary string
		if totalChanges == 1 {
			changeSummary = "Apply 1 change?"
		} else {
			changeSummary = fmt.Sprintf("Apply %d changes?", totalChanges)
		}
		var confirmErr error
		// TODO: Replace with palette-based confirm prompt
		fmt.Printf("%s [y/N]: ", changeSummary)
		var resp string
		_, confirmErr = fmt.Scanln(&resp)
		if confirmErr != nil && confirmErr.Error() != "unexpected newline" {
			return fmt.Errorf("confirmation prompt error: %w", confirmErr)
		}
		if strings.ToLower(resp) != "y" && strings.ToLower(resp) != "yes" {
			fmt.Println()
			fmt.Println("→ Apply cancelled.")
			return nil
		}
		confirmed = true
	}

	// ==== Now, setup progress and actually apply ====
	// TODO: Add progress bar with new UI toolkit
	appOpts := engine.ApplyOptions{Confirm: confirmed}

	if err := eng.Apply(ctx, result, appOpts); err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}
    fmt.Println(style.StyledIconSuccess + " Changes applied successfully.")
	return nil

}
