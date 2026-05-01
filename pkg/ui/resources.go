package ui

import "unicode/utf8"

// ResourceRow is the canonical input format for the human-facing resource table.
type ResourceRow struct {
	Status string // add/remove/update/drift/sync/warn/info
	ID     string // composite ID e.g. Kind/Group[Name]
	Kind   string
	Group  string
	Name   string
	Info   string
}

// RenderResourceTable renders a canonical resource table using the unified
// column layout shared by plan/apply/state list.
func RenderResourceTable(width int, rows []ResourceRow, showHeader bool) string {
	// Compute content-driven widths for fixed columns — never truncate.
	kindWidth := utf8.RuneCountInString("Kind")
	groupWidth := utf8.RuneCountInString("Group")
	nameWidth := utf8.RuneCountInString("Name")
	for _, r := range rows {
		if n := utf8.RuneCountInString(r.Kind); n > kindWidth {
			kindWidth = n
		}
		if n := utf8.RuneCountInString(r.Group); n > groupWidth {
			groupWidth = n
		}
		if n := utf8.RuneCountInString(r.Name); n > nameWidth {
			nameWidth = n
		}
	}

	// Columns: Status, ID, Kind, Group, Name, Info
	table := NewTable([]Column{
		{Title: "Status", Width: 3, Align: Center},
		{Title: "ID", Flex: true},
		{Title: "Kind", Width: kindWidth},
		{Title: "Group", Width: groupWidth},
		{Title: "Name", Width: nameWidth},
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
