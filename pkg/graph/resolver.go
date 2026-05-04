package graph

import (
	"fmt"
	"strings"

	"github.com/wasilak/dotisan/pkg/resource"
)

// ResourceNodeID builds a NodeID for a resource group (no item).
// Format: "namespace/Kind/Group"
func ResourceNodeID(namespace, kind, group string) NodeID {
	return NodeID(fmt.Sprintf("%s/%s/%s", namespace, kind, group))
}

// ItemNodeID builds a NodeID for a specific item within a resource group.
// Format: "namespace/Kind/Group[item]"
func ItemNodeID(namespace, kind, group, item string) NodeID {
	return NodeID(fmt.Sprintf("%s/%s/%s[%s]", namespace, kind, group, item))
}

// ResolveAddress resolves a raw dependency address string to a canonical NodeID.
// The address may be in any of the forms accepted by resource.ParseResourceID.
// defaultNamespace is used when the address contains no namespace component.
func ResolveAddress(addr, defaultNamespace string) (NodeID, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", fmt.Errorf("empty address")
	}

	rid, err := resource.ParseResourceID(addr)
	if err != nil {
		return "", fmt.Errorf("invalid address %q: %w", addr, err)
	}

	ns := rid.Namespace
	if ns == "" {
		ns = defaultNamespace
	}

	if rid.Item != "" {
		return ItemNodeID(ns, rid.Kind, rid.Group, rid.Item), nil
	}
	return ResourceNodeID(ns, rid.Kind, rid.Group), nil
}
