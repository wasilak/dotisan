package diff

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/pterm/pterm"
)

var ansiStripRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

const defaultContextLines = 3

type rowKind int

const (
	rowNormal    rowKind = iota
	rowHunkBreak         // collapsed unchanged region
)

type sideBySideRow struct {
	kind      rowKind
	leftText  string // raw text, no ANSI
	rightText string
	leftType  ChangeType
	rightType ChangeType
	skipCount int // only for rowHunkBreak
}

// SideBySideRenderer renders two file versions as a side-by-side terminal diff.
type SideBySideRenderer struct {
	contextLines int
}

// NewSideBySideRenderer creates a renderer with default 3 context lines.
func NewSideBySideRenderer() *SideBySideRenderer {
	return &SideBySideRenderer{contextLines: defaultContextLines}
}

// Render returns a formatted side-by-side diff string.
// action is one of "add", "remove", "update".
func (r *SideBySideRenderer) Render(oldContent, newContent, action string) string {
	colWidth := (pterm.GetTerminalWidth() - 3) / 2
	if colWidth < 20 {
		colWidth = 20
	}

	var rows []sideBySideRow

	switch action {
	case "add":
		lines := splitRawLines(newContent)
		for _, ln := range lines {
			rows = append(rows, sideBySideRow{
				leftText: "", rightText: ln,
				leftType: LineUnchanged, rightType: LineAdded,
			})
		}
	case "remove":
		lines := splitRawLines(oldContent)
		for _, ln := range lines {
			rows = append(rows, sideBySideRow{
				leftText: ln, rightText: "",
				leftType: LineDeleted, rightType: LineUnchanged,
			})
		}
	default: // "update"
		changes := NewEngine().GenerateDiff(oldContent, newContent)
		rows = pairChanges(changes)
		rows = r.applyContextWindow(rows)
	}

	sep := pterm.NewStyle(pterm.FgGray).Sprint("│")
	divSep := pterm.NewStyle(pterm.FgGray).Sprint("┼")

	var b strings.Builder

	// Column header
	leftHdr := pterm.NewStyle(pterm.Bold).Sprint(padRight("  BEFORE", colWidth))
	rightHdr := pterm.NewStyle(pterm.Bold).Sprint(padRight("  AFTER", colWidth))
	b.WriteString(leftHdr + sep + rightHdr + "\n")

	// Divider under header
	divLine := pterm.NewStyle(pterm.FgGray).Sprint(strings.Repeat("─", colWidth))
	b.WriteString(divLine + divSep + divLine + "\n")

	for _, row := range rows {
		switch row.kind {
		case rowHunkBreak:
			msg := padRight(fmt.Sprintf("  ···  %d unchanged lines  ···", row.skipCount), colWidth)
			styled := pterm.NewStyle(pterm.FgGray).Sprint(msg)
			b.WriteString(styled + sep + styled + "\n")
		default:
			left := applyDiffStyle(padRight(row.leftText, colWidth), row.leftType)
			right := applyDiffStyle(padRight(row.rightText, colWidth), row.rightType)
			b.WriteString(left + sep + right + "\n")
		}
	}

	return b.String()
}

// pairChanges converts a flat []LineChange into side-by-side rows, pairing
// consecutive delete blocks with following add blocks.
func pairChanges(changes []LineChange) []sideBySideRow {
	var rows []sideBySideRow
	i := 0
	for i < len(changes) {
		c := changes[i]
		if c.Type == LineUnchanged {
			rows = append(rows, sideBySideRow{
				leftText: c.Content, rightText: c.Content,
				leftType: LineUnchanged, rightType: LineUnchanged,
			})
			i++
			continue
		}

		// Collect a contiguous run of deletions then additions.
		var deletes, adds []string
		for i < len(changes) && changes[i].Type == LineDeleted {
			deletes = append(deletes, changes[i].Content)
			i++
		}
		for i < len(changes) && changes[i].Type == LineAdded {
			adds = append(adds, changes[i].Content)
			i++
		}

		max := len(deletes)
		if len(adds) > max {
			max = len(adds)
		}
		for k := 0; k < max; k++ {
			var lt, rt string
			var lType, rType ChangeType
			if k < len(deletes) {
				lt = deletes[k]
				lType = LineDeleted
			} else {
				lType = LineUnchanged
			}
			if k < len(adds) {
				rt = adds[k]
				rType = LineAdded
			} else {
				rType = LineUnchanged
			}
			rows = append(rows, sideBySideRow{
				leftText: lt, rightText: rt,
				leftType: lType, rightType: rType,
			})
		}
	}
	return rows
}

// applyContextWindow collapses large unchanged regions into hunk-break rows.
func (r *SideBySideRenderer) applyContextWindow(rows []sideBySideRow) []sideBySideRow {
	if len(rows) == 0 {
		return rows
	}

	// Mark rows within contextLines of a changed row.
	near := make([]bool, len(rows))
	for i, row := range rows {
		if row.leftType != LineUnchanged || row.rightType != LineUnchanged {
			lo := i - r.contextLines
			if lo < 0 {
				lo = 0
			}
			hi := i + r.contextLines
			if hi >= len(rows) {
				hi = len(rows) - 1
			}
			for j := lo; j <= hi; j++ {
				near[j] = true
			}
		}
	}

	var out []sideBySideRow
	skipCount := 0
	for i, row := range rows {
		if near[i] {
			if skipCount > 0 {
				out = append(out, sideBySideRow{kind: rowHunkBreak, skipCount: skipCount})
				skipCount = 0
			}
			out = append(out, row)
		} else {
			skipCount++
		}
	}
	// Trailing skipped lines are silently dropped (unchanged tail after last change).
	return out
}

// splitRawLines splits content on newlines, dropping a single trailing empty element.
func splitRawLines(s string) []string {
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// padRight pads (or truncates) a raw string to exactly width visual columns.
func padRight(s string, width int) string {
	s = strings.ReplaceAll(s, "\t", "    ")
	// Strip any stray ANSI before measuring.
	raw := ansiStripRe.ReplaceAllString(s, "")
	vw := runewidth.StringWidth(raw)
	if vw > width {
		return truncateToWidth(raw, width-1) + "…"
	}
	return raw + strings.Repeat(" ", width-vw)
}

// truncateToWidth cuts s to at most width visual columns, rune-aware.
func truncateToWidth(s string, width int) string {
	acc := 0
	for i, r := range s {
		w := runewidth.RuneWidth(r)
		if acc+w > width {
			return s[:i]
		}
		acc += w
	}
	return s
}

// applyDiffStyle colours a pre-padded raw string based on the change type.
func applyDiffStyle(text string, ct ChangeType) string {
	switch ct {
	case LineDeleted:
		return pterm.NewStyle(pterm.FgRed).Sprint(text)
	case LineAdded:
		return pterm.NewStyle(pterm.FgGreen).Sprint(text)
	default:
		return pterm.NewStyle(pterm.FgGray).Sprint(text)
	}
}
