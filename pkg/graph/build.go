package graph

import (
	"fmt"

	"github.com/wasilak/dotisan/pkg/resource"
)

// Build constructs a DAG from a slice of resources.
// Each resource becomes one node; its Metadata.DependsOn entries are resolved
// to NodeIDs via ResolveAddress (namespace in addresses is stripped).
//
// Implicit edges are injected for brew kinds: every HomeBrewPackages and
// HomeBrewCasks node automatically depends on all HomeBrewTaps nodes, so
// taps are always applied before packages without requiring users to write
// explicit dependsOn in their config.
//
// Returns an error if any address is invalid or a duplicate node ID is detected.
func Build(resources []resource.Resource) (*DAG, error) {
	d := New()

	// Pass 1: collect all HomeBrewTaps NodeIDs for implicit dep injection.
	var tapIDs []NodeID
	for _, res := range resources {
		if res.GetKind() != resource.KindHomeBrewTaps {
			continue
		}
		meta := res.GetMetadata()
		tapIDs = append(tapIDs, ResourceNodeID(res.GetKind(), meta.Name))
	}

	// Pass 2: add all nodes, injecting implicit tap deps for brew package kinds.
	for _, res := range resources {
		meta := res.GetMetadata()
		id := ResourceNodeID(res.GetKind(), meta.Name)

		deps := make([]NodeID, 0, len(meta.DependsOn))
		for _, addr := range meta.DependsOn {
			depID, err := ResolveAddress(addr)
			if err != nil {
				return nil, fmt.Errorf("resource %s: invalid dependency %q: %w", id, addr, err)
			}
			deps = append(deps, depID)
		}

		// Inject implicit HomeBrewPackages/Casks → HomeBrewTaps edges.
		kind := res.GetKind()
		if kind == resource.KindHomeBrewPackages || kind == resource.KindHomeBrewCasks {
			deps = withImplicitTapDeps(deps, tapIDs)
		}

		if err := d.AddNode(&Node{
			ID:        id,
			Kind:      kind,
			DependsOn: deps,
		}); err != nil {
			return nil, fmt.Errorf("failed to add node for resource %s: %w", id, err)
		}
	}

	return d, nil
}

// withImplicitTapDeps appends tap NodeIDs that are not already in deps.
func withImplicitTapDeps(deps []NodeID, tapIDs []NodeID) []NodeID {
	existing := make(map[NodeID]bool, len(deps))
	for _, d := range deps {
		existing[d] = true
	}
	for _, id := range tapIDs {
		if !existing[id] {
			deps = append(deps, id)
		}
	}
	return deps
}
