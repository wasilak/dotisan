package style

import "testing"

func TestDefaultPaletteRoles(t *testing.T) {
	p := DefaultPalette()
	if p.Get("success") != Green {
		t.Errorf("expected success to map to Green")
	}
	if p.Get("error") != Red {
		t.Errorf("expected error to map to Red")
	}
	if p.Get("info") != Yellow {
		t.Errorf("expected info to map to Yellow")
	}
	if p.Get("row_error") != RowRed {
		t.Errorf("expected row_error to map to RowRed")
	}
}
