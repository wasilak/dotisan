package graph

import (
	"fmt"
	"strings"

	"github.com/wasilak/nim/pkg/resource"
)

// ResourceNodeID builds a NodeID for a resource group (no item).
// Format: "Kind/Group"
func ResourceNodeID(kind, group string) NodeID {
	return NodeID(fmt.Sprintf("%s/%s", kind, group))
}

// ItemNodeID builds a NodeID for a specific item within a resource group.
// Format: "Kind/Group[item]"
func ItemNodeID(kind, group, item string) NodeID {
	return NodeID(fmt.Sprintf("%s/%s[%s]", kind, group, item))
}

// ResolveAddress resolves a raw dependency address string to a canonical NodeID.
// The address may be in any of the forms accepted by resource.ParseResourceID.
// Namespace is accepted in addresses for backwards compatibility but stripped —
// all resources share a single namespace in the dependency graph.
func ResolveAddress(addr string) (NodeID, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", fmt.Errorf("empty address")
	}

	rid, err := resource.ParseResourceID(addr)
	if err != nil {
		return "", fmt.Errorf("invalid address %q: %w", addr, err)
	}

	if rid.Item != "" {
		return ItemNodeID(rid.Kind, rid.Group, rid.Item), nil
	}
	return ResourceNodeID(rid.Kind, rid.Group), nil
}
