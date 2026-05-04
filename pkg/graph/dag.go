package graph

import (
	"fmt"
	"strings"
)

// DAG is a directed acyclic graph of resource nodes.
type DAG struct {
	nodes map[NodeID]*Node
}

// New returns an empty DAG.
func New() *DAG {
	return &DAG{nodes: make(map[NodeID]*Node)}
}

// AddNode inserts a node. Returns an error if the ID is already registered.
func (d *DAG) AddNode(n *Node) error {
	if _, exists := d.nodes[n.ID]; exists {
		return fmt.Errorf("duplicate node: %s", n.ID)
	}
	d.nodes[n.ID] = n
	return nil
}

// DependenciesOf returns the direct dependencies of the node with the given ID.
// Returns nil if the node does not exist.
func (d *DAG) DependenciesOf(id NodeID) []NodeID {
	if n, ok := d.nodes[id]; ok {
		return n.DependsOn
	}
	return nil
}

// Nodes returns all nodes keyed by ID (read-only copy of the map).
func (d *DAG) Nodes() map[NodeID]*Node {
	out := make(map[NodeID]*Node, len(d.nodes))
	for k, v := range d.nodes {
		out[k] = v
	}
	return out
}

// Validate checks that:
//   - every dependency in DependsOn references a known node
//   - the graph contains no cycles
func (d *DAG) Validate() error {
	for id, node := range d.nodes {
		for _, dep := range node.DependsOn {
			if _, ok := d.nodes[dep]; !ok {
				return fmt.Errorf("node %s: unknown dependency %s", id, dep)
			}
		}
	}
	return d.detectCycles()
}

// TopologicalOrder returns nodes in an order where each node appears after
// all its dependencies. Uses Kahn's algorithm.
// Returns an error if the graph has a cycle or an unknown dependency.
func (d *DAG) TopologicalOrder() ([]NodeID, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}

	// Build in-degree map and adjacency list.
	inDegree := make(map[NodeID]int, len(d.nodes))
	dependents := make(map[NodeID][]NodeID, len(d.nodes)) // dep → nodes that depend on dep

	for id := range d.nodes {
		inDegree[id] = 0
	}
	for id, node := range d.nodes {
		for _, dep := range node.DependsOn {
			dependents[dep] = append(dependents[dep], id)
			inDegree[id]++
		}
	}

	// Seed queue with zero-in-degree nodes.
	queue := make([]NodeID, 0, len(d.nodes))
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	result := make([]NodeID, 0, len(d.nodes))
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		result = append(result, cur)

		for _, dependent := range dependents[cur] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(result) != len(d.nodes) {
		return nil, fmt.Errorf("cycle detected: topological sort incomplete")
	}
	return result, nil
}

// detectCycles uses DFS with three-colour marking to find cycles.
// On detection it returns an error that includes the full cycle path.
func (d *DAG) detectCycles() error {
	const (
		unvisited = 0
		visiting  = 1
		done      = 2
	)
	state := make(map[NodeID]int, len(d.nodes))

	var visit func(id NodeID, path []NodeID) error
	visit = func(id NodeID, path []NodeID) error {
		switch state[id] {
		case visiting:
			// Find where the cycle starts in the current path.
			cycleStart := 0
			for i, p := range path {
				if p == id {
					cycleStart = i
					break
				}
			}
			cycle := append(path[cycleStart:], id)
			parts := make([]string, len(cycle))
			for i, n := range cycle {
				parts[i] = string(n)
			}
			return fmt.Errorf("cycle detected: %s", strings.Join(parts, " → "))
		case done:
			return nil
		}
		state[id] = visiting
		for _, dep := range d.nodes[id].DependsOn {
			if err := visit(dep, append(path, id)); err != nil {
				return err
			}
		}
		state[id] = done
		return nil
	}

	for id := range d.nodes {
		if state[id] == unvisited {
			if err := visit(id, nil); err != nil {
				return err
			}
		}
	}
	return nil
}
