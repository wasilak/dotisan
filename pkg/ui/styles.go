// Package ui provides shared UI components and styling for dotisan CLI (Bubbletea, Lipgloss charm theme)
package ui

import "charm.land/lipgloss/v2"

// Emoji/state icon definitions. Keep simple ASCII for state markers to match
// historical UI (+, -, ~). Keep info/warn emoji.
var (
	EmojiAdd    = "+"
	EmojiRemove = "-"
	EmojiUpdate = "~"
	EmojiDrift  = "!"
	EmojiSync   = "✓"
	EmojiWarn   = "⚠️"
	EmojiInfo   = "🛈"
)

// Charm dark theme colors (hex)
var (
	ColorBg      = lipgloss.Color("#0b1220")
	ColorSurface = lipgloss.Color("#0f1724")
	ColorPrimary = lipgloss.Color("#7aa2f7") // sky blue
	ColorText    = lipgloss.Color("#e6eef6")
	ColorMuted   = lipgloss.Color("#94a3b8")
	ColorBorder  = lipgloss.Color("#1f2a44")
	ColorAdd     = lipgloss.Color("#7ee787") // mint green
	ColorRemove  = lipgloss.Color("#ff6b6b") // coral
	ColorUpdate  = lipgloss.Color("#ffd27f") // amber
	ColorDrift   = lipgloss.Color("#c77dff") // violet
	ColorInfo    = lipgloss.Color("#66b0ff")
	ColorWarning = lipgloss.Color("#ffb86b")
)

// Lipgloss styles for table rendering and icons
var (
	// Header and row styles share the surface background so the table blends
	// with the rest of the app. Avoid extra padding which caused column
	// misalignment with the Table rendering logic.
	// Use the global app background so the table blends into the page.
	// HeaderStyle: only set foreground (no background) so the header text sits
	// directly on the app background. Removing background avoids boxed cells.
	HeaderStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// RowStyle: only foreground, no background. This gives transparent rows
	// that match the application background and avoids the visible box art.
	RowStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	// No zebra backgrounds; keep styles empty so rows render transparently.
	ZebraEven = lipgloss.NewStyle()
	ZebraOdd  = lipgloss.NewStyle()

	StateAdd    = lipgloss.NewStyle().Foreground(ColorAdd).Bold(true)
	StateRemove = lipgloss.NewStyle().Foreground(ColorRemove).Bold(true)
	StateUpdate = lipgloss.NewStyle().Foreground(ColorUpdate).Bold(true)
	StateDrift  = lipgloss.NewStyle().Foreground(ColorDrift).Bold(true)
	// Use green for synced/present resources so they appear as success.
	StateSync = lipgloss.NewStyle().Foreground(ColorAdd).Bold(true)

	InfoStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	WarnStyle = lipgloss.NewStyle().Foreground(ColorWarning)
)

// TableStyle is the outer box style used to render the entire table. It
// provides a uniform surface background and border like the Bubbletea example
// while individual rows remain transparent (no per-row background).
var TableStyle = lipgloss.NewStyle().
	Background(ColorSurface).
	Border(lipgloss.NormalBorder()).
	BorderForeground(ColorBorder).
	Padding(0, 1)

// 256-color fallback mapping (for basic terminals)
// This is available but not yet used; could be picked up dynamically if necessary.
var (
	FgFallback = map[string]lipgloss.Style{
		"add":     lipgloss.NewStyle().Foreground(lipgloss.Color("42")),  // 7ee787 mint green ~ term green
		"remove":  lipgloss.NewStyle().Foreground(lipgloss.Color("203")), // ff6b6b coral ~ term red
		"update":  lipgloss.NewStyle().Foreground(lipgloss.Color("222")), // ffd27f amber ~ yellow
		"drift":   lipgloss.NewStyle().Foreground(lipgloss.Color("135")), // c77dff violet ~ magenta
		"sync":    lipgloss.NewStyle().Foreground(lipgloss.Color("244")), // muted grey
		"primary": lipgloss.NewStyle().Foreground(lipgloss.Color("75")),  // sky blue
		"info":    lipgloss.NewStyle().Foreground(lipgloss.Color("33")),  // info blue
		"warning": lipgloss.NewStyle().Foreground(lipgloss.Color("215")), // warning amber
	}
)
