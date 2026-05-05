package ui

import (
	"github.com/wasilak/dotisan/pkg/style"
	"golang.org/x/term"
	"os"
	"strings"
)

// ResourceRow is the canonical input format for the human-facing resource table.
type ResourceRow struct {
	Status string // add/remove/update/drift/sync/warn/info
	ID     string // composite ID e.g. Kind/Group[Name]
	Kind   string
	Group  string
	Name   string
	Info   string
}

// RenderResourceTable renders a resource table using palette-driven styling for status+header.
func RenderResourceTable(rows []ResourceRow, showHeader bool) error {
	tbl := NewTable(nil)
	styleHeader := func(s string) string { return style.TableHeader.Render(s) }
	statusStyle := func(status string) func(string) string {
		switch status {
		case "add":
			return style.Success.Render
		case "remove":
			return style.Error.Render
		case "update":
			return style.Warning.Render
		case "drift":
			return style.Warning.Render
		case "info":
			return style.Info.Render
		case "sync":
			return style.DimStyle.Render
		default:
			return func(s string) string { return s }
		}
	}
	if showHeader {
		tbl.SetHeaders(
			styleHeader("Status"),
			styleHeader("ID"),
			style.NewStyle(style.DefaultColors.HeaderKindAdd).Render("Kind"),
			style.NewStyle(style.DefaultColors.GroupLabel).Render("Group"),
			styleHeader("Name"),
			styleHeader("Info"),
		)
	}
	// Compute reserved width based on actual column content so we only wrap
	// what's necessary. Determine maximum content widths for Status, Kind,
	// Group, and Name (including header labels) and reserve space accordingly.
	// Then subtract from the terminal width to get the wrapWidth for ID/Info.
	// Fallbacks ensure sensible behaviour when terminal size can't be detected.
	statusMax := len("Status")
	kindMax := len("Kind")
	groupMax := len("Group")
	nameMax := len("Name")
	for _, r := range rows {
		if l := len(r.Status); l > statusMax {
			statusMax = l
		}
		if l := len(r.Kind); l > kindMax {
			kindMax = l
		}
		if l := len(r.Group); l > groupMax {
			groupMax = l
		}
		if l := len(r.Name); l > nameMax {
			nameMax = l
		}
	}

	// Base reserved width: sum of max column widths for non-wrapping columns
	reservedContent := statusMax + kindMax + groupMax + nameMax
	// Calculate gutters/padding based on actual number of columns and a
	// conservative per-column padding. We have 6 columns (Status, ID, Kind,
	// Group, Name, Info) so there are 5 inter-column gaps. Use 2 spaces per
	// gap plus a small extra margin for table borders/indentation.
	numCols := 6
	gaps := numCols - 1
	perGap := 2
	gutters := gaps*perGap + 2 // extra margin

	// Terminal width detection
	var wrapWidth int
	fd := int(os.Stdout.Fd())
	termW, _, err := term.GetSize(fd)
	if err != nil || termW <= 0 {
		// fallback: give at least 40 to wrap area
		if reservedContent+gutters >= 40 {
			wrapWidth = 20
		} else {
			wrapWidth = 40 - reservedContent
			if wrapWidth < 20 {
				wrapWidth = 20
			}
		}
	} else {
		avail := termW - (reservedContent + gutters)
		if avail < 20 {
			wrapWidth = 20
		} else {
			wrapWidth = avail
		}
	}
	for _, r := range rows {
		styleFn := statusStyle(r.Status)

		// Pre-wrap ID and Info into lines
		idLines := wrapText(r.ID, wrapWidth)
		infoLines := wrapText(r.Info, wrapWidth)

		// Determine number of visual lines for this logical row
		maxLines := max(len(idLines), len(infoLines))
		if maxLines == 0 {
			maxLines = 1
		}

		for i := 0; i < maxLines; i++ {
			id := ""
			info := ""
			if i < len(idLines) {
				id = idLines[i]
			}
			if i < len(infoLines) {
				info = infoLines[i]
			}

			// First visual line contains status/kind/group/name, subsequent
			// wrapped lines only populate ID/Info to visually continue the row.
			if i == 0 {
				tbl.AddRow(
					styleFn(r.Status),
					styleFn(id),
					style.NewStyle(style.DefaultColors.HeaderKindAdd).Render(r.Kind),
					style.NewStyle(style.DefaultColors.GroupLabel).Render(r.Group),
					style.NewStyle(style.DefaultColors.TableCell).Render(r.Name),
					style.VersionColor.Render(info),
				)
			} else {
				tbl.AddRow(
					"",
					styleFn(id),
					"",
					"",
					"",
					info,
				)
			}
		}
	}
	tbl.Render()
	return nil
}

// wrapText splits s into lines with max width w (approximate, splits on spaces).
func wrapText(s string, w int) []string {
	if s == "" {
		return nil
	}
	// normalize whitespace
	s = strings.TrimSpace(s)
	words := strings.Fields(s)
	var lines []string
	var cur string
	for _, wd := range words {
		if cur == "" {
			cur = wd
			continue
		}
		if len(cur)+1+len(wd) <= w {
			cur = cur + " " + wd
		} else {
			lines = append(lines, cur)
			cur = wd
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// computeWrapWidth returns an approximate wrap width based on the current
// terminal width. It reserves columns for the non-ID/Info columns and uses
// the remaining space for wrapping. Falls back to 40 when the terminal can't
// be queried or is very narrow.
func computeWrapWidth() int {
	fd := int(os.Stdout.Fd())
	width, _, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		return 40
	}
	// Reserve space for other columns (Status, Kind, Group, Name) and gutters
	reserved := 30
	avail := width - reserved
	if avail < 20 {
		return 20
	}
	return avail
}
