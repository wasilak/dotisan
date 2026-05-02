// Package ui provides shared UI components and styling for dotisan CLI (pterm-based)
package ui

import "github.com/pterm/pterm"

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

// pterm color definitions (formerly matched against Lipgloss palette)
var (
	// No direct background usage; pterm table/tree use their own defaults.
	ColorBg      = pterm.BgBlack
	ColorSurface = pterm.BgDarkGray  // fallback for subtle backgrounds
	ColorPrimary = pterm.FgLightBlue // sky blue
	ColorText    = pterm.FgWhite
	ColorMuted   = pterm.FgGray
	ColorBorder  = pterm.FgDarkGray
	ColorAdd     = pterm.FgLightGreen   // mint green
	ColorRemove  = pterm.FgLightRed     // coral
	ColorUpdate  = pterm.FgLightYellow  // amber
	ColorDrift   = pterm.FgLightMagenta // violet
	ColorInfo    = pterm.FgLightCyan
	ColorWarning = pterm.FgLightYellow
)

// pterm styles for table rendering and icons
var (
	// Header: Bold, primary text color
	HeaderStyle = &pterm.Style{ColorPrimary, pterm.Bold}

	// Row: Normal, white/gray text
	RowStyle = &pterm.Style{ColorText}

	// Zebra/odd-even: empty for now (pterm supports row-level color if needed)
	ZebraEven = &pterm.Style{}
	ZebraOdd  = &pterm.Style{}

	StateAdd    = &pterm.Style{ColorAdd, pterm.Bold}
	StateRemove = &pterm.Style{ColorRemove, pterm.Bold}
	StateUpdate = &pterm.Style{ColorUpdate, pterm.Bold}
	StateDrift  = &pterm.Style{ColorDrift, pterm.Bold}
	StateSync   = &pterm.Style{ColorAdd, pterm.Bold}

	InfoStyle = &pterm.Style{ColorMuted}
	WarnStyle = &pterm.Style{ColorWarning}
)

// TableStyle removed; use pterm.DefaultTable for table borders, padding, backgrounds, etc. (all pterm-native now)
// See /pkg/ui/table.go for structural changes to table rendering.

// Fallback: pterm handles 256-color/truecolor automatically, so the old mapping is now obsolete.
// If a specific fallback is needed for ultra-basic terminals, define here:
// var FgFallback = map[string]*pterm.Style{...}
