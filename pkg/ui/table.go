package ui

import (
	"github.com/aquasecurity/table"
	"github.com/wasilak/dotisan/pkg/style"
	"io"
	"os"
)

// Table is a wrapper for aquasecurity/table.Table.
type Table struct {
	t *table.Table
}

// NewTable creates a Table that writes to the specified writer (default os.Stdout if nil).
func NewTable(w io.Writer) *Table {
	if w == nil {
		w = os.Stdout
	}
	tbl := table.New(w)
	// Use the library's StyleBlue which corresponds to SGR 34 — keep table
	// borders in the same visual family as the rest of the UI.
	tbl.SetLineStyle(table.StyleBlue)
	return &Table{t: tbl}
}

// SetHeaders sets the headers for the table (variadic, not slice).
func (t *Table) SetHeaders(headers ...string) {
	// Ensure headers are rendered with the palette's TableHeader style so
	// callers don't need to remember to apply styling.
	styled := make([]string, 0, len(headers))
	for _, h := range headers {
		styled = append(styled, style.TableHeader.Render(h))
	}
	t.t.SetHeaders(styled...)
}

// AddRow adds a row to the table (variadic, not slice).
func (t *Table) AddRow(cols ...string) {
	t.t.AddRow(cols...)
}

// Render renders the table to the underlying writer.
func (t *Table) Render() {
	t.t.Render()
}
