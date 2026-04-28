package style

import (
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

// Styles for rendering full item lines (icon + text)
var (
	ItemSuccess  = Success
	ItemError   = Error
	ItemWarning = Warning
	ItemInfo   = Info
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
