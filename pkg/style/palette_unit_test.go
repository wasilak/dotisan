package style

import "testing"

func TestGetUnknownRoleReturnsEmpty(t *testing.T) {
	p := DefaultPalette()
	if got := p.GetColor("__does_not_exist__"); got != "" {
		t.Fatalf("expected unknown role to return empty string, got %q", got)
	}
}

func TestGetRainbowCycle(t *testing.T) {
	p := DefaultPalette()
	if len(p.NoChangesRainbow) == 0 {
		t.Skip("no rainbow colors configured")
	}
	first := p.GetRainbow(0)
	// ensure cycling
	l := len(p.NoChangesRainbow)
	if got := p.GetRainbow(l); got != first {
		t.Fatalf("expected rainbow to cycle back after %d entries", l)
	}
}
