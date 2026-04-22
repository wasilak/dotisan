package providers

import (
	"context"
	"testing"

	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

func TestNpmProvider_E2E_Lifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	p := NewNpmProvider()
	ctx := context.Background()

	available, _ := p.Available()
	if !available {
		t.Skip("npm not available on this system")
	}

	t.Run("FullLifecycle", func(t *testing.T) {
		plan := p.Reconcile(nil, nil)
		_ = plan.Additions // Verify no panic occurred
	})

	t.Run("Reconcile_Empty", func(t *testing.T) {
		plan := p.Reconcile([]resource.Resource{}, []provider.ResourceState{})
		if len(plan.Additions) != 0 || len(plan.Removals) != 0 {
			t.Errorf("empty reconcile should have no changes, got additions=%d, removals=%d", len(plan.Additions), len(plan.Removals))
		}
	})

	t.Run("DriftDetection", func(t *testing.T) {
		state := []provider.ResourceState{
			{
				ID:        "NpmPackages/test",
				Kind:     "NpmPackages",
				Name:     "test",
				Namespace: "default",
				Version:  "1.0.0",
				Extra: map[string]interface{}{
					"packages": map[string]interface{}{
						"nonexistent-pkg-xyz-123": "1.0.0",
					},
				},
			},
		}
		desired := []resource.Resource{
			&resource.NpmPackages{
				BaseResource: resource.BaseResource{
					Kind:     "NpmPackages",
					Metadata: resource.Metadata{Name: "test", Namespace: "default"},
				},
				Spec: resource.NpmPackagesSpec{
					Packages: []resource.Package{
						{Name: "typescript", Version: "latest"},
					},
				},
			},
		}
		plan := p.Reconcile(desired, state)

		if len(plan.Warnings) == 0 {
			t.Error("drift detection should warn about packages in state but not installed")
		}
	})

	t.Run("Import", func(t *testing.T) {
		state, err := p.Import(ctx, "")
		if err != nil {
			t.Fatalf("Import failed: %v", err)
		}

		if state.ID != "NpmPackages/global" {
			t.Errorf("Import ID = %q, want %q", state.ID, "NpmPackages/global")
		}

		if state.Kind != "NpmPackages" {
			t.Errorf("Import Kind = %q, want %q", state.Kind, "NpmPackages")
		}

		packages, ok := state.Extra["packages"]
		if !ok {
			t.Fatal("Import Extra should contain 'packages' key")
		}
		pkgs, ok := packages.(map[string]string)
		if !ok {
			t.Fatal("packages should be map[string]string")
		}
		if len(pkgs) == 0 {
			t.Error("Import should return at least one package (self)")
		}
	})

	t.Run("Apply_ContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		plan := provider.Plan{}
		err := p.Apply(ctx, plan)
		if err != nil {
			t.Errorf("Apply with cancelled context should not error, got: %v", err)
		}
	})
}