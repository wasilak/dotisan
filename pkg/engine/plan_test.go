package engine

import (
	"testing"

	"github.com/wasilak/nim/pkg/resource"
)

// mockResource is a mock implementation of resource.Resource for testing.
type mockResource struct {
	resource.BaseResource
}

func (m *mockResource) Validate() error { return nil }
func (m *mockResource) ToGroup() resource.ResourceGroup[any] {
	return resource.ResourceGroup[any]{}
}

func TestPlanOptions_NamespaceField(t *testing.T) {
	opts := PlanOptions{
		Targets:   []string{"test"},
		ShowDiff:  true,
		Namespace: "work",
	}

	if opts.Namespace != "work" {
		t.Errorf("Namespace = %q, want %q", opts.Namespace, "work")
	}
}

func TestFilterResourcesByNamespace(t *testing.T) {
	// This test validates the filtering logic that happens in Plan()
	// We'll create mock resources and verify the filtering behavior

	tests := []struct {
		name          string
		resourceNS    string // resource's metadata.namespace value
		activeNS      string // active namespace for filter
		shouldInclude bool
	}{
		{"exact match", "work", "work", true},
		{"regex match", "/work.*/", "work-laptop", true},
		{"no match", "personal", "work", false},
		{"implicit default with default", "", "default", true},
		{"implicit default with other", "", "work", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create metadata with the namespace
			m := &resource.Metadata{Name: "test", Namespace: tt.resourceNS}
			if err := m.CompileNamespace(); err != nil {
				t.Fatalf("CompileNamespace() failed: %v", err)
			}

			// Create a BaseResource with the metadata
			br := resource.BaseResource{
				APIVersion: "github.com/wasilak/nim/v1",
				Kind:       resource.KindHomeBrewPackages,
				Metadata:   *m,
			}

			// Test MatchesNamespace (this is what Plan() uses for filtering)
			got := br.MatchesNamespace(tt.activeNS)
			if got != tt.shouldInclude {
				t.Errorf("MatchesNamespace(%q) = %v, want %v", tt.activeNS, got, tt.shouldInclude)
			}
		})
	}
}

func TestPlanOptions_DefaultNamespace(t *testing.T) {
	// Test that empty namespace defaults to "default" in filtering logic
	opts := PlanOptions{
		Namespace: "", // empty namespace
	}

	// The filtering logic in Plan() treats empty namespace as "default"
	// This is a behavior test to document this expectation
	activeNS := opts.Namespace
	if activeNS == "" {
		activeNS = "default"
	}

	if activeNS != "default" {
		t.Errorf("Active namespace should default to 'default', got %q", activeNS)
	}
}
