// Package ui provides shared UI components for the CLI.
//
// This file exposes a small set of emoji/icon constants used by older UI
// code. These are thin shims that reuse the canonical values from the
// style package so there aren't duplicated definitions across packages.
package ui

import "github.com/wasilak/dotisan/pkg/style"

var (
	// Prefer canonical values from pkg/style when available.
	EmojiAdd    = style.EmojiAdd    // e.g. ➕
	EmojiRemove = style.EmojiRemove // e.g. ➖
	EmojiUpdate = style.EmojiUpdate // e.g. ✎
	// UI-only simple ASCII markers retained locally
	EmojiDrift = "!"
	EmojiSync  = "✓"
	// Keep full emoji sequences where we previously used them
	EmojiWarn = "⚠️"
	EmojiInfo = style.EmojiInfo
)
