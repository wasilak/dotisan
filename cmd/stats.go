package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wasilak/nim/pkg/engine"
	"github.com/wasilak/nim/pkg/style"
	"github.com/wasilak/nim/pkg/ui"
)

var (
	statsAllFlag    bool
	statsOutputFlag string
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show statistics about nim-managed resources",
	Long: `Display per-kind resource counts from nim state.

Use --all to query each package manager for all installed packages and
compute nim's coverage percentage per resource kind.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStats(cmd.Context(), statsAllFlag, statsOutputFlag)
	},
}

func init() {
	statsCmd.Flags().BoolVar(&statsAllFlag, "all", false, "query all installed packages to compute coverage")
	statsCmd.Flags().StringVarP(&statsOutputFlag, "output", "o", "plain", "output format: plain or json")
	rootCmd.AddCommand(statsCmd)
}

func runStats(ctx context.Context, withAll bool, outputFmt string) error {
	if ctx == nil {
		return fmt.Errorf("internal: context is nil")
	}
	eng, err := engine.NewEngine()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	var result *engine.StatsResult
	spinMsg := "Loading state..."
	if withAll {
		spinMsg = "Querying package managers..."
	}

	if err := ui.RunWithSpinner(ctx, style.Info, spinMsg, "stats cancelled", func(ctx context.Context, _ func(ui.MessageLevel, string)) error {
		var e error
		result, e = eng.Stats(ctx, withAll)
		return e
	}); err != nil {
		return fmt.Errorf("stats failed: %w", err)
	}

	if outputFmt == "json" {
		return printStatsJSON(result, withAll)
	}
	return printStatsPlain(result, withAll)
}

func printStatsPlain(result *engine.StatsResult, withAll bool) error {
	fmt.Println()
	fmt.Println(style.Bold.Render("  Managed Resources"))
	fmt.Println()

	tbl := ui.NewTable(os.Stdout)
	tbl.SetHeaders("Kind", "Groups", "Items")
	for _, ks := range result.KindStats {
		tbl.AddRow(
			ks.Kind,
			fmt.Sprintf("%d", ks.Groups),
			fmt.Sprintf("%d", ks.Items),
		)
	}
	tbl.Render()

	fmt.Printf("\n  %s  %s groups, %s items\n",
		style.DimStyle.Render("Total:"),
		style.TableCell.Render(fmt.Sprintf("%d", result.TotalGroups)),
		style.TableCell.Render(fmt.Sprintf("%d", result.TotalItems)),
	)

	if withAll && len(result.Coverage) > 0 {
		fmt.Println()
		fmt.Println(style.Bold.Render("  Coverage"))
		fmt.Println()

		// Find the longest kind name for alignment.
		maxKind := 0
		for _, cv := range result.Coverage {
			if len(cv.Kind) > maxKind {
				maxKind = len(cv.Kind)
			}
		}

		for _, cv := range result.Coverage {
			untracked := max(0, cv.Installed-cv.Tracked)
			if cv.Installed == 0 && cv.Tracked == 0 {
				continue
			}
			pct := 0.0
			if cv.Installed > 0 {
				pct = float64(cv.Tracked) / float64(cv.Installed) * 100
			}
			clr := coverageColor(pct)
			bar := clr.Render(coverageBar(pct, 24))
			pctStr := clr.Render(fmt.Sprintf("%5.1f%%", pct))
			detail := style.DimStyle.Render(fmt.Sprintf("%d/%d, %d untracked", cv.Tracked, cv.Installed, untracked))
			fmt.Printf("  %-*s  %s  %s  %s\n", maxKind, cv.Kind, bar, pctStr, detail)
		}
		fmt.Println()
	}

	return nil
}

// coverageBar renders a Unicode block progress bar of the given width.
// Filled cells use '█', the remainder uses '░'.
func coverageBar(pct float64, width int) string {
	filled := min(width, int(pct/100*float64(width)))
	bar := make([]rune, width)
	for i := range bar {
		if i < filled {
			bar[i] = '█'
		} else {
			bar[i] = '░'
		}
	}
	return string(bar)
}

// coverageColor returns a pastel style based on coverage percentage.
// Intentionally soft — low coverage is informational, not alarming.
func coverageColor(pct float64) style.Style {
	switch {
	case pct >= 75:
		return style.Success // mint
	case pct >= 40:
		return style.Warning // peachy coral
	default:
		return style.TableCell // soft lavender
	}
}

func printStatsJSON(result *engine.StatsResult, withAll bool) error {
	type jsonKind struct {
		Kind   string `json:"kind"`
		Groups int    `json:"groups"`
		Items  int    `json:"items"`
	}
	type jsonCoverage struct {
		Kind        string  `json:"kind"`
		Installed   int     `json:"installed"`
		Tracked     int     `json:"tracked"`
		Untracked   int     `json:"untracked"`
		CoveragePct float64 `json:"coverage_pct"`
	}
	type jsonOut struct {
		TotalGroups int            `json:"total_groups"`
		TotalItems  int            `json:"total_items"`
		Kinds       []jsonKind     `json:"kinds"`
		Coverage    []jsonCoverage `json:"coverage,omitempty"`
	}

	out := jsonOut{
		TotalGroups: result.TotalGroups,
		TotalItems:  result.TotalItems,
	}
	for _, ks := range result.KindStats {
		out.Kinds = append(out.Kinds, jsonKind{Kind: ks.Kind, Groups: ks.Groups, Items: ks.Items})
	}
	if withAll {
		for _, cv := range result.Coverage {
			untracked := max(0, cv.Installed-cv.Tracked)
			pct := 0.0
			if cv.Installed > 0 {
				pct = float64(cv.Tracked) / float64(cv.Installed) * 100
			}
			out.Coverage = append(out.Coverage, jsonCoverage{
				Kind:        cv.Kind,
				Installed:   cv.Installed,
				Tracked:     cv.Tracked,
				Untracked:   untracked,
				CoveragePct: pct,
			})
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
