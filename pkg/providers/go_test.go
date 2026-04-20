package providers

import (
	"context"
	"testing"

	"dotisan/pkg/provider"
	"dotisan/pkg/resource"
)

func TestNewGoProvider(t *testing.T) {
	p := NewGoProvider()

	if p == nil {
		t.Fatal("NewGoProvider() returned nil")
	}
}

func TestGoProvider_Name(t *testing.T) {
	p := NewGoProvider()

	if name := p.Name(); name != "go" {
		t.Errorf("Name() = %q, want %q", name, "go")
	}
}

func TestGoProvider_Available(t *testing.T) {
	p := NewGoProvider()

	available, msg := p.Available()

	t.Logf("Available() = %v, %s", available, msg)

	if msg == "" {
		t.Error("Available() should return a message")
	}
}

func TestGoProvider_Reconcile_Empty(t *testing.T) {
	p := NewGoProvider()

	desired := []resource.Resource{}
	state := []provider.ResourceState{}
	plan := p.Reconcile(desired, state)

	if len(plan.Additions) != 0 {
		t.Errorf("len(Additions) = %d, want 0", len(plan.Additions))
	}
	if len(plan.Removals) != 0 {
		t.Errorf("len(Removals) = %d, want 0", len(plan.Removals))
	}
}

func TestGoProvider_Reconcile_Additions(t *testing.T) {
	p := NewGoProvider()

	gp := &resource.GoPackages{
		BaseResource: resource.BaseResource{
			APIVersion: "dotisan/v1",
			Kind:       "GoPackages",
			Metadata:   resource.Metadata{Name: "tools", Namespace: "default"},
		},
		Spec: resource.GoPackagesSpec{
			Packages: []resource.GoPackage{
				{Module: "golang.org/x/tools/gopls", Version: "latest"},
				{Module: "github.com/air-verse/air"},
			},
		},
	}

	desired := []resource.Resource{gp}
	state := []provider.ResourceState{}

	plan := p.Reconcile(desired, state)

	t.Logf("Plan: %d additions, %d removals, %d in-sync",
		len(plan.Additions), len(plan.Removals), len(plan.InSync))
}

func TestGoProvider_isPackageInstalled(t *testing.T) {
	p := NewGoProvider()

	// Check common Go tools
	tests := []struct {
		name string
	}{
		{"go"},
		{"gofmt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installed := make(map[string]string)
			result := p.isPackageInstalled("/usr/local/go/bin/"+tt.name, installed)
			t.Logf("isPackageInstalled(%q) = %v", tt.name, result)
		})
	}
}

func TestGoProvider_Apply_ContextCancellation(t *testing.T) {
	p := NewGoProvider()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	plan := provider.Plan{}
	err := p.Apply(ctx, plan)

	if err != nil {
		t.Logf("Apply() returned error for cancelled context (may be expected): %v", err)
	}
}
