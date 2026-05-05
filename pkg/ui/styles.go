// Package ui provides shared UI components and styling for dotisan CLI (pterm-based)
package ui

// All pterm color/style constants removed. TODO: migrate to palette (see palettes.go).

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
