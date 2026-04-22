package providers

import (
	"context"
	"testing"

	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

func TestGoProvider_E2E_Lifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	p := NewGoProvider()
	ctx := context.Background()

	available, _ := p.Available()
	if !available {
		t.Skip("go not available on this system")
	}

	t.Run("Reconcile_Empty", func(t *testing.T) {
		plan := p.Reconcile([]resource.Resource{}, []provider.ResourceState{})
		if len(plan.Additions) != 0 || len(plan.Removals) != 0 {
			t.Errorf("empty reconcile should have no changes, got additions=%d, removals=%d", len(plan.Additions), len(plan.Removals))
		}
	})

	t.Run("DriftDetection", func(t *testing.T) {
		state := []provider.ResourceState{
			{
				ID:        "GoPackages/test",
				Kind:     "GoPackages",
				Name:     "test",
				Namespace: "default",
				Version:  "1.0.0",
				Extra: map[string]interface{}{
					"modules": map[string]interface{}{
						"github.com/nonexistent/pkg-xyz-123": "v1.0.0",
					},
				},
			},
		}
		desired := []resource.Resource{
			&resource.GoPackages{
				BaseResource: resource.BaseResource{
					Kind:     "GoPackages",
					Metadata: resource.Metadata{Name: "test", Namespace: "default"},
				},
				Spec: resource.GoPackagesSpec{
					Packages: []resource.GoPackage{
						{Module: "github.com/wasilak/dotisan", Version: "latest"},
					},
				},
			},
		}
		plan := p.Reconcile(desired, state)

		if len(plan.Warnings) == 0 {
			t.Error("drift detection should warn about modules in state but not installed")
		}
	})

	t.Run("Import", func(t *testing.T) {
		state, err := p.Import(ctx, "")
		if err != nil {
			t.Fatalf("Import failed: %v", err)
		}

		if state.ID != "GoPackages/global" {
			t.Errorf("Import ID = %q, want %q", state.ID, "GoPackages/global")
		}

		if state.Kind != "GoPackages" {
			t.Errorf("Import Kind = %q, want %q", state.Kind, "GoPackages")
		}

		modules, ok := state.Extra["modules"]
		if !ok {
			t.Fatal("Import Extra should contain 'modules' key")
		}
		mods, ok := modules.(map[string]string)
		if !ok {
			t.Fatal("modules should be map[string]string")
		}
		if len(mods) == 0 {
			t.Error("Import should return at least one module (the main module)")
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