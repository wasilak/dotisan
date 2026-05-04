package graph

import (
	"strings"
	"testing"
)

func TestDAG_TopologicalOrder_Linear(t *testing.T) {
	// A → B → C  (A depends on B, B depends on C)
	d := New()
	_ = d.AddNode(&Node{ID: "C", Kind: "k"})
	_ = d.AddNode(&Node{ID: "B", Kind: "k", DependsOn: []NodeID{"C"}})
	_ = d.AddNode(&Node{ID: "A", Kind: "k", DependsOn: []NodeID{"B"}})

	order, err := d.TopologicalOrder()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 3 {
		t.Fatalf("len(order) = %d, want 3", len(order))
	}
	pos := make(map[NodeID]int, 3)
	for i, id := range order {
		pos[id] = i
	}
	if pos["C"] >= pos["B"] || pos["B"] >= pos["A"] {
		t.Errorf("wrong order: C=%d B=%d A=%d", pos["C"], pos["B"], pos["A"])
	}
}

func TestDAG_TopologicalOrder_Diamond(t *testing.T) {
	//      A
	//     / \
	//    B   C
	//     \ /
	//      D
	d := New()
	_ = d.AddNode(&Node{ID: "D", Kind: "k"})
	_ = d.AddNode(&Node{ID: "B", Kind: "k", DependsOn: []NodeID{"D"}})
	_ = d.AddNode(&Node{ID: "C", Kind: "k", DependsOn: []NodeID{"D"}})
	_ = d.AddNode(&Node{ID: "A", Kind: "k", DependsOn: []NodeID{"B", "C"}})

	order, err := d.TopologicalOrder()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pos := make(map[NodeID]int, 4)
	for i, id := range order {
		pos[id] = i
	}
	if pos["D"] >= pos["B"] || pos["D"] >= pos["C"] {
		t.Errorf("D must come before B and C: %v", pos)
	}
	if pos["B"] >= pos["A"] || pos["C"] >= pos["A"] {
		t.Errorf("B and C must come before A: %v", pos)
	}
}

func TestDAG_Validate_Cycle(t *testing.T) {
	// A → B → A (cycle)
	d := New()
	_ = d.AddNode(&Node{ID: "A", Kind: "k", DependsOn: []NodeID{"B"}})
	_ = d.AddNode(&Node{ID: "B", Kind: "k", DependsOn: []NodeID{"A"}})

	err := d.Validate()
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "A") || !strings.Contains(msg, "B") {
		t.Errorf("cycle error should mention both nodes, got: %q", msg)
	}
	if !strings.Contains(msg, "→") {
		t.Errorf("cycle error should show path with →, got: %q", msg)
	}
}

func TestDAG_Validate_TransitiveCycle(t *testing.T) {
	// A → B → C → A (transitive cycle)
	d := New()
	_ = d.AddNode(&Node{ID: "A", Kind: "k", DependsOn: []NodeID{"B"}})
	_ = d.AddNode(&Node{ID: "B", Kind: "k", DependsOn: []NodeID{"C"}})
	_ = d.AddNode(&Node{ID: "C", Kind: "k", DependsOn: []NodeID{"A"}})

	err := d.Validate()
	if err == nil {
		t.Fatal("expected transitive cycle error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "→") {
		t.Errorf("cycle error should show path, got: %q", msg)
	}
}

func TestDAG_Validate_SelfLoop(t *testing.T) {
	d := New()
	_ = d.AddNode(&Node{ID: "A", Kind: "k", DependsOn: []NodeID{"A"}})

	if err := d.Validate(); err == nil {
		t.Fatal("expected self-loop error, got nil")
	}
}

func TestDAG_Validate_UnknownDependency(t *testing.T) {
	d := New()
	_ = d.AddNode(&Node{ID: "A", Kind: "k", DependsOn: []NodeID{"missing"}})

	if err := d.Validate(); err == nil {
		t.Fatal("expected unknown-dependency error, got nil")
	}
}

func TestDAG_AddNode_Duplicate(t *testing.T) {
	d := New()
	_ = d.AddNode(&Node{ID: "A", Kind: "k"})
	if err := d.AddNode(&Node{ID: "A", Kind: "k"}); err == nil {
		t.Fatal("expected duplicate-node error, got nil")
	}
}

func TestDAG_TopologicalOrder_NoDeps(t *testing.T) {
	d := New()
	_ = d.AddNode(&Node{ID: "X", Kind: "k"})
	_ = d.AddNode(&Node{ID: "Y", Kind: "k"})

	order, err := d.TopologicalOrder()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 2 {
		t.Fatalf("len(order) = %d, want 2", len(order))
	}
}

func TestDAG_Empty(t *testing.T) {
	d := New()
	if err := d.Validate(); err != nil {
		t.Fatalf("empty DAG Validate() error: %v", err)
	}
	order, err := d.TopologicalOrder()
	if err != nil {
		t.Fatalf("empty DAG TopologicalOrder() error: %v", err)
	}
	if len(order) != 0 {
		t.Fatalf("empty DAG order len = %d, want 0", len(order))
	}
}
