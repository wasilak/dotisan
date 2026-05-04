package graph

import "testing"

func TestResolveAddress_ResourceLevel(t *testing.T) {
	id, err := ResolveAddress("HomeBrewPackages/core-tools", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := NodeID("default/HomeBrewPackages/core-tools")
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

func TestResolveAddress_ItemLevel(t *testing.T) {
	id, err := ResolveAddress("HomeBrewPackages/core-tools[ripgrep]", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := NodeID("default/HomeBrewPackages/core-tools[ripgrep]")
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

func TestResolveAddress_WithNamespace(t *testing.T) {
	id, err := ResolveAddress("myns/GoPackages/tools", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := NodeID("myns/GoPackages/tools")
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

func TestResolveAddress_Empty(t *testing.T) {
	if _, err := ResolveAddress("", "default"); err == nil {
		t.Fatal("expected error for empty address")
	}
}

func TestResourceNodeID(t *testing.T) {
	got := ResourceNodeID("default", "GoPackages", "tools")
	want := NodeID("default/GoPackages/tools")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestItemNodeID(t *testing.T) {
	got := ItemNodeID("default", "HomeBrewPackages", "core", "ripgrep")
	want := NodeID("default/HomeBrewPackages/core[ripgrep]")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
