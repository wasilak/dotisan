package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/pterm/pterm"
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
	if err = style.RunWithSpinner(ctx, "Planning...", func(ctx context.Context) error {
		result, planErr = eng.Plan(ctx, engine.PlanOptions{Targets: opts.Targets, ShowDiff: opts.ShowDiff})
		return planErr
	}); err != nil {
		return fmt.Errorf("plan failed: %w", err)
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
		if opts.IsApply {
			fmt.Println(style.Info.Render("No resources to apply for your targets."))
		} else {
			ui.RenderNoChanges()
		}
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
		confirmed, confirmErr = pterm.DefaultInteractiveConfirm.Show(changeSummary)
		if confirmErr != nil {
			return fmt.Errorf("confirmation prompt error: %w", confirmErr)
		}
		if !confirmed {
			fmt.Println()
			fmt.Println(style.Info.Render("→ Apply cancelled."))
			return nil
		}
	}

	// ==== Now, setup progress and actually apply ====
	progressItems := 0
	for _, providerPlan := range result.ProviderPlans {
		for _, add := range providerPlan.Additions {
			progressItems += len(add.Items)
		}
		for _, mod := range providerPlan.Modifications {
			progressItems += len(mod.Changes)
		}
		for _, rem := range providerPlan.Removals {
			progressItems += len(rem.Items)
		}
	}
	var progress *style.ApplyProgress
	if progressItems > 0 {
		progress = style.NewApplyProgress(progressItems)
	}
	appOpts := engine.ApplyOptions{Confirm: confirmed}
	if progress != nil {
		appOpts.OnItemStart = func(kind, group, item string) { progress.StartItem(kind, group, item) }
		appOpts.OnItemComplete = func(kind, group, item string, err error) { progress.CompleteItem(err) }
	}
	defer func() {
		if progress != nil {
			progress.Stop()
		}
	}()

	if err := eng.Apply(ctx, result, appOpts); err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}
	fmt.Println(style.IconSuccess + " Changes applied successfully.")
	return nil

}
