package ui

import (
	"encoding/json"
	"fmt"
	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/engine"
	"github.com/wasilak/dotisan/pkg/output"
	"github.com/wasilak/dotisan/pkg/style"
	"os"
	"strings"
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
			// top-level aggregated warnings for machine consumption
			"warnings": []map[string]interface{}{},
		}
		providers := make(map[string]interface{})
		allWarnings := make([]map[string]interface{}, 0)
		for name, plan := range result.ProviderPlans {
			prov := map[string]interface{}{
				"additions":     plan.Additions,
				"modifications": plan.Modifications,
				"removals":      plan.Removals,
				"cleanup":       plan.Cleanup,
				"in_sync":       plan.InSync,
				"drifted":       plan.Drifted,
			}
			// include per-provider warnings in the aggregated list and embed per-provider
			// warnings into the provider object for easier machine consumption.
			if len(plan.Warnings) > 0 {
				pw := make([]map[string]interface{}, 0, len(plan.Warnings))
				for _, w := range plan.Warnings {
					entry := map[string]interface{}{
						"group_id":   w.GroupID,
						"item_id":    w.ItemID,
						"severity":   w.Severity,
						"message":    w.Message,
						"suggestion": w.Suggestion,
					}
					pw = append(pw, entry)
					allWarnings = append(allWarnings, map[string]interface{}{
						"provider":   name,
						"group_id":   w.GroupID,
						"item_id":    w.ItemID,
						"severity":   w.Severity,
						"message":    w.Message,
						"suggestion": w.Suggestion,
					})
				}
				prov["warnings"] = pw
			}

			providers[name] = prov
		}
		output["providers"] = providers
		if len(allWarnings) > 0 {
			output["warnings"] = allWarnings
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	case output.FormatTree:
		treeFormatter := diff.NewTreeFormatter()
		// Render provider warnings first so they are visible in tree output as well.
		for pname, plan := range result.ProviderPlans {
			if len(plan.Warnings) > 0 {
				// Header uses the palette's header style so it's consistent with plain output.
				fmt.Println()
				fmt.Println(style.Header.Render(pname + " warnings"))
				for _, w := range plan.Warnings {
					id := w.GroupID
					if w.ItemID != "" {
						id = id + "/" + w.ItemID
					}
					icon := style.Warning.Render("⚠")
					// id bolded, message in warning colour
					fmt.Printf("  %s %s: %s\n", icon, style.Bold.Render(id), style.Warning.Render(w.Message))
					if w.Suggestion != "" {
						fmt.Printf("    %s %s\n", style.Info.Render("Suggestion:"), style.DimStyle.Render(w.Suggestion))
					}
				}
			}
		}
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
		// helper type for rendering warnings
		type providerWarning struct {
			GroupID    string
			ItemID     string
			Severity   string
			Message    string
			Suggestion string
		}

		// Collect any provider-generated warnings early so they can be shown even
		// when there are no actionable changes. Warnings are advisory and useful
		// independently of the plan counts.
		allWarnings := []providerWarning{}
		for _, plan := range result.ProviderPlans {
			for _, w := range plan.Warnings {
				allWarnings = append(allWarnings, providerWarning{
					GroupID:    w.GroupID,
					ItemID:     w.ItemID,
					Severity:   w.Severity,
					Message:    w.Message,
					Suggestion: w.Suggestion,
				})
			}
		}

		// If there are no changes and no warnings, render the no-changes banner.
		if !result.HasChanges && len(allWarnings) == 0 {
			RenderNoChanges()
			return nil
		}

		fmt.Println(style.Header.Render("Plan Summary"))
		fmt.Println()

		if len(allWarnings) > 0 {
			fmt.Println(style.Header.Render("Warnings"))
			for _, w := range allWarnings {
				icon := style.Warning.Render("⚠")
				id := w.GroupID
				if w.ItemID != "" {
					id = id + "/" + w.ItemID
				}
				fmt.Printf("  %s %s: %s\n", icon, style.Bold.Render(id), style.Warning.Render(w.Message))
				if w.Suggestion != "" {
					fmt.Printf("    %s %s\n", style.Info.Render("Suggestion:"), style.DimStyle.Render(w.Suggestion))
				}
			}
			fmt.Println()
		}
		type PlanItem struct {
			Action      string
			Name        string
			Kind        string
			Region      string
			Explanation string
			Details     string
		}
		var flatItems []PlanItem

		// (providerWarning already declared above)
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

        // --- Unified (inline) diffs ---
        if showDiff {
            // Use the styled diff engine which can generate and format unified diffs
            styled := diff.NewStyledEngine()

            for providerName, plan := range result.ProviderPlans {
                // MODIFICATIONS
                for _, mod := range plan.Modifications {
                    for _, ch := range mod.Changes {
                        if ch.OldContent != "" || ch.NewContent != "" {
                            filePath := ch.ItemName
                            if mod.Kind != "" && mod.Group != "" {
                                filePath = fmt.Sprintf("%s/%s[%s]", mod.Kind, mod.Group, ch.ItemName)
                            }
                            printDiffHeader("update", filePath, providerName)
                            diffText, err := styled.GenerateUnifiedDiff("before", "after", ensureTrailingNewline(ch.OldContent), ensureTrailingNewline(ch.NewContent))
                            if err != nil {
                                // Fallback to raw ch.Diff or simple content print
                                if ch.Diff != "" {
                                    fmt.Print(ch.Diff)
                                } else {
                                    fmt.Printf("- %s\n+ %s\n", ch.OldContent, ch.NewContent)
                                }
                                continue
                            }
                            fmt.Print(styled.FormatUnifiedDiff(diffText))
                        } else if ch.Diff != "" {
                            printDiffHeader("update", ch.ItemName, providerName)
                            fmt.Print(ch.Diff)
                        }
                    }
                }

                // ADDITIONS
                for _, add := range plan.Additions {
                    for _, item := range add.Items {
                        if add.Contents != nil && add.Contents[item.Name] != "" {
                            filePath := fmt.Sprintf("%s/%s[%s]", add.Kind, add.Group, item.Name)
                            printDiffHeader("add", filePath, providerName)
                            diffText, err := styled.GenerateUnifiedDiff("/dev/null", filePath, "", ensureTrailingNewline(add.Contents[item.Name]))
                            if err != nil {
                                fmt.Print(ensureTrailingNewline(add.Contents[item.Name]))
                                continue
                            }
                            fmt.Print(styled.FormatUnifiedDiff(diffText))
                        }
                    }
                }

                // REMOVALS
                for _, rem := range plan.Removals {
                    for _, item := range rem.Items {
                        if rem.Contents != nil && rem.Contents[item.Name] != "" {
                            filePath := fmt.Sprintf("%s/%s[%s]", rem.Kind, rem.Group, item.Name)
                            printDiffHeader("remove", filePath, providerName)
                            diffText, err := styled.GenerateUnifiedDiff(filePath, "/dev/null", ensureTrailingNewline(rem.Contents[item.Name]), "")
                            if err != nil {
                                fmt.Print(ensureTrailingNewline(rem.Contents[item.Name]))
                                continue
                            }
                            fmt.Print(styled.FormatUnifiedDiff(diffText))
                        }
                    }
                }
            }
            fmt.Println()
        }

		planParts := []string{
			fmt.Sprintf("%s to add", style.TableStatusAdd.Render(fmt.Sprintf("%d", result.TotalAdditions))),
			fmt.Sprintf("%s to change", style.TableStatusUpdate.Render(fmt.Sprintf("%d", result.TotalModifications))),
			fmt.Sprintf("%s to destroy", style.TableStatusRemove.Render(fmt.Sprintf("%d", result.TotalRemovals))),
		}

		if result.TotalCleanup > 0 {
			planParts = append(planParts, fmt.Sprintf("%s cleanup (will be removed from state)", style.TableStatusCleanup.Render(fmt.Sprintf("%d", result.TotalCleanup))))
		}
		fmt.Printf("Plan: %s\n", strings.Join(planParts, ", "))
		return nil
	}
}

// printDiffHeader renders a visual divider and a colour-coded action label
// with the file path before each diff block.
func printDiffHeader(action, filePath, providerName string) {
	// TODO: migrate to os terminal or use default width
	width := 80
	if width < 20 {
		width = 72
	}
	rule := strings.Repeat("─", width) // removed pterm, replace with palette later

	var badge string
	switch action {
	case "add":
		badge = style.DiffBadgeAdd.Render("+ add   ")
	case "remove":
		badge = style.DiffBadgeRemove.Render("- remove")
	default:
		badge = style.DiffBadgeUpdate.Render("~ update")
	}

	path := style.DiffPath.Render(filePath)
	prov := style.DiffProvider.Render("(" + providerName + ")")

	fmt.Println()
	fmt.Println(rule)
	fmt.Printf("  %s  %s  %s\n", badge, path, prov)
	fmt.Println(rule)
	// (All pterm styles stubbed, now using color palette)
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
