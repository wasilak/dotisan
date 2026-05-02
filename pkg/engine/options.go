package engine

import (
	"strings"
)

// PlanOptions contains options for the Plan operation.
type PlanOptions struct {
	Targets []string
}

// TargetMatch represents a parsed target expression.
type TargetMatch struct {
	Kind  string
	Group string
	Item  string
}

// ParseTargets converts raw target strings into TargetMatch structs.
// Supports target forms: Kind, Kind/Group, Kind/Group[Item]
func ParseTargets(targets []string) []TargetMatch {
	var out []TargetMatch
	for _, t := range targets {
		t = strings.ReplaceAll(t, "\\", "/")
		t = strings.TrimSpace(t)

		tm := TargetMatch{}
		// Handle bracketed item notation: Kind/Group[Item]
		bracketStart := strings.LastIndex(t, "[")
		bracketEnd := strings.LastIndex(t, "]")
		if bracketStart != -1 && bracketEnd != -1 && bracketEnd == len(t)-1 && bracketStart < bracketEnd {
			prefix := t[:bracketStart]
			tm.Item = t[bracketStart+1 : bracketEnd]
			parts := strings.SplitN(prefix, "/", 2)
			if len(parts) >= 1 {
				tm.Kind = parts[0]
			}
			if len(parts) == 2 {
				tm.Group = parts[1]
			}
		} else {
			// No brackets - support legacy Kind/Group/Item or shorter
			parts := strings.SplitN(t, "/", 3)
			if len(parts) >= 1 {
				tm.Kind = parts[0]
			}
			if len(parts) >= 2 {
				tm.Group = parts[1]
			}
			if len(parts) == 3 {
				tm.Item = parts[2]
			}
		}
		out = append(out, tm)
	}
	return out
}


// Matches returns true if the given kind/group/item matches this target.
func (t TargetMatch) Matches(kind, group, item string) bool {
	if t.Kind != "" {
		if !strings.EqualFold(t.Kind, kind) {
			return false
		}
	}
	if t.Group != "" && !strings.EqualFold(t.Group, group) {
		return false
	}
	if t.Item != "" && !strings.EqualFold(t.Item, item) {
		return false
	}
	return true
}

// Note: no alias normalization is performed. Users must specify canonical
// Kind names (case-insensitive). Aliases like "npm" are not supported.
