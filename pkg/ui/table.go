// Package ui provides table rendering using pterm (migrated from Bubbletea/Lipgloss).
package ui

import (
	"github.com/pterm/pterm"
)

// RenderTable renders a table given a header row and data rows.
// header should be a slice of column names, data is a slice of rows (each a []string).
// Optionally, provide a slice of *pterm.Style for column-specific styling (pterm replaces prior Bubbletea/Lipgloss approach).
func RenderTable(header []string, data [][]string, colStyles []*pterm.Style) error {
	if len(header) == 0 {
		return nil
	}

	// Data as required by pterm [ [header...], [row1...], ... ]. Was normalized from Bubbletea/Lipgloss table formats.
	rows := make([][]string, 1+len(data))
	rows[0] = header
	copy(rows[1:], data)

	table := pterm.DefaultTable.WithHasHeader().WithData(rows)
	// Apply column styles to header row if provided
	if colStyles != nil && len(colStyles) == len(header) {
		styledHeader := make([]string, len(header))
		for i, h := range header {
			if colStyles[i] != nil {
				styledHeader[i] = colStyles[i].Sprint(h)
			} else {
				styledHeader[i] = h
			}
		}
		rows[0] = styledHeader
	}
	return table.Render()
}

// StateIcon returns the emoji for given action (add, remove, update, drift, sync/warn/info)
// and the pterm Style to use for state column coloring (previously Lipgloss styles, now all pterm).
func StateIcon(state string) (string, *pterm.Style) {
	switch state {
	case "add":
		return EmojiAdd, StateAdd
	case "remove":
		return EmojiRemove, StateRemove
	case "update", "modify", "change":
		return EmojiUpdate, StateUpdate
	case "drift":
		return EmojiDrift, StateDrift
	case "sync", "in_sync", "in-sync", "insync", "present":
		return EmojiSync, StateSync
	case "warn":
		return EmojiWarn, WarnStyle
	case "info":
		return EmojiInfo, InfoStyle
	default:
		return EmojiSync, StateSync
	}
}
