package diff

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/mattn/go-runewidth"
	// TODO: All color/styles migrated to palette/ANSI – no pterm
)

var ansiStripRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

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
type SideBySideRenderer struct{}

// NewSideBySideRenderer creates a new renderer.
func NewSideBySideRenderer() *SideBySideRenderer {
	return &SideBySideRenderer{}
}

// Render returns a formatted side-by-side diff string.
// action is one of "add", "remove", "update".
func (r *SideBySideRenderer) Render(oldContent, newContent, action string) string {
	// TODO: replace with dynamic terminal width query, for now use 80 default
	colWidth := (80 - 3) / 2
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
		diffText, _ := NewEngine().GenerateUnifiedDiff("before", "after", oldContent, newContent)
		rows = parseUnifiedDiff(diffText)
	}

	// TODO: with palette, use gray. Now, use plain.
	sep := "│"
	divSep := "┼"

	var b strings.Builder

	// TODO: Apply styling/palette once available; for now, just plain headers
	leftHdr := padRight("  BEFORE", colWidth)
	rightHdr := padRight("  AFTER", colWidth)
	b.WriteString(leftHdr + sep + rightHdr + "\n")

	// Divider under header
	divLine := strings.Repeat("─", colWidth)
	b.WriteString(divLine + divSep + divLine + "\n")

	if len(rows) == 0 {
			note := padRight("  (no textual differences — state checksum may be stale)", colWidth*2+1)
			b.WriteString(note + "\n") // TODO: style as hint/gray

	} else {
		for _, row := range rows {
			switch row.kind {
			case rowHunkBreak:
				msg := padRight(fmt.Sprintf("  ···  %d unchanged lines  ···", row.skipCount), colWidth)
				b.WriteString(msg + sep + msg + "\n") // TODO: style as gray
			default:
				left := applyDiffStyle(padRight(row.leftText, colWidth), row.leftType)
				right := applyDiffStyle(padRight(row.rightText, colWidth), row.rightType)
				b.WriteString(left + sep + right + "\n")
			}
		}
	}

	return b.String()
}

// parseUnifiedDiff converts unified diff text into side-by-side rows.
// Uses the properly LCS-computed diff from GenerateUnifiedDiff so that
// unchanged lines are correctly identified even when changes appear early.
// Gaps between hunks (omitted context) become rowHunkBreak rows.
func parseUnifiedDiff(diffText string) []sideBySideRow {
	var rows []sideBySideRow
	var deletes, adds []string

	// Track old-file line position to compute gap sizes between hunks.
	prevHunkOldEnd := 0
	firstHunk := true

	flush := func() {
		if len(deletes) == 0 && len(adds) == 0 {
			return
		}
		n := len(deletes)
		if len(adds) > n {
			n = len(adds)
		}
		for k := 0; k < n; k++ {
			row := sideBySideRow{}
			if k < len(deletes) {
				row.leftText = deletes[k]
				row.leftType = LineDeleted
			} else {
				row.leftType = LineUnchanged
			}
			if k < len(adds) {
				row.rightText = adds[k]
				row.rightType = LineAdded
			} else {
				row.rightType = LineUnchanged
			}
			rows = append(rows, row)
		}
		deletes = nil
		adds = nil
	}

	for _, line := range strings.Split(diffText, "\n") {
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			continue
		}
		if strings.HasPrefix(line, "@@") {
			flush()
			oldStart, oldCount := parseHunkHeader(line)
			if !firstHunk {
				skip := oldStart - prevHunkOldEnd
				if skip > 0 {
					rows = append(rows, sideBySideRow{kind: rowHunkBreak, skipCount: skip})
				}
			}
			firstHunk = false
			prevHunkOldEnd = oldStart + oldCount
			continue
		}
		if len(line) == 0 || line[0] == '\\' { // "\ No newline at end of file"
			continue
		}
		switch line[0] {
		case '-':
			deletes = append(deletes, line[1:])
		case '+':
			adds = append(adds, line[1:])
		case ' ':
			flush()
			content := line[1:]
			rows = append(rows, sideBySideRow{
				leftText: content, rightText: content,
				leftType: LineUnchanged, rightType: LineUnchanged,
			})
		}
	}
	flush()
	return rows
}

// parseHunkHeader extracts the old-file start line and count from a unified
// diff hunk header of the form "@@ -start[,count] +start[,count] @@".
func parseHunkHeader(line string) (oldStart, oldCount int) {
	// Find the "-" section between "@@ " and the next space.
	rest := strings.TrimPrefix(line, "@@ -")
	if rest == line {
		return 0, 0
	}
	part := strings.SplitN(rest, " ", 2)[0] // e.g. "10,5" or "10"
	nums := strings.SplitN(part, ",", 2)
	oldStart, _ = strconv.Atoi(nums[0])
	if len(nums) == 2 {
		oldCount, _ = strconv.Atoi(nums[1])
	} else {
		oldCount = 1
	}
	return
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
	// TODO: switch to palette color functions
	switch ct {
	case LineDeleted:
		return "\033[31m" + text + "\033[0m" // red
	case LineAdded:
		return "\033[32m" + text + "\033[0m" // green
	default:
		return text // gray not used; would be "\033[90m"
	}
}

