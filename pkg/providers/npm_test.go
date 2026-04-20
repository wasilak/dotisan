package providers

import (
	"context"
	"testing"

	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

func TestNewNpmProvider(t *testing.T) {
	p := NewNpmProvider()

	if p == nil {
		t.Fatal("NewNpmProvider() returned nil")
	}
}

func TestNpmProvider_Name(t *testing.T) {
	p := NewNpmProvider()

	if name := p.Name(); name != "npm" {
		t.Errorf("Name() = %q, want %q", name, "npm")
	}
}

func TestNpmProvider_Available(t *testing.T) {
	p := NewNpmProvider()

	available, msg := p.Available()

	// We can't know if npm is installed in test environment
	// Just verify the method runs without panic
	t.Logf("Available() = %v, %s", available, msg)

	if msg == "" {
		t.Error("Available() should return a message")
	}
}

func TestNpmProvider_Reconcile_Empty(t *testing.T) {
	p := NewNpmProvider()

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

func TestNpmProvider_Reconcile_Additions(t *testing.T) {
	p := NewNpmProvider()

	np := &resource.NpmPackages{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "NpmPackages",
			Metadata:   resource.Metadata{Name: "globals", Namespace: "default"},
		},
		Spec: resource.NpmPackagesSpec{
			Packages: []resource.Package{
				{Name: "typescript"},
				{Name: "prettier"},
			},
		},
	}

	desired := []resource.Resource{np}
	state := []provider.ResourceState{}

	// Note: This test may behave differently depending on npm availability
	plan := p.Reconcile(desired, state)

	t.Logf("Plan: %d additions, %d removals, %d in-sync",
		len(plan.Additions), len(plan.Removals), len(plan.InSync))
}

func TestNpmProvider_isPackageInstalled(t *testing.T) {
	p := NewNpmProvider()

	installed := map[string]string{
		"typescript": "5.4.0",
		"prettier":   "3.2.0",
	}

	tests := []struct {
		name     string
		expected bool
	}{
		{"typescript", true},
		{"prettier", true},
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.isPackageInstalled(tt.name, installed)
			if result != tt.expected {
				t.Errorf("isPackageInstalled(%q) = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestNpmProvider_Apply_ContextCancellation(t *testing.T) {
	p := NewNpmProvider()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	plan := provider.Plan{}
	err := p.Apply(ctx, plan)

	if err != nil {
		t.Logf("Apply() returned error for cancelled context (may be expected): %v", err)
	}
}
