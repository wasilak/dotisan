package ui

// ResourceRow is the canonical input format for the human-facing resource table.
type ResourceRow struct {
	Status string // add/remove/update/drift/sync/warn/info
	ID     string // composite ID e.g. Kind/Group/Name
	Kind   string
	Group  string
	Name   string
	Info   string
}

// RenderResourceTable renders a canonical resource table using the unified
// column layout shared by plan/apply/state list.
func RenderResourceTable(width int, rows []ResourceRow, showHeader bool) string {
	// Columns: Status, ID, Kind, Group, Name, Info
	table := NewTable([]Column{
		{Title: "Status", Width: 3, Align: Center},
		{Title: "ID", Flex: true},
		{Title: "Kind", Width: 20},
		{Title: "Group", Width: 20},
		{Title: "Name", Width: 20},
		{Title: "Info", Flex: true},
	}, showHeader)

	out := make([]Row, 0, len(rows))
	for _, r := range rows {
		icon, style := StateIcon(r.Status)
		infoStyle := &InfoStyle
		// Build cells: status glyph, id, kind, group, name, info
		row := Row{
			Cell{Text: icon, Style: style},
			Cell{Text: r.ID},
			Cell{Text: r.Kind},
			Cell{Text: r.Group},
			Cell{Text: r.Name},
			Cell{Text: r.Info, Style: infoStyle},
		}
		out = append(out, row)
	}
	table.SetRows(out)
	return table.RenderPlain(width)
}
