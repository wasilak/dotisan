// Package graph provides a dependency graph (DAG) for dotisan resources.
package graph

// NodeID is the unique identifier for a node in the dependency graph.
// Format: "Kind/Group" or "Kind/Group[Item]"
type NodeID string

// Node represents a single vertex in the dependency graph.
// It may represent a whole resource group or a single item within one.
type Node struct {
	ID        NodeID
	Kind      string
	DependsOn []NodeID
}
