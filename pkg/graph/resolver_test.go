package graph

import "testing"

func TestResolveAddress_ResourceLevel(t *testing.T) {
	id, err := ResolveAddress("HomeBrewPackages/core-tools")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := NodeID("HomeBrewPackages/core-tools")
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

func TestResolveAddress_ItemLevel(t *testing.T) {
	id, err := ResolveAddress("HomeBrewPackages/core-tools[ripgrep]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := NodeID("HomeBrewPackages/core-tools[ripgrep]")
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

func TestResolveAddress_NamespaceStripped(t *testing.T) {
	// Namespace prefix is accepted (backwards compat) but stripped from NodeID.
	id, err := ResolveAddress("myns/GoPackages/tools")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := NodeID("GoPackages/tools")
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

func TestResolveAddress_Empty(t *testing.T) {
	if _, err := ResolveAddress(""); err == nil {
		t.Fatal("expected error for empty address")
	}
}

func TestResourceNodeID(t *testing.T) {
	got := ResourceNodeID("GoPackages", "tools")
	want := NodeID("GoPackages/tools")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestItemNodeID(t *testing.T) {
	got := ItemNodeID("HomeBrewPackages", "core", "ripgrep")
	want := NodeID("HomeBrewPackages/core[ripgrep]")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
