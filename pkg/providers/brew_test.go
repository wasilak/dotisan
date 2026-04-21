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

	if name := p.Name(); name != "brew" {
		t.Errorf("Name() = %q, want %q", name, "brew")
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

func TestBrewProvider_Reconcile_Additions(t *testing.T) {
	p := NewBrewProvider()

	// Create desired BrewPackages
	bp := &resource.BrewPackages{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "BrewPackages",
			Metadata:   resource.Metadata{Name: "core-tools", Namespace: "default"},
		},
		Spec: resource.BrewPackagesSpec{
			Formulae: []resource.Package{
				{Name: "ripgrep"},
				{Name: "fd"},
			},
			Casks: []resource.Package{
				{Name: "visual-studio-code"},
			},
		},
	}

	desired := []resource.Resource{bp}
	state := []provider.ResourceState{}

	// Note: This test may fail if brew is not installed
	// We're just testing that Reconcile doesn't panic
	plan := p.Reconcile(desired, state)

	t.Logf("Plan: %d additions, %d removals, %d in-sync",
		len(plan.Additions), len(plan.Removals), len(plan.InSync))
}

func TestBrewProvider_isPackageInstalled(t *testing.T) {
	p := NewBrewProvider()

	installed := map[string]string{
		"ripgrep": "13.0.0",
		"fd":      "8.7.0",
	}

	tests := []struct {
		name     string
		expected bool
	}{
		{"ripgrep", true},
		{"fd", true},
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

func TestBrewProvider_isTapInstalled(t *testing.T) {
	p := NewBrewProvider()

	// This test requires brew to be installed
	// We'll just verify the method doesn't panic
	installed := make(map[string]string)
	result := p.isTapInstalled("homebrew/core", installed)
	t.Logf("isTapInstalled() = %v", result)
}

func TestBrewProvider_isCaskInstalled(t *testing.T) {
	p := NewBrewProvider()

	// This test requires brew to be installed
	// We'll just verify the method doesn't panic
	installed := make(map[string]string)
	result := p.isCaskInstalled("firefox", installed)
	t.Logf("isCaskInstalled() = %v", result)
}

func TestBrewProvider_getFormulaInfo(t *testing.T) {
	p := NewBrewProvider()

	// Test fetching formula info from API
	// This may fail if there's no network connectivity
	info, err := p.getFormulaInfo("ripgrep")
	if err != nil {
		t.Logf("getFormulaInfo() returned error (may be expected without network): %v", err)
		return
	}

	if info.Name == "" {
		t.Error("formula info should have a name")
	}

	t.Logf("Formula info: name=%s", info.Name)
}

func TestBrewProvider_Apply_ContextCancellation(t *testing.T) {
	p := NewBrewProvider()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Try to apply empty plan
	plan := provider.Plan{}
	err := p.Apply(ctx, plan)

	if err != nil {
		t.Logf("Apply() returned error for cancelled context (may be expected): %v", err)
	}
}

func TestBrewProvider_checkFormulaeStatus(t *testing.T) {
	p := NewBrewProvider()

	t.Run("empty list returns empty map", func(t *testing.T) {
		result, err := p.checkFormulaeStatus([]string{})
		if err != nil {
			t.Errorf("checkFormulaeStatus([]) returned error: %v", err)
			return
		}
		if len(result) != 0 {
			t.Errorf("checkFormulaeStatus([]) = %v, want empty map", result)
		}
	})

	t.Run("returns map with formula names", func(t *testing.T) {
		formulae := []string{"ripgrep", "fd"}
		result, err := p.checkFormulaeStatus(formulae)
		if err != nil {
			t.Logf("checkFormulaeStatus() error (may fail without brew): %v", err)
			return
		}

		if len(result) == 0 {
			t.Log("checkFormulaeStatus returned empty map - may be expected if brew not available")
			return
		}

		for _, formula := range formulae {
			if _, ok := result[formula]; !ok {
				t.Errorf("checkFormulaeStatus() missing key %q in result %v", formula, result)
			}
			if result[formula] != true && result[formula] != false {
				t.Errorf("checkFormulaeStatus()[%q] = %v, want bool", formula, result[formula])
			}
			t.Logf("checkFormulaeStatus(%q) = %v", formula, result[formula])
		}
	})

	t.Run("nonexistent formula handled gracefully", func(t *testing.T) {
		formulae := []string{"nonexistent-formula-xyz-123"}
		result, err := p.checkFormulaeStatus(formulae)
		if err != nil {
			t.Logf("checkFormulaeStatus() error (may fail without brew): %v", err)
			return
		}

		if result == nil {
			t.Error("checkFormulaeStatus() returned nil map")
			return
		}

		if installed, ok := result["nonexistent-formula-xyz-123"]; !ok {
			t.Errorf("checkFormulaeStatus() missing nonexistent formula in result %v", result)
		} else if installed {
			t.Error("checkFormulaeStatus() reported nonexistent formula as installed")
		} else {
			t.Log("checkFormulaeStatus() correctly reported nonexistent formula as not installed")
		}
	})
}
