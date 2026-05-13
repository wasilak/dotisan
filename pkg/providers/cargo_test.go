package providers

import (
	"context"
	"testing"

	"github.com/wasilak/nim/pkg/provider"
	"github.com/wasilak/nim/pkg/resource"
)

func TestNewCargoProvider(t *testing.T) {
	p := NewCargoProvider()

	if p == nil {
		t.Fatal("NewCargoProvider() returned nil")
	}
}

func TestCargoProvider_Name(t *testing.T) {
	p := NewCargoProvider()

	if name := p.Name(); name != "cargo" {
		t.Errorf("Name() = %q, want %q", name, "cargo")
	}
}

func TestCargoProvider_Available(t *testing.T) {
	p := NewCargoProvider()

	available, msg := p.Available()

	t.Logf("Available() = %v, %s", available, msg)

	if msg == "" {
		t.Error("Available() should return a message")
	}
}

func TestCargoProvider_Reconcile_Empty(t *testing.T) {
	p := NewCargoProvider()

	desired := []resource.ResourceGroup[any]{}
	state := []provider.ResourceState{}
	plan := p.Reconcile(context.Background(), desired, state)

	if len(plan.Additions) != 0 {
		t.Errorf("len(Additions) = %d, want 0", len(plan.Additions))
	}
	if len(plan.Removals) != 0 {
		t.Errorf("len(Removals) = %d, want 0", len(plan.Removals))
	}
}

func TestCargoProvider_Reconcile_Additions(t *testing.T) {
	p := NewCargoProvider()

	desired := []resource.ResourceGroup[any]{
		{
			Kind:  "CargoPackages",
			Name:  "dev-tools",
			Items: []resource.ResourceItem{{Name: "bat"}},
		},
	}
	state := []provider.ResourceState{}
	plan := p.Reconcile(context.Background(), desired, state)

	t.Logf("Plan Additions: %d, Modifications: %d, Removals: %d, InSync: %d",
		len(plan.Additions), len(plan.Modifications), len(plan.Removals), len(plan.InSync))
}

func TestCargoProvider_Apply(t *testing.T) {
	p := NewCargoProvider()

	plan := provider.GroupPlan{}
	_, err := p.Apply(context.Background(), plan)

	if err != nil {
		t.Errorf("Apply() with empty plan error: %v", err)
	}
}

func TestCargoProvider_Import(t *testing.T) {
	p := NewCargoProvider()

	state, err := p.Import(context.Background(), "dev-tools")
	if err != nil {
		t.Logf("Import() error (may be expected without cargo): %v", err)
		return
	}

	if state.Kind != "CargoPackages" {
		t.Errorf("Import() Kind = %q, want CargoPackages", state.Kind)
	}
}
