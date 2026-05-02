package diff

import (
	"testing"
)

// Sample minimal, non-UI test to ensure highlight works and does not error.
func TestHighlightUnifiedDiff(t *testing.T) {
	diffText := `--- old.go\n+++ new.go\n@@ -1,4 +1,5 @@\n-func Hello() int {\n-\treturn 1\n+func Hello(name string) string {\n+\treturn \"Hi, " + name\n }\n`
	out, err := HighlightUnifiedDiff(diffText, "github-dark")
	if err != nil {
		t.Fatalf("Highlight error: %v", err)
	}
	if len(out) == 0 {
		t.Errorf("Output was empty")
	}
	// Should contain ANSI escape code for red/green (\u001b[), at minimum
	if idx := findAnsi(out); idx == -1 {
		t.Logf("Output: %q", out)
		t.Error("Expected ANSI escape codes in output")
	}
}

// Helper: find first ESC sequence
func findAnsi(s string) int {
	for i := range s {
		if s[i] == '\u001b' {
			return i
		}
	}
	return -1
}
