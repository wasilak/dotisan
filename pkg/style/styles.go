package style

import (
	"github.com/pterm/pterm"
)

// Color constants
// Use pterm named colors for primary accents (green, red, yellow, blue) for clarity.
// Custom palette indices used for non-standard tints and row alternates, preserving prior visual fidelity.
const (
	Green  = pterm.FgGreen  // Standard green
	Red    = pterm.FgRed    // Standard red
	Yellow = pterm.FgYellow // Standard yellow
	Blue   = pterm.FgBlue   // Standard blue

	Orange = pterm.Color(216) // Orange: not in named set, preserved index
	Gray   = pterm.Color(245) // Gray: lighter gray for dim style

	// Row colors (lighter for better visibility on dark bg)
	RowGreen  = pterm.Color(77)  // Custom tint, not a named pterm color
	RowRed    = pterm.Color(204) // Custom tint
	RowYellow = pterm.Color(180) // Custom tint
)

// Icon constants
const (
	IconAdd    = "✚"
	IconRemove = "✖"
	IconEdit   = "✎"
	IconWarn   = "⚠"
	IconOK     = "✓"
	IconInfo   = "ℹ"
	IconTrash  = "🗑️"
)

// Style wraps pterm.Style and provides a Render() method (formerly lipgloss-compatible, now pterm-native).
// All callers of the style package use .Render() — this wrapper preserves that API.
type Style struct {
	style *pterm.Style
}

// NewStyle creates a new Style from pterm colors.
// Accepts pterm.Color values like pterm.FgGreen, pterm.Bold, Green, etc.
func NewStyle(colors ...pterm.Color) Style {
	return Style{style: pterm.NewStyle(colors...)}
}

// Render formats text with the style (was compatible with lipgloss .Render(); now pterm-native).
func (s Style) Render(text string) string {
	if s.style == nil {
		return text
	}
	return s.style.Sprint(text)
}

// Base text styles
var (
	Success    = NewStyle(Green)
	Error      = NewStyle(Red)
	Warning    = NewStyle(Orange)
	Info       = NewStyle(Yellow)
	Dim        = NewStyle(Gray)
	RowSuccess = NewStyle(RowGreen)
	RowError   = NewStyle(RowRed)
	RowWarning = NewStyle(RowYellow)
)

// Prerendered icons
var (
	IconSuccess    = Success.Render(IconOK)
	IconError      = Error.Render(IconRemove)
	IconWarning    = Warning.Render(IconWarn)
	StyledIconInfo = Info.Render(IconInfo)
	StyledIconAdd  = Success.Render(IconAdd)
	IconTrashBin   = Dim.Render(IconTrash)
)

// Styles for rendering full item lines (icon + text)
var (
	ItemSuccess = Success
	ItemError   = Error
	ItemWarning = Warning
	ItemInfo    = Info
)

// Header styles
var (
	Header = NewStyle(pterm.Bold)
	Bold   = NewStyle(pterm.Bold)
)

// Box styles — simplified to text styles (pterm replaces prior lipgloss border/box API)
var (
	SuccessBox = NewStyle(pterm.FgGreen)
	ErrorBox   = NewStyle(pterm.FgRed)
	WarningBox = NewStyle(pterm.FgYellow)
	InfoBox    = NewStyle(pterm.FgYellow)
)

// Table styles with width constraints
var (
	TableHeader = NewStyle(pterm.Bold) // pterm has no underline support
	TableRow    = NewStyle()
	TableCell   = NewStyle()
)

// TableBorder is kept for source compatibility but is a no-op (pterm has no border styles)
var TableBorder = ""

// tableStyles defines the styling for table components.
// Uses Style type so callers can use .Render().
type tableStyles struct {
	Header Style
	Row    Style
	RowAlt Style
	Cell   Style
}

// TableStyles is the default table styling
var TableStyles = tableStyles{
	Header: NewStyle(pterm.Bold, Blue),
	Row:    NewStyle(),
	RowAlt: NewStyle(),
	Cell:   NewStyle(),
}

// Plan-specific styles with icons
var (
	PlanIconSuccess = NewStyle(Green)
	PlanIconError   = NewStyle(Red)
	PlanIconWarn    = NewStyle(Orange)
	PlanIconInfo    = NewStyle(Yellow)

	PlanSection = NewStyle(pterm.Bold, Blue)
	PlanSummary = NewStyle(pterm.Bold)
)
