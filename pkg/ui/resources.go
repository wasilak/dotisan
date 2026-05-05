package ui

import "github.com/wasilak/dotisan/pkg/style"

// ResourceRow is the canonical input format for the human-facing resource table.
type ResourceRow struct {
	Status string // add/remove/update/drift/sync/warn/info
	ID     string // composite ID e.g. Kind/Group[Name]
	Kind   string
	Group  string
	Name   string
	Info   string
}

// RenderResourceTable renders a resource table using palette-driven styling for status+header.
func RenderResourceTable(rows []ResourceRow, showHeader bool) error {
	tbl := NewTable(nil)
	styleHeader := func(s string) string { return style.TableHeader.Render(s) }
	statusStyle := func(status string) func(string) string {
		switch status {
		case "add":
			return style.Success.Render
		case "remove":
			return style.Error.Render
		case "update":
			return style.Warning.Render
		case "drift":
			return style.Warning.Render
		case "info":
			return style.Info.Render
		case "sync":
			return style.DimStyle.Render
		default:
			return func(s string) string { return s }
		}
	}
	if showHeader {
		tbl.SetHeaders(
			styleHeader("Status"),
			styleHeader("ID"),
			styleHeader("Kind"),
			styleHeader("Group"),
			styleHeader("Name"),
			styleHeader("Info"),
		)
	}
	for _, r := range rows {
		row := []string{
			statusStyle(r.Status)(r.Status),
			r.ID,
			r.Kind,
			r.Group,
			r.Name,
			r.Info,
		}
		tbl.AddRow(row...)
	}
	tbl.Render()
	return nil
}
