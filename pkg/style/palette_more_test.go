package style

import "testing"

func TestDefaultPaletteFieldsPopulated(t *testing.T) {
	p := DefaultPalette()
	roles := []string{
		"success", "error", "warning", "info", "dim",
		"table_status_add", "table_status_remove", "table_status_update",
		"diff_badge_add", "diff_badge_remove", "diff_provider", "diff_path",
	}
	for _, r := range roles {
		if v := p.Get(r); v == "" {
			t.Fatalf("expected palette role %s to be populated", r)
		}
	}
}

func TestStyleWrappersRender(t *testing.T) {
	// Ensure the commonly used style wrappers exist and render without panic.
	_ = Success.Render("ok")
	_ = Error.Render("err")
	_ = Warning.Render("warn")
	_ = DimStyle.Render("hint")
	_ = DiffBadgeAdd.Render("+add")
	_ = DiffBadgeRemove.Render("-rem")
	_ = DiffProvider.Render("(prov)")
	_ = DiffPath.Render("/path")
}
