package style

import "testing"

func TestGetSetColor(t *testing.T) {
	p := DefaultPalette()

	// Remember original value so we can assert change and not disrupt other tests
	orig := p.GetColor("success")
	newSeq := "\033[38;5;200m"

	p.SetColor("success", newSeq)
	if got := p.GetColor("success"); got != newSeq {
		t.Fatalf("expected success color to be %q, got %q", newSeq, got)
	}

	// Setting an unknown role should be a no-op and should not panic
	p.SetColor("unknown_role", "\033[31m")

	// Restore original to avoid cross-test interference
	p.SetColor("success", orig)
}
