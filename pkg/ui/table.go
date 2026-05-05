package ui

import (
	"os"
	"io"
	"github.com/aquasecurity/table"
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
	return &Table{t: tbl}
}

// SetHeaders sets the headers for the table (variadic, not slice).
func (t *Table) SetHeaders(headers ...string) {
	t.t.SetHeaders(headers...)
}

// AddRow adds a row to the table (variadic, not slice).
func (t *Table) AddRow(cols ...string) {
	t.t.AddRow(cols...)
}

// Render renders the table to the underlying writer.
func (t *Table) Render() {
	t.t.Render()
}
