package style

import (
	"testing"
)

func TestSetAndGetColor(t *testing.T) {
	t.Parallel()
	p := DefaultPalette()
	// Use an existing palette constant instead of a raw escape sequence.
	seq := Magenta
	p.SetColor("info", seq)
	if got := p.GetColor("info"); got != seq {
		t.Fatalf("expected info color %q, got %q", seq, got)
	}

	// Unknown set should be a no-op and not panic
	p.SetColor("__unknown__", "x")
}

func TestApplyPaletteAndRefresh(t *testing.T) {

	// Keep original to restore later
	orig := DefaultColors
	defer ApplyPalette(orig)

	// Apply new palette with a distinct Success sequence
	p := DefaultPalette()
	// Use a distinct existing constant for Success to verify ApplyPalette
	p.Success = Green
	ApplyPalette(p)

	// DefaultColors should reflect the new value
	if DefaultColors.Success != p.Success {
		t.Fatalf("DefaultColors.Success not updated: want %q got %q", p.Success, DefaultColors.Success)
	}

	// Ensure exported Style wrappers were refreshed: Success.Render should
	// include the new sequence as prefix.
	out := Success.Render("OK")
	if len(out) < len(p.Success) || out[:len(p.Success)] != p.Success {
		t.Fatalf("Success.Render did not use updated palette sequence: got %q", out)
	}
}

func TestApplyToDefaultsMerge(t *testing.T) {

	orig := DefaultColors
	defer ApplyPalette(orig)

	// Partial palette: only change Error
	p := ColorPalette{Error: Red}
	p.ApplyToDefaults()

	if DefaultColors.Error != p.Error {
		t.Fatalf("expected DefaultColors.Error to be updated to %q, got %q", p.Error, DefaultColors.Error)
	}
	// Other roles should remain from original defaults (e.g., Success)
	if DefaultColors.Success == "" {
		t.Fatalf("expected DefaultColors.Success to remain populated")
	}
}
