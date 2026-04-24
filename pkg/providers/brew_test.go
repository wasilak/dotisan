package providers

import (
	"context"
	"testing"

	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

func TestNewBrewProvider(t *testing.T) {
	p := NewBrewProvider()

	if p == nil {
		t.Fatal("NewBrewProvider() returned nil")
	}

	if p.httpClient == nil {
		t.Error("httpClient should be initialized")
	}
}

func TestBrewProvider_Name(t *testing.T) {
	p := NewBrewProvider()

	if name := p.Name(); name != "homebrew" {
		t.Errorf("Name() = %q, want %q", name, "homebrew")
	}
}

func TestBrewProvider_Available(t *testing.T) {
	p := NewBrewProvider()

	available, msg := p.Available()

	// We can't know if brew is installed in test environment
	// Just verify the method runs without panic
	t.Logf("Available() = %v, %s", available, msg)

	if msg == "" {
		t.Error("Available() should return a message")
	}
}

func TestBrewProvider_Reconcile_Empty(t *testing.T) {
	p := NewBrewProvider()

	// Reconcile with no desired resources
	desired := []resource.ResourceGroup{}
	state := []provider.ResourceState{}
	plan := p.Reconcile(desired, state)

	if len(plan.Additions) != 0 {
		t.Errorf("len(Additions) = %d, want 0", len(plan.Additions))
	}
	if len(plan.Removals) != 0 {
		t.Errorf("len(Removals) = %d, want 0", len(plan.Removals))
	}
}

func TestBrewProvider_Reconcile_Additions(t *testing.T) {
	p := NewBrewProvider()

	// Reconcile with desired resources
	desired := []resource.ResourceGroup{
		{
			Kind:  "BrewPackages",
			Name:  "core-tools",
			Items: []resource.ResourceItem{{Name: "ripgrep"}},
		},
	}
	state := []provider.ResourceState{}
	plan := p.Reconcile(desired, state)

	// Note: result depends on whether brew is installed
	// Just verify Reconcile runs without error
	t.Logf("Plan Additions: %d, Modifications: %d, Removals: %d, InSync: %d",
		len(plan.Additions), len(plan.Modifications), len(plan.Removals), len(plan.InSync))
}

func TestBrewProvider_Apply(t *testing.T) {
	p := NewBrewProvider()

	// Apply empty plan
	plan := provider.GroupPlan{}
	err := p.Apply(context.Background(), plan)

	// Should not error with empty plan
	if err != nil {
		t.Errorf("Apply() with empty plan error: %v", err)
	}
}

func TestBrewProvider_Import(t *testing.T) {
	p := NewBrewProvider()

	// Import should handle non-empty group name
	state, err := p.Import(context.Background(), "core-tools")
	if err != nil {
		t.Logf("Import() error (may be expected without brew): %v", err)
		return
	}

	if state.Kind != "BrewPackages" {
		t.Errorf("Import() Kind = %q, want BrewPackages", state.Kind)
	}
}

func TestBrewProvider_ImportItem(t *testing.T) {
	p := NewBrewProvider()

	// ImportItem should handle non-empty args
	state, err := p.ImportItem(context.Background(), "core-tools", "ripgrep")
	if err != nil {
		t.Logf("ImportItem() error (may be expected): %v", err)
		return
	}

	if state.Group != "core-tools" {
		t.Errorf("ImportItem() Group = %q, want core-tools", state.Group)
	}
}