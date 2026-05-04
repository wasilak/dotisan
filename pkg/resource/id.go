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

	// Handle optional namespace prefix: namespace/Kind/...
	parts := strings.Split(id, "/")

	// If there are 3 or more parts, treat first as namespace
	if len(parts) >= 3 {
		rid.Namespace = parts[0]
		// Rejoin remainder for further parsing
		id = strings.Join(parts[1:], "/")
	}

	// Extract item if present using bracket syntax
	if idx := strings.IndexByte(id, '['); idx >= 0 {
		if !strings.HasSuffix(id, "]") {
			return rid, fmt.Errorf("unclosed '[' in id: %s", id)
		}
		rid.Item = id[idx+1 : len(id)-1]
		id = id[:idx]
	}

	// Remaining format: Kind or Kind/Group
	rem := strings.SplitN(id, "/", 2)
	if len(rem) >= 1 {
		rid.Kind = rem[0]
	}
	if rid.Kind == "" {
		return rid, fmt.Errorf("empty kind in id")
	}
	if len(rem) == 2 {
		rid.Group = rem[1]
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
