package ui

import "github.com/pterm/pterm"

// ResourceRow is the canonical input format for the human-facing resource table (pterm-based).
type ResourceRow struct {
	Status string // add/remove/update/drift/sync/warn/info
	ID     string // composite ID e.g. Kind/Group[Name]
	Kind   string
	Group  string
	Name   string
	Info   string
}

// RenderResourceTable renders a canonical resource table using pterm (migrated from Bubbletea/Lipgloss) with a shared column layout.
func RenderResourceTable(rows []ResourceRow, showHeader bool) error {
	header := []string{"Status", "ID", "Kind", "Group", "Name", "Info"}
	data := make([][]string, 0, len(rows))
	// Prepare to collect column styles for the header
	colStyles := make([]*pterm.Style, len(header))
	colStyles[0] = HeaderStyle // Status header colored (optional)
	colStyles[5] = InfoStyle   // Info header colored (optional)

	for _, r := range rows {
		icon, statusStyle := StateIcon(r.Status)
		row := []string{
			statusStyle.Sprint(icon),
			r.ID,
			r.Kind,
			r.Group,
			r.Name,
			InfoStyle.Sprint(r.Info),
		}
		data = append(data, row)
	}
	// Only show header if requested
	if !showHeader {
		// pterm doesn't support hiding header row, logic present if header need must be faked
		// (could be extended for interactive modes if needed)
		header = nil
	}
	return RenderTable(header, data, colStyles)
}
