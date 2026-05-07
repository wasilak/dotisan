package provider

import (
	"context"
	"testing"

	"github.com/wasilak/dotisan/pkg/resource"
)

// mockProvider is a test provider implementation.
type mockProvider struct {
	name      string
	available bool
	message   string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Available() (bool, string) {
	return m.available, m.message
}

func (m *mockProvider) Reconcile(ctx context.Context, desired []resource.ResourceGroup[any], state []ResourceState) GroupPlan {
	return GroupPlan{}
}

func (m *mockProvider) Apply(ctx context.Context, plan GroupPlan) error {
	return nil
}

func (m *mockProvider) Import(ctx context.Context, group string) (ResourceState, error) {
	return ResourceState{Kind: "test", Group: group}, nil
}

// Note: provider-level ImportItem removed; Import should return a group state.

func TestRegisterAndGet(t *testing.T) {
	// Create a fresh registry for testing
	reg := &Registry{
		providers: make(map[string]Provider),
	}

	p := &mockProvider{name: "test", available: true}

	// Register
	err := reg.Register("test", p)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	// Get
	got, err := reg.Get("test")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if got.Name() != "test" {
		t.Errorf("Get() name = %q, want %q", got.Name(), "test")
	}

	// Duplicate registration should fail
	err = reg.Register("test", p)
	if err == nil {
		t.Error("Register() duplicate should error")
	}
}

func TestGet_NotFound(t *testing.T) {
	reg := &Registry{
		providers: make(map[string]Provider),
	}

	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Error("Get() nonexistent should error")
	}
}

func TestList(t *testing.T) {
	reg := &Registry{
		providers: make(map[string]Provider),
	}

	// Register multiple providers
	reg.Register("a", &mockProvider{name: "a"})
	reg.Register("b", &mockProvider{name: "b"})
	reg.Register("c", &mockProvider{name: "c"})

	names := reg.List()
	if len(names) != 3 {
		t.Errorf("List() returned %d names, want 3", len(names))
	}
}

func TestGetAll(t *testing.T) {
	reg := &Registry{
		providers: make(map[string]Provider),
	}

	reg.Register("test", &mockProvider{name: "test"})

	providers := reg.GetAll()
	if len(providers) != 1 {
		t.Errorf("GetAll() returned %d providers, want 1", len(providers))
	}
}

func TestCheckAvailable(t *testing.T) {
	reg := &Registry{
		providers: make(map[string]Provider),
	}

	reg.Register("available", &mockProvider{name: "available", available: true, message: "ok"})
	reg.Register("unavailable", &mockProvider{name: "unavailable", available: false, message: "not found"})

	results := reg.CheckAvailable()

	if len(results) != 2 {
		t.Errorf("CheckAvailable() returned %d results, want 2", len(results))
	}

	if !results["available"].Available {
		t.Error("available provider should be available")
	}

	if results["unavailable"].Available {
		t.Error("unavailable provider should not be available")
	}
}

func TestCheckExecutable(t *testing.T) {
	// Test with a known executable (should exist on most systems)
	available, message := CheckExecutable("ls")
	if !available {
		t.Errorf("CheckExecutable(\"ls\") should be available, got: %s", message)
	}

	// Test with a non-existent executable
	available, message = CheckExecutable("this-definitely-does-not-exist-12345")
	if available {
		t.Error("CheckExecutable() nonexistent should not be available")
	}
	if message == "" {
		t.Error("CheckExecutable() should return a message for missing executable")
	}
}

func TestItemStatus_Constants(t *testing.T) {
	if ItemSkipped != "skipped" {
		t.Errorf("ItemSkipped = %q, want %q", ItemSkipped, "skipped")
	}
	if ItemApplied != "applied" {
		t.Errorf("ItemApplied = %q, want %q", ItemApplied, "applied")
	}
	if ItemFailed != "failed" {
		t.Errorf("ItemFailed = %q, want %q", ItemFailed, "failed")
	}
}

func TestGroupSkip_InGroupPlan(t *testing.T) {
	plan := GroupPlan{
		Skipped: []GroupSkip{
			{Kind: "HomeBrewPackages", Group: "fonts", Reason: "dependency failed"},
		},
	}
	if len(plan.Skipped) != 1 {
		t.Fatalf("len(Skipped) = %d, want 1", len(plan.Skipped))
	}
	skip := plan.Skipped[0]
	if skip.Kind != "HomeBrewPackages" || skip.Group != "fonts" {
		t.Errorf("skip = %+v", skip)
	}
	if ItemStatus(skip.Reason) != ItemSkipped {
		// Reason is a free-form string; just verify the constant is usable as a value
		_ = ItemSkipped
	}
}

func TestCheckExecutables(t *testing.T) {
	// Test with multiple executables
	available, message := CheckExecutables("ls", "cat")
	if !available {
		t.Errorf("CheckExecutables() should be available, got: %s", message)
	}

	// Test with one missing
	available, message = CheckExecutables("ls", "this-definitely-does-not-exist-12345")
	if available {
		t.Error("CheckExecutables() should not be available when one is missing")
	}
	if message == "" {
		t.Error("CheckExecutables() should return a message")
	}
}
