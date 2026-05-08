package engine

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/wasilak/nim/pkg/graph"
	"github.com/wasilak/nim/pkg/provider"
	"github.com/wasilak/nim/pkg/resource"
	"github.com/wasilak/nim/pkg/state"
)

// fakeProvider is a controllable provider for testing apply ordering and failures.
type fakeProvider struct {
	name       string
	applyOrder []string        // records "kind/group" in the order Apply was called
	failGroups map[string]bool // groups to fail (key: "kind/group")
}

func newFakeProvider(name string) *fakeProvider {
	return &fakeProvider{name: name, failGroups: make(map[string]bool)}
}

func (f *fakeProvider) Name() string              { return f.name }
func (f *fakeProvider) Available() (bool, string) { return true, "" }
func (f *fakeProvider) Import(_ context.Context, _ string) (provider.ResourceState, error) {
	return provider.ResourceState{}, nil
}
func (f *fakeProvider) Reconcile(_ context.Context, _ []resource.ResourceGroup[any], _ []provider.ResourceState) provider.GroupPlan {
	return provider.GroupPlan{}
}

func (f *fakeProvider) Apply(_ context.Context, plan provider.GroupPlan) error {
	for _, a := range plan.Additions {
		key := fmt.Sprintf("%s/%s", a.Kind, a.Group)
		f.applyOrder = append(f.applyOrder, key)
		if f.failGroups[key] {
			return errors.New("injected failure")
		}
	}
	return nil
}

// buildTestPlanResult constructs a PlanResult with a DAG for two resources where
// "fonts" depends on "tools". Both have additions so they show up as HasChanges.
func buildTestPlanResult(t *testing.T, providerName string) *PlanResult {
	t.Helper()

	toolsID := graph.ResourceNodeID("HomeBrewPackages", "tools")
	fontsID := graph.ResourceNodeID("HomeBrewPackages", "fonts")

	d := graph.New()
	_ = d.AddNode(&graph.Node{ID: toolsID, Kind: "HomeBrewPackages"})
	_ = d.AddNode(&graph.Node{ID: fontsID, Kind: "HomeBrewPackages", DependsOn: []graph.NodeID{toolsID}})

	order, err := d.TopologicalOrder()
	if err != nil {
		t.Fatalf("TopologicalOrder: %v", err)
	}

	plan := provider.GroupPlan{
		Additions: []provider.GroupAddition{
			{Kind: "HomeBrewPackages", Group: "tools", Items: []resource.ResourceItem{{Name: "ripgrep"}}},
			{Kind: "HomeBrewPackages", Group: "fonts", Items: []resource.ResourceItem{{Name: "font-fira-code"}}},
		},
	}

	return &PlanResult{
		HasChanges:      true,
		DependencyOrder: order,
		DAG:             d,
		ProviderPlans:   map[string]provider.GroupPlan{providerName: plan},
	}
}

func TestApply_DependencyOrder(t *testing.T) {
	fake := newFakeProvider("brew")

	eng := &Engine{
		Providers:    map[string]provider.Provider{"brew": fake},
		StateBackend: &noopStateBackend{},
	}

	// Register kinds for provider lookup.
	provider.Register("brew", fake, "HomeBrewPackages")

	result := buildTestPlanResult(t, "brew")
	err := eng.Apply(context.Background(), result, ApplyOptions{Confirm: true})
	if err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}

	if len(fake.applyOrder) != 2 {
		t.Fatalf("expected 2 apply calls, got %d: %v", len(fake.applyOrder), fake.applyOrder)
	}
	if fake.applyOrder[0] != "HomeBrewPackages/tools" {
		t.Errorf("expected tools first, got %q", fake.applyOrder[0])
	}
	if fake.applyOrder[1] != "HomeBrewPackages/fonts" {
		t.Errorf("expected fonts second, got %q", fake.applyOrder[1])
	}
}

func TestApply_SkipPropagation(t *testing.T) {
	fake := newFakeProvider("brew")
	fake.failGroups["HomeBrewPackages/tools"] = true

	eng := &Engine{
		Providers:    map[string]provider.Provider{"brew": fake},
		StateBackend: &noopStateBackend{},
	}

	provider.Register("brew", fake, "HomeBrewPackages")

	result := buildTestPlanResult(t, "brew")
	err := eng.Apply(context.Background(), result, ApplyOptions{Confirm: true})
	// Apply should return an error (tools failed).
	if err == nil {
		t.Fatal("expected error when tools fails, got nil")
	}

	// fonts should have been skipped — only one Apply call (tools), not two.
	if len(fake.applyOrder) != 1 {
		t.Errorf("expected 1 apply call (tools only), got %d: %v", len(fake.applyOrder), fake.applyOrder)
	}

	// The fonts group should appear in ProviderPlans["brew"].Skipped.
	brewPlan := result.ProviderPlans["brew"]
	if len(brewPlan.Skipped) != 1 {
		t.Fatalf("expected 1 skipped group, got %d", len(brewPlan.Skipped))
	}
	if brewPlan.Skipped[0].Group != "fonts" {
		t.Errorf("skipped group = %q, want %q", brewPlan.Skipped[0].Group, "fonts")
	}
}

// buildChainPlanResult constructs a PlanResult with a three-node chain:
// base → middle → top (top depends on middle, middle depends on base).
func buildChainPlanResult(t *testing.T, providerName string) *PlanResult {
	t.Helper()

	baseID := graph.ResourceNodeID("HomeBrewPackages", "base")
	midID := graph.ResourceNodeID("HomeBrewPackages", "middle")
	topID := graph.ResourceNodeID("HomeBrewPackages", "top")

	d := graph.New()
	_ = d.AddNode(&graph.Node{ID: baseID, Kind: "HomeBrewPackages"})
	_ = d.AddNode(&graph.Node{ID: midID, Kind: "HomeBrewPackages", DependsOn: []graph.NodeID{baseID}})
	_ = d.AddNode(&graph.Node{ID: topID, Kind: "HomeBrewPackages", DependsOn: []graph.NodeID{midID}})

	order, err := d.TopologicalOrder()
	if err != nil {
		t.Fatalf("TopologicalOrder: %v", err)
	}

	plan := provider.GroupPlan{
		Additions: []provider.GroupAddition{
			{Kind: "HomeBrewPackages", Group: "base", Items: []resource.ResourceItem{{Name: "gcc"}}},
			{Kind: "HomeBrewPackages", Group: "middle", Items: []resource.ResourceItem{{Name: "cmake"}}},
			{Kind: "HomeBrewPackages", Group: "top", Items: []resource.ResourceItem{{Name: "llvm"}}},
		},
	}

	return &PlanResult{
		HasChanges:      true,
		DependencyOrder: order,
		DAG:             d,
		ProviderPlans:   map[string]provider.GroupPlan{providerName: plan},
	}
}

func TestApply_MultiLevelSkipPropagation(t *testing.T) {
	fake := newFakeProvider("brew")
	fake.failGroups["HomeBrewPackages/base"] = true

	eng := &Engine{
		Providers:    map[string]provider.Provider{"brew": fake},
		StateBackend: &noopStateBackend{},
	}

	provider.Register("brew", fake, "HomeBrewPackages")

	result := buildChainPlanResult(t, "brew")
	err := eng.Apply(context.Background(), result, ApplyOptions{Confirm: true})
	if err == nil {
		t.Fatal("expected error when base fails, got nil")
	}

	// Only base should have been attempted — middle and top both skipped.
	if len(fake.applyOrder) != 1 {
		t.Errorf("expected 1 apply call (base only), got %d: %v", len(fake.applyOrder), fake.applyOrder)
	}

	brewPlan := result.ProviderPlans["brew"]
	skippedGroups := make(map[string]bool, len(brewPlan.Skipped))
	for _, s := range brewPlan.Skipped {
		skippedGroups[s.Group] = true
	}

	if !skippedGroups["middle"] {
		t.Error("middle should be in Skipped")
	}
	if !skippedGroups["top"] {
		t.Error("top should be in Skipped")
	}
}

// noopStateBackend satisfies state.StateBackend without touching the filesystem.
type noopStateBackend struct{}

func (n *noopStateBackend) Load(_ context.Context) (*state.State, error) {
	return state.NewState(), nil
}
func (n *noopStateBackend) Save(_ context.Context, _ *state.State) error { return nil }
