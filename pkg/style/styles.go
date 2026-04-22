package style

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	Green  = "114"
	Red    = "174"
	Yellow = "222"
	Orange = "216"
	Gray   = "245"
	Blue   = "110"
)

const (
	IconAdd    = "✚"
	IconRemove = "✖"
	IconEdit   = "✎"
	IconWarn   = "⚠"
	IconOK     = "✓"
	IconInfo   = "ℹ"
)

var (
	Success    = lipgloss.NewStyle().Foreground(lipgloss.Color(Green))
	Error      = lipgloss.NewStyle().Foreground(lipgloss.Color(Red))
	Warning    = lipgloss.NewStyle().Foreground(lipgloss.Color(Orange))
	Info       = lipgloss.NewStyle().Foreground(lipgloss.Color(Yellow))
	Dim        = lipgloss.NewStyle().Foreground(lipgloss.Color(Gray))
	RowSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("77"))
	RowError   = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	RowWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("180"))
)

var (
	IconSuccess    = Success.Render(IconOK)
	IconError      = Error.Render(IconRemove)
	IconWarning    = Warning.Render(IconWarn)
	StyledIconInfo = Info.Render(IconInfo)
	StyledIconAdd  = Success.Render(IconAdd)
)

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

var (
	Header = lipgloss.NewStyle().Bold(true)
	Bold   = lipgloss.NewStyle().Bold(true)
)

var (
	TableHeader = lipgloss.NewStyle().Bold(true).Underline(true)
	TableRow    = lipgloss.NewStyle()
	TableCell   = lipgloss.NewStyle()
)

var (
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
		Cell: lipgloss.NewStyle().Padding(0, 1),
	}
)

type tableStyles struct {
	Header lipgloss.Style
	Row    lipgloss.Style
	RowAlt lipgloss.Style
	Cell   lipgloss.Style
}

func RenderTableRow(cols []string, widths []int, isAlt bool) string {
	style := TableStyles.Row
	if isAlt {
		style = TableStyles.RowAlt
	}
	var result string
	for i, col := range cols {
		w := widths[i]
		result += TableStyles.Cell.Render(padCol(col, w))
	}
	return style.Render(result)
}

func RenderTableHeader(cols []string, widths []int) string {
	var result string
	for i, col := range cols {
		result += TableStyles.Cell.Render(padCol(col, widths[i]))
	}
	return TableStyles.Header.Render(result)
}

func RenderTableBorder(widths []int) string {
	var line string
	for _, w := range widths {
		line += "+" + strings.Repeat("-", w+2)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(Gray)).Render(line + "+")
}

func padCol(s string, w int) string {
	runes := []rune(s)
	if len(runes) >= w {
		return string(runes[:w-3]) + "..."
	}
	return s + strings.Repeat(" ", w-len(runes))
}
