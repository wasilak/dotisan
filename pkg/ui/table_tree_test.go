package ui

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/wasilak/nim/pkg/diff"
	"github.com/wasilak/nim/pkg/style"
)

// captureOutput temporarily redirects os.Stdout to a pipe and returns the
// captured output. Simple and used only for this test.
// captureStdout duplicates small capture helper but uses a unique name to
// avoid colliding with other test helpers in the package.
func captureStdout(f func()) string {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	_ = w.Close()
	outBytes, _ := io.ReadAll(r)
	os.Stdout = orig
	return string(outBytes)
}

func TestTableAndTreeUseTableBlue(t *testing.T) {
	// Use the palette value rather than a hard-coded escape sequence.
	esc := style.DefaultColors.NoChangesBorder

	// Capture table output
	outTable := captureStdout(func() {
		tbl := NewTable(nil)
		tbl.SetHeaders("status", "name")
		tbl.AddRow("present", "example/pkg")
		tbl.Render()
	})
	if !strings.Contains(outTable, esc) {
		t.Fatalf("expected table output to contain SGR 34 sequence %q, got: %q", esc, outTable)
	}

	// Capture tree output
	outTree := captureStdout(func() {
		tf := diff.NewTreeFormatter()
		resources := []diff.StateResource{
			{Kind: "Kind", Group: "grp", Items: []diff.StateItem{{Name: "example"}}},
		}
		if err := tf.FormatStateAsTree(resources); err != nil {
			t.Fatalf("unexpected error rendering tree: %v", err)
		}
	})
	if !strings.Contains(outTree, esc) {
		t.Fatalf("expected tree output to contain SGR 34 sequence %q, got: %q", esc, outTree)
	}
}
