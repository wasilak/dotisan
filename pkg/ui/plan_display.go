package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/engine"
	"github.com/wasilak/dotisan/pkg/output"
	"github.com/wasilak/dotisan/pkg/style"
)

// DisplayPlanResult handles displaying a PlanResult in plain, tree, or JSON modes.
func DisplayPlanResult(result *engine.PlanResult, outputFormat output.Format, showDiff bool) error {
	switch outputFormat {
	case output.FormatJSON:
		output := map[string]interface{}{
			"summary": map[string]int{
				"additions":     result.TotalAdditions,
				"modifications": result.TotalModifications,
				"removals":      result.TotalRemovals,
				"cleanup":       result.TotalCleanup,
				"in_sync":       result.TotalInSync,
				"drifted":       result.TotalDrifted,
			},
			"has_changes": result.HasChanges,
			"providers":   map[string]interface{}{},
		}
		providers := make(map[string]interface{})
		for name, plan := range result.ProviderPlans {
			providers[name] = map[string]interface{}{
				"additions":     plan.Additions,
				"modifications": plan.Modifications,
				"removals":      plan.Removals,
				"cleanup":       plan.Cleanup,
				"in_sync":       plan.InSync,
				"drifted":       plan.Drifted,
			}
		}
		output["providers"] = providers
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	case output.FormatTree:
		treeFormatter := diff.NewTreeFormatter()
		for providerName, plan := range result.ProviderPlans {
			if len(plan.Additions) > 0 || len(plan.Removals) > 0 || len(plan.Modifications) > 0 {
				fmt.Printf("\n%s:\n", providerName)
				if err := treeFormatter.FormatGroupPlanAsTree(diff.GroupPlanInfo{Plan: plan}); err != nil {
					fmt.Fprintf(os.Stderr, "tree render error: %v\n", err)
				}
			}
		}
		return nil
	default:
		if !result.HasChanges {
			fmt.Println(style.Info.Render("No changes to apply."))
			return nil
		}
		fmt.Println(style.Header.Render("Plan Summary"))
		fmt.Println()
		type PlanItem struct {
			Action      string
			Name        string
			Kind        string
			Region      string
			Explanation string
			Details     string
		}
		var flatItems []PlanItem
		for _, groupPlan := range result.ProviderPlans {
			for _, add := range groupPlan.Additions {
				for _, item := range add.Items {
					flatItems = append(flatItems, PlanItem{
						Action:      "add",
						Name:        item.Name,
						Kind:        add.Kind,
						Region:      add.Group,
						Details:     item.Version,
						Explanation: "",
					})
				}
			}
			for _, rem := range groupPlan.Removals {
				for _, item := range rem.Items {
					flatItems = append(flatItems, PlanItem{
						Action:      "remove",
						Name:        item.Name,
						Kind:        rem.Kind,
						Region:      rem.Group,
						Details:     item.Version,
						Explanation: "",
					})
				}
			}
			for _, cl := range groupPlan.Cleanup {
				for _, item := range cl.Items {
					flatItems = append(flatItems, PlanItem{
						Action:      "cleanup",
						Name:        item.Name,
						Kind:        cl.Kind,
						Region:      cl.Group,
						Details:     item.Version,
						Explanation: "will be removed from state",
					})
				}
			}
			for _, mod := range groupPlan.Modifications {
				for _, ch := range mod.Changes {
					flatItems = append(flatItems, PlanItem{
						Action:      "update",
						Name:        ch.ItemName,
						Kind:        mod.Kind,
						Region:      mod.Group,
						Details:     ch.NewState.Version,
						Explanation: "",
					})
				}
			}
			for _, drift := range groupPlan.Drifted {
				flatItems = append(flatItems, PlanItem{
					Action:      "drift",
					Name:        drift.Item,
					Kind:        "",
					Region:      "",
					Details:     "",
					Explanation: "actual vs expected drift",
				})
			}
		}
		rows := make([]ResourceRow, 0, len(flatItems))
		for _, it := range flatItems {
			var id string
			if it.Kind != "" && it.Region != "" && it.Name != "" {
				id = fmt.Sprintf("%s/%s[%s]", it.Kind, it.Region, it.Name)
			} else if it.Kind != "" && it.Region != "" {
				id = fmt.Sprintf("%s/%s", it.Kind, it.Region)
			} else {
				id = it.Name
			}
			info := it.Explanation
			if info == "" {
				info = it.Details
			}
			rows = append(rows, ResourceRow{
				Status: it.Action,
				ID:     id,
				Kind:   it.Kind,
				Group:  it.Region,
				Name:   it.Name,
				Info:   info,
			})
		}
		if err := RenderResourceTable(rows, true); err != nil {
			fmt.Fprintf(os.Stderr, "resource table error: %v\n", err)
		}
		fmt.Println()

		// --- Unified Diffs ---
		if showDiff {
			for providerName, plan := range result.ProviderPlans {
				// MODIFICATIONS
				for _, mod := range plan.Modifications {
					for _, ch := range mod.Changes {
						if ch.OldContent != "" || ch.NewContent != "" {
							id := ch.ItemName
							label := "[update] " + id
							oldName, newName := "before", "after"
							if mod.Kind != "" && mod.Group != "" {
								oldName = mod.Kind + "/" + mod.Group + "/" + ch.ItemName
								newName = oldName
							}
							// Normalize newlines before diff generation so the
							// unified diff has consistent, readable hunks.
							oldC := ensureTrailingNewline(ch.OldContent)
							newC := ensureTrailingNewline(ch.NewContent)
							diffText, _ := diff.NewEngine().GenerateUnifiedDiff(oldName, newName, oldC, newC)
							// Truncate very large diffs to keep output readable in the plan
							dt := truncateUnifiedDiff(diffText, 3)
							colored, _ := diff.HighlightUnifiedDiff(dt, "github-dark")
							fmt.Printf("\n\033[1m%s (%s)\033[0m\n%s\n", label, providerName, colored)
						} else if ch.Diff != "" {
							// For package diffs: just render Diff field
							fmt.Printf("\n\033[1m[update] %s (%s)\033[0m\n%s\n", ch.ItemName, providerName, ch.Diff)
						}
					}
				}
				// ADDITIONS
				for _, add := range plan.Additions {
					for _, item := range add.Items {
						if add.Contents != nil && add.Contents[item.Name] != "" {
							id := item.Name
							label := "[add] " + id
							fileName := add.Kind + "/" + add.Group + "/" + id
							content := ensureTrailingNewline(add.Contents[item.Name])
							diffText, _ := diff.NewEngine().GenerateUnifiedDiff("/dev/null", fileName, "", content)
							dt := truncateUnifiedDiff(diffText, 3)
							colored, _ := diff.HighlightUnifiedDiff(dt, "github-dark")
							fmt.Printf("\n\033[1m%s (%s)\033[0m\n%s\n", label, providerName, colored)
						}
					}
				}
				// REMOVALS
				for _, rem := range plan.Removals {
					for _, item := range rem.Items {
						if rem.Contents != nil && rem.Contents[item.Name] != "" {
							id := item.Name
							label := "[remove] " + id
							fileName := rem.Kind + "/" + rem.Group + "/" + id
							content := ensureTrailingNewline(rem.Contents[item.Name])
							diffText, _ := diff.NewEngine().GenerateUnifiedDiff(fileName, "/dev/null", content, "")
							dt := truncateUnifiedDiff(diffText, 3)
							colored, _ := diff.HighlightUnifiedDiff(dt, "github-dark")
							fmt.Printf("\n\033[1m%s (%s)\033[0m\n%s\n", label, providerName, colored)
						}
					}
				}
			}
		}

		planParts := []string{
			fmt.Sprintf("%s to add", style.Success.Render(fmt.Sprintf("%d", result.TotalAdditions))),
			fmt.Sprintf("%s to change", style.Info.Render(fmt.Sprintf("%d", result.TotalModifications))),
			fmt.Sprintf("%s to destroy", style.Error.Render(fmt.Sprintf("%d", result.TotalRemovals))),
		}

		if result.TotalCleanup > 0 {
			planParts = append(planParts, fmt.Sprintf("%s cleanup (will be removed from state)", style.Dim.Render(fmt.Sprintf("%d", result.TotalCleanup))))
		}
		fmt.Printf("Plan: %s\n", strings.Join(planParts, ", "))
		return nil
	}
}

// truncateUnifiedDiff keeps the diff headers and the first `maxHunks` hunks
// to avoid flooding the plan output when files are large. It returns the
// possibly-truncated diff string; if truncation occurs a footer note is
// appended to explain the truncation.
func truncateUnifiedDiff(diffText string, maxHunks int) string {
	if maxHunks <= 0 {
		return diffText
	}
	// Line-based safe hunk counting. Keep the file headers (starting with
	// --- and +++) and the first `maxHunks` hunk sections which start with
	// "@@". This preserves the @@ markers exactly as they appear.
	lines := strings.Split(diffText, "\n")
	if maxHunks <= 0 {
		return diffText
	}
	var b strings.Builder
	hunkCount := 0
	inHunk := false
	for i, ln := range lines {
		// Always append header lines until we hit first @@
		if strings.HasPrefix(ln, "@@ ") || strings.HasPrefix(ln, "@@-") || strings.HasPrefix(ln, "@@") {
			// Starting a new hunk
			hunkCount++
			if hunkCount > maxHunks {
				// We've included enough hunks; append truncation note and stop
				b.WriteString("\n\n--- Diff truncated (showing first ")
				b.WriteString(fmt.Sprintf("%d hunks). Use --diff to view full diff.", maxHunks))
				break
			}
			inHunk = true
		}
		b.WriteString(ln)
		// Don't re-add a trailing newline for the last line to avoid
		// introducing extra blank lines.
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
		if inHunk && strings.HasPrefix(ln, "@@") {
			// continue capturing until next hunk or end
		}
	}
	return b.String()
}

// ensureTrailingNewline mirrors the behavior in pkg/diff to guarantee
// the UI uses normalized content when generating diffs.
func ensureTrailingNewline(s string) string {
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimRight(s, "\n")
	return s + "\n"
}
