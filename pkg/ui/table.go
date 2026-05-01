// Package ui: Bubbletea Table with Lipgloss charm theme and emoji state
package ui

import (
	"charm.land/lipgloss/v2"
	"strings"
	"unicode/utf8"
)

// Column defines table columns
// Flex allocates any leftover width proportionally
// Fixed uses given Width (if >0)
type Column struct {
	Title string
	Width int
	Align lipgloss.Position // Left/Center/Right
	Flex  bool
}

// Cell is a single table cell (styled text and optional style key)
type Cell struct {
	Text  string
	Style *lipgloss.Style
}
type Row []Cell

// Table holds all state (non-interactive plain output model)
type Table struct {
	Columns []Column
	Rows    []Row
	Header  bool // show header?
}

// NewTable creates a table with optional header
func NewTable(cols []Column, header bool) *Table {
	return &Table{Columns: cols, Header: header}
}

// SetRows sets table rows
func (t *Table) SetRows(rows []Row) { t.Rows = rows }

// RenderPlain renders the table to plain string (using Lipgloss for style)
func (t *Table) RenderPlain(width int) string {
	if width < 20 {
		width = 20
	} // minimum width
	// Calculate column widths
	used := 0
	flexCols, totalFlex := 0, 0
	colWidths := make([]int, len(t.Columns))
	for i, c := range t.Columns {
		if c.Flex {
			flexCols++
			totalFlex += 1 // count flex cols
		} else if c.Width > 0 {
			colWidths[i] = c.Width
			used += c.Width
		} else {
			colWidths[i] = 12 // fallback
			used += 12
		}
	}
	// Distribute remaining width among flex cols. One space between
	// columns is used as separator so subtract (n-1)*1
	remain := width - used - (len(t.Columns)-1)*1
	if remain < flexCols*8 {
		remain = flexCols * 8
	}
	flexWidth := remain / max(flexCols, 1)
	for i, c := range t.Columns {
		if c.Flex {
			colWidths[i] = max(flexWidth, 8)
		}
	}
	// Build lines
	var b strings.Builder
	if t.Header {
		for i, c := range t.Columns {
			text := c.Title
			style := HeaderStyle
			b.WriteString(renderCell(text, colWidths[i], c.Align, &style))
			if i != len(t.Columns)-1 {
				b.WriteString(" ")
			}
		}
		b.WriteString("\n")
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			cellStyle := cell.Style
			if cellStyle == nil {
				// Default to RowStyle (no background) so rows are transparent
				cellStyle = &RowStyle
			}
			// Render state column specially: show small colored glyph and leave
			// remaining cell transparent. This reproduces the thin status stripe
			// effect rather than full-cell backgrounds.
			// Render cell using renderCell for consistent alignment. State
			// glyphs/styles are foreground-only so this keeps the background
			// transparent while ensuring header/row alignment.
			b.WriteString(renderCell(cell.Text, colWidths[i], t.Columns[i].Align, cellStyle))
			if i != len(t.Columns)-1 {
				b.WriteString(" ")
			}
		}
		b.WriteString("\n")
	}

	// Wrap entire result with TableStyle so it has a border and background
	boxed := TableStyle.Render(b.String())
	return boxed
}
func renderCell(text string, width int, align lipgloss.Position, s *lipgloss.Style) string {
	// Truncate by rune count to avoid slicing in middle of UTF-8.
	// Only truncate when the text would exceed the available width.
	// Keep total visible characters <= width; reserve one rune for ellipsis when
	// truncating.
	runeCount := utf8.RuneCountInString(text)
	if runeCount > width {
		runes := []rune(text)
		if width <= 1 {
			// No room for content, show ellipsis (or empty for width==0)
			if width == 1 {
				text = "…"
			} else {
				text = ""
			}
		} else {
			text = string(runes[:width-1]) + "…"
		}
	}

	// Render the visible text with style (if any), but do not include padding
	// inside the styled section. This prevents styled backgrounds from filling
	// the entire cell and creating boxed columns.
	var renderedText string
	if s != nil {
		renderedText = s.Render(text)
	} else {
		renderedText = text
	}

	// Compute padding based on rune length of the visible (unstyled) text
	pad := width - utf8.RuneCountInString(text)
	if pad < 0 {
		pad = 0
	}
	switch align {
	case lipgloss.Center:
		l := pad / 2
		r := pad - l
		return strings.Repeat(" ", l) + renderedText + strings.Repeat(" ", r)
	case lipgloss.Right:
		return strings.Repeat(" ", pad) + renderedText
	default:
		return renderedText + strings.Repeat(" ", pad)
	}
}
func padAlign(text string, width int, align lipgloss.Position) string {
	pad := width - len(text)
	if pad < 0 {
		pad = 0
	}
	switch align {
	case lipgloss.Center:
		l := pad / 2
		r := pad - l
		return strings.Repeat(" ", l) + text + strings.Repeat(" ", r)
	case lipgloss.Right:
		return strings.Repeat(" ", pad) + text
	default:
		return text + strings.Repeat(" ", pad)
	}
}
func max(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

// (omitting Bubbletea Model interface for now since not used for plain CLI)

// StateIcon returns emoji + style for given action (add, remove, update, drift, sync)
func StateIcon(state string) (string, *lipgloss.Style) {
	switch state {
	case "add":
		return EmojiAdd, &StateAdd
	case "remove":
		return EmojiRemove, &StateRemove
	case "update", "modify", "change":
		return EmojiUpdate, &StateUpdate
	case "drift":
		return EmojiDrift, &StateDrift
	case "sync", "in_sync":
		return EmojiSync, &StateSync
	case "warn":
		return EmojiWarn, &WarnStyle
	case "info":
		return EmojiInfo, &InfoStyle
	default:
		return "?", nil
	}
}

// Example: usage in adapters/conversion functions
