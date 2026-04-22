package style

import (
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

// Color constants (ANSI 256-color palette)
const (
	Green  = "114"
	Red    = "174"
	Yellow = "222"
	Orange = "216"
	Gray   = "245"
	Blue   = "110"

	// Row colors (lighter for better visibility on dark bg)
	RowGreen  = "77"
	RowRed    = "204"
	RowYellow = "180"
)

// Icon constants
const (
	IconAdd    = "✚"
	IconRemove = "✖"
	IconEdit  = "✎"
	IconWarn = "⚠"
	IconOK   = "✓"
	IconInfo = "ℹ"
)

// Base text styles
var (
	Success    = lipgloss.NewStyle().Foreground(lipgloss.Color(Green))
	Error      = lipgloss.NewStyle().Foreground(lipgloss.Color(Red))
	Warning    = lipgloss.NewStyle().Foreground(lipgloss.Color(Orange))
	Info      = lipgloss.NewStyle().Foreground(lipgloss.Color(Yellow))
	Dim       = lipgloss.NewStyle().Foreground(lipgloss.Color(Gray))
	RowSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color(RowGreen))
	RowError  = lipgloss.NewStyle().Foreground(lipgloss.Color(RowRed))
	RowWarning = lipgloss.NewStyle().Foreground(lipgloss.Color(RowYellow))
)

// Prerendered icons
var (
	IconSuccess    = Success.Render(IconOK)
	IconError      = Error.Render(IconRemove)
	IconWarning   = Warning.Render(IconWarn)
	StyledIconInfo = Info.Render(IconInfo)
	StyledIconAdd = Success.Render(IconAdd)
)

// Header styles
var (
	Header = lipgloss.NewStyle().Bold(true)
	Bold   = lipgloss.NewStyle().Bold(true)
)

// Box styles with borders
var (
	SuccessBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(Green)).
		Padding(1, 2)

	ErrorBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(Red)).
		Padding(1, 2)

	WarningBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(Orange)).
		Padding(1, 2)

	InfoBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(Yellow)).
		Padding(1, 2)
)

// Table styles with width constraints
var (
	TableHeader = lipgloss.NewStyle().
		Bold(true).
		Underline(true)

	TableRow  = lipgloss.NewStyle()
	TableCell = lipgloss.NewStyle()

	TableBorder = lipgloss.RoundedBorder()
	TableStyles = tableStyles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(Blue)).
			Border(TableBorder).
			BorderTop(false).
			BorderBottom(false).
			Padding(0, 1),
		Row: lipgloss.NewStyle().
			Border(TableBorder).
			BorderTop(false).
			BorderBottom(false).
			Padding(0, 1),
		RowAlt: lipgloss.NewStyle().
			Border(TableBorder).
			BorderTop(false).
			BorderBottom(false).
			Padding(0, 1),
		Cell: lipgloss.NewStyle().
			Padding(0, 1).
			Width(30),
	}
)

// Plan-specific styles with icons
var (
	PlanIconSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color(Green))
	PlanIconError   = lipgloss.NewStyle().Foreground(lipgloss.Color(Red))
	PlanIconWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color(Orange))
	PlanIconInfo   = lipgloss.NewStyle().Foreground(lipgloss.Color(Yellow))

	PlanSection = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(Blue)).
		MarginBottom(1)

	PlanSummary = lipgloss.NewStyle().Bold(true)
)

type tableStyles struct {
	Header lipgloss.Style
	Row    lipgloss.Style
	RowAlt lipgloss.Style
	Cell   lipgloss.Style
}

// RenderTableRow renders a table row with cells.
func RenderTableRow(cols []string, widths []int, isAlt bool) string {
    st := TableStyles.Row
    if isAlt {
        st = TableStyles.RowAlt
    }
    var result string
    for i, col := range cols {
        // Let lipgloss handle width, padding and truncation so measurements stay consistent.
        // In v2, methods return new styles, so we can chain directly.
        cellStyle := TableStyles.Cell.Width(widths[i])
        result += cellStyle.Render(col)
    }
    return st.Render(result)
}

// RenderTableHeader renders the table header.
func RenderTableHeader(cols []string, widths []int) string {
    var result string
    for i, col := range cols {
        // Use the cell style with the requested width to ensure
        // the rendered header cell and the border measurement match exactly.
        // In v2, methods return new styles, so we can chain directly.
        cellStyle := TableStyles.Cell.Width(widths[i])
        result += cellStyle.Render(col)
    }
    return TableStyles.Header.Render(result)
}

// RenderTableBorder renders the table border line.
func RenderTableBorder(widths []int) string {
    var line string
    // For each column, render a sample cell at the desired width and measure
    // its visible width using lipgloss.Width. This avoids fragile assumptions
    // about padding/truncation and keeps the border aligned with rendered cells.
    for _, w := range widths {
        // In v2, methods return new styles, so we can chain directly.
        cellStyle := TableStyles.Cell.Width(w)
        // Render a sequence of characters of length w to force the cell to
        // occupy the intended width; then measure the visible width including
        // padding applied by the style.
        sample := strings.Repeat("X", w)
        rendered := cellStyle.Render(sample)
        segWidth := lipgloss.Width(rendered)
        if segWidth <= 0 {
            // Fallback to the old heuristic if measurement fails for some reason.
            segWidth = w + 2
        }
        line += "+" + strings.Repeat("-", segWidth)
    }
    return lipgloss.NewStyle().Foreground(lipgloss.Color(Gray)).Render(line + "+")
}

// padCol pads a column value to fit the width.
// Uses lipgloss.Width() equivalent for proper width calculation.
func padCol(s string, w int) string {
	runes := []rune(s)
	if len(runes) >= w {
		return string(runes[:w-3]) + "..."
	}
	return s + strings.Repeat(" ", w-len(runes))
}

// WrapText wraps text to a given width using lipgloss.
func WrapText(text string, width int) string {
	return lipgloss.NewStyle().Width(width).Render(text)
}

// Indent indents text by the given number of spaces.
func Indent(text string, spaces int) string {
	indent := strings.Repeat(" ", spaces)
	return indent + text
}

// Truncate truncates text to maxWidth with ellipsis.
func Truncate(text string, maxWidth int) string {
	runes := []rune(text)
	if len(runes) <= maxWidth {
		return text
	}
	if maxWidth < 3 {
		return strings.Repeat(".", maxWidth)
	}
	return string(runes[:maxWidth-3]) + "..."
}
