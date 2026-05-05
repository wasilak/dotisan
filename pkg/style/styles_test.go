package style

import "testing"

func TestDefaultPaletteRoles(t *testing.T) {
	p := DefaultPalette()
	// Ensure core roles are populated and accessible via Get (avoid
	// depending on low-level color constants in tests).
	roles := []string{"success", "error", "info", "row_error"}
	for _, r := range roles {
		if v := p.Get(r); v == "" {
			t.Errorf("expected palette role %s to be populated", r)
		}
	}
}
