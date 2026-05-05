package ui

import (
	"bytes"
	"github.com/aquasecurity/table"
	"github.com/wasilak/dotisan/pkg/style"
	"io"
	"os"
)

// Table is a wrapper for aquasecurity/table.Table.
type Table struct {
	t *table.Table
	w *capturingWriter
}

// NewTable creates a Table that writes to the specified writer (default os.Stdout if nil).
func NewTable(w io.Writer) *Table {
	if w == nil {
		w = os.Stdout
	}
	// Wrap the output writer so we can post-process the rendered table (for
	// example, replace the library's numeric SGR border color with our
	// truecolor Border sequence). We capture output in a buffer and then
	// flush processed bytes to the original writer on Render().
	cw := &capturingWriter{out: w}
	tbl := table.New(cw)
	// Use bright magenta for table lines — approximate the NoChangesBorder
	// purple. aquasecurity/table uses numeric SGR codes for line styling, so
	// pick the closest available constant.
	// Use a darker blue-ish line style (StyleBlue) as a closer fit to the
	// desired dark purple when rendered on many terminals. The table library
	// only accepts these numeric SGR constants.
	tbl.SetLineStyle(table.StyleBlue)
	return &Table{t: tbl, w: cw}
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
	// Render into the capturing writer's buffer
	t.t.Render()
	if t.w == nil {
		return
	}
	s := t.w.buf.String()
	// Replace border glyphs with the exact Border color from our palette.
	var b bytes.Buffer
	for _, r := range s {
		switch r {
		case '─', '│', '├', '└', '┘', '┌', '┐', '┤', '┬', '┴', '┼':
			b.WriteString(style.Border.Render(string(r)))
		default:
			b.WriteRune(r)
		}
	}
	// Reset the capture buffer and write processed output to underlying writer
	t.w.buf.Reset()
	t.w.out.Write(b.Bytes())
}

// capturingWriter buffers writes so we can post-process table output.
type capturingWriter struct {
	buf bytes.Buffer
	out io.Writer
}

func (c *capturingWriter) Write(p []byte) (int, error) {
	return c.buf.Write(p)
}
