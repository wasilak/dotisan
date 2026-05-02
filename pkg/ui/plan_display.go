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
func DisplayPlanResult(result *engine.PlanResult, outputFormat output.Format) error {
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