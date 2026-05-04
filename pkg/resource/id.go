package resource

import (
	"fmt"
	"strings"
)

// ResourceID represents a parsed resource identifier.
// Expected forms:
//
//	Kind
//	Kind/Group
//	Kind/Group[Item]
//
// Optionally a namespace may be included as namespace/Kind/Group[Item]
type ResourceID struct {
	Namespace string
	Kind      string
	Group     string
	Item      string
}

// ParseResourceID parses an ID in the canonical forms described above.
// Returns an error for invalid formats. Parsing is strict and deterministic.
func ParseResourceID(id string) (ResourceID, error) {
	var rid ResourceID
	id = strings.TrimSpace(id)
	if id == "" {
		return rid, fmt.Errorf("empty id")
	}

	// Extract the item part FIRST so that slashes inside [...] (e.g. tap names
	// like "stigoleg/homebrew-tap") don't interfere with path splitting.
	if idx := strings.IndexByte(id, '['); idx >= 0 {
		if !strings.HasSuffix(id, "]") {
			return rid, fmt.Errorf("unclosed '[' in id: %s", id)
		}
		rid.Item = id[idx+1 : len(id)-1]
		id = id[:idx]
	}

	// Parse the path prefix: Kind | Kind/Group | namespace/Kind/Group
	parts := strings.SplitN(id, "/", 3)
	switch len(parts) {
	case 1:
		rid.Kind = parts[0]
	case 2:
		rid.Kind, rid.Group = parts[0], parts[1]
	case 3:
		rid.Namespace, rid.Kind, rid.Group = parts[0], parts[1], parts[2]
	}

	if rid.Kind == "" {
		return rid, fmt.Errorf("empty kind in id")
	}

	return rid, nil
}

// MustParseResourceID is like ParseResourceID but panics on error.
func MustParseResourceID(id string) ResourceID {
	rid, err := ParseResourceID(id)
	if err != nil {
		panic(err)
	}
	return rid
}
